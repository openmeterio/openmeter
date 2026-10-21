package stdinvoicesanity

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/pkg/models"
)

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
