package ingest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/samber/lo"
)

// NewService creates a new ingestion service.
func NewService(collector Collector, logger *slog.Logger, batchSize int) Service {
	return &service{
		collector: collector,
		logger:    logger,
		batchSize: batchSize,
	}
}

// Service implements the ingestion service.
type service struct {
	collector Collector
	logger    *slog.Logger
	batchSize int
}

// IngestEventsRequest is the request for ingesting events.
type IngestEventsRequest struct {
	Namespace string
	Events    []event.Event
}

// IngestEvents ingests events.
func (s service) IngestEvents(ctx context.Context, request IngestEventsRequest) (bool, error) {
	if len(request.Events) == 1 {
		return true, s.processEvent(ctx, request.Events[0], request.Namespace)
	}

	// Split events into chunks of size s.batchSize and process events in chunks in parallel
	chunks := lo.Chunk(request.Events, s.batchSize)

	for _, chunk := range chunks {
		wg := sync.WaitGroup{}
		wg.Add(len(chunk))
		chErr := make(chan error, len(chunk))

		for _, ev := range chunk {
			go func(ev event.Event, wg *sync.WaitGroup) {
				defer wg.Done()

				err := s.processEvent(ctx, ev, request.Namespace)
				if err != nil {
					chErr <- err
				}
			}(ev, &wg)
		}

		wg.Wait()
		close(chErr)

		var errs []error

		for err := range chErr {
			errs = append(errs, err)
		}

		if len(errs) > 0 {
			return false, errors.Join(errs...)
		}
	}

	return true, nil
}

// processEvent processes a single event.
func (s service) processEvent(ctx context.Context, event event.Event, namespace string) error {
	logger := s.logger.With(
		slog.String("event_id", event.ID()),
		slog.String("event_subject", event.Subject()),
		slog.String("event_source", event.Source()),
		slog.String("namespace", namespace),
	)

	if event.Time().IsZero() {
		logger.DebugContext(ctx, "event does not have a timestamp")

		event.SetTime(time.Now().UTC())
	} else {
		event.SetTime(event.Time().UTC())
	}

	err := s.collector.Ingest(ctx, namespace, event)
	if err != nil {
		// TODO: attach context to error and log at a higher level
		logger.ErrorContext(ctx, "unable to forward event to collector", "error", err)

		return fmt.Errorf("forwarding event to collector: %w", err)
	}

	logger.DebugContext(ctx, "event forwarded to downstream collector")

	return nil
}
