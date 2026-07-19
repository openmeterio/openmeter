package lineengine

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/invoicing/legacy/splitlinegroup"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	"github.com/samber/lo"
)

type StandardLineWithSplitLineHierarchy struct {
	*billing.StandardLine
	SplitLineHierarchy *splitlinegroup.SplitLineHierarchy
}

func (i StandardLineWithSplitLineHierarchy) Validate() error {
	if i.StandardLine == nil {
		return models.NewNillableGenericValidationError(errors.New("standard line is required"))
	}

	var errs []error

	if err := i.StandardLine.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.StandardLine.SplitLineGroupID != nil {
		if i.SplitLineHierarchy == nil {
			errs = append(errs, errors.New("split line hierarchy is required"))
		}
	}

	return errors.Join(errs...)
}

func (l StandardLineWithSplitLineHierarchy) IsProgressivelyBilled() bool {
	return l.SplitLineGroupID != nil
}

func (i StandardLineWithSplitLineHierarchy) GetProgressivelyBilledServicePeriod() (timeutil.ClosedPeriod, error) {
	if i.SplitLineGroupID == nil {
		return timeutil.ClosedPeriod{
			From: i.Period.From,
			To:   i.Period.To,
		}, nil
	}

	if i.SplitLineHierarchy == nil {
		return timeutil.ClosedPeriod{}, errors.New("split line hierarchy is required")
	}

	return i.SplitLineHierarchy.Group.ServicePeriod, nil
}

func (i StandardLineWithSplitLineHierarchy) GetPreviouslyBilledAmount() (alpacadecimal.Decimal, error) {
	if i.SplitLineGroupID == nil {
		return alpacadecimal.Zero, nil
	}

	if i.SplitLineHierarchy == nil {
		return alpacadecimal.Zero, fmt.Errorf("line[%s] does not have a progressive line hierarchy, but is a progressive billed line", i.ID)
	}

	return i.SplitLineHierarchy.SumNetAmount(splitlinegroup.SumNetAmountInput{
		PeriodEndLTE: i.Period.From,
	})
}

type StandardLinesWithSplitLineHierarchy []StandardLineWithSplitLineHierarchy

func (i StandardLinesWithSplitLineHierarchy) Validate() error {
	return errors.Join(lo.Map(i, func(line StandardLineWithSplitLineHierarchy, _ int) error {
		return line.Validate()
	})...)
}

func (i StandardLinesWithSplitLineHierarchy) AsStandardLines() billing.StandardLines {
	return lo.Map(i, func(line StandardLineWithSplitLineHierarchy, _ int) *billing.StandardLine {
		return line.StandardLine
	})
}

func (i StandardLinesWithSplitLineHierarchy) GetReferencedFeatureKeys() ([]string, error) {
	return i.AsStandardLines().GetReferencedFeatureKeys()
}
