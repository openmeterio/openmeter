package invoicecalc

import (
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

// FillGatheringDetailedLineMeta fills the meta fields for the detailed lines in a gathering invoice.
// This is needed because the detailed lines are not created in the database, so we need to fill the meta fields
// manually.
func FillGatheringDetailedLineMeta(invoice *billing.Invoice, deps CalculatorDependencies) error {
	invoice.Lines = invoice.Lines.Map(func(line *billing.Line) *billing.Line {
		line.DetailedLines = line.DetailedLines.Map(func(detailedLine billing.DetailedLine) billing.DetailedLine {
			if detailedLine.ID == "" {
				detailedLine.ID = ulid.Make().String()
			}

			if detailedLine.CreatedAt.IsZero() {
				detailedLine.CreatedAt = time.Now()
			}

			if detailedLine.UpdatedAt.IsZero() {
				detailedLine.UpdatedAt = time.Now()
			}

			return detailedLine
		})

		return line
	})

	return nil
}
