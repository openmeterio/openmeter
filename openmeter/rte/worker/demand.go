package worker

import (
	"context"
	"sync"
	"time"
)

// pollDemands starts the configured background polling workers and blocks until
// cancellation and all in-flight attempts have finished. Each worker drains ready
// demand, waiting pollInterval after an empty or failed claim.
func (w *worker) pollDemands(ctx context.Context) {
	var workers sync.WaitGroup
	for range w.calculationConcurrency {
		workers.Go(func() {
			for ctx.Err() == nil {
				// PostgreSQL holds the claim through calculation and completion;
				// SKIP LOCKED excludes keys held by other slots or replicas.
				processed, err := w.demand.Process(ctx)
				if err != nil && ctx.Err() == nil {
					w.logger.ErrorContext(ctx, "process demanded calculation", "error", err)
				}

				// A failed calculation with a recorded retry still counts as
				// processed work. Drain other ready keys without waiting.
				if processed {
					continue
				}

				// Start the idle wait after the empty or failed claim. A ticker
				// could leave a tick queued during a long calculation.
				timer := time.NewTimer(w.pollInterval)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
			}
		})
	}

	// Process receives cancellation too; wait for in-flight transactions to finish.
	workers.Wait()
}
