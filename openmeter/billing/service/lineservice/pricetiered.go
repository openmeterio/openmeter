package lineservice

import (
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

// tieredPricer is a pricer that can handle both volume and graduated tiered pricing acts as
// a router between the two pricers
type tieredPricer struct {
	volume    volumeTieredPricer
	graduated graduatedTieredPricer
}

var _ Pricer = (*tieredPricer)(nil)

func (p tieredPricer) Calculate(l PricerCalculateInput) (newDetailedLinesInput, error) {
	price, err := l.line.UsageBased.Price.AsTiered()
	if err != nil {
		return nil, fmt.Errorf("converting price to tiered price: %w", err)
	}

	switch price.Mode {
	case productcatalog.VolumeTieredPrice:
		return p.volume.Calculate(l)
	case productcatalog.GraduatedTieredPrice:
		return p.graduated.Calculate(l)
	default:
		return nil, fmt.Errorf("unsupported tiered price mode: %s", price.Mode)
	}
}

func (p tieredPricer) CanBeInvoicedAsOf(l usageBasedLine, asOf time.Time) (bool, error) {
	price, err := l.line.UsageBased.Price.AsTiered()
	if err != nil {
		return false, fmt.Errorf("converting price to tiered price: %w", err)
	}

	switch price.Mode {
	case productcatalog.VolumeTieredPrice:
		return p.volume.CanBeInvoicedAsOf(l, asOf)
	case productcatalog.GraduatedTieredPrice:
		return p.graduated.CanBeInvoicedAsOf(l, asOf)
	default:
		return false, fmt.Errorf("unsupported tiered price mode: %s", price.Mode)
	}
}
