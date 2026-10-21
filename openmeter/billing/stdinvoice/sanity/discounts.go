package stdinvoicesanity

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

// validateDiscountTotals checks that each active detail's DiscountsTotal matches
// its active discount entries, including RoundingAmount corrections. Use billing's
// SumAmount rounding so the total agrees with the breakdown sent to the app.
// No entries means a zero discount total. Deleted parents, details and discounts
// no longer contribute charges or discounts. Compare exactly without repairing
// the invoice or deciding which rate-card discounts should have applied.
func validateDiscountTotals(invoice billing.StandardInvoice) error {
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

			discounts := lo.Filter(detail.AmountDiscounts,
				func(discount billing.AmountLineDiscountManaged, _ int) bool {
					return discount.DeletedAt == nil
				})
			expected := discounts.SumAmount(currency)
			actual := detail.Totals.DiscountsTotal
			if expected.Equal(actual) {
				continue
			}

			errs = append(errs, billing.ValidationWithFieldPrefix(
				fmt.Sprintf("lines/%d/detailedLines/%d/totals/discountsTotal", lineIdx, detailIdx),
				billing.ValidationWithAttributes(models.Annotations{
					"invoice_id":       invoice.ID,
					"line_id":          line.ID,
					"detailed_line_id": detail.ID,
					"expected":         expected.String(),
					"actual":           actual.String(),
				}, ErrDiscountTotalMismatch),
			))
		}
	}

	return errors.Join(errs...)
}

// validateDiscountSigns expects each live amount-discount entry to reduce the
// detailed line in the direction of its amount: a positive line carries
// non-negative discounts, a charge reversal with a negative amount carries
// non-positive ones. A zero discount is always consistent. Discounts of a
// different sign would increase the line instead of discounting it.
func validateDiscountSigns(invoice billing.StandardInvoice) error {
	var errs []error
	for lineIdx, line := range invoice.Lines.OrEmpty() {
		if line == nil || line.DeletedAt != nil {
			continue
		}

		for detailIdx, detail := range line.DetailedLines {
			if detail.DeletedAt != nil {
				continue
			}

			for discountIdx, discount := range detail.AmountDiscounts {
				if discount.DeletedAt != nil || discount.Amount.IsZero() {
					continue
				}

				if discount.Amount.IsNegative() == detail.Totals.Amount.IsNegative() {
					continue
				}

				errs = append(errs, billing.ValidationWithFieldPrefix(
					fmt.Sprintf("lines/%d/detailedLines/%d/amountDiscounts/%d/amount", lineIdx, detailIdx, discountIdx),
					billing.ValidationWithAttributes(models.Annotations{
						"invoice_id":       invoice.ID,
						"line_id":          line.ID,
						"detailed_line_id": detail.ID,
						"line_amount":      detail.Totals.Amount.String(),
						"discount_amount":  discount.Amount.String(),
					}, ErrDiscountSignMismatch),
				))
			}
		}
	}

	return errors.Join(errs...)
}
