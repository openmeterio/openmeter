package rate

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// Tiered is a pricer that can handle both volume and graduated tiered pricing acts as
// a router between the two pricers
type Tiered struct {
	volume    VolumeTiered
	graduated GraduatedTiered
}

var _ Pricer = (*Tiered)(nil)

func (p Tiered) GenerateDetailedLines(l PricerCalculateInput) (rating.DetailedLines, error) {
	price, err := l.GetPrice().AsTiered()
	if err != nil {
		return nil, fmt.Errorf("converting price to tiered price: %w", err)
	}

	switch price.Mode {
	case productcatalog.VolumeTieredPrice:
		return p.volume.GenerateDetailedLines(l)
	case productcatalog.GraduatedTieredPrice:
		return p.graduated.GenerateDetailedLines(l)
	default:
		return nil, fmt.Errorf("unsupported tiered price mode: %s", price.Mode)
	}
}

func (p Tiered) ResolveBillablePeriod(in rating.ResolveBillablePeriodInput) (*timeutil.ClosedPeriod, error) {
	price, err := in.Line.GetPrice().AsTiered()
	if err != nil {
		return nil, fmt.Errorf("converting price to tiered price: %w", err)
	}

	switch price.Mode {
	case productcatalog.VolumeTieredPrice:
		return p.volume.ResolveBillablePeriod(in)
	case productcatalog.GraduatedTieredPrice:
		return p.graduated.ResolveBillablePeriod(in)
	default:
		return nil, fmt.Errorf("unsupported tiered price mode: %s", price.Mode)
	}
}
