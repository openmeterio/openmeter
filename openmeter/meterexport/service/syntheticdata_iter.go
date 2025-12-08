package meterexportservice

import (
	"context"
	"fmt"
	"iter"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/meterexport"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// ExportSyntheticMeterDataIter wraps ExportSyntheticMeterData with an iterator interface.
// If the caller stops iterating early, the underlying operation is canceled.
func (s *service) ExportSyntheticMeterDataIter(ctx context.Context, config meterexport.DataExportConfig) (meterexport.TargetMeterDescriptor, iter.Seq2[streaming.RawEvent, error], error) {
	// Validate config upfront so we can return an error before creating the iterator
	if err := config.Validate(); err != nil {
		return meterexport.TargetMeterDescriptor{}, nil, fmt.Errorf("validate config: %w", err)
	}

	// Get meter upfront to validate and return descriptor synchronously
	m, err := s.MeterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
		IDOrSlug:  config.MeterID.ID,
		Namespace: config.MeterID.Namespace,
	})
	if err != nil {
		return meterexport.TargetMeterDescriptor{}, nil, fmt.Errorf("get meter: %w", err)
	}

	// Validate meter aggregation upfront
	switch m.Aggregation {
	case meter.MeterAggregationSum, meter.MeterAggregationCount:
	default:
		return meterexport.TargetMeterDescriptor{}, nil, fmt.Errorf("unsupported meter aggregation: %s", m.Aggregation)
	}

	descriptor := meterexport.TargetMeterDescriptor{
		Aggregation:   meter.MeterAggregationSum,
		EventType:     m.EventType,
		ValueProperty: lo.ToPtr(SUM_VALUE_PROPERTY_KEY),
	}

	seq := func(yield func(streaming.RawEvent, error) bool) {
		// Create a cancellable context so we can stop the operation if the caller breaks early
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		resultCh := make(chan streaming.RawEvent, 100)
		errCh := make(chan error, 10)

		// Start the export in a goroutine
		go func() {
			// We ignore the returned descriptor since we already have it
			_, _ = s.ExportSyntheticMeterData(ctx, config, resultCh, errCh)
		}()

		// Interleave results and errors
		for {
			select {
			case event, ok := <-resultCh:
				if !ok {
					// Results channel closed, drain remaining errors
					for err := range errCh {
						if !yield(streaming.RawEvent{}, err) {
							return
						}
					}
					return
				}
				if !yield(event, nil) {
					return // Caller stopped iterating, context will be canceled by defer
				}
			case err, ok := <-errCh:
				if !ok {
					// Error channel closed, drain remaining results
					for event := range resultCh {
						if !yield(event, nil) {
							return
						}
					}
					return
				}
				if !yield(streaming.RawEvent{}, err) {
					return // Caller stopped iterating
				}
			}
		}
	}

	return descriptor, seq, nil
}
