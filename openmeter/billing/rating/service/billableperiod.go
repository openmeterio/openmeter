package service

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (s *service) ResolveBillablePeriod(in rating.ResolveBillablePeriodInput) (*timeutil.ClosedPeriod, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	linePricer, err := getPricerFor(in.Line, rating.NewGenerateDetailedLinesOptions())
	if err != nil {
		return nil, err
	}

	linePrice := in.Line.GetPrice()
	if linePrice == nil {
		return nil, fmt.Errorf("price is nil")
	}

	meterTypeAllowsProgressiveBilling := false
	if linePrice.Type() != productcatalog.FlatPriceType && in.ProgressiveBilling {
		isDependingOnIncreaseOnlyMeters, err := isDependingOnIncreaseOnlyMeters(in)
		if err != nil {
			return nil, err
		}

		meterTypeAllowsProgressiveBilling = isDependingOnIncreaseOnlyMeters
	}

	// Force disable progressive billing if the meter type does not allow it
	if !meterTypeAllowsProgressiveBilling {
		in.ProgressiveBilling = false
	}

	billablePeriod, err := linePricer.ResolveBillablePeriod(rating.ResolveBillablePeriodInput{
		AsOf:               in.AsOf,
		ProgressiveBilling: in.ProgressiveBilling,
		Line:               in.Line,
		FeatureMeters:      in.FeatureMeters,
	})
	if err != nil {
		return nil, err
	}

	return billablePeriod, nil
}

// isDependingOnIncreaseOnlyMeters checks if the line is depending on meters that can decrease the totals over time
// (note: this is somewhat of a lie, as we can input negative values in events, which will have the same effect)
func isDependingOnIncreaseOnlyMeters(in rating.ResolveBillablePeriodInput) (bool, error) {
	featureKey := in.Line.GetFeatureKey()
	if featureKey == "" {
		return false, fmt.Errorf("feature key is required")
	}

	// Let's check if the underlying meter can be billed in a progressive manner
	featureMeter, err := in.FeatureMeters.Get(featureKey, true)
	if err != nil {
		return false, err
	}

	if featureMeter.Meter == nil {
		return false, fmt.Errorf("meter is nil for feature[%s]", featureKey)
	}

	meterEntity := *featureMeter.Meter

	switch meterEntity.Aggregation {
	case meter.MeterAggregationSum, meter.MeterAggregationCount,
		meter.MeterAggregationMax, meter.MeterAggregationUniqueCount:
		return true, nil
	default:
		// Other types need to be billed in arrears truncated by window size
		return false, nil
	}
}
