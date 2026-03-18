package price

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/pricer"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type Pricer interface {
	// GenerateDetailedLines generates the detailed lines for a line.
	GenerateDetailedLines(line PricerCalculateInput) (pricer.DetailedLines, error)

	// ResolveBillablePeriod checks if the line can be invoiced as of the given time and returns the service
	// period that can be invoiced.
	ResolveBillablePeriod(pricer.ResolveBillablePeriodInput) (*timeutil.ClosedPeriod, error)
}

type PricerCalculateInput struct {
	pricer.StandardLineAccessor
	CurrencyCalculator currencyx.Calculator

	FullProgressivelyBilledServicePeriod timeutil.ClosedPeriod
	Usage                                *pricer.Usage
	StandardLineDiscounts                billing.StandardLineDiscounts
}

func (i PricerCalculateInput) Validate() error {
	if lo.IsEmpty(i.FullProgressivelyBilledServicePeriod) {
		return fmt.Errorf("full service period is required")
	}

	if i.StandardLineAccessor == nil {
		return fmt.Errorf("standard line accessor is required")
	}

	return nil
}

func (i PricerCalculateInput) GetUsage() (pricer.Usage, error) {
	if i.Usage == nil {
		return pricer.Usage{}, fmt.Errorf("usage is nil")
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
