package meterexportservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/meterexport"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/clock"
)

// GetTargetMeterDescriptor validates the export config and returns the descriptor for the target meter.
func (s *service) GetTargetMeterDescriptor(ctx context.Context, config meterexport.DataExportConfig) (meterexport.TargetMeterDescriptor, error) {
	_, descriptor, err := s.validateAndGetMeter(ctx, config)
	return descriptor, err
}

// validateAndGetMeter validates the config, fetches the meter, and returns both the meter and descriptor.
// This is used internally by export methods to avoid duplicating validation logic.
func (s *service) validateAndGetMeter(ctx context.Context, config meterexport.DataExportConfig) (meter.Meter, meterexport.TargetMeterDescriptor, error) {
	if err := config.Validate(); err != nil {
		return meter.Meter{}, meterexport.TargetMeterDescriptor{}, fmt.Errorf("validate config: %w", err)
	}

	m, err := s.MeterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
		IDOrSlug:  config.MeterID.ID,
		Namespace: config.MeterID.Namespace,
	})
	if err != nil {
		return meter.Meter{}, meterexport.TargetMeterDescriptor{}, fmt.Errorf("get meter: %w", err)
	}

	// Validate the meter aggregation
	switch m.Aggregation {
	case meter.MeterAggregationSum, meter.MeterAggregationCount:
	default:
		return meter.Meter{}, meterexport.TargetMeterDescriptor{}, fmt.Errorf("unsupported meter aggregation: %s", m.Aggregation)
	}

	descriptor := meterexport.TargetMeterDescriptor{
		Aggregation:   meter.MeterAggregationSum,
		EventType:     m.EventType,
		ValueProperty: lo.ToPtr(SUM_VALUE_PROPERTY_KEY),
	}

	return m, descriptor, nil
}

func (s *service) ExportSyntheticMeterData(ctx context.Context, params meterexport.DataExportParams, resultCh chan<- streaming.RawEvent, errCh chan<- error) error {
	defer func() {
		close(resultCh)
		close(errCh)
	}()

	m, _, err := s.validateAndGetMeter(ctx, params.DataExportConfig)
	if err != nil {
		return err
	}

	// We're gonna do some things in parallel here
	meterRowCh := make(chan meter.MeterQueryRow, 1000)
	meterRowErrCh := make(chan error, 10)

	g, ctx := errgroup.WithContext(ctx)

	// Ensure context cancellation error is only sent once
	// (both funnel and consumer can detect context cancellation)
	var sendCtxErrOnce sync.Once
	sendCtxErr := func() {
		sendCtxErrOnce.Do(func() {
			if err := ctx.Err(); err != nil {
				errCh <- err
			}
		})
	}

	// Let's start consuming the rows
	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				sendCtxErr()
				return nil
			case err, ok := <-meterRowErrCh:
				// Filter out context errors as they're handled via sendCtxErr
				if ok && err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
					errCh <- err
				}
			case row, ok := <-meterRowCh:
				if !ok {
					// Before returning, check if context was canceled
					// This ensures we always report context cancellation to the caller
					sendCtxErr()
					return nil
				}

				event, err := s.createEventFromMeterRow(m, row)
				if err != nil {
					errCh <- fmt.Errorf("create event from meter row: %w", err)

					// Is this error critical from our perspective?
					// The caller should be able to determine if skipping rows is acceptable or not.
					continue
				}

				resultCh <- event
			}
		}
	})

	// Then let's start producing them
	g.Go(func() error {
		return s.funnel(ctx, funnelParams{
			meter: m,
			queryParams: streaming.QueryParams{
				From:           &params.Period.From,
				To:             params.Period.To,
				WindowSize:     &params.ExportWindowSize,
				WindowTimeZone: params.ExportWindowTimeZone,
				GroupBy:        []string{"subject"},
			},
		}, meterRowCh, meterRowErrCh)
	})

	if err := g.Wait(); err != nil {
		return fmt.Errorf("export synthetic meter data: %w", err)
	}

	return nil
}

const SUM_VALUE_PROPERTY_KEY = "value"

func (s *service) createEventFromMeterRow(m meter.Meter, row meter.MeterQueryRow) (streaming.RawEvent, error) {
	// For SUM and COUNT type source meters, all event rows can be represented as SUM meter events
	baseEvent := streaming.RawEvent{
		Namespace:  m.Namespace,
		ID:         ulid.Make().String(),
		Type:       m.EventType, // We reuse the same type as the source meter
		Source:     fmt.Sprintf("%s:%s/%s", s.EventSourceGroup, m.Namespace, m.ID),
		Subject:    lo.FromPtr(row.Subject),
		IngestedAt: clock.Now(),
		Time:       row.WindowStart,
		CustomerID: nil,
	}

	// Let's add the value data to the event
	switch m.Aggregation {
	case meter.MeterAggregationCount, meter.MeterAggregationSum:
		data := map[string]interface{}{
			SUM_VALUE_PROPERTY_KEY: row.Value,
		}

		dataBytes, err := json.Marshal(data)
		if err != nil {
			return baseEvent, fmt.Errorf("marshal data: %w", err)
		}

		baseEvent.Data = string(dataBytes)
	default:
		return baseEvent, fmt.Errorf("unsupported meter aggregation: %s", m.Aggregation)
	}

	return baseEvent, nil
}
