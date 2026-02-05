package lineservice

import (
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type PriceAccessor interface {
	GetPrice() *productcatalog.Price
	GetServicePeriod() timeutil.ClosedPeriod
	GetFeatureKey() string
}

func IsPeriodEmptyConsideringTruncations(line PriceAccessor) (bool, error) {
	price := line.GetPrice()
	if price == nil {
		return false, fmt.Errorf("price is nil")
	}

	if price.Type() == productcatalog.FlatPriceType {
		// Flat prices are always billable even if the period is empty
		return false, nil
	}

	return line.GetServicePeriod().Truncate(streaming.MinimumWindowSizeDuration).IsEmpty(), nil
}

type GetLinesWithBillablePeriodsInput[T PriceAccessor] struct {
	AsOf               time.Time
	ProgressiveBilling bool
	Lines              []T
	FeatureMeters      billing.FeatureMeters
}

type LineWithBillablePeriod[T PriceAccessor] struct {
	Line           T
	BillablePeriod timeutil.ClosedPeriod
}

func GetLinesWithBillablePeriods[T PricerCanBeInvoicedAsOfAccessor](in GetLinesWithBillablePeriodsInput[T]) ([]LineWithBillablePeriod[T], error) {
	out := make([]LineWithBillablePeriod[T], 0, len(in.Lines))
	for _, line := range in.Lines {
		billablePeriod, err := ResolveBillablePeriod(ResolveBillablePeriodInput[T]{
			AsOf:               in.AsOf,
			ProgressiveBilling: in.ProgressiveBilling,
			Line:               line,
			FeatureMeters:      in.FeatureMeters,
		})
		if err != nil {
			return nil, fmt.Errorf("line[%s]: %w", line.GetID(), err)
		}

		if billablePeriod == nil {
			continue
		}

		out = append(out, LineWithBillablePeriod[T]{
			Line:           line,
			BillablePeriod: *billablePeriod,
		})
	}

	return out, nil
}

type ResolveBillablePeriodInput[T PricerCanBeInvoicedAsOfAccessor] struct {
	AsOf               time.Time
	ProgressiveBilling bool
	Line               T
	FeatureMeters      billing.FeatureMeters
}

func ResolveBillablePeriod[T PricerCanBeInvoicedAsOfAccessor](in ResolveBillablePeriodInput[T]) (*timeutil.ClosedPeriod, error) {
	pricer, err := newPricerFor(in.Line)
	if err != nil {
		return nil, err
	}

	price := in.Line.GetPrice()
	if price == nil {
		return nil, fmt.Errorf("price is nil")
	}

	meterTypeAllowsProgressiveBilling := false
	if price.Type() != productcatalog.FlatPriceType && in.ProgressiveBilling {
		isDependingOnIncreaseOnlyMeters, err := isDependingOnIncreaseOnlyMeters(CanBeInvoicedAsOfInput{
			AsOf:               in.AsOf,
			ProgressiveBilling: in.ProgressiveBilling,
			Line:               in.Line,
			FeatureMeters:      in.FeatureMeters,
		})
		if err != nil {
			return nil, err
		}

		meterTypeAllowsProgressiveBilling = isDependingOnIncreaseOnlyMeters
	}

	// Force disable progressive billing if the meter type does not allow it
	if !meterTypeAllowsProgressiveBilling {
		in.ProgressiveBilling = false
	}

	billablePeriod, err := pricer.CanBeInvoicedAsOf(CanBeInvoicedAsOfInput{
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
func isDependingOnIncreaseOnlyMeters(in CanBeInvoicedAsOfInput) (bool, error) {
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
