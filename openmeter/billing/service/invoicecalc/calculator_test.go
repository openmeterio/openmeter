package invoicecalc

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

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
