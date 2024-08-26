package ingest

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
)

// Service implements the ingestion service.
type Service struct {
	Collector Collector
	Logger    *slog.Logger
}

type IngestEventsRequest struct {
	Namespace string
	Events    []event.Event
}

func (s Service) IngestEvents(ctx context.Context, request IngestEventsRequest) (bool, error) {
	for _, ev := range request.Events {
		err := s.processEvent(ctx, ev, request.Namespace)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

func (s Service) processEvent(ctx context.Context, event event.Event, namespace string) error {
	logger := s.Logger.With(
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

	err := s.Collector.Ingest(ctx, namespace, event)
	if err != nil {
		// TODO: attach context to error and log at a higher level
		logger.ErrorContext(ctx, "unable to forward event to collector", "error", err)

		return fmt.Errorf("forwarding event to collector: %w", err)
	}

	logger.DebugContext(ctx, "event forwarded to downstream collector")

	return nil
}
