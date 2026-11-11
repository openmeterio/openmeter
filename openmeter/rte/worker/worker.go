package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	"github.com/openmeterio/openmeter/openmeter/rte/upstream"
	ingestevents "github.com/openmeterio/openmeter/openmeter/sink/flushhandler/ingestnotification/events"
	"github.com/openmeterio/openmeter/openmeter/watermill/grouphandler"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Config struct {
	Service           rte.Service
	SystemEventsTopic string
	IngestEventsTopic string
	Demand            rte.DemandService
	Sources           rte.UpstreamResolver
	PollInterval      time.Duration
	Marshaler         marshaler.Marshaler
	Router            router.Options
}

func (c Config) Validate() error {
	var errs []error
	if c.Service == nil {
		errs = append(errs, errors.New("service is required"))
	}

	if c.Demand == nil || c.Sources == nil {
		errs = append(errs, errors.New("demand service and upstream resolver are required"))
	}

	if c.IngestEventsTopic == "" || c.IngestEventsTopic == c.SystemEventsTopic {
		errs = append(errs, errors.New("ingest events require a separate topic"))
	}

	if c.PollInterval <= 0 {
		errs = append(errs, errors.New("poll interval must be positive"))
	}

	if c.SystemEventsTopic == "" {
		errs = append(errs, errors.New("system events topic is required"))
	}

	if c.Marshaler == nil {
		errs = append(errs, errors.New("marshaler is required"))
	}

	if err := c.Router.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("router: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type worker struct {
	service      rte.Service
	router       *message.Router
	demand       rte.DemandService
	sources      rte.UpstreamResolver
	pollInterval time.Duration
	logger       *slog.Logger
	sourceEvents metric.Int64Counter
	sourceLag    metric.Float64Histogram
}

func New(config Config) (rte.Worker, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	r, err := router.NewDefaultRouter(config.Router)
	if err != nil {
		return nil, err
	}

	sourceEvents, err := config.Router.MetricMeter.Int64Counter("rte.upstream.events", metric.WithDescription("Supported source events by routing outcome"))
	if err != nil {
		return nil, err
	}

	sourceLag, err := config.Router.MetricMeter.Float64Histogram("rte.upstream.lag", metric.WithUnit("s"), metric.WithDescription("Source publication to RTE processing delay"))
	if err != nil {
		return nil, err
	}

	w := &worker{
		service: config.Service, router: r, demand: config.Demand, sources: config.Sources,
		pollInterval: config.PollInterval, logger: config.Router.Logger,
		sourceEvents: sourceEvents, sourceLag: sourceLag,
	}

	handler, err := grouphandler.NewNoPublishingHandler(config.Marshaler, config.Router.MetricMeter,
		grouphandler.NewGroupEventHandler(w.handleRequested),
	)
	if err != nil {
		return nil, err
	}

	handle := func(msg *message.Message) error {
		name := config.Marshaler.NameFromMessage(msg)
		if event, ok := upstream.NewEvent(name); ok {
			return w.handleSource(msg, event)
		}

		return handler.Handle(msg)
	}
	r.AddConsumerHandler("rte_system_events", config.SystemEventsTopic, config.Router.Subscriber, handle)
	r.AddConsumerHandler("rte_ingest_events", config.IngestEventsTopic, config.Router.Subscriber, handle)

	return w, nil
}

func (w *worker) handleSource(msg *message.Message, event marshaler.Event) error {
	ctx := msg.Context()
	name := attribute.String("event", event.EventName())
	outcome := "error"
	defer func() {
		w.sourceEvents.Add(ctx, 1, metric.WithAttributes(name, attribute.String("outcome", outcome)))
	}()

	envelope, err := marshaler.DecodeCloudEvent(msg, event)
	if err != nil {
		return err
	}

	if envelope.Type() != event.EventName() {
		return fmt.Errorf("event type mismatch: %q", envelope.Type())
	}

	if !envelope.Time().IsZero() {
		w.sourceLag.Record(ctx, max(0, time.Since(envelope.Time()).Seconds()), metric.WithAttributes(name))
	}

	selectors, err := w.sources.Resolve(ctx, event)
	if err != nil {
		return err
	}

	if len(selectors) == 0 {
		outcome = "no_target"
		return nil
	}

	if batch, ok := event.(*ingestevents.EventBatchedIngest); ok && slices.ContainsFunc(selectors, func(selector rte.DemandSelector) bool {
		return selector.Kind == rte.CalculationKindCreditBalance
	}) {
		if err := waitForCreditUsage(ctx, batch.StoredAt); err != nil {
			return err
		}
	}

	if err := w.demand.Invalidate(ctx, rte.InvalidateInput{
		Source: envelope.Source(), EventID: envelope.ID(), Selectors: selectors,
	}); err != nil {
		return err
	}

	outcome = "routed"

	return nil
}

func (w *worker) handleRequested(ctx context.Context, event *rte.CalculationRequestedEvent) error {
	if policy := event.Freshness; policy != nil {
		return w.demand.EnsureFresh(ctx, rte.EnsureFreshInput{
			Target:  event.CalculationTarget,
			PullTTL: time.Duration(policy.TrackForSeconds) * time.Second,
			MaxAge:  time.Duration(policy.MaxAgeSeconds) * time.Second,
		})
	}

	key, err := event.DecodeKey()
	if err != nil {
		return err
	}

	switch key := key.(type) {
	case creditbalance.Key:
		_, err = w.service.CreditBalances().Calculate(ctx, key)
	case meteredentitlement.Key:
		_, err = w.service.MeteredEntitlements().Calculate(ctx, key)
	}

	return err
}

func (w *worker) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		w.processDemand(ctx)
	}()

	err := w.router.Run(ctx)
	cancel()
	<-done

	return err
}

func (w *worker) processDemand(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for ctx.Err() == nil {
		processed, err := w.demand.Process(ctx)
		if err != nil && ctx.Err() == nil {
			w.logger.ErrorContext(ctx, "process demanded calculation", "error", err)
		}

		if processed && err == nil {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *worker) Close() error {
	return w.router.Close()
}

func (w *worker) Running() <-chan struct{} {
	return w.router.Running()
}
