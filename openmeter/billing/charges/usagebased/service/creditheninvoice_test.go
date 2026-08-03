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
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	chargestatemachine "github.com/openmeterio/openmeter/openmeter/billing/charges/statemachine"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils/currency"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestUnsupportedExtendOperation(t *testing.T) {
	for _, status := range []usagebased.Status{
		usagebased.StatusActiveRealizationIssuing,
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

func TestUnsupportedExtendOperationIsConfiguredForFinalRealizationBoundary(t *testing.T) {
	for _, status := range []usagebased.Status{
		usagebased.StatusActiveRealizationIssuing,
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
	machine.Adapter = newCreditThenInvoiceStateMachineAdapter(charge)

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
	machine.Adapter = newCreditThenInvoiceStateMachineAdapter(charge)

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

func newCreditThenInvoiceStateMachineWithChargeForTest(t *testing.T, charge usagebased.Charge) *CreditThenInvoiceStateMachine {
	t.Helper()

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

type creditThenInvoiceStateMachineAdapter struct {
	usagebased.Adapter

	runs map[string]usagebased.RealizationRunBase
}

func newCreditThenInvoiceStateMachineAdapter(charge usagebased.Charge) *creditThenInvoiceStateMachineAdapter {
	runs := make(map[string]usagebased.RealizationRunBase, len(charge.Realizations))
	for _, run := range charge.Realizations {
		runs[run.ID.ID] = run.RealizationRunBase
	}

	return &creditThenInvoiceStateMachineAdapter{
		runs: runs,
	}
}

func (a *creditThenInvoiceStateMachineAdapter) CreateChargeOverride(_ context.Context, charge usagebased.ChargeBase, override usagebased.IntentMutableFields) (usagebased.ChargeBase, error) {
	charge.Intent = usagebased.NewOverridableIntent(charge.Intent.GetBaseIntent(), &override)
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
