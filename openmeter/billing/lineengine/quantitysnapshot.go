package lineengine

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
	"github.com/openmeterio/openmeter/pkg/ref"
)

type SnapshotLineQuantityInput struct {
	Invoice *billing.StandardInvoice
	Line    *billing.StandardLine
}

func (i SnapshotLineQuantityInput) Validate() error {
	var errs []error
	if i.Invoice == nil {
		errs = append(errs, errors.New("invoice is required"))
	}

	if i.Line == nil {
		errs = append(errs, errors.New("line is required"))
	}

	return errors.Join(errs...)
}

// SnapshotLineQuantity snapshots the quantity of a standard line, this is an external API for invoice updater as needed.
// This is only needed for the invoiceupdater to try to minimize unnecessary immutable invoice updates, but generally should be an internal affair
// of the lineengine itself.
func (e *Engine) SnapshotLineQuantity(ctx context.Context, input SnapshotLineQuantityInput) (*billing.StandardLine, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	linesWithHierarchy, err := e.ResolveSplitLineGroupHeaders(ctx, input.Invoice.Namespace, billing.StandardLines{input.Line})
	if err != nil {
		return nil, fmt.Errorf("resolving split line group headers: %w", err)
	}

	featureMeters, err := e.resolveFeatureMeters(ctx, input.Invoice.Namespace, linesWithHierarchy)
	if err != nil {
		return nil, fmt.Errorf("line[%s]: %w", input.Line.ID, err)
	}

	if len(linesWithHierarchy) != 1 {
		return nil, fmt.Errorf("expected 1 line with hierarchy, got %d [line_id: %s]", len(linesWithHierarchy), input.Line.ID)
	}

	err = e.snapshotLineQuantity(ctx, input.Invoice.Customer, linesWithHierarchy[0], featureMeters)
	if err != nil {
		return nil, err
	}

	return linesWithHierarchy[0].StandardLine, nil
}

func (e *Engine) SnapshotLineQuantities(ctx context.Context, invoice billing.StandardInvoice, lines StandardLinesWithSplitLineHierarchy) error {
	featureMeters, err := e.resolveFeatureMeters(ctx, invoice.Namespace, lines)
	if err != nil {
		return fmt.Errorf("resolving feature meters: %w", err)
	}

	if err := e.snapshotLineQuantitiesInParallel(ctx, invoice.Customer, lines, featureMeters); err != nil {
		return fmt.Errorf("snapshotting lines: %w", err)
	}

	return nil
}

func (e *Engine) resolveFeatureMeters(ctx context.Context, namespace string, lines StandardLinesWithSplitLineHierarchy) (feature.FeatureMeters, error) {
	keys, err := lines.GetReferencedFeatureKeys()
	if err != nil {
		return nil, fmt.Errorf("getting referenced feature keys: %w", err)
	}

	featureMeters, err := e.featureService.ResolveFeatureMeters(ctx, namespace, lo.Map(keys, func(key string, _ int) ref.IDOrKey {
		return ref.IDOrKey{Key: key}
	})...)
	if err != nil {
		return nil, fmt.Errorf("resolving feature meters: %w", err)
	}

	return featureMetersErrorWrapper{featureMeters}, nil
}

// featureMetersErrorWrapper returns ErrSnapshotInvalidDatabaseState when a feature meter is unavailable during snapshotting.
type featureMetersErrorWrapper struct {
	feature.FeatureMeters
}

func (w featureMetersErrorWrapper) Get(featureKey string, requireMeter bool) (feature.FeatureMeter, error) {
	featureMeter, err := w.FeatureMeters.Get(featureKey, requireMeter)
	if err != nil {
		return feature.FeatureMeter{}, &billing.ErrSnapshotInvalidDatabaseState{
			Err: err,
		}
	}

	return featureMeter, nil
}

func (e *Engine) snapshotMeteredLineQuantity(ctx context.Context, line StandardLineWithSplitLineHierarchy, customer billing.InvoiceCustomer, featureMeters feature.FeatureMeters) error {
	featureMeter, err := featureMeters.Get(line.UsageBased.FeatureKey, true)
	if err != nil {
		return err
	}

	usage, err := e.getFeatureUsage(ctx,
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

func (e *Engine) snapshotFlatPriceLineQuantity(_ context.Context, line StandardLineWithSplitLineHierarchy) error {
	line.UsageBased.MeteredQuantity = lo.ToPtr(alpacadecimal.NewFromInt(1))
	line.UsageBased.Quantity = lo.ToPtr(alpacadecimal.NewFromInt(1))
	line.UsageBased.PreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
	line.UsageBased.MeteredPreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
	return nil
}

func (e *Engine) snapshotLineQuantity(ctx context.Context, customer billing.InvoiceCustomer, line StandardLineWithSplitLineHierarchy, featureMeters feature.FeatureMeters) error {
	if !line.DependsOnMeteredQuantity() {
		return e.snapshotFlatPriceLineQuantity(ctx, line)
	}

	return e.snapshotMeteredLineQuantity(ctx, line, customer, featureMeters)
}

func (e *Engine) snapshotLineQuantitiesInParallel(ctx context.Context, customer billing.InvoiceCustomer, lines StandardLinesWithSplitLineHierarchy, featureMeters feature.FeatureMeters) error {
	workerCount := e.maxParallelQuantitySnapshots
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

			err = e.snapshotLineQuantity(ctx, customer, line, featureMeters)
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
	Line     StandardLineWithSplitLineHierarchy
	Meter    meter.Meter
	Feature  feature.Feature
	Customer billing.InvoiceCustomer
}

func (i getFeatureUsageInput) Validate() error {
	if i.Line.StandardLine == nil {
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
	// TODO[BeforeMerge]: Make sure we have a test for this.

	if i.Line.SplitLineGroupID != nil && i.Line.SplitLineHierarchy == nil {
		return fmt.Errorf("split line group id is set but split line hierarchy is not expanded")
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

func (e *Engine) getFeatureUsage(ctx context.Context, in getFeatureUsageInput) (*featureUsageResponse, error) {
	// Validation
	if err := in.Validate(); err != nil {
		return nil, err
	}

	meterQueryParams := streaming.QueryParams{
		FilterCustomer: []streaming.Customer{in.Customer},
		From:           &in.Line.Period.From,
		To:             &in.Line.Period.To,
		FilterGroupBy:  in.Feature.MeterGroupByFilters,
	}

	lineHierarchy := in.Line.SplitLineHierarchy

	// If we are the first line in the split, we don't need to calculate the pre period
	if lineHierarchy == nil || lineHierarchy.Group.ServicePeriod.From.Equal(in.Line.Period.From) {
		meterValues, err := e.streamingConnector.QueryMeter(
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
	preLineQuery.From = &lineHierarchy.Group.ServicePeriod.From
	preLineQuery.To = &in.Line.Period.From

	preLineResult, err := e.streamingConnector.QueryMeter(
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
	upToLineEnd.From = &lineHierarchy.Group.ServicePeriod.From
	upToLineEnd.To = &in.Line.Period.To

	upToLineEndResult, err := e.streamingConnector.QueryMeter(
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
