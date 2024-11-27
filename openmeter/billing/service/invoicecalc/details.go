package invoicecalc

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
)

func RecalculateDetailedLinesAndTotals(invoice *billingentity.Invoice, deps CalculatorDependencies) error {
	if invoice.Lines.IsAbsent() {
		return errors.New("cannot recaulculate invoice without expanded lines")
	}

	lines, err := deps.LineService().FromEntities(invoice.Lines.OrEmpty())
	if err != nil {
		return fmt.Errorf("creating line services: %w", err)
	}

	var outErr error

	for _, line := range lines {
		if line.IsDeleted() {
			continue
		}

		if err := line.CalculateDetailedLines(); err != nil {
			outErr = errors.Join(outErr,
				billingentity.ValidationWithFieldPrefix(fmt.Sprintf("line[%s]", line.ID()),
					fmt.Errorf("calculating detailed lines: %w", err)))

			line.ResetTotals()
			continue
		}

		if err := line.UpdateTotals(); err != nil {
			outErr = errors.Join(outErr,
				billingentity.ValidationWithFieldPrefix(fmt.Sprintf("line[%s]", line.ID()),
					fmt.Errorf("updating totals: %w", err)))
		}
	}

	totals := billingentity.Totals{}

	totals = totals.Add(lo.Map(invoice.Lines.OrEmpty(), func(line *billingentity.Line, _ int) billingentity.Totals {
		// Deleted lines are not contributing to the totals
		if line.DeletedAt != nil {
			return billingentity.Totals{}
		}

		return line.Totals
	})...)

	invoice.Totals = totals

	return outErr
}
