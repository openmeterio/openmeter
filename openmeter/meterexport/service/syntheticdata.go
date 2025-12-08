package meterexportservice

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/meterexport"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

func (s *service) ExportSyntheticMeterData(ctx context.Context, config meterexport.DataExportConfig, resultCh chan<- streaming.RawEvent, errCh chan<- error) (meterexport.TargetMeterDescriptor, error) {
	defer func() {
		close(resultCh)
		close(errCh)
	}()

	if err := config.Validate(); err != nil {
		return meterexport.TargetMeterDescriptor{}, fmt.Errorf("validate config: %w", err)
	}

	m, err := s.MeterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
		IDOrSlug:  config.MeterID.ID,
		Namespace: config.MeterID.Namespace,
	})
	if err != nil {
		return meterexport.TargetMeterDescriptor{}, fmt.Errorf("get meter: %w", err)
	}

	// Let's validate the meter aggregation upfront
	switch m.Aggregation {
	case meter.MeterAggregationSum, meter.MeterAggregationCount:
	default:
		return meterexport.TargetMeterDescriptor{}, fmt.Errorf("unsupported meter aggregation: %s", m.Aggregation)
	}

	// We're gonna do some things in parallel here
	meterRowCh := make(chan meter.MeterQueryRow, 1000)
	meterRowErrCh := make(chan error, 10)

	g, ctx := errgroup.WithContext(ctx)

	// Let's start consuming the rows
	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return nil
			case err := <-meterRowErrCh:
				errCh <- err
			case row, ok := <-meterRowCh:
				if !ok {
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
		}, meterRowCh, errCh)
	})

	if err := g.Wait(); err != nil {
		return meterexport.TargetMeterDescriptor{}, fmt.Errorf("export synthetic meter data: %w", err)
	}

	return meterexport.TargetMeterDescriptor{
		Aggregation:   meter.MeterAggregationSum,
		EventType:     m.EventType,
		ValueProperty: lo.ToPtr(SUM_VALUE_PROPERTY_KEY),
	}, nil
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
		Time:       row.WindowStart,
		CustomerID: row.CustomerID,
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
