// Package stdinvoicesanity checks that calculated invoice totals agree with the
// details sent to invoicing apps. It does not recalculate invoices or decide
// which charges and discounts should apply.
package stdinvoicesanity

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Validate checks an invoice with loaded, calculated lines and returns all
// inconsistencies as critical billing validation issues. It leaves the invoice
// unchanged. Keep it separate from StandardInvoice.Validate so we can still read
// incomplete historical invoices without requiring them to pass these checks.
func Validate(invoice billing.StandardInvoice) error {
	return billing.ValidationWithComponent(ValidationComponent, models.Validate(invoice,
		validateExpandedLines,
		validateCurrency,
		validateDiscountTotals,
		validateCreditTotals,
		validateGrossAmounts,
		validateCommitmentPlacement,
		validateTotalArithmetic,
		validateLineTotals,
		validateInvoiceTotals,
		validateCreditConservation,
		validateCreditEntries,
		validateCreditAllocationLimits,
		validateCurrenciesAndPrecision,
		validateIdentities,
		validateServicePeriods,
		validateParentTotalsNonNegative,
		validateLineShape,
	))
}

// validateExpandedLines requires loaded lines with no nil entries. A loaded,
// empty collection is valid; unloaded lines do not tell us whether there are
// charges. Other checks skip nil entries so one missing line does not prevent us
// from reporting problems in the rest of the invoice.
func validateExpandedLines(invoice billing.StandardInvoice) error {
	if invoice.Lines.IsAbsent() {
		return billing.ValidationWithFieldPrefix("lines", ErrLinesNotExpanded)
	}

	var errs []error
	for idx, line := range invoice.Lines.OrEmpty() {
		if line == nil {
			errs = append(errs, billing.ValidationWithFieldPrefix(fmt.Sprintf("lines/%d", idx),
				ErrNilLine))
		}
	}

	return errors.Join(errs...)
}

// validateCurrency reports an unsupported currency once. Checks that need
// currency rounding skip it; checks that can compare stored totals still run.
func validateCurrency(invoice billing.StandardInvoice) error {
	if _, err := invoice.Currency.AsFiatCurrency(); err != nil {
		return billing.ValidationWithFieldPrefix("currency", billing.ValidationWithAttributes(models.Annotations{
			"currency": string(invoice.Currency),
			"error":    err.Error(),
		}, ErrInvalidCurrency))
	}

	return nil
}
