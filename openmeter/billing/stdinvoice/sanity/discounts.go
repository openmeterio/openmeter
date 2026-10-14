package stdinvoicesanity

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

// validateDiscountTotals expects each live detailed line's DiscountsTotal to
// equal its live amount-discount entries, including RoundingAmount corrections.
// Entries use billing's SumAmount currency rounding; the stored total must match
// exactly, without a tolerance or repair. Deleted parents, details and discounts
// do not contribute to billable value. Missing entries therefore only reconcile
// with a zero discount total. This checks calculated facts, not rate-card intent.
func validateDiscountTotals(invoice billing.StandardInvoice) error {
	currency, err := invoice.Currency.AsFiatCurrency()
	if err != nil {
		return billing.ValidationWithFieldPrefix("currency", billing.ValidationWithAttributes(models.Annotations{
			"currency": string(invoice.Currency),
			"error":    err.Error(),
		}, ErrInvalidCurrency))
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
