package service

import (
	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// nonProgressiveLineAccessor lets ordinary rating share the legacy pricing
// pipeline without requiring charge inputs to supply split-period history.
type nonProgressiveLineAccessor struct {
	rating.StandardLineAccessor
}

var _ rating.LegacyProgressivelyBillableLineAccessor = nonProgressiveLineAccessor{}

func (l nonProgressiveLineAccessor) GetMeteredPreLinePeriodQuantity() (*alpacadecimal.Decimal, error) {
	return lo.ToPtr(alpacadecimal.Zero), nil
}

func (l nonProgressiveLineAccessor) IsProgressivelyBilled() bool {
	return false
}

func (l nonProgressiveLineAccessor) GetProgressivelyBilledServicePeriod() (timeutil.ClosedPeriod, error) {
	return l.GetServicePeriod(), nil
}

func (l nonProgressiveLineAccessor) GetPreviouslyBilledAmount() (alpacadecimal.Decimal, error) {
	return alpacadecimal.Zero, nil
}
