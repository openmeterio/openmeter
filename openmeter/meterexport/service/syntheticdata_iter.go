package meterexportservice

import (
	"context"
	"iter"

	"github.com/openmeterio/openmeter/openmeter/meterexport"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// ExportSyntheticMeterDataIter wraps ExportSyntheticMeterData with an iterator interface.
// If the caller stops iterating early, the underlying operation is canceled.
func (s *service) ExportSyntheticMeterDataIter(ctx context.Context, config meterexport.DataExportConfig) (iter.Seq2[streaming.RawEvent, error], error) {
	// Validate upfront so we can return an error before creating the iterator
	if _, _, err := s.validateAndGetMeter(ctx, config); err != nil {
		return nil, err
	}

	seq := func(yield func(streaming.RawEvent, error) bool) {
		// Create a cancellable context so we can stop the operation if the caller breaks early
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		resultCh := make(chan streaming.RawEvent, 100)
		errCh := make(chan error, 10)

		// Start the export in a goroutine
		go func() {
			_ = s.ExportSyntheticMeterData(ctx, config, resultCh, errCh)
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

	return seq, nil
}
