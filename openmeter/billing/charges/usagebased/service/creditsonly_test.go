package service

import (
	"context"
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargestatemachine "github.com/openmeterio/openmeter/openmeter/billing/charges/statemachine"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
	usagebasedrun "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/run"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestCreditsOnlyPeriodPatchIsConfiguredForFinalRealizationBoundaries(t *testing.T) {
	for _, status := range []usagebased.Status{
		usagebased.StatusActiveFinalRealizationStarted,
		usagebased.StatusActiveFinalRealizationWaitingForCollection,
		usagebased.StatusActiveFinalRealizationCompleted,
		usagebased.StatusFinal,
	} {
		t.Run(string(status), func(t *testing.T) {
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

func TestCreditsOnlyExtendWhileFinalRealizationInProgressVoidsCurrentRunAndMovesActive(t *testing.T) {
	for _, status := range []usagebased.Status{
		usagebased.StatusActiveFinalRealizationStarted,
		usagebased.StatusActiveFinalRealizationWaitingForCollection,
	} {
		t.Run(string(status), func(t *testing.T) {
			servicePeriod := timeutil.ClosedPeriod{
				From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			}
			extendedServicePeriodTo := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
			currentRunID := "run-1"

			machine := newCreditsOnlyStateMachineWithChargeForTest(t, usagebased.Charge{
				ChargeBase: usagebased.ChargeBase{
					ManagedResource: newUsageBasedChargeTestManagedResource("charge-id"),
					Intent:          newUsageBasedIntentForCreditOnlyTest(servicePeriod),
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
			Intent:          newUsageBasedIntentForCreditOnlyTest(servicePeriod),
			Status:          usagebased.StatusActiveFinalRealizationCompleted,
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
			Intent:          newUsageBasedIntentForCreditOnlyTest(servicePeriod),
			Status:          usagebased.StatusActiveFinalRealizationCompleted,
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

func newUsageBasedIntentForCreditOnlyTest(servicePeriod timeutil.ClosedPeriod) usagebased.OverridableIntent {
	return usagebased.Intent{
		Intent: meta.Intent{
			ManagedBy:  billing.SubscriptionManagedLine,
			CustomerID: "customer-id",
			Currency:   currencyx.Code("USD"),
		},
		IntentMutableFields: usagebased.IntentMutableFields{
			IntentMutableFields: meta.IntentMutableFields{
				Name:              "usage",
				ServicePeriod:     servicePeriod,
				FullServicePeriod: servicePeriod,
				BillingPeriod:     servicePeriod,
				TaxConfig: productcatalog.TaxCodeConfig{
					TaxCodeID: "tax-code-id",
				},
			},
			InvoiceAt:  servicePeriod.To,
			FeatureKey: "feature-key",
			Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: decimal.NewFromInt(1),
			}),
		},
		SettlementMode: productcatalog.CreditOnlySettlementMode,
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
			Intent:          newUsageBasedIntentForCreditOnlyTest(servicePeriod),
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

	currencyCalculator, err := currencyx.Code("USD").Calculator()
	require.NoError(t, err)

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

	out := &CreditsOnlyStateMachine{
		stateMachine: &stateMachine{
			Machine:            machine,
			Adapter:            adapter,
			Runs:               runService,
			CurrencyCalculator: currencyCalculator,
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

	runs map[string]usagebased.RealizationRunBase
}

func newCreditsOnlyStateMachineAdapter(charge usagebased.Charge) *creditsOnlyStateMachineAdapter {
	runs := make(map[string]usagebased.RealizationRunBase, len(charge.Realizations))
	for _, run := range charge.Realizations {
		runs[run.ID.ID] = run.RealizationRunBase
	}

	return &creditsOnlyStateMachineAdapter{
		runs: runs,
	}
}

func (a *creditsOnlyStateMachineAdapter) UpdateCharge(_ context.Context, base usagebased.ChargeBase) (usagebased.ChargeBase, error) {
	return base, nil
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
