package ingest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Service interface {
	IngestEvents(ctx context.Context, request IngestEventsRequest) (bool, error)
}

type Config struct {
	Collector   Collector
	Logger      *slog.Logger
	MetricMeter metric.Meter
}

func (c Config) Validate() error {
	var errs []error

	if c.Collector == nil {
		errs = append(errs, errors.New("collector is required"))
	}

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	if c.MetricMeter == nil {
		errs = append(errs, errors.New("metric meter is required"))
	}

	return errors.Join(errs...)
}

// service implements the ingestion service.
type service struct {
	collector               Collector
	logger                  *slog.Logger
	requestEventCountMetric metric.Int64Histogram
}

func NewService(config Config) (Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	requestEventCountMetric, err := config.MetricMeter.Int64Histogram(
		"openmeter.ingest.request.events",
		metric.WithDescription("Number of events in an ingest request"),
		metric.WithUnit("{event}"),
		// The SDK default boundaries are latency-shaped: they start at 0 and jump
		// straight to 5. In the deployments we have observed, the large majority of
		// ingest requests carry between one and five events, so that first bucket
		// swallows the whole distribution and the reported percentiles become
		// interpolation rather than measurement. Resolve one through five exactly
		// and keep a coarse tail for clients that batch.
		metric.WithExplicitBucketBoundaries(0, 1, 2, 3, 4, 5, 10, 25, 100, 1000, 10000),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request event count histogram: %w", err)
	}

	return &service{
		collector:               config.Collector,
		logger:                  config.Logger,
		requestEventCountMetric: requestEventCountMetric,
	}, nil
}

type IngestEventsRequest struct {
	Namespace string
	Events    []event.Event
}

func (s service) IngestEvents(ctx context.Context, request IngestEventsRequest) (bool, error) {
	s.requestEventCountMetric.Record(ctx, int64(len(request.Events)), metric.WithAttributes(
		attribute.String("namespace", request.Namespace),
	))

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
