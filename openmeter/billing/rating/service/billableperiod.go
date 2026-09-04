package service

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
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

	progressiveBillingAllowed := false
	// We can allow progressive billing only if:
	// - the line is not flat price (=> usage based)
	// - the usage based line's features can be resolved
	//
	// As any usage-based line is billable in arrears, thus we are safe to continue with non-progressive billing.
	//
	// By allowing empty feature/meter for usage-based lines we can create a standard invoice which immediately
	// enters into failed state, thus we are communicating the issue to the user.
	if linePrice.Type() != productcatalog.FlatPriceType && in.ProgressiveBilling && in.Feature != nil && in.Meter != nil {
		progressiveBillingAllowed = isDependingOnIncreaseOnlyMeters(in)
	}

	in.ProgressiveBilling = progressiveBillingAllowed
	if !in.ProgressiveBilling && in.AsOf.Truncate(streaming.MinimumWindowSizeDuration).Before(in.Line.GetInvoiceAt().Truncate(streaming.MinimumWindowSizeDuration)) {
		return billing.IsLineBillableAsOfResult{}, nil
	}

	result, err := linePricer.ResolveBillablePeriod(rating.ResolveBillablePeriodInput{
		AsOf:               in.AsOf,
		ProgressiveBilling: in.ProgressiveBilling,
		Line:               in.Line,
		Feature:            in.Feature,
		Meter:              in.Meter,
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
func isDependingOnIncreaseOnlyMeters(in rating.ResolveBillablePeriodInput) bool {
	switch in.Meter.Aggregation {
	case meter.MeterAggregationSum, meter.MeterAggregationCount,
		meter.MeterAggregationMax, meter.MeterAggregationUniqueCount:
		return true
	default:
		// Other types need to be billed in arrears truncated by window size
		return false
	}
}
