package invoicecalc

import (
	"errors"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
)

// RecalculateTotals aggregates already-calculated standard lines without
// invoking their line engines. Deleted lines do not contribute to the invoice.
func RecalculateTotals(invoice *billing.StandardInvoice) error {
	if invoice == nil {
		return errors.New("invoice is required")
	}

	if invoice.Lines.IsAbsent() {
		return errors.New("cannot recalculate invoice totals without expanded lines")
	}

	invoice.Totals = totals.Sum(
		lo.Map(invoice.Lines.OrEmpty(), func(line *billing.StandardLine, _ int) totals.Totals {
			if line.IsDeleted() {
				return totals.Totals{}
			}

			return line.Totals
		})...,
	)

	return nil
}
