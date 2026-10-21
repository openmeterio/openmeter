package stdinvoicesanity

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

// validateParentTotalsNonNegative checks that each active parent's final payable
// total is zero or positive, as StandardLine requires. Corrections may have
// negative quantities or detail amounts as long as the parent total stays valid.
func validateParentTotalsNonNegative(invoice billing.StandardInvoice) error {
	var errs []error
	for idx, line := range invoice.Lines.OrEmpty() {
		if line == nil || line.DeletedAt != nil || !line.Totals.Total.IsNegative() {
			continue
		}

		errs = append(errs, billing.ValidationWithFieldPrefix(fmt.Sprintf("lines/%d/totals/total", idx), billing.ValidationWithAttributes(models.Annotations{
			"invoice_id": invoice.ID,
			"line_id":    line.ID,
			"actual":     line.Totals.Total.String(),
		}, ErrNegativeParentTotal)))
	}

	return errors.Join(errs...)
}

// validateLineShape requires UsageBased and Price, even for flat prices:
// line accessors such as DependsOnMeteredQuantity dereference both pointers.
// Price configuration and other business rules keep their existing validators.
func validateLineShape(invoice billing.StandardInvoice) error {
	var errs []error
	for idx, line := range invoice.Lines.OrEmpty() {
		if line == nil || line.DeletedAt != nil {
			continue
		}

		var err error
		if line.UsageBased == nil {
			err = ErrMissingUsageBased
		} else if line.UsageBased.Price == nil {
			err = billing.ValidationWithFieldPrefix("price", ErrMissingPrice)
		}

		errs = append(errs, billing.ValidationWithFieldPrefix(fmt.Sprintf("lines/%d/usageBased", idx), billing.ValidationWithAttributes(models.Annotations{"invoice_id": invoice.ID, "line_id": line.ID}, err)))
	}

	return errors.Join(errs...)
}
