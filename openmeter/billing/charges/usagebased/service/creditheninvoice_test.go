package service

import (
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
