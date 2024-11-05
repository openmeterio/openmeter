package lineservice

import (
	"context"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"

	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

type getFeatureUsageInput struct {
	Line       *billingentity.Line
	ParentLine *billingentity.Line
	Meter      models.Meter
	Feature    feature.Feature
	Subjects   []string
}

func (i getFeatureUsageInput) Validate() error {
	if i.Line == nil {
		return fmt.Errorf("line is required")
	}

	if i.Line.ParentLineID != nil && i.ParentLine == nil {
		return fmt.Errorf("parent line is required for split lines")
	}

	if i.Line.ParentLineID == nil && i.ParentLine != nil {
		return fmt.Errorf("parent line is not allowed for non-split lines")
	}

	if i.ParentLine != nil {
		if i.ParentLine.Status != billingentity.InvoiceLineStatusSplit {
			return fmt.Errorf("parent line is not split")
		}
	}

	if slices.Contains([]models.MeterAggregation{
		models.MeterAggregationAvg,
		models.MeterAggregationMin,
	}, i.Meter.Aggregation) {
		if i.ParentLine != nil {
			return fmt.Errorf("aggregation %s is not supported for split lines", i.Meter.Aggregation)
		}
	}

	if len(i.Subjects) == 0 {
		return fmt.Errorf("subjects are required")
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

	meterQueryParams := &streaming.QueryParams{
		Aggregation:   in.Meter.Aggregation,
		FilterSubject: in.Subjects,
		From:          &in.Line.Period.Start,
		To:            &in.Line.Period.End,
	}

	if in.Feature.MeterGroupByFilters != nil {
		meterQueryParams.FilterGroupBy = map[string][]string{}
		for k, v := range in.Feature.MeterGroupByFilters {
			meterQueryParams.FilterGroupBy[k] = []string{v}
		}
	}

	meterValues, err := s.StreamingConnector.QueryMeter(
		ctx,
		in.Line.Namespace,
		in.Meter.Slug,
		meterQueryParams,
	)
	if err != nil {
		return nil, fmt.Errorf("querying line[%s] meter[%s]: %w", in.Line.ID, in.Meter.Slug, err)
	}

	res := &featureUsageResponse{
		LinePeriodQty: summarizeMeterQueryRow(meterValues),
	}

	// If we are the first line in the split, we don't need to calculate the pre period
	if in.ParentLine == nil || in.ParentLine.Period.Start.Equal(in.Line.Period.Start) {
		return res, nil
	}

	// Let's get the usage for the parent line to calculate the pre period
	meterQueryParams.From = &in.ParentLine.Period.Start

	meterValues, err = s.StreamingConnector.QueryMeter(
		ctx,
		in.Line.Namespace,
		in.Meter.Slug,
		meterQueryParams,
	)
	if err != nil {
		return nil, fmt.Errorf("querying parent line[%s] meter[%s]: %w", in.ParentLine.ID, in.Meter.Slug, err)
	}

	fullPeriodQty := summarizeMeterQueryRow(meterValues)
	res.PreLinePeriodQty = fullPeriodQty.Sub(res.LinePeriodQty)

	return res, nil
}

func summarizeMeterQueryRow(in []models.MeterQueryRow) alpacadecimal.Decimal {
	sum := alpacadecimal.Decimal{}
	for _, row := range in {
		sum = sum.Add(alpacadecimal.NewFromFloat(row.Value))
	}

	return sum
}
