package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	rtemetrics "github.com/openmeterio/openmeter/openmeter/rte/metrics"
	"github.com/openmeterio/openmeter/openmeter/rte/upstream"
	ingestevents "github.com/openmeterio/openmeter/openmeter/sink/flushhandler/ingestnotification/events"
	"github.com/openmeterio/openmeter/openmeter/watermill/grouphandler"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Config struct {
	Service                rte.Service
	SystemEventsTopic      string
	IngestEventsTopic      string
	Demand                 rte.DemandService
	Sources                rte.UpstreamResolver
	PollInterval           time.Duration
	CalculationConcurrency int
	Marshaler              marshaler.Marshaler
	Router                 router.Options
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

	if c.CalculationConcurrency <= 0 {
		errs = append(errs, errors.New("calculation concurrency must be positive"))
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
	service                rte.Service
	router                 *message.Router
	demand                 rte.DemandService
	sources                rte.UpstreamResolver
	pollInterval           time.Duration
	calculationConcurrency int
	logger                 *slog.Logger
	upstream               *rtemetrics.Upstream
	comparison             *rtemetrics.Comparison
}

func New(config Config) (rte.Worker, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	r, err := router.NewDefaultRouter(config.Router)
	if err != nil {
		return nil, err
	}

	upstreamMetrics, err := rtemetrics.NewUpstream(config.Router.MetricMeter)
	if err != nil {
		return nil, err
	}

	comparison, err := rtemetrics.NewComparison(config.Router.MetricMeter)
	if err != nil {
		return nil, err
	}

	w := &worker{
		service:                config.Service,
		router:                 r,
		demand:                 config.Demand,
		sources:                config.Sources,
		pollInterval:           config.PollInterval,
		calculationConcurrency: config.CalculationConcurrency,
		logger:                 config.Router.Logger,
		upstream:               upstreamMetrics,
		comparison:             comparison,
	}

	handler, err := grouphandler.NewNoPublishingHandler(config.Marshaler, config.Router.MetricMeter,
		grouphandler.NewGroupEventHandler(w.handleRequested),
		grouphandler.NewGroupEventHandler(w.handleComparison),
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
	name := event.EventName()
	outcome := rtemetrics.UpstreamOutcomeError
	defer func() {
		w.upstream.Event(ctx, name, outcome)
	}()

	envelope, err := marshaler.DecodeCloudEvent(msg, event)
	if err != nil {
		return err
	}

	if envelope.Type() != event.EventName() {
		return fmt.Errorf("event type mismatch: %q", envelope.Type())
	}

	if !envelope.Time().IsZero() {
		w.upstream.Lag(ctx, name, envelope.Time())
	}

	if batch, ok := event.(*ingestevents.EventBatchedIngest); ok {
		// A first pull can register and cache a balance during the stored
		// second. Resolve after the cutoff so that key gets invalidated too.
		if err := waitForCreditUsage(ctx, batch.StoredAt); err != nil {
			return err
		}
	}

	// Measure routing separately from waiting for the stored usage to become visible.
	start := time.Now()
	defer func() {
		w.upstream.Routing(ctx, name, start, outcome)
	}()

	selectors, err := w.sources.Resolve(ctx, event)
	if err != nil {
		return err
	}

	if len(selectors) == 0 {
		outcome = rtemetrics.UpstreamOutcomeNoTarget
		return nil
	}

	if err := w.demand.Invalidate(ctx, rte.InvalidateInput{
		Source: envelope.Source(), EventID: envelope.ID(), Selectors: selectors,
	}); err != nil {
		return err
	}

	outcome = rtemetrics.UpstreamOutcomeRouted

	return nil
}

func (w *worker) handleRequested(ctx context.Context, event *rte.CalculationRequestedEvent) error {
	if policy := event.Freshness; policy != nil {
		return w.demand.EnsureFresh(ctx, policy.EnsureFreshInput(event.CalculationTarget))
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
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		w.pollDemands(ctx)
	}()

	err := w.router.Run(ctx)
	cancel()
	<-done

	return err
}

func (w *worker) Close() error {
	return w.router.Close()
}

func (w *worker) Running() <-chan struct{} {
	return w.router.Running()
}
