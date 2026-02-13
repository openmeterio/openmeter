package lineservice

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// tieredPricer is a pricer that can handle both volume and graduated tiered pricing acts as
// a router between the two pricers
type tieredPricer struct {
	volume    volumeTieredPricer
	graduated graduatedTieredPricer
}

var _ Pricer = (*tieredPricer)(nil)

func (p tieredPricer) Calculate(l PricerCalculateInput) (newDetailedLinesInput, error) {
	price, err := l.line.Price.AsTiered()
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

func (p tieredPricer) CanBeInvoicedAsOf(in CanBeInvoicedAsOfInput) (*timeutil.ClosedPeriod, error) {
	price, err := in.Line.GetPrice().AsTiered()
	if err != nil {
		return nil, fmt.Errorf("converting price to tiered price: %w", err)
	}

	switch price.Mode {
	case productcatalog.VolumeTieredPrice:
		return p.volume.CanBeInvoicedAsOf(in)
	case productcatalog.GraduatedTieredPrice:
		return p.graduated.CanBeInvoicedAsOf(in)
	default:
		return nil, fmt.Errorf("unsupported tiered price mode: %s", price.Mode)
	}
}
