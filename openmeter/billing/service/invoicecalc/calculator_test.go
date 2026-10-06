package invoicecalc

import (
	"errors"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestInvoicePipelinesAggregateAuthoritativeLines(t *testing.T) {
	for _, status := range []billing.StandardInvoiceStatus{billing.StandardInvoiceStatusDraftCreated, billing.StandardInvoiceStatusGathering} {
		t.Run(string(status), func(t *testing.T) {
			// Given authoritative line totals that differ from the supplied flat price.
			now := time.Now().UTC()
			line := billing.NewFlatFeeLine(billing.NewFlatFeeLineInput{
				ID: "legacy", Name: "legacy", Currency: "USD", InvoiceID: "invoice",
				Period:        timeutil.ClosedPeriod{From: now.Add(-time.Hour), To: now},
				PerUnitAmount: alpacadecimal.NewFromInt(100), PaymentTerm: productcatalog.InAdvancePaymentTerm,
			})
			line.Totals = totals.Totals{Amount: alpacadecimal.NewFromInt(12), Total: alpacadecimal.NewFromInt(12)}
			chargeLine, err := line.Clone()
			require.NoError(t, err)
			chargeLine.ID = "charge"
			chargeLine.Engine = billing.LineEngineTypeChargeFlatFee
			chargeLine.Totals = totals.Totals{Amount: alpacadecimal.NewFromInt(8), Total: alpacadecimal.NewFromInt(8)}
			deleted, err := line.Clone()
			require.NoError(t, err)
			deleted.ID = "deleted"
			deleted.DeletedAt = &now
			invoice := billing.StandardInvoice{
				StandardInvoiceBase: billing.StandardInvoiceBase{ID: "invoice", Status: status},
				Lines:               billing.NewStandardInvoiceLines(billing.StandardLines{line, chargeLine, deleted}),
			}

			// When either pipeline runs without a line-engine registry or rating service.
			if status == billing.StandardInvoiceStatusGathering {
				err = New().CalculateGatheringInvoiceWithLiveData(&invoice, StandardInvoiceCalculatorDependencies{})
			} else {
				err = New().Calculate(&invoice, StandardInvoiceCalculatorDependencies{})
			}
			require.NoError(t, err)

			// Then the invoice sums authoritative live lines without rerating them.
			require.Equal(t, float64(20), invoice.Totals.Total.InexactFloat64())
			require.Equal(t, float64(12), line.Totals.Total.InexactFloat64())
			require.Equal(t, float64(8), chargeLine.Totals.Total.InexactFloat64())
		})
	}
}

func TestApplyCalculationsPreservesErrorClassification(t *testing.T) {
	t.Run("merges typed validation issues", func(t *testing.T) {
		invoice := billing.StandardInvoice{}
		warning := billing.NewValidationWarning("rating_warning", "rating warning")

		err := (&calculator{}).applyCalculations(
			&invoice,
			[]StandardInvoiceCalculation{func(*billing.StandardInvoice, StandardInvoiceCalculatorDependencies) error {
				return warning
			}},
			StandardInvoiceCalculatorDependencies{},
		)

		require.NoError(t, err)
		require.Equal(t, billing.ValidationIssues{{
			Severity:  warning.Severity,
			Message:   warning.Message,
			Code:      warning.Code,
			Component: billing.ValidationComponentOpenMeter,
		}}, invoice.ValidationIssues)
	})

	t.Run("returns system errors", func(t *testing.T) {
		invoice := billing.StandardInvoice{}
		systemErr := errors.New("rating service unavailable")

		err := (&calculator{}).applyCalculations(
			&invoice,
			[]StandardInvoiceCalculation{func(*billing.StandardInvoice, StandardInvoiceCalculatorDependencies) error {
				return systemErr
			}},
			StandardInvoiceCalculatorDependencies{},
		)

		require.ErrorIs(t, err, systemErr)
		require.Empty(t, invoice.ValidationIssues)

		issues, extractionErr := billing.ToValidationIssues(err)
		require.Nil(t, issues)
		require.Equal(t, err, extractionErr)
	})
}
