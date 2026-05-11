package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
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
