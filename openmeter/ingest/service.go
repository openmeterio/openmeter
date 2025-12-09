package ingest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
)

type Service interface {
	IngestEvents(ctx context.Context, request IngestEventsRequest) (bool, error)
}

type Config struct {
	Collector Collector
	Logger    *slog.Logger
}

func (c Config) Validate() error {
	var errs []error

	if c.Collector == nil {
		errs = append(errs, errors.New("collector is required"))
	}

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	return errors.Join(errs...)
}

// service implements the ingestion service.
type service struct {
	collector Collector
	logger    *slog.Logger
}

func NewService(config Config) (Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{
		collector: config.Collector,
		logger:    config.Logger,
	}, nil
}

type IngestEventsRequest struct {
	Namespace string
	Events    []event.Event
}

func (s service) IngestEvents(ctx context.Context, request IngestEventsRequest) (bool, error) {
	for _, ev := range request.Events {
		err := s.processEvent(ctx, ev, request.Namespace)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

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
