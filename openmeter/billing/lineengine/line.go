package lineengine

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type linesWithHierarchy []lineWithHierarchy

// Clone isolates mutable line state while retaining read-only
// history for the same calculation operation.
func (lines linesWithHierarchy) Clone() (linesWithHierarchy, error) {
	return slicesx.MapWithErr(lines, func(line lineWithHierarchy) (lineWithHierarchy, error) {
		clonedLine, err := line.StandardLine.Clone()
		if err != nil {
			return lineWithHierarchy{}, err
		}

		return lineWithHierarchy{StandardLine: clonedLine, hierarchy: line.hierarchy}, nil
	})
}

func (lines linesWithHierarchy) MapStandardLines() billing.StandardLines {
	return slicesx.Map(lines, func(line lineWithHierarchy) *billing.StandardLine {
		return line.StandardLine
	})
}

type lineWithHierarchy struct {
	*billing.StandardLine
	hierarchy *billing.SplitLineHierarchy
}

var _ rating.LegacyProgressivelyBillableLineAccessor = lineWithHierarchy{}

func (l lineWithHierarchy) GetProgressivelyBilledServicePeriod() (timeutil.ClosedPeriod, error) {
	if !l.IsProgressivelyBilled() {
		return l.Period, nil
	}

	if l.hierarchy == nil {
		return timeutil.ClosedPeriod{}, errors.New("split line hierarchy is required")
	}

	return l.hierarchy.Group.ServicePeriod, nil
}

func (l lineWithHierarchy) GetPreviouslyBilledAmount() (alpacadecimal.Decimal, error) {
	if !l.IsProgressivelyBilled() {
		return alpacadecimal.Zero, nil
	}

	if l.hierarchy == nil {
		return alpacadecimal.Zero, fmt.Errorf("line[%s] does not have a progressive line hierarchy, but is a progressive billed line", l.ID)
	}

	return l.hierarchy.SumNetAmount(billing.SumNetAmountInput{PeriodEndLTE: l.Period.From})
}
