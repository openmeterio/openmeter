package calculation

import (
	"github.com/samber/lo"

	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
)

func Totals(invoice *billingentity.Invoice) (bool, error) {
	// Let's calculate the line totals

	totals := billingentity.Totals{}

	totals = totals.Add(lo.Map(invoice.Lines, func(line *billingentity.Line, _ int) billingentity.Totals {
		return line.Totals
	})...)

	invoice.Totals = totals

	return true, nil
}
