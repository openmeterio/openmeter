package service

import (
	"context"
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargestatemachine "github.com/openmeterio/openmeter/openmeter/billing/charges/statemachine"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
	usagebasedrun "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/run"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestCreditsOnlyPeriodPatchIsConfiguredForPatchableStates(t *testing.T) {
	for _, status := range []usagebased.Status{
		usagebased.StatusCreated,
		usagebased.StatusActive,
		usagebased.StatusActiveRealizationStarted,
		usagebased.StatusActiveRealizationWaitingForCollection,
		usagebased.StatusActiveRealizationCompleted,
		usagebased.StatusFinal,
	} {
		t.Run(string(status), func(t *testing.T) {
			// given:
			// - a credit-only usage-based charge state machine in a patchable state
			// when:
			// - shrink and extend patch triggers are checked
			// then:
			// - both period patches are accepted by the state machine
			machine := newCreditsOnlyStateMachineForTest(t, status)

			canFire, err := machine.CanFire(t.Context(), meta.TriggerShrink)
			require.NoError(t, err)
			require.True(t, canFire)

			canFire, err = machine.CanFire(t.Context(), meta.TriggerExtend)
			require.NoError(t, err)
			require.True(t, canFire)
		})
	}
}

func TestCreditsOnlySetOverrideVoidsRealizationsAndReturnsChargeToActive(t *testing.T) {
	// given:
	// - a credit-only usage-based charge with a final realization in progress
	// when:
	// - an API override changes its effective intent
	// then:
	// - the realization is voided, the override becomes effective, and the charge is rerated from active
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	currentRunID := "run-1"
	charge := usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: newUsageBasedChargeTestManagedResource("charge-id"),
			Intent:          newUsageBasedIntentForCreditOnlyTest(t, servicePeriod),
			Status:          usagebased.StatusActiveRealizationWaitingForCollection,
			State: usagebased.State{
				CurrentRealizationRunID: &currentRunID,
				FeatureID:               "feature-id",
				RatingEngine:            usagebased.RatingEngineDelta,
			},
		},
		Realizations: usagebased.RealizationRuns{
			newUsageBasedRunForShrinkTest(currentRunID, usagebased.RealizationRunTypeFinalRealization, servicePeriod.To),
		},
	}
	machine := newCreditsOnlyStateMachineWithChargeForTest(t, charge)

	overrideFields := charge.Intent.GetEffectiveIntent().IntentMutableFields.Clone()
	overrideFields.Name = "overridden usage"
	overrideFields.ServicePeriod.To = servicePeriod.To.AddDate(0, 1, 0)
	overrideFields.FullServicePeriod.To = servicePeriod.To.AddDate(0, 1, 0)
	overrideFields.BillingPeriod.To = servicePeriod.To.AddDate(0, 1, 0)
	overrideFields.InvoiceAt = servicePeriod.To.AddDate(0, 1, 0)

	patch, err := meta.NewPatchSetOverride(usagebased.NewPatchSetOverrideInput{
		ChangeSource:        billing.ChangeSourceAPIRequest,
		IntentMutableFields: overrideFields,
	})
	require.NoError(t, err)

	err = machine.FireAndActivate(t.Context(), patch.Trigger(), patch)
	require.NoError(t, err)

	updated := machine.GetCharge()
	require.Equal(t, usagebased.StatusActive, updated.Status)
	require.Nil(t, updated.State.CurrentRealizationRunID)
	require.NotNil(t, updated.State.AdvanceAfter)
	require.Equal(t, overrideFields.ServicePeriod.To, *updated.State.AdvanceAfter)
	require.Equal(t, servicePeriod.To, updated.Intent.GetBaseIntent().ServicePeriod.To)
	require.True(t, updated.Intent.HasOverrideLayer())
	require.Equal(t, overrideFields, *updated.Intent.GetOverrideLayerMutableFields())

	run, err := updated.Realizations.GetByID(currentRunID)
	require.NoError(t, err)
	require.NotNil(t, run.DeletedAt)
}

func TestCreditsOnlySetOverrideUpdatesExistingOverrideWithoutStacking(t *testing.T) {
	// given:
	// - a created credit-only charge with an existing manual override
	// when:
	// - the API sets another override snapshot
	// then:
	// - the existing override is replaced while the base intent and created schedule remain intact
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	baseIntent := newUsageBasedIntentForCreditOnlyTest(t, servicePeriod).GetBaseIntent()
	existingOverride := baseIntent.IntentMutableFields.Clone()
	existingOverride.Name = "first override"
	charge := usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: newUsageBasedChargeTestManagedResource("charge-id"),
			Intent:          usagebased.NewOverridableIntent(baseIntent, &existingOverride),
			Status:          usagebased.StatusCreated,
			State: usagebased.State{
				FeatureID:    "feature-id",
				RatingEngine: usagebased.RatingEngineDelta,
			},
		},
	}
	machine := newCreditsOnlyStateMachineWithChargeForTest(t, charge)

	overrideFields := existingOverride.Clone()
	overrideFields.Name = "second override"
	patch, err := meta.NewPatchSetOverride(usagebased.NewPatchSetOverrideInput{
		ChangeSource:        billing.ChangeSourceAPIRequest,
		IntentMutableFields: overrideFields,
	})
	require.NoError(t, err)

	err = machine.FireAndActivate(t.Context(), patch.Trigger(), patch)
	require.NoError(t, err)

	updated := machine.GetCharge()
	require.Equal(t, usagebased.StatusCreated, updated.Status)
	require.Equal(t, baseIntent.IntentMutableFields, updated.Intent.GetBaseIntent().IntentMutableFields)
	require.Equal(t, overrideFields, *updated.Intent.GetOverrideLayerMutableFields())
	require.NotNil(t, updated.State.AdvanceAfter)
	require.Equal(t, servicePeriod.From, *updated.State.AdvanceAfter)
}

func TestCreditsOnlyClearOverrideVoidsRealizationsAndRestoresBase(t *testing.T) {
	// given:
	// - a credit-only charge with an active override and mutable realization history
	// when:
	// - the override is cleared
	// then:
	// - credits are voided, the source intent becomes effective, and rerating resumes from active
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	baseIntent := newUsageBasedIntentForCreditOnlyTest(t, servicePeriod).GetBaseIntent()
	overrideFields := baseIntent.IntentMutableFields.Clone()
	overrideFields.Name = "overridden usage"
	overrideFields.ServicePeriod.To = servicePeriod.To.AddDate(0, 1, 0)
	overrideFields.FullServicePeriod.To = servicePeriod.To.AddDate(0, 1, 0)
	overrideFields.BillingPeriod.To = servicePeriod.To.AddDate(0, 1, 0)
	overrideFields.InvoiceAt = servicePeriod.To.AddDate(0, 1, 0)
	currentRunID := "run-1"
	charge := usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: newUsageBasedChargeTestManagedResource("charge-id"),
			Intent:          usagebased.NewOverridableIntent(baseIntent, &overrideFields),
			Status:          usagebased.StatusActiveRealizationWaitingForCollection,
			State: usagebased.State{
				CurrentRealizationRunID: &currentRunID,
				FeatureID:               "feature-id",
				RatingEngine:            usagebased.RatingEngineDelta,
			},
		},
		Realizations: usagebased.RealizationRuns{
			newUsageBasedRunForShrinkTest(currentRunID, usagebased.RealizationRunTypeFinalRealization, overrideFields.ServicePeriod.To),
		},
	}
	machine := newCreditsOnlyStateMachineWithChargeForTest(t, charge)

	err := machine.FireAndActivate(t.Context(), meta.TriggerClearOverride, mustNewPatchClearOverride(t))
	require.NoError(t, err)
	require.Equal(t, usagebased.StatusActiveClearOverride, machine.GetCharge().Status)
	require.NoError(t, machine.FireAndActivate(t.Context(), meta.TriggerNext))

	updated := machine.GetCharge()
	require.False(t, updated.Intent.HasOverrideLayer())
	require.Equal(t, baseIntent.IntentMutableFields, updated.Intent.GetEffectiveIntent().IntentMutableFields)
	require.Equal(t, usagebased.StatusActive, updated.Status)
	require.Nil(t, updated.State.CurrentRealizationRunID)
	require.NotNil(t, updated.State.AdvanceAfter)
	require.Equal(t, servicePeriod.To, *updated.State.AdvanceAfter)

	run, err := updated.Realizations.GetByID(currentRunID)
	require.NoError(t, err)
	require.NotNil(t, run.DeletedAt)
}

func TestCreditsOnlyClearDeletionOverrideRestoresLiveBase(t *testing.T) {
	// given:
	// - a deleted credit-only override hiding a live source intent
	// when:
	// - the override is cleared
	// then:
	// - the charge is reactivated from the base lifecycle instead of remaining deleted
	clock.FreezeTime(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	baseIntent := newUsageBasedIntentForCreditOnlyTest(t, servicePeriod).GetBaseIntent()
	overrideFields := baseIntent.IntentMutableFields.Clone()
	deletedAt := clock.Now()
	overrideFields.IntentDeletedAt = &deletedAt
	charge := usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: newUsageBasedChargeTestManagedResource("charge-id"),
			Intent:          usagebased.NewOverridableIntent(baseIntent, &overrideFields),
			Status:          usagebased.StatusDeleted,
			State: usagebased.State{
				FeatureID:    "feature-id",
				RatingEngine: usagebased.RatingEngineDelta,
			},
		},
	}
	machine := newCreditsOnlyStateMachineWithChargeForTest(t, charge)

	err := machine.FireAndActivate(t.Context(), meta.TriggerClearOverride, mustNewPatchClearOverride(t))
	require.NoError(t, err)
	require.Equal(t, usagebased.StatusActiveClearOverride, machine.GetCharge().Status)
	require.NoError(t, machine.FireAndActivate(t.Context(), meta.TriggerNext))

	updated := machine.GetCharge()
	require.False(t, updated.Intent.HasOverrideLayer())
	require.Equal(t, usagebased.StatusActive, updated.Status)
	require.NotNil(t, updated.State.AdvanceAfter)
	require.Equal(t, servicePeriod.To, *updated.State.AdvanceAfter)
}

func TestCreditsOnlyClearOverrideKeepsDeletedStatusWhenBaseIsDeleted(t *testing.T) {
	// given:
	// - a deleted charge whose override is hiding an already-deleted base intent
	// when:
	// - the override is cleared
	// then:
	// - only the layer is removed; the charge remains deleted
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	baseIntent := newUsageBasedIntentForCreditOnlyTest(t, servicePeriod).GetBaseIntent()
	deletedAt := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	baseIntent.IntentDeletedAt = &deletedAt
	overrideFields := baseIntent.IntentMutableFields.Clone()
	overrideFields.IntentDeletedAt = nil
	charge := usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: newUsageBasedChargeTestManagedResource("charge-id"),
			Intent:          usagebased.NewOverridableIntent(baseIntent, &overrideFields),
			Status:          usagebased.StatusDeleted,
			State: usagebased.State{
				FeatureID:    "feature-id",
				RatingEngine: usagebased.RatingEngineDelta,
			},
		},
	}
	machine := newCreditsOnlyStateMachineWithChargeForTest(t, charge)

	err := machine.FireAndActivate(t.Context(), meta.TriggerClearOverride, mustNewPatchClearOverride(t))
	require.NoError(t, err)

	updated := machine.GetCharge()
	require.False(t, updated.Intent.HasOverrideLayer())
	require.Equal(t, usagebased.StatusDeleted, updated.Status)
	require.Equal(t, deletedAt, *updated.Intent.GetDeletedAt())
}

func TestCreditsOnlyClearOverrideTransitionsToDeletedBase(t *testing.T) {
	// given:
	// - an active override hiding a source intent deleted by subscription reconciliation
	// when:
	// - the override is cleared
	// then:
	// - the source deletion becomes effective through the normal deletion lifecycle
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	baseIntent := newUsageBasedIntentForCreditOnlyTest(t, servicePeriod).GetBaseIntent()
	deletedAt := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	baseIntent.IntentDeletedAt = &deletedAt
	overrideFields := baseIntent.IntentMutableFields.Clone()
	overrideFields.IntentDeletedAt = nil
	charge := usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: newUsageBasedChargeTestManagedResource("charge-id"),
			Intent:          usagebased.NewOverridableIntent(baseIntent, &overrideFields),
			Status:          usagebased.StatusActive,
			State: usagebased.State{
				FeatureID:    "feature-id",
				RatingEngine: usagebased.RatingEngineDelta,
			},
		},
	}
	machine := newCreditsOnlyStateMachineWithChargeForTest(t, charge)

	require.NoError(t, machine.FireAndActivate(t.Context(), meta.TriggerClearOverride, mustNewPatchClearOverride(t)))
	require.Equal(t, usagebased.StatusDeletedClearOverride, machine.GetCharge().Status)
	_, err := machine.AdvanceUntilStateStable(t.Context())
	require.NoError(t, err)

	updated := machine.GetCharge()
	require.False(t, updated.Intent.HasOverrideLayer())
	require.Equal(t, usagebased.StatusDeleted, updated.Status)
	require.Equal(t, deletedAt, *updated.Intent.GetDeletedAt())
}

func TestCreditsOnlyPeriodPatchWhileCreatedUpdatesIntentAndKeepsCreatedSchedule(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	for _, tc := range []struct {
		name string
		to   time.Time
		new  func(t *testing.T, to time.Time) meta.Patch
	}{
		{
			name: "shrink",
			to:   time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
			new: func(t *testing.T, to time.Time) meta.Patch {
				return mustNewPatchShrink(t, to)
			},
		},
		{
			name: "extend",
			to:   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			new: func(t *testing.T, to time.Time) meta.Patch {
				return mustNewPatchExtend(t, to)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given:
			// - a created credit-only usage-based charge before realization
			// when:
			// - a period patch is applied
			// then:
			// - the charge stays created and remains scheduled from service-period start
			machine := newCreditsOnlyStateMachineWithChargeForTest(t, usagebased.Charge{
				ChargeBase: usagebased.ChargeBase{
					ManagedResource: newUsageBasedChargeTestManagedResource("charge-id"),
					Intent:          newUsageBasedIntentForCreditOnlyTest(t, servicePeriod),
					Status:          usagebased.StatusCreated,
					State: usagebased.State{
						FeatureID:    "feature-id",
						RatingEngine: usagebased.RatingEngineDelta,
					},
				},
			})

			patch := tc.new(t, tc.to)
			err := machine.FireAndActivate(t.Context(), patch.Trigger(), patch)
			require.NoError(t, err)

			charge := machine.GetCharge()
			require.Equal(t, usagebased.StatusCreated, charge.Status)
			require.Nil(t, charge.State.CurrentRealizationRunID)
			require.NotNil(t, charge.State.AdvanceAfter)
			require.Equal(t, servicePeriod.From, *charge.State.AdvanceAfter)
			require.Equal(t, tc.to, charge.Intent.GetEffectiveServicePeriod().To)
		})
	}
}

func TestCreditsOnlyExtendWhileFinalRealizationInProgressVoidsCurrentRunAndMovesActive(t *testing.T) {
	for _, status := range []usagebased.Status{
		usagebased.StatusActiveRealizationStarted,
		usagebased.StatusActiveRealizationWaitingForCollection,
	} {
		t.Run(string(status), func(t *testing.T) {
			// given:
			// - a credit-only usage-based charge with final realization in progress
			// when:
			// - the service period is extended
			// then:
			// - the current final run is voided and the charge moves back to active
			servicePeriod := timeutil.ClosedPeriod{
				From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			}
			extendedServicePeriodTo := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
			currentRunID := "run-1"

			machine := newCreditsOnlyStateMachineWithChargeForTest(t, usagebased.Charge{
				ChargeBase: usagebased.ChargeBase{
					ManagedResource: newUsageBasedChargeTestManagedResource("charge-id"),
					Intent:          newUsageBasedIntentForCreditOnlyTest(t, servicePeriod),
					Status:          status,
					State: usagebased.State{
						CurrentRealizationRunID: &currentRunID,
						FeatureID:               "feature-id",
						RatingEngine:            usagebased.RatingEngineDelta,
					},
				},
				Realizations: usagebased.RealizationRuns{
					newUsageBasedRunForShrinkTest(currentRunID, usagebased.RealizationRunTypeFinalRealization, servicePeriod.To),
				},
			})

			patch := mustNewPatchExtend(t, extendedServicePeriodTo)
			err := machine.FireAndActivate(t.Context(), patch.Trigger(), patch)
			require.NoError(t, err)

			charge := machine.GetCharge()
			require.Equal(t, usagebased.StatusActive, charge.Status)
			require.Nil(t, charge.State.CurrentRealizationRunID)
			require.NotNil(t, charge.State.AdvanceAfter)
			require.Equal(t, extendedServicePeriodTo, *charge.State.AdvanceAfter)
			require.Equal(t, extendedServicePeriodTo, charge.Intent.GetEffectiveServicePeriod().To)

			run, err := charge.Realizations.GetByID(currentRunID)
			require.NoError(t, err)
			require.NotNil(t, run.DeletedAt)
		})
	}
}

func TestCreditsOnlyShrinkWhileCompletedVoidsRunBeyondNewEndAndMovesActive(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	newServicePeriodTo := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	machine := newCreditsOnlyStateMachineWithChargeForTest(t, usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: newUsageBasedChargeTestManagedResource("charge-id"),
			Intent:          newUsageBasedIntentForCreditOnlyTest(t, servicePeriod),
			Status:          usagebased.StatusActiveRealizationCompleted,
			State: usagebased.State{
				FeatureID:    "feature-id",
				RatingEngine: usagebased.RatingEngineDelta,
			},
		},
		Realizations: usagebased.RealizationRuns{
			newUsageBasedRunForShrinkTest("run-1", usagebased.RealizationRunTypeFinalRealization, servicePeriod.To),
		},
	})

	patch := mustNewPatchShrink(t, newServicePeriodTo)
	err := machine.FireAndActivate(t.Context(), patch.Trigger(), patch)
	require.NoError(t, err)

	charge := machine.GetCharge()
	require.Equal(t, usagebased.StatusActive, charge.Status)
	require.Nil(t, charge.State.CurrentRealizationRunID)
	require.NotNil(t, charge.State.AdvanceAfter)
	require.Equal(t, newServicePeriodTo, *charge.State.AdvanceAfter)
	require.Equal(t, newServicePeriodTo, charge.Intent.GetEffectiveServicePeriod().To)

	run, err := charge.Realizations.GetByID("run-1")
	require.NoError(t, err)
	require.NotNil(t, run.DeletedAt)
}

func TestCreditsOnlyExtendWhileCompletedVoidsRunAndMovesActive(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	extendedServicePeriodTo := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	machine := newCreditsOnlyStateMachineWithChargeForTest(t, usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: newUsageBasedChargeTestManagedResource("charge-id"),
			Intent:          newUsageBasedIntentForCreditOnlyTest(t, servicePeriod),
			Status:          usagebased.StatusActiveRealizationCompleted,
			State: usagebased.State{
				FeatureID:    "feature-id",
				RatingEngine: usagebased.RatingEngineDelta,
			},
		},
		Realizations: usagebased.RealizationRuns{
			newUsageBasedRunForShrinkTest("run-1", usagebased.RealizationRunTypeFinalRealization, servicePeriod.To),
		},
	})

	patch := mustNewPatchExtend(t, extendedServicePeriodTo)
	err := machine.FireAndActivate(t.Context(), patch.Trigger(), patch)
	require.NoError(t, err)

	charge := machine.GetCharge()
	require.Equal(t, usagebased.StatusActive, charge.Status)
	require.Nil(t, charge.State.CurrentRealizationRunID)
	require.NotNil(t, charge.State.AdvanceAfter)
	require.Equal(t, extendedServicePeriodTo, *charge.State.AdvanceAfter)
	require.Equal(t, extendedServicePeriodTo, charge.Intent.GetEffectiveServicePeriod().To)

	run, err := charge.Realizations.GetByID("run-1")
	require.NoError(t, err)
	require.NotNil(t, run.DeletedAt)
}

func newUsageBasedIntentForCreditOnlyTest(t testing.TB, servicePeriod timeutil.ClosedPeriod) usagebased.OverridableIntent {
	t.Helper()

	return usagebased.Intent{
		Intent: meta.Intent{
			ManagedBy:  billing.SubscriptionManagedLine,
			CustomerID: "customer-id",
			Currency:   currenciestestutils.NewFiatCurrency(t, "USD"),
			TaxConfig: productcatalog.TaxCodeConfig{
				TaxCodeID: "tax-code-id",
			},
		},
		IntentMutableFields: usagebased.IntentMutableFields{
			IntentMutableFields: meta.IntentMutableFields{
				Name:              "usage",
				ServicePeriod:     servicePeriod,
				FullServicePeriod: servicePeriod,
				BillingPeriod:     servicePeriod,
			},
			InvoiceAt: servicePeriod.To,
			Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: decimal.NewFromInt(1),
			}),
		},
		SettlementMode: productcatalog.CreditOnlySettlementMode,
		FeatureKey:     "feature-key",
	}.AsOverridableIntent()
}

func newCreditsOnlyStateMachineForTest(t *testing.T, status usagebased.Status) *CreditsOnlyStateMachine {
	t.Helper()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	return newCreditsOnlyStateMachineWithChargeForTest(t, usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: newUsageBasedChargeTestManagedResource("charge-id"),
			Intent:          newUsageBasedIntentForCreditOnlyTest(t, servicePeriod),
			Status:          status,
			State: usagebased.State{
				FeatureID:    "feature-id",
				RatingEngine: usagebased.RatingEngineDelta,
			},
		},
	})
}

func newCreditsOnlyStateMachineWithChargeForTest(t *testing.T, charge usagebased.Charge) *CreditsOnlyStateMachine {
	t.Helper()

	adapter := newCreditsOnlyStateMachineAdapter(charge)
	runService, err := usagebasedrun.New(usagebasedrun.Config{
		Adapter: adapter,
		Rater:   creditsOnlyStateMachineRater{},
		Handler: usagebased.UnimplementedHandler{},
		Lineage: creditsOnlyStateMachineLineage{},
	})
	require.NoError(t, err)
	cur, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
		WithCode(currencyx.Code(currency.USD)).
		Build()
	require.NoError(t, err)

	machine, err := chargestatemachine.New(chargestatemachine.Config[usagebased.Charge, usagebased.ChargeBase, usagebased.Status]{
		Charge: charge,
		Persistence: chargestatemachine.Persistence[usagebased.Charge, usagebased.ChargeBase]{
			UpdateBase: adapter.UpdateCharge,
			Refetch: func(_ context.Context, _ meta.ChargeID) (usagebased.Charge, error) {
				return adapter.charge, nil
			},
		},
	})
	require.NoError(t, err)

	out := &CreditsOnlyStateMachine{
		stateMachine: &stateMachine{
			Machine:            machine,
			Adapter:            adapter,
			Runs:               runService,
			CurrencyCalculator: cur,
		},
	}
	out.configureStates()

	return out
}

func newUsageBasedChargeTestManagedResource(id string) meta.ManagedResource {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	return meta.ManagedResource{
		NamespacedModel: models.NamespacedModel{Namespace: "namespace"},
		ManagedModel: models.ManagedModel{
			CreatedAt: now,
			UpdatedAt: now,
		},
		ID: id,
	}
}

type creditsOnlyStateMachineAdapter struct {
	usagebased.Adapter

	charge usagebased.Charge
	runs   map[string]usagebased.RealizationRunBase
}

func newCreditsOnlyStateMachineAdapter(charge usagebased.Charge) *creditsOnlyStateMachineAdapter {
	runs := make(map[string]usagebased.RealizationRunBase, len(charge.Realizations))
	for _, run := range charge.Realizations {
		runs[run.ID.ID] = run.RealizationRunBase
	}

	return &creditsOnlyStateMachineAdapter{
		charge: charge,
		runs:   runs,
	}
}

func (a *creditsOnlyStateMachineAdapter) UpdateCharge(_ context.Context, base usagebased.ChargeBase) (usagebased.ChargeBase, error) {
	a.charge.ChargeBase = base
	return base, nil
}

func (a *creditsOnlyStateMachineAdapter) DeleteCharge(_ context.Context, charge usagebased.Charge) error {
	a.charge = charge
	return nil
}

func (a *creditsOnlyStateMachineAdapter) CreateChargeOverride(_ context.Context, charge usagebased.ChargeBase, override usagebased.IntentMutableFields) (usagebased.ChargeBase, error) {
	charge.Intent = usagebased.NewOverridableIntent(charge.Intent.GetBaseIntent(), &override)
	return charge, nil
}

func (a *creditsOnlyStateMachineAdapter) DeleteChargeOverride(_ context.Context, charge usagebased.ChargeBase) (usagebased.ChargeBase, error) {
	charge.Intent = charge.Intent.GetBaseIntent().AsOverridableIntent()
	return charge, nil
}

func (a *creditsOnlyStateMachineAdapter) UpdateRealizationRun(_ context.Context, input usagebased.UpdateRealizationRunInput) (usagebased.RealizationRunBase, error) {
	run, ok := a.runs[input.ID.ID]
	if !ok {
		return usagebased.RealizationRunBase{}, nil
	}

	if input.DeletedAt.IsPresent() {
		run.DeletedAt = input.DeletedAt.OrEmpty()
	}

	a.runs[input.ID.ID] = run
	return run, nil
}

type creditsOnlyStateMachineRater struct {
	usagebasedrating.Service
}

type creditsOnlyStateMachineLineage struct {
	lineage.Service
}

func (creditsOnlyStateMachineLineage) LoadActiveSegmentsByRealizationID(_ context.Context, _ string, _ []string) (lineage.ActiveSegmentsByRealizationID, error) {
	return lineage.ActiveSegmentsByRealizationID{}, nil
}
