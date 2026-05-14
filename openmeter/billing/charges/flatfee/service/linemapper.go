package service

import (
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
)

func populateFlatFeeStandardLineFromRun(stdLine *billing.StandardLine, run flatfee.RealizationRun) error {
	creditsApplied, err := run.CreditRealizations.AsCreditsApplied()
	if err != nil {
		return err
	}

	stdLine.CreditsApplied = creditsApplied

	mappedDetailedLines, err := mapFlatFeeDetailedLines(stdLine, run)
	if err != nil {
		return fmt.Errorf("mapping run detailed lines: %w", err)
	}

	stdLine.DetailedLines = stdLine.DetailedLinesWithIDReuse(mappedDetailedLines)
	stdLine.Totals = stdLine.DetailedLines.SumTotals()

	if !stdLine.Totals.Equal(run.Totals) {
		return fmt.Errorf("mapped line totals do not match run totals [line_id=%s run_id=%s line_total=%s run_total=%s]",
			stdLine.ID, run.ID.ID, stdLine.Totals.Total.String(), run.Totals.Total.String())
	}

	return nil
}

func mapFlatFeeDetailedLines(stdLine *billing.StandardLine, run flatfee.RealizationRun) (billing.DetailedLines, error) {
	if run.DetailedLines.IsAbsent() {
		return nil, fmt.Errorf("run %s detailed lines must be expanded", run.ID.ID)
	}

	return lo.Map(run.DetailedLines.OrEmpty(), func(line flatfee.DetailedLine, _ int) billing.DetailedLine {
		base := line.Clone()
		base.Namespace = stdLine.Namespace
		base.ID = ""
		base.CreatedAt = time.Time{}
		base.UpdatedAt = time.Time{}
		base.DeletedAt = nil

		return billing.DetailedLine{
			DetailedLineBase: billing.DetailedLineBase{
				Base:      base,
				InvoiceID: stdLine.InvoiceID,
			},
		}
	}), nil
}
