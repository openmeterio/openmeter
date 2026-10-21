package stdinvoicesanity

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// validateServicePeriods checks that periods do not end before they start and
// that the invoice spans the earliest start and latest end of its active parents.
// Equal start and end times are valid. Rating corrections can put detail periods
// outside their parent period. If parents are missing or invalid, we cannot check
// the invoice span; no active parents means the invoice period must be nil.
func validateServicePeriods(invoice billing.StandardInvoice) error {
	var errs []error
	var expected *timeutil.ClosedPeriod
	known := !invoice.Lines.IsAbsent()
	for lineIdx, line := range invoice.Lines.OrEmpty() {
		if line == nil {
			known = false
			continue
		}

		if line.DeletedAt != nil {
			continue
		}

		var lineErrs []error
		if err := line.Period.Validate(); err != nil {
			lineErrs = append(lineErrs, billing.ValidationWithFieldPrefix("period", ErrInvalidServicePeriod))
			known = false
		}

		if expected == nil {
			period := line.Period
			expected = &period
		} else {
			if line.Period.From.Before(expected.From) {
				expected.From = line.Period.From
			}

			if line.Period.To.After(expected.To) {
				expected.To = line.Period.To
			}
		}

		for detailIdx, detail := range line.DetailedLines {
			if detail.DeletedAt != nil {
				continue
			}

			if err := detail.ServicePeriod.Validate(); err != nil {
				lineErrs = append(lineErrs, billing.ValidationWithFieldPrefix(fmt.Sprintf("detailedLines/%d/servicePeriod", detailIdx), billing.ValidationWithAttributes(models.Annotations{"detailed_line_id": detail.ID}, ErrInvalidServicePeriod)))
			}
		}

		errs = append(errs, billing.ValidationWithFieldPrefix(fmt.Sprintf("lines/%d", lineIdx), billing.ValidationWithAttributes(models.Annotations{"line_id": line.ID}, errors.Join(lineErrs...))))
	}

	if known && ((expected == nil) != (invoice.Period == nil) || (expected != nil && invoice.Period != nil && !expected.Equal(*invoice.Period))) {
		errs = append(errs, billing.ValidationWithFieldPrefix("period", billing.ValidationWithAttributes(models.Annotations{"expected": fmt.Sprint(expected), "actual": fmt.Sprint(invoice.Period)}, ErrInvoicePeriodMismatch)))
	}

	return billing.ValidationWithAttributes(models.Annotations{"invoice_id": invoice.ID}, errors.Join(errs...))
}
