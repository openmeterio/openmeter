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
		if line.Children.IsAbsent() {
			return line
		}

		line.Children = line.Children.Map(func(child *billing.Line) *billing.Line {
			if child.ID == "" {
				child.ID = ulid.Make().String()
			}

			if child.CreatedAt.IsZero() {
				child.CreatedAt = time.Now()
			}

			if child.UpdatedAt.IsZero() {
				child.UpdatedAt = time.Now()
			}

			return child
		})

		return line
	})

	return nil
}
