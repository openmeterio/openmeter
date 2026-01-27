package lineservice

import (
	"context"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

type getFeatureUsageInput struct {
	Line     *billing.StandardLine
	Meter    meter.Meter
	Feature  feature.Feature
	Customer billing.InvoiceCustomer
}

func (i getFeatureUsageInput) Validate() error {
	if i.Line == nil {
		return fmt.Errorf("line is required")
	}

	// So that we can safely determine the IsFirst/IsLastInPeriod flags
	if i.Line.SplitLineGroupID != nil && i.Line.SplitLineHierarchy == nil {
		return fmt.Errorf("split line group id is set but split line hierarchy is not expanded")
	}

	if slices.Contains([]meter.MeterAggregation{
		meter.MeterAggregationAvg,
		meter.MeterAggregationMin,
	}, i.Meter.Aggregation) {
		if i.Line.SplitLineHierarchy != nil {
			return fmt.Errorf("aggregation %s is not supported for split lines", i.Meter.Aggregation)
		}
	}

	if err := i.Customer.Validate(); err != nil {
		return fmt.Errorf("customer: %w", err)
	}

	return nil
}

type featureUsageResponse struct {
	// LinePeriodQty is the quantity of the usage for the line for the period
	LinePeriodQty alpacadecimal.Decimal
	// PreLinePeriodQty is the quantity of the usage for the line for the period before the current period
	PreLinePeriodQty alpacadecimal.Decimal
}

func (s *Service) getFeatureUsage(ctx context.Context, in getFeatureUsageInput) (*featureUsageResponse, error) {
	// Validation
	if err := in.Validate(); err != nil {
		return nil, err
	}

	meterQueryParams := streaming.QueryParams{
		FilterCustomer: []streaming.Customer{in.Customer},
		From:           &in.Line.Period.Start,
		To:             &in.Line.Period.End,
		FilterGroupBy:  in.Feature.MeterGroupByFilters,
	}

	lineHierarchy := in.Line.SplitLineHierarchy

	// If we are the first line in the split, we don't need to calculate the pre period
	if lineHierarchy == nil || lineHierarchy.Group.ServicePeriod.Start.Equal(in.Line.Period.Start) {
		meterValues, err := s.StreamingConnector.QueryMeter(
			ctx,
			in.Line.Namespace,
			in.Meter,
			meterQueryParams,
		)
		if err != nil {
			return nil, fmt.Errorf("querying line[%s] meter[%s]: %w", in.Line.ID, in.Meter.Key, err)
		}

		return &featureUsageResponse{
			LinePeriodQty: summarizeMeterQueryRow(meterValues),
		}, nil
	}

	// Let's calculate [parent.start ... line.start] values
	preLineQuery := meterQueryParams
	preLineQuery.From = &lineHierarchy.Group.ServicePeriod.Start
	preLineQuery.To = &in.Line.Period.Start

	preLineResult, err := s.StreamingConnector.QueryMeter(
		ctx,
		in.Line.Namespace,
		in.Meter,
		preLineQuery,
	)
	if err != nil {
		return nil, fmt.Errorf("querying pre line[%s] period meter[%s]: %w", in.Line.ID, in.Meter.Key, err)
	}

	preLineQty := summarizeMeterQueryRow(preLineResult)

	// Let's calculate [parent.start ... line.end] values
	upToLineEnd := meterQueryParams
	upToLineEnd.From = &lineHierarchy.Group.ServicePeriod.Start
	upToLineEnd.To = &in.Line.Period.End

	upToLineEndResult, err := s.StreamingConnector.QueryMeter(
		ctx,
		in.Line.Namespace,
		in.Meter,
		upToLineEnd,
	)
	if err != nil {
		return nil, fmt.Errorf("querying up to line[%s] end meter[%s]: %w", in.Line.ID, in.Meter.Key, err)
	}

	upToLineQty := summarizeMeterQueryRow(upToLineEndResult)

	return &featureUsageResponse{
		LinePeriodQty:    upToLineQty.Sub(preLineQty),
		PreLinePeriodQty: preLineQty,
	}, nil
}

func summarizeMeterQueryRow(in []meter.MeterQueryRow) alpacadecimal.Decimal {
	sum := alpacadecimal.Decimal{}
	for _, row := range in {
		sum = sum.Add(alpacadecimal.NewFromFloat(row.Value))
	}

	return sum
}
