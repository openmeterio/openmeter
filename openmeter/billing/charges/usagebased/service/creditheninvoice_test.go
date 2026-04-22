package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

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
