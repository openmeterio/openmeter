package lineservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

var pricerByPriceType = map[productcatalog.PriceType]Pricer{
	productcatalog.UnitPriceType:    &unitPricer{},
	productcatalog.DynamicPriceType: &dynamicPricer{},
	productcatalog.PackagePriceType: &packagePricer{},
	productcatalog.TieredPriceType:  &tieredPricer{},
}

type PricerCapabilities struct {
	AllowsProgressiveBilling bool
}

type Pricer interface {
	// Calculate calculates the detailed lines for a line.
	Calculate(ctx context.Context, line usageBasedLine) (newDetailedLinesInput, error)

	// TODO most probably requires Feature/Meter as input
	Capabilities(usageBasedLine) (PricerCapabilities, error)
}

// PricerMiddleware is a middleware that can be used to modify the pricer behavior (discounts and commitments)
type PricerMiddleware interface {
	BeforeCalculate(ctx context.Context, line usageBasedLine) (usageBasedLine, error)
	AfterCalculate(ctx context.Context, line usageBasedLine, pricerResult newDetailedLinesInput) (newDetailedLinesInput, error)

	Capabilities(line usageBasedLine, pricerCapabilities PricerCapabilities) (PricerCapabilities, error)
}

// PricerMiddlewareBase is a no-op pricer middleware, can be used to prevent unneeded middleware interface member implementation
type PricerMiddlewareBase struct{}

func (p PricerMiddlewareBase) BeforeCalculate(ctx context.Context, line usageBasedLine) (usageBasedLine, error) {
	return line, nil
}

func (p PricerMiddlewareBase) AfterCalculate(ctx context.Context, line usageBasedLine, pricerResult newDetailedLinesInput) (newDetailedLinesInput, error) {
	return pricerResult, nil
}

func (p PricerMiddlewareBase) Capabilities(line usageBasedLine, pricerCapabilities PricerCapabilities) (PricerCapabilities, error) {
	return pricerCapabilities, nil
}
