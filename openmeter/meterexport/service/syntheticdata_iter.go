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

		// Channel to capture any startup error from ExportSyntheticMeterData
		startupErrCh := make(chan error, 1)

		// Start the export in a goroutine
		go func() {
			if err := s.ExportSyntheticMeterData(ctx, config, resultCh, errCh); err != nil {
				startupErrCh <- err
			}
			close(startupErrCh)
		}()

		// Interleave results and errors
		for {
			select {
			case err := <-startupErrCh:
				// Startup error from ExportSyntheticMeterData (should be rare given upfront validation)
				if err != nil {
					yield(streaming.RawEvent{}, err)
				}
				return
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
