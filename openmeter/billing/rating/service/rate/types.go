package rate

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type Pricer interface {
	// GenerateDetailedLines generates the detailed lines for a line.
	GenerateDetailedLines(line PricerCalculateInput) (rating.DetailedLines, error)

	// ResolveBillablePeriod checks if the line can be invoiced as of the given time and returns the service
	// period that can be invoiced.
	ResolveBillablePeriod(rating.ResolveBillablePeriodInput) (*timeutil.ClosedPeriod, error)
}

type PricerCalculateInput struct {
	rating.ProgressiveBilledLineAccessor
	CurrencyCalculator currencyx.Currency

	FullProgressivelyBilledServicePeriod timeutil.ClosedPeriod
	Usage                                *rating.Usage
	StandardLineDiscounts                billing.StandardLineDiscounts
}

func (i PricerCalculateInput) Validate() error {
	if i.CurrencyCalculator == nil {
		return fmt.Errorf("currency is required")
	}

	if lo.IsEmpty(i.FullProgressivelyBilledServicePeriod) {
		return fmt.Errorf("full service period is required")
	}

	if i.ProgressiveBilledLineAccessor == nil {
		return fmt.Errorf("progressively billed line accessor is required")
	}

	return nil
}

func (i PricerCalculateInput) GetUsage() (rating.Usage, error) {
	if i.Usage == nil {
		return rating.Usage{}, fmt.Errorf("usage is nil")
	}

	return *i.Usage, nil
}

func (i PricerCalculateInput) IsLastInPeriod() bool {
	// If the line is not progressively billed, it is always the last line in period
	if !i.IsProgressivelyBilled() {
		return true
	}

	servicePeriod := i.GetServicePeriod()
	return servicePeriod.To.Equal(i.FullProgressivelyBilledServicePeriod.To)
}

func (i PricerCalculateInput) IsFirstInPeriod() bool {
	// If the line is not progressively billed, it is always the first line in period
	if !i.IsProgressivelyBilled() {
		return true
	}

	servicePeriod := i.GetServicePeriod()
	return servicePeriod.From.Equal(i.FullProgressivelyBilledServicePeriod.From)
}
