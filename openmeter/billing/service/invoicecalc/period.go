package invoicecalc

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// CalculateStandardInvoiceServicePeriod calculates the period of the invoice based on the lines.
func CalculateStandardInvoiceServicePeriod(invoice *billing.StandardInvoice) error {
	var period *timeutil.ClosedPeriod

	for _, line := range invoice.Lines.OrEmpty() {
		if line.DeletedAt != nil {
			continue
		}

		if period == nil {
			period = &timeutil.ClosedPeriod{
				From: line.Period.From,
				To:   line.Period.To,
			}
			continue
		}

		if line.Period.From.Before(period.From) {
			period.From = line.Period.From
		}

		if line.Period.To.After(period.To) {
			period.To = line.Period.To
		}
	}

	invoice.Period = period

	return nil
}

func CalculateGatheringInvoiceServicePeriod(invoice *billing.GatheringInvoice) error {
	var period timeutil.ClosedPeriod

	for _, line := range invoice.Lines.OrEmpty() {
		if line.DeletedAt != nil {
			continue
		}

		if lo.IsEmpty(period) {
			period = line.ServicePeriod
			continue
		}

		if line.ServicePeriod.From.Before(period.From) {
			period.From = line.ServicePeriod.From
		}

		if line.ServicePeriod.To.After(period.To) {
			period.To = line.ServicePeriod.To
		}
	}

	invoice.ServicePeriod = period

	return nil
}
