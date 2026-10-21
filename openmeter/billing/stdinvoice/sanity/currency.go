package stdinvoicesanity

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

// validateCurrenciesAndPrecision checks that lines use the invoice currency
// and that invoice, parent and detail totals are already rounded to its precision.
// Otherwise, apps could change the amounts when converting them to currency units.
// Unit prices, quantities and allocation inputs keep their full precision.
// validateCurrency reports unsupported currencies.
func validateCurrenciesAndPrecision(invoice billing.StandardInvoice) error {
	currency, err := invoice.Currency.AsFiatCurrency()
	if err != nil {
		return nil
	}

	comparison := totalsComparison{
		Expected:      invoice.Totals.RoundToPrecision(currency),
		Actual:        invoice.Totals,
		MismatchError: ErrTotalsPrecision,
	}

	errs := []error{billing.ValidationWithFieldPrefix("totals", comparison.Validate())}
	for lineIdx, line := range invoice.Lines.OrEmpty() {
		if line == nil || line.DeletedAt != nil {
			continue
		}

		var lineErrs []error
		if line.Currency != invoice.Currency {
			lineErrs = append(lineErrs, billing.ValidationWithFieldPrefix("currency", billing.ValidationWithAttributes(models.Annotations{"expected": string(invoice.Currency), "actual": string(line.Currency)}, ErrCurrencyMismatch)))
		}

		comparison = totalsComparison{
			Expected:      line.Totals.RoundToPrecision(currency),
			Actual:        line.Totals,
			MismatchError: ErrTotalsPrecision,
		}

		lineErrs = append(lineErrs, billing.ValidationWithFieldPrefix("totals", comparison.Validate()))
		for detailIdx, detail := range line.DetailedLines {
			if detail.DeletedAt != nil {
				continue
			}

			comparison = totalsComparison{
				Expected:      detail.Totals.RoundToPrecision(currency),
				Actual:        detail.Totals,
				MismatchError: ErrTotalsPrecision,
			}

			lineErrs = append(lineErrs, billing.ValidationWithFieldPrefix(fmt.Sprintf("detailedLines/%d/totals", detailIdx), billing.ValidationWithAttributes(models.Annotations{"detailed_line_id": detail.ID}, comparison.Validate())))
		}

		errs = append(errs, billing.ValidationWithFieldPrefix(fmt.Sprintf("lines/%d", lineIdx), billing.ValidationWithAttributes(models.Annotations{"line_id": line.ID}, errors.Join(lineErrs...))))
	}

	return billing.ValidationWithAttributes(models.Annotations{"invoice_id": invoice.ID}, errors.Join(errs...))
}
