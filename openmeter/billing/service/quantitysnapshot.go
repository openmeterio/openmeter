package billingservice

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"golang.org/x/sync/semaphore"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

func (s *Service) SnapshotLineQuantity(ctx context.Context, input billing.SnapshotLineQuantityInput) (*billing.StandardLine, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	featureMeters, err := s.resolveFeatureMeters(ctx, input.Line.Namespace, billing.StandardLines{input.Line})
	if err != nil {
		return nil, fmt.Errorf("line[%s]: %w", input.Line.ID, err)
	}

	err = s.snapshotLineQuantity(ctx, input.Invoice.Customer, input.Line, featureMeters)
	if err != nil {
		return nil, err
	}

	return input.Line, nil
}

func (s *Service) snapshotMeteredLineQuantity(ctx context.Context, line *billing.StandardLine, customer billing.InvoiceCustomer, featureMeters billing.FeatureMeters) error {
	featureMeter, err := featureMeters.Get(line.UsageBased.FeatureKey, true)
	if err != nil {
		return err
	}

	usage, err := s.getFeatureUsage(ctx,
		getFeatureUsageInput{
			Line:     line,
			Feature:  featureMeter.Feature,
			Meter:    *featureMeter.Meter,
			Customer: customer,
		},
	)
	if err != nil {
		return err
	}

	// MeteredQuantity is not mutable by the price mutators, that's why we have this redundancy
	line.UsageBased.MeteredQuantity = lo.ToPtr(usage.LinePeriodQty)
	line.UsageBased.Quantity = lo.ToPtr(usage.LinePeriodQty)
	line.UsageBased.PreLinePeriodQuantity = lo.ToPtr(usage.PreLinePeriodQty)
	line.UsageBased.MeteredPreLinePeriodQuantity = lo.ToPtr(usage.PreLinePeriodQty)
	return nil
}

func (s *Service) snapshotFlatPriceLineQuantity(_ context.Context, line *billing.StandardLine) error {
	line.UsageBased.MeteredQuantity = lo.ToPtr(alpacadecimal.NewFromInt(1))
	line.UsageBased.Quantity = lo.ToPtr(alpacadecimal.NewFromInt(1))
	line.UsageBased.PreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
	line.UsageBased.MeteredPreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
	return nil
}

func (s *Service) snapshotLineQuantity(ctx context.Context, customer billing.InvoiceCustomer, line *billing.StandardLine, featureMeters billing.FeatureMeters) error {
	if !line.DependsOnMeteredQuantity() {
		return s.snapshotFlatPriceLineQuantity(ctx, line)
	}

	return s.snapshotMeteredLineQuantity(ctx, line, customer, featureMeters)
}

func (s *Service) snapshotLineQuantitiesInParallel(ctx context.Context, customer billing.InvoiceCustomer, lines billing.StandardLines, featureMeters billing.FeatureMeters) error {
	workerCount := s.maxParallelQuantitySnapshots
	if workerCount <= 0 {
		workerCount = 1
	}

	sem := semaphore.NewWeighted(int64(workerCount))

	errCh := make(chan error, len(lines))

	var wg sync.WaitGroup

	for _, line := range lines {
		err := sem.Acquire(ctx, 1)
		if err != nil {
			// Clean up and stop the loop
			errCh <- fmt.Errorf("acquiring worker slot: %w", err)
			break
		}

		wg.Go(func() {
			defer sem.Release(1)

			var err error
			defer func() {
				if err != nil {
					errCh <- err
				}
			}()

			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("snapshotting line quantity: %v", r)
				}
			}()

			err = s.snapshotLineQuantity(ctx, customer, line, featureMeters)
		})
	}

	wg.Wait()

	close(errCh)

	var errs []error

	for err := range errCh {
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

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

	if slices.Contains([]meter.MeterAggregation{
		meter.MeterAggregationAvg,
		meter.MeterAggregationMin,
	}, i.Meter.Aggregation) {
		if i.Line.SplitLineHierarchy != nil {
			return fmt.Errorf("aggregation %s is not supported for split lines", i.Meter.Aggregation)
		}
	}

	// TODO[OM-160]: We need to have this check to make sure that usage discounts are properly accounted for
	// but we seem to have a bug in syncing progressively billed lines, so let's address this as a separate pr.

	// if i.Line.SplitLineGroupID != nil && i.Line.SplitLineHierarchy == nil {
	// 	return fmt.Errorf("split line group id is set but split line hierarchy is not expanded")
	// }

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
		meterValues, err := s.streamingConnector.QueryMeter(
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

	preLineResult, err := s.streamingConnector.QueryMeter(
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

	upToLineEndResult, err := s.streamingConnector.QueryMeter(
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
	sum := alpacadecimal.Zero
	for _, row := range in {
		sum = sum.Add(alpacadecimal.NewFromFloat(row.Value))
	}

	return sum
}
