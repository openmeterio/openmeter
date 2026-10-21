package stdinvoicesanity

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/pkg/models"
)

// validateTotalArithmetic checks that the invoice and each active parent and
// detail have Total = amount + charges + exclusive tax - discounts - credits.
// Inclusive tax is already in the gross amount; adding it again would double count
// it. Negative correction details are valid. Compare stored values exactly so
// rounding cannot hide an error, even if lines are missing or currency is invalid.
func validateTotalArithmetic(invoice billing.StandardInvoice) error {
	errs := []error{validateTotal(invoice.Totals)}
	for lineIdx, line := range invoice.Lines.OrEmpty() {
		if line == nil || line.DeletedAt != nil {
			continue
		}

		lineErrs := []error{validateTotal(line.Totals)}
		for detailIdx, detail := range line.DetailedLines {
			if detail.DeletedAt != nil {
				continue
			}

			lineErrs = append(lineErrs, billing.ValidationWithFieldPrefix(
				fmt.Sprintf("detailedLines/%d", detailIdx),
				billing.ValidationWithAttributes(models.Annotations{"detailed_line_id": detail.ID}, validateTotal(detail.Totals)),
			))
		}

		errs = append(errs, billing.ValidationWithFieldPrefix(fmt.Sprintf("lines/%d", lineIdx),
			billing.ValidationWithAttributes(models.Annotations{"line_id": line.ID}, errors.Join(lineErrs...))))
	}

	return billing.ValidationWithAttributes(models.Annotations{"invoice_id": invoice.ID}, errors.Join(errs...))
}

func validateTotal(value totals.Totals) error {
	expected := value.CalculateTotal()
	if expected.Equal(value.Total) {
		return nil
	}

	return billing.ValidationWithFieldPrefix("totals/total", billing.ValidationWithAttributes(models.Annotations{
		"expected": expected.String(),
		"actual":   value.Total.String(),
	}, ErrTotalArithmeticMismatch))
}

// validateLineTotals checks that every active parent total field equals the sum
// of that field across its active details. Apps use the details, so an adjustment
// stored only on the parent would be lost. No active details means zero totals;
// this allows provisional lines that have no charges yet.
func validateLineTotals(invoice billing.StandardInvoice) error {
	var errs []error
	for lineIdx, line := range invoice.Lines.OrEmpty() {
		if line == nil || line.DeletedAt != nil {
			continue
		}

		var expected totals.Totals
		for _, detail := range line.DetailedLines {
			if detail.DeletedAt == nil {
				expected = expected.Add(detail.Totals)
			}
		}

		comparison := totalsComparison{
			Expected:      expected,
			Actual:        line.Totals,
			MismatchError: ErrLineTotalsMismatch,
		}
		errs = append(errs, billing.ValidationWithFieldPrefix(fmt.Sprintf("lines/%d/totals", lineIdx),
			billing.ValidationWithAttributes(models.Annotations{
				"invoice_id": invoice.ID,
				"line_id":    line.ID,
			}, comparison.Validate())))
	}

	return errors.Join(errs...)
}

// validateInvoiceTotals checks that each invoice total field equals the sum of
// that field across active parents. Missing or nil parents leave us unable to add
// up the lines; validateExpandedLines reports those. A loaded collection with no
// active parents must have zero invoice totals.
func validateInvoiceTotals(invoice billing.StandardInvoice) error {
	if invoice.Lines.IsAbsent() {
		return nil
	}

	var expected totals.Totals
	for _, line := range invoice.Lines.OrEmpty() {
		if line == nil {
			return nil
		}

		if line.DeletedAt == nil {
			expected = expected.Add(line.Totals)
		}
	}

	comparison := totalsComparison{
		Expected:      expected,
		Actual:        invoice.Totals,
		MismatchError: ErrInvoiceTotalsMismatch,
	}

	return billing.ValidationWithFieldPrefix("totals", billing.ValidationWithAttributes(
		models.Annotations{"invoice_id": invoice.ID}, comparison.Validate()))
}

type totalsComparison struct {
	Expected      totals.Totals
	Actual        totals.Totals
	MismatchError billing.ValidationIssue
}

// Validate compares each total field exactly and reports its path and both
// values. Checking only the payable total could hide errors that cancel out,
// such as an extra charge paired with an extra discount.
func (c totalsComparison) Validate() error {
	var errs []error
	for _, field := range []struct {
		name     string
		expected alpacadecimal.Decimal
		actual   alpacadecimal.Decimal
	}{
		{name: "amount", expected: c.Expected.Amount, actual: c.Actual.Amount},
		{name: "chargesTotal", expected: c.Expected.ChargesTotal, actual: c.Actual.ChargesTotal},
		{name: "discountsTotal", expected: c.Expected.DiscountsTotal, actual: c.Actual.DiscountsTotal},
		{name: "creditsTotal", expected: c.Expected.CreditsTotal, actual: c.Actual.CreditsTotal},
		{name: "taxesInclusiveTotal", expected: c.Expected.TaxesInclusiveTotal, actual: c.Actual.TaxesInclusiveTotal},
		{name: "taxesExclusiveTotal", expected: c.Expected.TaxesExclusiveTotal, actual: c.Actual.TaxesExclusiveTotal},
		{name: "taxesTotal", expected: c.Expected.TaxesTotal, actual: c.Actual.TaxesTotal},
		{name: "total", expected: c.Expected.Total, actual: c.Actual.Total},
	} {
		if field.expected.Equal(field.actual) {
			continue
		}

		errs = append(errs, billing.ValidationWithFieldPrefix(field.name, billing.ValidationWithAttributes(models.Annotations{
			"expected": field.expected.String(),
			"actual":   field.actual.String(),
		}, c.MismatchError)))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
