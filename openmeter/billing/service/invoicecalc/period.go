package invoicecalc

import "github.com/openmeterio/openmeter/openmeter/billing"

// CalculateInvoicePeriod calculates the period of the invoice based on the lines.
func CalculateInvoicePeriod(invoice *billing.StandardInvoice) error {
	var period *billing.Period

	for _, line := range invoice.Lines.OrEmpty() {
		if line.DeletedAt != nil {
			continue
		}

		if period == nil {
			period = &billing.Period{
				Start: line.Period.Start,
				End:   line.Period.End,
			}
			continue
		}

		if line.Period.Start.Before(period.Start) {
			period.Start = line.Period.Start
		}

		if line.Period.End.After(period.End) {
			period.End = line.Period.End
		}
	}

	invoice.Period = period

	return nil
}
