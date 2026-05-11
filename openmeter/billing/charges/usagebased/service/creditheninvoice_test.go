package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	chargestatemachine "github.com/openmeterio/openmeter/openmeter/billing/charges/statemachine"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestUnsupportedExtendOperation(t *testing.T) {
	for _, status := range []usagebased.Status{
		usagebased.StatusActiveFinalRealizationIssuing,
		usagebased.StatusActiveFinalRealizationCompleted,
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
		usagebased.StatusActiveFinalRealizationIssuing,
		usagebased.StatusActiveFinalRealizationCompleted,
	} {
		t.Run(string(status), func(t *testing.T) {
			machine := newCreditThenInvoiceStateMachineForTest(t, status)
			patch, err := meta.NewPatchExtend(meta.NewPatchExtendInput{
				NewServicePeriodTo:     time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC),
				NewFullServicePeriodTo: time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC),
				NewBillingPeriodTo:     time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC),
				NewInvoiceAt:           time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC),
			})
			require.NoError(t, err)

			canFire, err := machine.CanFire(t.Context(), meta.TriggerExtend)
			require.NoError(t, err)
			require.True(t, canFire)

			err = machine.FireAndActivate(t.Context(), patch.Trigger(), patch.TriggerParams())
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
		usagebased.StatusActivePartialInvoiceIssuing,
		usagebased.StatusActivePartialInvoiceCompleted,
		usagebased.StatusActiveFinalRealizationIssuing,
		usagebased.StatusActiveFinalRealizationCompleted,
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
		usagebased.StatusActivePartialInvoiceIssuing,
		usagebased.StatusActivePartialInvoiceCompleted,
		usagebased.StatusActiveFinalRealizationIssuing,
		usagebased.StatusActiveFinalRealizationCompleted,
		usagebased.StatusDeleted,
	} {
		t.Run(string(status), func(t *testing.T) {
			machine := newCreditThenInvoiceStateMachineForTest(t, status)
			patch, err := meta.NewPatchShrink(meta.NewPatchShrinkInput{
				NewServicePeriodTo:     time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
				NewFullServicePeriodTo: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
				NewBillingPeriodTo:     time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
				NewInvoiceAt:           time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
			})
			require.NoError(t, err)

			canFire, err := machine.CanFire(t.Context(), meta.TriggerShrink)
			require.NoError(t, err)
			require.True(t, canFire)

			err = machine.FireAndActivate(t.Context(), patch.Trigger(), patch.TriggerParams())
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
			Intent: usagebased.Intent{
				Intent: meta.Intent{
					ServicePeriod:     servicePeriod,
					FullServicePeriod: servicePeriod,
					BillingPeriod:     servicePeriod,
				},
				InvoiceAt: servicePeriod.To,
			},
			Status: usagebased.StatusActivePartialInvoiceProcessing,
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
	require.Equal(t, usagebased.StatusActivePartialInvoiceProcessing, charge.Status)
	require.Equal(t, currentRunID, *charge.State.CurrentRealizationRunID)
	require.Equal(t, currentAdvanceAfter, *charge.State.AdvanceAfter)

	patches := machine.InvoicePatches()
	require.Len(t, patches, 1)
	require.Equal(t, invoiceupdater.PatchOpUpdateGatheringLineByChargeID, patches[0].Op())

	updatePatch, err := patches[0].AsUpdateGatheringLineByChargeIDPatch()
	require.NoError(t, err)
	require.Equal(t, "charge-id", updatePatch.ChargeID)
	require.Equal(t, newServicePeriodTo, updatePatch.ServicePeriodTo)
	require.Equal(t, newServicePeriodTo, updatePatch.InvoiceAt)
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
			Intent: usagebased.Intent{
				Intent: meta.Intent{
					ServicePeriod:     servicePeriod,
					FullServicePeriod: servicePeriod,
					BillingPeriod:     servicePeriod,
				},
				InvoiceAt: servicePeriod.To,
			},
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
		Mutable: true,
	}

	machine := newCreditThenInvoiceStateMachineWithChargeForTest(t, usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "namespace"},
				ID:              "charge-id",
			},
			Intent: usagebased.Intent{
				Intent: meta.Intent{
					ServicePeriod:     servicePeriod,
					FullServicePeriod: servicePeriod,
					BillingPeriod:     servicePeriod,
				},
				InvoiceAt: servicePeriod.To,
			},
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

	err := machine.startInvoiceCreatedRun(
		t.Context(),
		invoiceCreatedInput{
			LineID:    "line-1",
			InvoiceID: "invoice-1",
		},
		usagebased.RealizationRunTypePartialInvoice,
	)

	require.Error(t, err)
	require.ErrorContains(t, err, "validate invoice created input")
	require.ErrorContains(t, err, "service period to is required")
}

func TestResolveInvoiceCreatedTrigger(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}

	charge := usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			Intent: usagebased.Intent{
				Intent: meta.Intent{
					ServicePeriod: servicePeriod,
				},
			},
		},
	}

	t.Run("partial invoice period", func(t *testing.T) {
		billedPeriod := timeutil.ClosedPeriod{
			From: servicePeriod.From,
			To:   time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
		}

		trigger := resolveInvoiceCreatedTrigger(charge, billedPeriod)
		require.Equal(t, meta.TriggerPartialInvoiceCreated, trigger)
	})

	t.Run("final realization period", func(t *testing.T) {
		trigger := resolveInvoiceCreatedTrigger(charge, servicePeriod)
		require.Equal(t, meta.TriggerFinalInvoiceCreated, trigger)
	})
}
