package service

import (
	"context"
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	chargestatemachine "github.com/openmeterio/openmeter/openmeter/billing/charges/statemachine"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestZeroFiatAmountOverageRunCanResumeCompletionAfterPersistence(t *testing.T) {
	// given: a persisted custom-currency run whose converted fiat overage is zero
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	charge := newUsageBasedCustomCurrencyCreditThenInvoiceChargeForTest(t, servicePeriod)
	run := usagebased.RealizationRun{
		RealizationRunBase: usagebased.RealizationRunBase{
			ID: usagebased.RealizationRunID{
				Namespace: "namespace",
				ID:        "run-id",
			},
			Type:                      usagebased.RealizationRunTypeFinalRealization,
			ServicePeriodTo:           servicePeriod.To,
			NoFiatTransactionRequired: true,
		},
	}
	charge.Status = usagebased.StatusActiveRealizationProcessing
	charge.Realizations = usagebased.RealizationRuns{run}
	currentRunID := run.ID.ID
	charge.State.CurrentRealizationRunID = &currentRunID

	// when: the state machine resumes while the run is still current
	machine := newCreditThenInvoiceStateMachineWithChargeForTest(t, charge)
	canFire, err := machine.CanFire(t.Context(), meta.TriggerNext)

	// then: it can complete the zero-overage transition
	require.NoError(t, err)
	require.True(t, canFire)

	// when: the same persisted run is reloaded after the current-run reference was cleared
	charge.State.CurrentRealizationRunID = nil
	machine = newCreditThenInvoiceStateMachineWithChargeForTest(t, charge)
	canFire, err = machine.CanFire(t.Context(), meta.TriggerNext)

	// then: it remains terminal and settled without requiring an invoice line
	require.NoError(t, err)
	require.False(t, canFire)
	require.True(t, machine.HasTerminalCompletedRealizationWithoutCurrentRun())
	allSettled, err := areAllRealizationRunsSettled(charge)
	require.NoError(t, err)
	require.True(t, allSettled)
}

func TestCalculateFiatOverageForRun(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	charge := newUsageBasedCustomCurrencyCreditThenInvoiceChargeForTest(t, servicePeriod)

	tests := []struct {
		name                      string
		runTotals                 totals.Totals
		noFiatTransactionRequired bool
		conversionFails           bool
		expectFiatOverage         float64
		expectOmitInvoiceLine     bool
	}{
		{
			name: "zero converted overage does not depend on transaction requirement",
			runTotals: totals.Totals{
				Amount:       decimal.NewFromInt(3),
				CreditsTotal: decimal.NewFromInt(3),
			},
			noFiatTransactionRequired: false,
			expectOmitInvoiceLine:     true,
		},
		{
			name: "positive converted overage is retained when no transaction is required",
			runTotals: totals.Totals{
				Amount: decimal.NewFromInt(3),
				Total:  decimal.NewFromInt(3),
			},
			noFiatTransactionRequired: true,
			expectFiatOverage:         6,
		},
		{
			name: "conversion failure is returned",
			runTotals: totals.Totals{
				Amount: decimal.NewFromInt(3),
				Total:  decimal.NewFromInt(3),
			},
			conversionFails: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given: a custom-currency run where converted overage and transaction requirement intentionally differ
			charge := charge
			if test.conversionFails {
				charge.State.ResolvedCostBasis = nil
			}
			run := usagebased.RealizationRun{
				RealizationRunBase: usagebased.RealizationRunBase{
					Totals:                    test.runTotals,
					NoFiatTransactionRequired: test.noFiatTransactionRequired,
				},
			}

			// when: the run's fiat overage and invoice-line omission decision are calculated
			fiatOverage, err := calculateFiatOverageForRun(charge, run)

			// then: conversion errors are returned and successful conversion alone controls omission
			if test.conversionFails {
				require.ErrorContains(t, err, "resolved cost basis is required")
				require.False(t, fiatOverage.ShouldOmitInvoiceLine)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.expectFiatOverage, fiatOverage.FiatOverage.InexactFloat64())
			require.Equal(t, test.expectOmitInvoiceLine, fiatOverage.ShouldOmitInvoiceLine)
		})
	}
}

func TestUnsupportedExtendOperation(t *testing.T) {
	for _, status := range []usagebased.Status{
		usagebased.StatusActiveRealizationIssuing,
		usagebased.StatusActiveRealizationZeroFiatAmountOverageCompleted,
		usagebased.StatusActiveRealizationCompleted,
	} {
		t.Run(string(status), func(t *testing.T) {
			machine := CreditThenInvoiceStateMachine{
				stateMachine: &stateMachine{
					Machine: &chargestatemachine.Machine[usagebased.Charge, usagebased.ChargeBase, usagebased.Status]{
						Charge: usagebased.Charge{
							ChargeBase: usagebased.ChargeBase{
								Status: status,
							},
						},
					},
				},
			}

			err := machine.UnsupportedExtendOperation(t.Context(), meta.PatchExtend{})
			require.Error(t, err)
			require.True(t, models.IsGenericPreConditionFailedError(err))
			require.ErrorContains(t, err, "cannot extend usage-based charge in status "+string(status))
			require.Empty(t, machine.InvoicePatches())
		})
	}
}

func TestCreditThenInvoiceSetOverrideBeforeRealizationUpdatesGatheringLine(t *testing.T) {
	for _, tc := range []struct {
		name            string
		status          usagebased.Status
		expectedAdvance func(timeutil.ClosedPeriod) time.Time
	}{
		{
			name:   "created",
			status: usagebased.StatusCreated,
			expectedAdvance: func(period timeutil.ClosedPeriod) time.Time {
				return period.From
			},
		},
		{
			name:   "active",
			status: usagebased.StatusActive,
			expectedAdvance: func(period timeutil.ClosedPeriod) time.Time {
				return period.To
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given:
			// - a credit-then-invoice usage-based charge before any realization has started
			// when:
			// - the API sets a full override snapshot
			// then:
			// - the source intent remains unchanged and the pending gathering line follows the override
			servicePeriod := timeutil.ClosedPeriod{
				From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			}
			charge := usagebased.Charge{
				ChargeBase: usagebased.ChargeBase{
					ManagedResource: newUsageBasedChargeTestManagedResource("charge-id"),
					Intent:          newUsageBasedIntentForCreditThenInvoiceTest(t, servicePeriod),
					Status:          tc.status,
					State: usagebased.State{
						FeatureID:    "feature-id",
						RatingEngine: usagebased.RatingEngineDelta,
					},
				},
			}
			machine := newCreditThenInvoiceStateMachineWithChargeForTest(t, charge)

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
			require.Equal(t, tc.status, updated.Status)
			require.Equal(t, servicePeriod.To, updated.Intent.GetBaseIntent().ServicePeriod.To)
			require.True(t, updated.Intent.HasOverrideLayer())
			require.Equal(t, overrideFields, *updated.Intent.GetOverrideLayerMutableFields())
			require.NotNil(t, updated.State.AdvanceAfter)
			require.Equal(t, tc.expectedAdvance(overrideFields.ServicePeriod), *updated.State.AdvanceAfter)

			patches := machine.InvoicePatches()
			require.Len(t, patches, 1)
			require.Equal(t, invoiceupdater.PatchOpUpsertGatheringLineByChargeID, patches[0].Op())
			linePatch, err := patches[0].AsUpsertGatheringLineByChargeIDPatch()
			require.NoError(t, err)
			require.Equal(t, overrideFields.ServicePeriod, linePatch.TargetState.ServicePeriod)
			require.Equal(t, overrideFields.InvoiceAt, linePatch.TargetState.InvoiceAt)
		})
	}
}

func TestCreditThenInvoiceSetOverrideRejectsAfterRealizationStarted(t *testing.T) {
	// given:
	// - an active credit-then-invoice charge with a non-voided realization
	// when:
	// - the API sets an override
	// then:
	// - the request is rejected before it changes intent or invoice state
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	charge := usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: newUsageBasedChargeTestManagedResource("charge-id"),
			Intent:          newUsageBasedIntentForCreditThenInvoiceTest(t, servicePeriod),
			Status:          usagebased.StatusActive,
		},
		Realizations: usagebased.RealizationRuns{
			newUsageBasedRunForShrinkTest("run-1", usagebased.RealizationRunTypePartialInvoice, servicePeriod.To),
		},
	}
	machine := newCreditThenInvoiceStateMachineWithChargeForTest(t, charge)
	patch, err := meta.NewPatchSetOverride(usagebased.NewPatchSetOverrideInput{
		ChangeSource:        billing.ChangeSourceAPIRequest,
		IntentMutableFields: charge.Intent.GetEffectiveIntent().IntentMutableFields,
	})
	require.NoError(t, err)

	err = machine.FireAndActivate(t.Context(), patch.Trigger(), patch)
	require.Error(t, err)
	require.True(t, models.IsGenericPreConditionFailedError(err))
	require.ErrorContains(t, err, "after realization has started")

	updated := machine.GetCharge()
	require.False(t, updated.Intent.HasOverrideLayer())
	require.Empty(t, machine.InvoicePatches())
}

func TestCreditThenInvoiceSetOverrideIsRejectedAtRealizationBoundaries(t *testing.T) {
	for _, status := range []usagebased.Status{
		usagebased.StatusActiveRealizationStarted,
		usagebased.StatusActiveRealizationWaitingForCollection,
		usagebased.StatusActiveRealizationProcessing,
		usagebased.StatusActiveRealizationIssuing,
		usagebased.StatusActiveRealizationZeroFiatAmountOverageCompleted,
		usagebased.StatusActiveRealizationCompleted,
		usagebased.StatusActiveAwaitingPaymentSettlement,
		usagebased.StatusFinal,
		usagebased.StatusDeleted,
	} {
		t.Run(string(status), func(t *testing.T) {
			// given:
			// - a credit-then-invoice charge at a realization or terminal boundary
			// when:
			// - an override is requested
			// then:
			// - the state machine returns a precondition failure without emitting invoice patches
			servicePeriod := timeutil.ClosedPeriod{
				From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			}
			charge := usagebased.Charge{
				ChargeBase: usagebased.ChargeBase{
					ManagedResource: newUsageBasedChargeTestManagedResource("charge-id"),
					Intent:          newUsageBasedIntentForCreditThenInvoiceTest(t, servicePeriod),
					Status:          status,
				},
			}
			machine := newCreditThenInvoiceStateMachineWithChargeForTest(t, charge)
			patch, err := meta.NewPatchSetOverride(usagebased.NewPatchSetOverrideInput{
				ChangeSource:        billing.ChangeSourceAPIRequest,
				IntentMutableFields: charge.Intent.GetEffectiveIntent().IntentMutableFields,
			})
			require.NoError(t, err)

			err = machine.FireAndActivate(t.Context(), patch.Trigger(), patch)
			require.Error(t, err)
			require.True(t, models.IsGenericPreConditionFailedError(err))
			require.ErrorContains(t, err, "cannot set override for usage-based charge in status "+string(status))
			require.False(t, machine.GetCharge().Intent.HasOverrideLayer())
			require.Empty(t, machine.InvoicePatches())
		})
	}
}

func TestCreditThenInvoiceClearOverrideBeforeRealizationRestoresGatheringLine(t *testing.T) {
	for _, tc := range []struct {
		name            string
		status          usagebased.Status
		expectedAdvance func(timeutil.ClosedPeriod) time.Time
	}{
		{
			name:   "created",
			status: usagebased.StatusCreated,
			expectedAdvance: func(period timeutil.ClosedPeriod) time.Time {
				return period.From
			},
		},
		{
			name:   "active",
			status: usagebased.StatusActive,
			expectedAdvance: func(period timeutil.ClosedPeriod) time.Time {
				return period.To
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given:
			// - a pre-realization credit-then-invoice charge with a manual override
			// when:
			// - the override is cleared
			// then:
			// - the base intent becomes effective and the pending gathering line is restored from it
			servicePeriod := timeutil.ClosedPeriod{
				From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			}
			baseIntent := newUsageBasedIntentForCreditThenInvoiceTest(t, servicePeriod).GetBaseIntent()
			overrideFields := baseIntent.IntentMutableFields.Clone()
			overrideFields.Name = "overridden usage"
			overrideFields.ServicePeriod.To = servicePeriod.To.AddDate(0, 1, 0)
			overrideFields.FullServicePeriod.To = servicePeriod.To.AddDate(0, 1, 0)
			overrideFields.BillingPeriod.To = servicePeriod.To.AddDate(0, 1, 0)
			overrideFields.InvoiceAt = servicePeriod.To.AddDate(0, 1, 0)
			charge := usagebased.Charge{
				ChargeBase: usagebased.ChargeBase{
					ManagedResource: newUsageBasedChargeTestManagedResource("charge-id"),
					Intent:          usagebased.NewOverridableIntent(baseIntent, &overrideFields),
					Status:          tc.status,
					State: usagebased.State{
						FeatureID:    "feature-id",
						RatingEngine: usagebased.RatingEngineDelta,
					},
				},
			}
			machine := newCreditThenInvoiceStateMachineWithChargeForTest(t, charge)

			err := machine.FireAndActivate(t.Context(), meta.TriggerClearOverride, mustNewPatchClearOverride(t))
			require.NoError(t, err)

			updated := machine.GetCharge()
			require.Equal(t, tc.status, updated.Status)
			require.False(t, updated.Intent.HasOverrideLayer())
			require.Equal(t, baseIntent.IntentMutableFields, updated.Intent.GetEffectiveIntent().IntentMutableFields)
			require.NotNil(t, updated.State.AdvanceAfter)
			require.Equal(t, tc.expectedAdvance(servicePeriod), *updated.State.AdvanceAfter)

			patches := machine.InvoicePatches()
			require.Len(t, patches, 1)
			linePatch, err := patches[0].AsUpsertGatheringLineByChargeIDPatch()
			require.NoError(t, err)
			require.Equal(t, servicePeriod, linePatch.TargetState.ServicePeriod)
			require.Equal(t, servicePeriod.To, linePatch.TargetState.InvoiceAt)
		})
	}
}

func TestCreditThenInvoiceClearOverrideTransitionsToDeletedBase(t *testing.T) {
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
	baseIntent := newUsageBasedIntentForCreditThenInvoiceTest(t, servicePeriod).GetBaseIntent()
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
	machine := newCreditThenInvoiceStateMachineWithChargeForTest(t, charge)

	require.NoError(t, machine.FireAndActivate(t.Context(), meta.TriggerClearOverride, mustNewPatchClearOverride(t)))
	require.Equal(t, usagebased.StatusDeletedClearOverride, machine.GetCharge().Status)
	_, err := machine.AdvanceUntilStateStable(t.Context())
	require.NoError(t, err)

	updated := machine.GetCharge()
	require.False(t, updated.Intent.HasOverrideLayer())
	require.Equal(t, usagebased.StatusDeleted, updated.Status)
	require.Equal(t, deletedAt, *updated.Intent.GetDeletedAt())
	require.Len(t, machine.InvoicePatches(), 1)
	require.Equal(t, invoiceupdater.PatchOpDeleteGatheringLineByChargeID, machine.InvoicePatches()[0].Op())
}

func TestCreditThenInvoiceClearOverrideRejectsAfterRealizationStarted(t *testing.T) {
	// given:
	// - a credit-then-invoice charge with an override and a non-voided realization
	// when:
	// - the override is cleared
	// then:
	// - the operation fails without exposing the source intent or changing invoice state
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	baseIntent := newUsageBasedIntentForCreditThenInvoiceTest(t, servicePeriod).GetBaseIntent()
	overrideFields := baseIntent.IntentMutableFields.Clone()
	overrideFields.Name = "overridden usage"
	charge := usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: newUsageBasedChargeTestManagedResource("charge-id"),
			Intent:          usagebased.NewOverridableIntent(baseIntent, &overrideFields),
			Status:          usagebased.StatusActive,
		},
		Realizations: usagebased.RealizationRuns{
			newUsageBasedRunForShrinkTest("run-1", usagebased.RealizationRunTypePartialInvoice, servicePeriod.To),
		},
	}
	machine := newCreditThenInvoiceStateMachineWithChargeForTest(t, charge)

	err := machine.FireAndActivate(t.Context(), meta.TriggerClearOverride, mustNewPatchClearOverride(t))
	require.Error(t, err)
	require.True(t, models.IsGenericPreConditionFailedError(err))
	require.ErrorContains(t, err, "after realization has started")
	require.True(t, machine.GetCharge().Intent.HasOverrideLayer())
	require.Empty(t, machine.InvoicePatches())
}

func TestCreditThenInvoiceClearDeletionOverrideRestoresLiveBase(t *testing.T) {
	// given:
	// - a deleted override hiding a live, pre-realization source intent
	// when:
	// - the override is cleared
	// then:
	// - the charge resumes its source lifecycle and restores the gathering line
	clock.FreezeTime(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	baseIntent := newUsageBasedIntentForCreditThenInvoiceTest(t, servicePeriod).GetBaseIntent()
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
	machine := newCreditThenInvoiceStateMachineWithChargeForTest(t, charge)

	err := machine.FireAndActivate(t.Context(), meta.TriggerClearOverride, mustNewPatchClearOverride(t))
	require.NoError(t, err)
	require.Equal(t, usagebased.StatusActiveClearOverride, machine.GetCharge().Status)
	require.NoError(t, machine.FireAndActivate(t.Context(), meta.TriggerNext))

	updated := machine.GetCharge()
	require.False(t, updated.Intent.HasOverrideLayer())
	require.Equal(t, usagebased.StatusActive, updated.Status)
	require.NotNil(t, updated.State.AdvanceAfter)
	require.Equal(t, servicePeriod.To, *updated.State.AdvanceAfter)
	patches := machine.InvoicePatches()
	require.Len(t, patches, 1)
	require.Equal(t, invoiceupdater.PatchOpUpsertGatheringLineByChargeID, patches[0].Op())
}

func TestUnsupportedExtendOperationIsConfiguredForFinalRealizationBoundary(t *testing.T) {
	for _, status := range []usagebased.Status{
		usagebased.StatusActiveRealizationIssuing,
		usagebased.StatusActiveRealizationZeroFiatAmountOverageCompleted,
		usagebased.StatusActiveRealizationCompleted,
	} {
		t.Run(string(status), func(t *testing.T) {
			machine := newCreditThenInvoiceStateMachineForTest(t, status)
			patch, err := meta.NewPatchExtend(meta.NewPatchExtendInput{
				ChangeSource:           billing.ChangeSourceSystem,
				NewServicePeriodTo:     time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC),
				NewFullServicePeriodTo: time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC),
				NewBillingPeriodTo:     time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC),
				NewInvoiceAt:           time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC),
			})
			require.NoError(t, err)

			canFire, err := machine.CanFire(t.Context(), meta.TriggerExtend)
			require.NoError(t, err)
			require.True(t, canFire)

			err = machine.FireAndActivate(t.Context(), patch.Trigger(), patch)
			require.Error(t, err)
			require.True(t, models.IsGenericPreConditionFailedError(err))
			require.ErrorContains(t, err, "cannot extend usage-based charge in status "+string(status))
			require.Empty(t, machine.InvoicePatches())
			require.Equal(t, status, machine.GetCharge().Status)
		})
	}
}

func TestUnsupportedShrinkOperation(t *testing.T) {
	for _, status := range []usagebased.Status{
		usagebased.StatusActiveRealizationIssuing,
		usagebased.StatusActiveRealizationZeroFiatAmountOverageCompleted,
		usagebased.StatusActiveRealizationCompleted,
		usagebased.StatusDeleted,
	} {
		t.Run(string(status), func(t *testing.T) {
			machine := CreditThenInvoiceStateMachine{
				stateMachine: &stateMachine{
					Machine: &chargestatemachine.Machine[usagebased.Charge, usagebased.ChargeBase, usagebased.Status]{
						Charge: usagebased.Charge{
							ChargeBase: usagebased.ChargeBase{
								Status: status,
							},
						},
					},
				},
			}

			err := machine.UnsupportedShrinkOperation(t.Context(), meta.PatchShrink{})
			require.Error(t, err)
			require.True(t, models.IsGenericPreConditionFailedError(err))
			require.ErrorContains(t, err, "cannot shrink usage-based charge in status "+string(status))
			require.Empty(t, machine.InvoicePatches())
		})
	}
}

func TestUnsupportedShrinkOperationIsConfiguredForImmutableBoundaries(t *testing.T) {
	for _, status := range []usagebased.Status{
		usagebased.StatusActiveRealizationIssuing,
		usagebased.StatusActiveRealizationZeroFiatAmountOverageCompleted,
		usagebased.StatusActiveRealizationCompleted,
		usagebased.StatusDeleted,
	} {
		t.Run(string(status), func(t *testing.T) {
			machine := newCreditThenInvoiceStateMachineForTest(t, status)
			patch, err := meta.NewPatchShrink(meta.NewPatchShrinkInput{
				ChangeSource:           billing.ChangeSourceSystem,
				NewServicePeriodTo:     time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
				NewFullServicePeriodTo: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
				NewBillingPeriodTo:     time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
				NewInvoiceAt:           time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
			})
			require.NoError(t, err)

			canFire, err := machine.CanFire(t.Context(), meta.TriggerShrink)
			require.NoError(t, err)
			require.True(t, canFire)

			err = machine.FireAndActivate(t.Context(), patch.Trigger(), patch)
			require.Error(t, err)
			require.True(t, models.IsGenericPreConditionFailedError(err))
			require.ErrorContains(t, err, "cannot shrink usage-based charge in status "+string(status))
			require.Empty(t, machine.InvoicePatches())
			require.Equal(t, status, machine.GetCharge().Status)
		})
	}
}

func TestShrinkChargeKeepsCurrentRunStateWhenCurrentRunSurvivesShrink(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	}
	currentRunID := "run-1"
	currentAdvanceAfter := time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC)
	currentRunEnd := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	newServicePeriodTo := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	machine := newCreditThenInvoiceStateMachineWithChargeForTest(t, usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "namespace"},
				ID:              "charge-id",
			},
			Intent: newUsageBasedIntentForCreditThenInvoiceTest(t, servicePeriod),
			Status: usagebased.StatusActiveRealizationProcessing,
			State: usagebased.State{
				CurrentRealizationRunID: &currentRunID,
				AdvanceAfter:            &currentAdvanceAfter,
			},
		},
		Realizations: usagebased.RealizationRuns{
			newUsageBasedRunForShrinkTest(currentRunID, usagebased.RealizationRunTypePartialInvoice, currentRunEnd),
		},
	})

	err := machine.ShrinkCharge(t.Context(), mustNewPatchShrink(t, newServicePeriodTo))
	require.NoError(t, err)

	charge := machine.GetCharge()
	require.Equal(t, usagebased.StatusActiveRealizationProcessing, charge.Status)
	require.Equal(t, currentRunID, *charge.State.CurrentRealizationRunID)
	require.Equal(t, currentAdvanceAfter, *charge.State.AdvanceAfter)

	patches := machine.InvoicePatches()
	require.Len(t, patches, 1)
	require.Equal(t, invoiceupdater.PatchOpUpsertGatheringLineByChargeID, patches[0].Op())

	updatePatch, err := patches[0].AsUpsertGatheringLineByChargeIDPatch()
	require.NoError(t, err)
	require.Equal(t, "charge-id", updatePatch.ChargeID)
	require.Equal(t, currentRunEnd, updatePatch.TargetState.ServicePeriod.From)
	require.Equal(t, newServicePeriodTo, updatePatch.TargetState.ServicePeriod.To)
	require.Equal(t, newServicePeriodTo, updatePatch.TargetState.InvoiceAt)
}

func TestExtendChargeDeletesPendingGatheringLineWhenRunsCoverExtendedPeriod(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	extendedServicePeriodTo := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	machine := newCreditThenInvoiceStateMachineWithChargeForTest(t, usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "namespace"},
				ID:              "charge-id",
			},
			Intent: newUsageBasedIntentForCreditThenInvoiceTest(t, servicePeriod),
			Status: usagebased.StatusActive,
		},
		Realizations: usagebased.RealizationRuns{
			newUsageBasedRunForShrinkTest("run-1", usagebased.RealizationRunTypePartialInvoice, extendedServicePeriodTo),
		},
	})

	err := machine.ExtendCharge(t.Context(), mustNewPatchExtend(t, extendedServicePeriodTo))
	require.NoError(t, err)

	patches := machine.InvoicePatches()
	require.Len(t, patches, 1)
	require.Equal(t, invoiceupdater.PatchOpDeleteGatheringLineByChargeID, patches[0].Op())

	deletePatch, err := patches[0].AsDeleteGatheringLineByChargeIDPatch()
	require.NoError(t, err)
	require.Equal(t, "charge-id", deletePatch.ChargeID)
}

func TestShrinkChargeMovesToAwaitingPaymentWhenKeptRunCoversNewEnd(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	}
	newServicePeriodTo := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	currentRunID := "run-1"
	currentAdvanceAfter := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	machine := newCreditThenInvoiceStateMachineWithChargeForTest(t, usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "namespace"},
				ID:              "charge-id",
			},
			Intent: newUsageBasedIntentForCreditThenInvoiceTest(t, servicePeriod),
			Status: usagebased.StatusActive,
			State: usagebased.State{
				AdvanceAfter: &currentAdvanceAfter,
			},
		},
		Realizations: usagebased.RealizationRuns{
			newUsageBasedRunForShrinkTest(currentRunID, usagebased.RealizationRunTypeFinalRealization, newServicePeriodTo),
		},
	})

	err := machine.ShrinkCharge(t.Context(), mustNewPatchShrink(t, newServicePeriodTo))
	require.NoError(t, err)

	charge := machine.GetCharge()
	require.Equal(t, usagebased.StatusActiveAwaitingPaymentSettlement, charge.Status)
	require.Nil(t, charge.State.CurrentRealizationRunID)
	require.Nil(t, charge.State.AdvanceAfter)

	patches := machine.InvoicePatches()
	require.Len(t, patches, 1)
	require.Equal(t, invoiceupdater.PatchOpDeleteGatheringLineByChargeID, patches[0].Op())
}

func TestShrinkChargeMovesToFinalWhenKeptRunCoversNewEndAndSettlementIsComplete(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	}
	newServicePeriodTo := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	currentRunID := "run-1"
	currentAdvanceAfter := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	run := newUsageBasedRunForShrinkTest(currentRunID, usagebased.RealizationRunTypeFinalRealization, newServicePeriodTo)
	run.NoFiatTransactionRequired = true
	run.InvoiceUsage = &invoicedusage.AccruedUsage{
		ServicePeriod: timeutil.ClosedPeriod{
			From: servicePeriod.From,
			To:   newServicePeriodTo,
		},
	}

	machine := newCreditThenInvoiceStateMachineWithChargeForTest(t, usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "namespace"},
				ID:              "charge-id",
			},
			Intent: newUsageBasedIntentForCreditThenInvoiceTest(t, servicePeriod),
			Status: usagebased.StatusActiveAwaitingPaymentSettlement,
			State: usagebased.State{
				AdvanceAfter: &currentAdvanceAfter,
			},
		},
		Realizations: usagebased.RealizationRuns{run},
	})

	err := machine.ShrinkCharge(t.Context(), mustNewPatchShrink(t, newServicePeriodTo))
	require.NoError(t, err)

	charge := machine.GetCharge()
	require.Equal(t, usagebased.StatusFinal, charge.Status)
	require.Nil(t, charge.State.CurrentRealizationRunID)
	require.Nil(t, charge.State.AdvanceAfter)

	patches := machine.InvoicePatches()
	require.Len(t, patches, 1)
	require.Equal(t, invoiceupdater.PatchOpDeleteGatheringLineByChargeID, patches[0].Op())
}

func TestShrinkToRealizedPeriodFinalizesKeptPartialRunAndPreservesChargeState(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	}
	newServicePeriodTo := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	currentRunID := "run-1"
	currentAdvanceAfter := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	charge := usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "namespace"},
				ID:              "charge-id",
			},
			Intent: newUsageBasedIntentForCreditThenInvoiceTest(t, servicePeriod),
			Status: usagebased.StatusActive,
			State: usagebased.State{
				AdvanceAfter: &currentAdvanceAfter,
			},
		},
		Realizations: usagebased.RealizationRuns{
			newUsageBasedRunForShrinkTest(currentRunID, usagebased.RealizationRunTypePartialInvoice, newServicePeriodTo),
		},
	}
	machine := newCreditThenInvoiceStateMachineWithChargeForTest(t, charge)

	err := machine.ShrinkToRealizedPeriod(t.Context(), mustNewPatchShrinkToRealizedPeriod(t, newServicePeriodTo))
	require.NoError(t, err)

	charge = machine.GetCharge()
	require.Equal(t, usagebased.StatusActive, charge.Status)
	require.Nil(t, charge.State.CurrentRealizationRunID)
	require.Equal(t, currentAdvanceAfter, *charge.State.AdvanceAfter)
	require.Equal(t, servicePeriod.To, charge.Intent.GetBaseIntent().ServicePeriod.To)
	require.True(t, charge.Intent.HasOverrideLayer())
	require.Equal(t, newServicePeriodTo, charge.Intent.GetEffectiveServicePeriod().To)
	require.Equal(t, servicePeriod.To, charge.Intent.GetEffectiveIntent().FullServicePeriod.To)
	require.Equal(t, servicePeriod.To, charge.Intent.GetEffectiveIntent().BillingPeriod.To)
	require.Equal(t, servicePeriod.To, charge.Intent.GetEffectiveInvoiceAt())

	run, err := charge.Realizations.GetByID(currentRunID)
	require.NoError(t, err)
	require.Equal(t, usagebased.RealizationRunTypeFinalRealization, run.Type)
	require.Equal(t, usagebased.RealizationRunTypePartialInvoice, run.InitialType)

	patches := machine.InvoicePatches()
	require.Len(t, patches, 1)
	require.Equal(t, invoiceupdater.PatchOpDeleteGatheringLineByChargeID, patches[0].Op())
}

func TestShrinkToRealizedPeriodRejectsPeriodNotCoveredByLatest(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	}
	firstRunEnd := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	latestRunEnd := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	machine := newCreditThenInvoiceStateMachineWithChargeForTest(t, usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "namespace"},
				ID:              "charge-id",
			},
			Intent: newUsageBasedIntentForCreditThenInvoiceTest(t, servicePeriod),
			Status: usagebased.StatusActive,
		},
		Realizations: usagebased.RealizationRuns{
			newUsageBasedRunForShrinkTest("run-1", usagebased.RealizationRunTypePartialInvoice, firstRunEnd),
			newUsageBasedRunForShrinkTest("run-2", usagebased.RealizationRunTypePartialInvoice, latestRunEnd),
		},
	})

	err := machine.ShrinkToRealizedPeriod(t.Context(), mustNewPatchShrinkToRealizedPeriod(t, firstRunEnd))

	require.ErrorContains(t, err, billing.ErrCannotEditProgressivelyBilledUsageBasedLine.Error())
}

func TestShrinkToRealizedPeriodFinalizesCurrentPartialRunAndPreservesChargeState(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	}
	currentRunID := "run-1"
	currentAdvanceAfter := time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC)
	newServicePeriodTo := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	charge := usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "namespace"},
				ID:              "charge-id",
			},
			Intent: newUsageBasedIntentForCreditThenInvoiceTest(t, servicePeriod),
			Status: usagebased.StatusActiveRealizationProcessing,
			State: usagebased.State{
				CurrentRealizationRunID: &currentRunID,
				AdvanceAfter:            &currentAdvanceAfter,
			},
		},
		Realizations: usagebased.RealizationRuns{
			newUsageBasedRunForShrinkTest(currentRunID, usagebased.RealizationRunTypePartialInvoice, newServicePeriodTo),
		},
	}
	machine := newCreditThenInvoiceStateMachineWithChargeForTest(t, charge)

	err := machine.ShrinkToRealizedPeriod(t.Context(), mustNewPatchShrinkToRealizedPeriod(t, newServicePeriodTo))
	require.NoError(t, err)

	charge = machine.GetCharge()
	require.Equal(t, usagebased.StatusActiveRealizationProcessing, charge.Status)
	require.Equal(t, currentRunID, *charge.State.CurrentRealizationRunID)
	require.Equal(t, currentAdvanceAfter, *charge.State.AdvanceAfter)
	require.True(t, charge.Intent.HasOverrideLayer())
	require.Equal(t, newServicePeriodTo, charge.Intent.GetEffectiveServicePeriod().To)
	require.Equal(t, servicePeriod.To, charge.Intent.GetEffectiveIntent().FullServicePeriod.To)
	require.Equal(t, servicePeriod.To, charge.Intent.GetEffectiveIntent().BillingPeriod.To)
	require.Equal(t, servicePeriod.To, charge.Intent.GetEffectiveInvoiceAt())

	run, err := charge.Realizations.GetByID(currentRunID)
	require.NoError(t, err)
	require.Equal(t, usagebased.RealizationRunTypeFinalRealization, run.Type)
	require.Equal(t, usagebased.RealizationRunTypePartialInvoice, run.InitialType)

	patches := machine.InvoicePatches()
	require.Len(t, patches, 1)
	require.Equal(t, invoiceupdater.PatchOpDeleteGatheringLineByChargeID, patches[0].Op())
}

func newCreditThenInvoiceStateMachineForTest(t *testing.T, status usagebased.Status) *CreditThenInvoiceStateMachine {
	t.Helper()

	charge := usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{
					Namespace: "namespace",
				},
				ID: "charge-id",
			},
			Status: status,
		},
	}

	machine, err := chargestatemachine.New(chargestatemachine.Config[usagebased.Charge, usagebased.ChargeBase, usagebased.Status]{
		Charge: charge,
		Persistence: chargestatemachine.Persistence[usagebased.Charge, usagebased.ChargeBase]{
			UpdateBase: func(_ context.Context, base usagebased.ChargeBase) (usagebased.ChargeBase, error) {
				return base, nil
			},
			Refetch: func(_ context.Context, _ meta.ChargeID) (usagebased.Charge, error) {
				return charge, nil
			},
		},
	})
	require.NoError(t, err)

	out := &CreditThenInvoiceStateMachine{
		stateMachine: &stateMachine{
			Machine: machine,
		},
	}
	out.configureStates()

	return out
}

func newUsageBasedIntentForCreditThenInvoiceTest(t testing.TB, servicePeriod timeutil.ClosedPeriod) usagebased.OverridableIntent {
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
		SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
		FeatureKey:     "feature-key",
	}.AsOverridableIntent()
}

func newUsageBasedCustomCurrencyCreditThenInvoiceChargeForTest(t testing.TB, servicePeriod timeutil.ClosedPeriod) usagebased.Charge {
	t.Helper()

	customCurrency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode("TOKENS").
		WithName("Tokens").
		WithPrecision(4).
		Build()
	require.NoError(t, err)

	fiatCurrency, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)
	costBasisIntent := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: fiatCurrency,
		Rate:         decimal.NewFromInt(2),
	})

	intent := newUsageBasedIntentForCreditThenInvoiceTest(t, servicePeriod).GetBaseIntent()
	intent.Intent.Currency = currencies.Currency{
		NamespacedID: models.NamespacedID{
			Namespace: "namespace",
			ID:        "currency-id",
		},
		Currency: customCurrency,
	}
	intent.CostBasis = &costBasisIntent

	return usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "namespace"},
				ID:              "charge-id",
			},
			Intent: usagebased.NewOverridableIntent(intent, nil),
			State: usagebased.State{
				ResolvedCostBasis: &costbasis.State{
					CostBasis:  decimal.NewFromInt(2),
					ResolvedAt: servicePeriod.From,
				},
			},
		},
	}
}

func newCreditThenInvoiceStateMachineWithChargeForTest(t *testing.T, charge usagebased.Charge) *CreditThenInvoiceStateMachine {
	t.Helper()

	adapter := newCreditThenInvoiceStateMachineAdapter(charge)
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

	out := &CreditThenInvoiceStateMachine{
		stateMachine: &stateMachine{
			Machine: machine,
			Adapter: adapter,
		},
	}
	out.configureStates()

	return out
}

type creditThenInvoiceStateMachineAdapter struct {
	usagebased.Adapter

	charge usagebased.Charge
	runs   map[string]usagebased.RealizationRunBase
}

func newCreditThenInvoiceStateMachineAdapter(charge usagebased.Charge) *creditThenInvoiceStateMachineAdapter {
	runs := make(map[string]usagebased.RealizationRunBase, len(charge.Realizations))
	for _, run := range charge.Realizations {
		runs[run.ID.ID] = run.RealizationRunBase
	}

	return &creditThenInvoiceStateMachineAdapter{
		charge: charge,
		runs:   runs,
	}
}

func (a *creditThenInvoiceStateMachineAdapter) UpdateCharge(_ context.Context, base usagebased.ChargeBase) (usagebased.ChargeBase, error) {
	a.charge.ChargeBase = base
	return base, nil
}

func (a *creditThenInvoiceStateMachineAdapter) DeleteCharge(_ context.Context, charge usagebased.Charge) error {
	a.charge = charge
	return nil
}

func (a *creditThenInvoiceStateMachineAdapter) CreateChargeOverride(_ context.Context, charge usagebased.ChargeBase, override usagebased.IntentMutableFields) (usagebased.ChargeBase, error) {
	charge.Intent = usagebased.NewOverridableIntent(charge.Intent.GetBaseIntent(), &override)
	return charge, nil
}

func (a *creditThenInvoiceStateMachineAdapter) DeleteChargeOverride(_ context.Context, charge usagebased.ChargeBase) (usagebased.ChargeBase, error) {
	charge.Intent = charge.Intent.GetBaseIntent().AsOverridableIntent()
	return charge, nil
}

func (a *creditThenInvoiceStateMachineAdapter) UpdateRealizationRun(_ context.Context, input usagebased.UpdateRealizationRunInput) (usagebased.RealizationRunBase, error) {
	run, ok := a.runs[input.ID.ID]
	if !ok {
		return usagebased.RealizationRunBase{}, nil
	}

	if input.Type.IsPresent() {
		run.Type = input.Type.OrEmpty()
	}

	a.runs[input.ID.ID] = run
	return run, nil
}

func newUsageBasedRunForShrinkTest(id string, typ usagebased.RealizationRunType, servicePeriodTo time.Time) usagebased.RealizationRun {
	return usagebased.RealizationRun{
		RealizationRunBase: usagebased.RealizationRunBase{
			ID: usagebased.RealizationRunID{
				Namespace: "namespace",
				ID:        id,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: servicePeriodTo.Add(-time.Hour),
				UpdatedAt: servicePeriodTo.Add(-time.Hour),
			},
			Type:            typ,
			InitialType:     typ,
			FeatureID:       "feature-id",
			StoredAtLT:      servicePeriodTo,
			ServicePeriodTo: servicePeriodTo,
		},
	}
}

func mustNewPatchShrink(t *testing.T, newServicePeriodTo time.Time) meta.PatchShrink {
	t.Helper()

	patch, err := meta.NewPatchShrink(meta.NewPatchShrinkInput{
		ChangeSource:           billing.ChangeSourceSystem,
		NewServicePeriodTo:     newServicePeriodTo,
		NewFullServicePeriodTo: newServicePeriodTo,
		NewBillingPeriodTo:     newServicePeriodTo,
		NewInvoiceAt:           newServicePeriodTo,
	})
	require.NoError(t, err)

	return patch
}

func mustNewPatchClearOverride(t *testing.T) meta.PatchClearOverride {
	t.Helper()

	patch, err := meta.NewPatchClearOverride(meta.NewPatchClearOverrideInput{
		ChangeSource: billing.ChangeSourceAPIRequest,
	})
	require.NoError(t, err)

	return patch
}

func mustNewPatchShrinkToRealizedPeriod(t *testing.T, newServicePeriodTo time.Time) meta.PatchShrinkToRealizedPeriod {
	t.Helper()

	patch, err := meta.NewPatchShrinkToRealizedPeriod(meta.NewPatchShrinkToRealizedPeriodInput{
		ChangeSource:        billing.ChangeSourceAPIRequest,
		NewServicePeriodEnd: newServicePeriodTo,
	})
	require.NoError(t, err)

	return patch
}

func mustNewPatchExtend(t *testing.T, newServicePeriodTo time.Time) meta.PatchExtend {
	t.Helper()

	patch, err := meta.NewPatchExtend(meta.NewPatchExtendInput{
		ChangeSource:           billing.ChangeSourceSystem,
		NewServicePeriodTo:     newServicePeriodTo,
		NewFullServicePeriodTo: newServicePeriodTo,
		NewBillingPeriodTo:     newServicePeriodTo,
		NewInvoiceAt:           newServicePeriodTo,
	})
	require.NoError(t, err)

	return patch
}

func TestStartInvoiceCreatedRunValidatesInput(t *testing.T) {
	var machine CreditThenInvoiceStateMachine

	err := machine.StartInvoiceRun(
		t.Context(),
		invoiceCreatedInput{
			LineID:    "line-1",
			InvoiceID: "invoice-1",
		},
	)

	require.Error(t, err)
	require.ErrorContains(t, err, "validate invoice created input")
	require.ErrorContains(t, err, "service period")
	require.ErrorContains(t, err, "from is required")
	require.ErrorContains(t, err, "to is required")
}

func TestGetInvoiceRealizationRunType(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}

	charge := usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			Intent: newUsageBasedIntentForCreditThenInvoiceTest(t, servicePeriod),
		},
	}

	t.Run("partial invoice period", func(t *testing.T) {
		runType := getInvoiceRealizationRunType(charge, timeutil.ClosedPeriod{
			From: servicePeriod.From,
			To:   time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
		})
		require.Equal(t, usagebased.RealizationRunTypePartialInvoice, runType)
	})

	t.Run("final realization period", func(t *testing.T) {
		runType := getInvoiceRealizationRunType(charge, servicePeriod)
		require.Equal(t, usagebased.RealizationRunTypeFinalRealization, runType)
	})
}
