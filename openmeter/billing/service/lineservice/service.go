package lineservice

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func FromEntity(line *billing.StandardLine, featureMeters feature.FeatureMeters) (Line, error) {
	currencyCalc, err := line.Currency.Calculator()
	if err != nil {
		return nil, fmt.Errorf("creating currency calculator: %w", err)
	}

	base := lineBase{
		line:          line,
		currency:      currencyCalc,
		featureMeters: featureMeters,
	}

	if line.UsageBased.Price.Type() == productcatalog.FlatPriceType {
		return &ubpFlatFeeLine{
			lineBase: base,
		}, nil
	}

	return &usageBasedLine{
		lineBase: base,
	}, nil
}

func FromEntities(line []*billing.StandardLine, featureMeters feature.FeatureMeters) (Lines, error) {
	return slicesx.MapWithErr(line, func(l *billing.StandardLine) (Line, error) {
		return FromEntity(l, featureMeters)
	})
}

// UpdateTotalsFromDetailedLines is a helper method to update the totals of a line from its detailed lines.
func UpdateTotalsFromDetailedLines(line *billing.StandardLine) error {
	// Calculate the line totals
	for idx, detailedLine := range line.DetailedLines {
		if detailedLine.DeletedAt != nil {
			continue
		}

		totals, err := calculateDetailedLineTotals(detailedLine)
		if err != nil {
			return fmt.Errorf("updating totals for line[%s]: %w", line.ID, err)
		}

		line.DetailedLines[idx].Totals = totals
	}

	// WARNING: Even if tempting to add discounts etc. here to the totals, we should always keep the logic as is.
	// The usageBasedLine will never be synchronized directly to stripe or other apps, only the detailed lines.
	//
	// Given that the external systems will have their own logic for calculating the totals, we cannot expect
	// any custom logic implemented here to be carried over to the external systems.

	// UBP line's value is the sum of all the children
	res := billing.Totals{}

	res = res.Add(lo.Map(line.DetailedLines, func(l billing.DetailedLine, _ int) billing.Totals {
		// Deleted lines are not contributing to the totals
		if l.DeletedAt != nil {
			return billing.Totals{}
		}

		return l.Totals
	})...)

	line.Totals = res

	return nil
}

type Line interface {
	LineBase

	CalculateDetailedLines() error
	UpdateTotals() error
}

type Lines []Line

func (s Lines) ToEntities() []*billing.StandardLine {
	return lo.Map(s, func(service Line, _ int) *billing.StandardLine {
		return service.ToEntity()
	})
}
