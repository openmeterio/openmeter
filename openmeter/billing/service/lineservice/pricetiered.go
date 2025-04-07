package lineservice

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

// tieredPricer is a pricer that works as a router between the graduated and volume tiered pricers.
type tieredPricer struct {
	graduatedPricer graduatedTieredPricer
	volumePricer    volumeTieredPricer
}

var _ Pricer = (*tieredPricer)(nil)

func (p *tieredPricer) Calculate(ctx context.Context, l usageBasedLine) (newDetailedLinesInput, error) {
	price, err := l.line.UsageBased.Price.AsTiered()
	if err != nil {
		return nil, err
	}

	switch price.Mode {
	case productcatalog.VolumeTieredPrice:
		return p.volumePricer.Calculate(ctx, l)
	case productcatalog.GraduatedTieredPrice:
		return p.graduatedPricer.Calculate(ctx, l)
	default:
		return nil, fmt.Errorf("unsupported tiered price mode: %s", price.Mode)
	}
}

func (p *tieredPricer) Capabilities(l usageBasedLine) (PricerCapabilities, error) {
	price, err := l.line.UsageBased.Price.AsTiered()
	if err != nil {
		return PricerCapabilities{}, err
	}

	switch price.Mode {
	case productcatalog.GraduatedTieredPrice:
		return p.graduatedPricer.Capabilities(l)
	case productcatalog.VolumeTieredPrice:
		return p.volumePricer.Capabilities(l)
	default:
		return PricerCapabilities{}, fmt.Errorf("unsupported tiered price mode: %s", price.Mode)
	}
}
