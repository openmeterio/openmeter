package stdinvoicesanity

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/pkg/models"
)

// validateGrossAmounts checks that each active detail's gross amount matches
// unit amount times quantity, rounded only after multiplication. Rounding the
// inputs first could change the charge. Negative correction quantities are valid.
// validateCommitmentPlacement handles unknown categories; this check uses the
// calculated detail without rerating the parent price or metered usage.
func validateGrossAmounts(invoice billing.StandardInvoice) error {
	currency, err := invoice.Currency.AsFiatCurrency()
	if err != nil {
		// validateCurrency already reports this; avoid repeating the same issue.
		return nil
	}

	var errs []error
	for lineIdx, line := range invoice.Lines.OrEmpty() {
		if line == nil || line.DeletedAt != nil {
			continue
		}

		for detailIdx, detail := range line.DetailedLines {
			if detail.DeletedAt != nil {
				continue
			}

			field := "amount"
			actual := detail.Totals.Amount
			switch detail.Category {
			case stddetailedline.CategoryRegular:
			case stddetailedline.CategoryCommitment:
				field = "chargesTotal"
				actual = detail.Totals.ChargesTotal
			default:
				continue
			}

			expected := currency.RoundToPrecision(detail.PerUnitAmount.Mul(detail.Quantity))
			if expected.Equal(actual) {
				continue
			}

			errs = append(errs, billing.ValidationWithFieldPrefix(
				fmt.Sprintf("lines/%d/detailedLines/%d/totals/%s", lineIdx, detailIdx, field),
				billing.ValidationWithAttributes(models.Annotations{
					"invoice_id":       invoice.ID,
					"line_id":          line.ID,
					"detailed_line_id": detail.ID,
					"expected":         expected.String(),
					"actual":           actual.String(),
				}, ErrGrossAmountMismatch),
			))
		}
	}

	return errors.Join(errs...)
}

// validateCommitmentPlacement requires regular details to use Amount and
// commitments to use ChargesTotal, with the other field zero. Apps choose one
// field by category, so an unknown category or amounts in both fields could
// lose a charge. Zero amounts and negative corrections are valid; deleted
// parents and details are skipped because they no longer contribute charges.
func validateCommitmentPlacement(invoice billing.StandardInvoice) error {
	var errs []error
	for lineIdx, line := range invoice.Lines.OrEmpty() {
		if line == nil || line.DeletedAt != nil {
			continue
		}

		for detailIdx, detail := range line.DetailedLines {
			if detail.DeletedAt != nil {
				continue
			}

			path := fmt.Sprintf("lines/%d/detailedLines/%d", lineIdx, detailIdx)
			attributes := models.Annotations{
				"invoice_id":       invoice.ID,
				"line_id":          line.ID,
				"detailed_line_id": detail.ID,
				"category":         string(detail.Category),
			}
			if err := detail.Category.Validate(); err != nil {
				errs = append(errs, billing.ValidationWithFieldPrefix(path+"/category",
					billing.ValidationWithAttributes(attributes, ErrInvalidDetailCategory)))

				continue
			}

			field := "chargesTotal"
			actual := detail.Totals.ChargesTotal
			if detail.Category == stddetailedline.CategoryCommitment {
				field = "amount"
				actual = detail.Totals.Amount
			}

			if actual.IsZero() {
				continue
			}

			attributes["expected"] = "0"
			attributes["actual"] = actual.String()
			errs = append(errs, billing.ValidationWithFieldPrefix(path+"/totals/"+field,
				billing.ValidationWithAttributes(attributes, ErrCommitmentPlacementMismatch)))
		}
	}

	return errors.Join(errs...)
}
