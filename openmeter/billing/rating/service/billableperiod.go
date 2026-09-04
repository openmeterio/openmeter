package service

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func (s *service) ResolveBillablePeriod(in rating.ResolveBillablePeriodInput) (billing.IsLineBillableAsOfResult, error) {
	if err := in.Validate(); err != nil {
		return billing.IsLineBillableAsOfResult{}, err
	}

	linePricer, err := getPricerFor(in.Line, rating.NewGenerateDetailedLinesOptions(), s.unitConfigEnabled)
	if err != nil {
		return billing.IsLineBillableAsOfResult{}, err
	}

	linePrice := in.Line.GetPrice()
	if linePrice == nil {
		return billing.IsLineBillableAsOfResult{}, fmt.Errorf("price is nil")
	}

	meterTypeAllowsProgressiveBilling := false
	if linePrice.Type() != productcatalog.FlatPriceType && in.ProgressiveBilling {
		isDependingOnIncreaseOnlyMeters, err := isDependingOnIncreaseOnlyMeters(in)
		if err != nil {
			return billing.IsLineBillableAsOfResult{}, err
		}

		meterTypeAllowsProgressiveBilling = isDependingOnIncreaseOnlyMeters
	}

	// Force disable progressive billing if the meter type does not allow it
	if !meterTypeAllowsProgressiveBilling {
		in.ProgressiveBilling = false
	}

	result, err := linePricer.ResolveBillablePeriod(rating.ResolveBillablePeriodInput{
		AsOf:               in.AsOf,
		ProgressiveBilling: in.ProgressiveBilling,
		Line:               in.Line,
		FeatureMeters:      in.FeatureMeters,
	})
	if err != nil {
		return billing.IsLineBillableAsOfResult{}, err
	}

	if err := result.Validate(); err != nil {
		return billing.IsLineBillableAsOfResult{}, fmt.Errorf("validating billable period result: %w", err)
	}

	return result, nil
}

// isDependingOnIncreaseOnlyMeters checks if the line is depending on meters that can decrease the totals over time
// (note: this is somewhat of a lie, as we can input negative values in events, which will have the same effect)
func isDependingOnIncreaseOnlyMeters(in rating.ResolveBillablePeriodInput) (bool, error) {
	featureKey := in.Line.GetFeatureKey()
	if featureKey == "" {
		return false, fmt.Errorf("feature key is required")
	}

	// Let's check if the underlying meter can be billed in a progressive manner
	featureMeter, err := in.FeatureMeters.Get(in.Line)
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
