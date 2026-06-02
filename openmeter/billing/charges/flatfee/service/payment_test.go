package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestGetPaymentTotal(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	baseRun := flatfee.RealizationRun{
		RealizationRunBase: flatfee.RealizationRunBase{
			ID: flatfee.RealizationRunID{
				Namespace: "ns",
				ID:        "run",
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: servicePeriod.From,
				UpdatedAt: servicePeriod.From,
			},
			Type:                 flatfee.RealizationRunTypeFinalRealization,
			InitialType:          flatfee.RealizationRunTypeFinalRealization,
			ServicePeriod:        servicePeriod,
			AmountAfterProration: alpacadecimal.NewFromInt(5),
		},
	}

	t.Run("missing accrued usage returns an error", func(t *testing.T) {
		// given:
		// - a run has no accrued invoice usage
		// when:
		// - the payment total is requested
		// then:
		// - the payment path fails instead of silently booking zero
		_, err := getPaymentTotal(baseRun)
		require.Error(t, err)
		require.Contains(t, err.Error(), "accrued invoice usage is required")
	})

	t.Run("zero total on fiat-backed run returns an error", func(t *testing.T) {
		// given:
		// - a fiat-backed run has accrued usage with a zero total
		// when:
		// - the payment total is requested
		// then:
		// - the payment path fails instead of hiding the inconsistent state
		run := baseRun
		run.AccruedUsage = &invoicedusage.AccruedUsage{
			ServicePeriod: servicePeriod,
			Totals:        totals.Totals{},
		}

		_, err := getPaymentTotal(run)
		require.Error(t, err)
		require.Contains(t, err.Error(), "non-zero accrued invoice usage total is required")
	})

	t.Run("no-fiat run returns an error", func(t *testing.T) {
		// given:
		// - a no-fiat run has accrued usage with a zero total
		// when:
		// - the payment total is requested
		// then:
		// - the payment path fails because no-fiat runs should skip payment booking
		run := baseRun
		run.NoFiatTransactionRequired = true
		run.AccruedUsage = &invoicedusage.AccruedUsage{
			ServicePeriod: servicePeriod,
			Totals:        totals.Totals{},
		}

		_, err := getPaymentTotal(run)
		require.Error(t, err)
		require.Contains(t, err.Error(), "fiat payment total is not required")
	})

	t.Run("positive total is returned", func(t *testing.T) {
		// given:
		// - a run has positive accrued invoice usage
		// when:
		// - the payment total is requested
		// then:
		// - the accrued total is returned exactly
		run := baseRun
		run.AccruedUsage = &invoicedusage.AccruedUsage{
			ServicePeriod: servicePeriod,
			Totals: totals.Totals{
				Total: alpacadecimal.NewFromInt(5),
			},
		}

		total, err := getPaymentTotal(run)
		require.NoError(t, err)
		require.Equal(t, float64(5), total.InexactFloat64())
	})
}
