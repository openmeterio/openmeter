// Package stdinvoicesanity checks consistency between calculated invoice totals
// and the supporting detail consumed by invoicing apps. It does not recalculate
// invoices or validate their commercial intent.
package stdinvoicesanity

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Validate expects an invoice with expanded, calculated lines. It reports all
// detected inconsistencies as critical billing validation issues without changing
// the invoice. It is separate from StandardInvoice.Validate so incomplete
// historical invoices can still be read without passing these checks.
func Validate(invoice billing.StandardInvoice) error {
	return billing.ValidationWithComponent(ValidationComponent, models.Validate(invoice,
		validateExpandedLines,
		validateDiscountTotals,
	))
}

// validateExpandedLines requires a loaded line collection with no nil entries.
// A loaded but empty collection is valid; absent lines cannot establish that
// there is no billable detail. Other validators skip nil entries so this check
// can report malformed input without preventing independent checks from running.
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
