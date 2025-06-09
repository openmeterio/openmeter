package invoicecalc

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

func RecalculateDetailedLinesAndTotals(invoice *billing.Invoice, deps CalculatorDependencies) error {
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
				billing.ValidationWithFieldPrefix(fmt.Sprintf("line[%s]", line.ID()),
					fmt.Errorf("calculating detailed lines: %w", err)))

			line.ResetTotals()
			continue
		}

		if err := line.UpdateTotals(); err != nil {
			outErr = errors.Join(outErr,
				billing.ValidationWithFieldPrefix(fmt.Sprintf("line[%s]", line.ID()),
					fmt.Errorf("updating totals: %w", err)))
		}
	}

	totals := billing.Totals{}

	totals = totals.Add(lo.Map(invoice.Lines.OrEmpty(), func(line *billing.Line, _ int) billing.Totals {
		// Deleted lines are not contributing to the totals
		if line.DeletedAt != nil {
			return billing.Totals{}
		}

		return line.Totals
	})...)

	invoice.Totals = totals

	return outErr
}
