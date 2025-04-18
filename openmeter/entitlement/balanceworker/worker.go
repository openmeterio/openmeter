package balanceworker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/entitlement/balanceworker/estimator"
	"github.com/openmeterio/openmeter/openmeter/entitlement/edge"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/registry"
	ingestevents "github.com/openmeterio/openmeter/openmeter/sink/flushhandler/ingestnotification/events"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/grouphandler"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
	pkgmodels "github.com/openmeterio/openmeter/pkg/models"
)

const (
	defaultHighWatermarkCacheSize = 100_000

	// defaultClockDrift specifies how much clock drift is allowed when calculating the current time between the worker nodes.
	// with AWS, Google Cloud 1ms is guaranteed, this should work well for any NTP based setup.
	defaultClockDrift = time.Millisecond
)

type NamespacedID = pkgmodels.NamespacedID

type SubjectResolver interface {
	GetSubjectByKey(ctx context.Context, namespace, key string) (models.Subject, error)
}

type BatchedIngestEventHandler = func(ctx context.Context, event ingestevents.EventBatchedIngest) error

type WorkerOptions struct {
	SystemEventsTopic string
	IngestEventsTopic string

	Router   router.Options
	EventBus eventbus.Publisher

	Entitlement *registry.Entitlement
	Repo        BalanceWorkerRepository
	// External connectors
	SubjectResolver SubjectResolver
	MeterService    meter.Service

	Logger *slog.Logger

	Estimator EstimatorOptions
}

func (o *WorkerOptions) Validate() error {
	if err := o.Estimator.Validate(); err != nil {
		return fmt.Errorf("failed to validate estimator options: %w", err)
	}

	if o.Entitlement == nil {
		return errors.New("entitlement is required")
	}

	if o.Repo == nil {
		return errors.New("repo is required")
	}

	if o.EventBus == nil {
		return errors.New("event bus is required")
	}

	if o.Logger == nil {
		return errors.New("logger is required")
	}

	if o.SystemEventsTopic == "" {
		return errors.New("system events topic is required")
	}

	if o.IngestEventsTopic == "" {
		return errors.New("ingest events topic is required")
	}

	if o.MeterService == nil {
		return errors.New("meter service is required")
	}

	return nil
}

type EstimatorOptions struct {
	Enabled            bool
	RedisURL           string
	ValidationRate     float64
	LockTimeout        time.Duration
	CacheTTL           time.Duration
	ThresholdProviders []ThresholdProvider
}

func (o *EstimatorOptions) Validate() error {
	if !o.Enabled {
		return nil
	}

	if o.RedisURL == "" {
		return errors.New("redis url is required")
	}

	if o.ValidationRate < 0 || o.ValidationRate > 1 {
		return errors.New("validation rate must be between 0 and 1")
	}

	if o.LockTimeout <= 0 {
		return errors.New("lock timeout must be greater than 0")
	}

	// Or we won't emit any events
	if len(o.ThresholdProviders) == 0 {
		return errors.New("threshold providers are required")
	}

	return nil
}

type highWatermarkCacheEntry struct {
	HighWatermark time.Time
	IsDeleted     bool
}

type estimatorConfig struct {
	estimator.Estimator

	enabled bool

	thresholdProviders []ThresholdProvider
	validationRate     float64

	lockTimeout time.Duration
}

type Worker struct {
	opts        WorkerOptions
	entitlement *registry.Entitlement
	meter       meter.Service
	repo        BalanceWorkerRepository
	router      *message.Router

	highWatermarkCache *lru.Cache[string, highWatermarkCacheEntry]

	// Estimator debounce engine
	estimator estimatorConfig

	metricRecalculationTime       metric.Int64Histogram
	metricHighWatermarkCacheStats metric.Int64Counter

	// Estimator metrics
	metricEstimatorRequestsTotal         metric.Int64Counter
	metricEstimatorValidationErrorsTotal metric.Int64Counter
	metricEstimatorActionTotal           metric.Int64Counter

	// Handlers
	nonPublishingHandler *grouphandler.NoPublishingHandler
}

func (w *Worker) initMetrics() error {
	var err error

	w.metricRecalculationTime, err = w.opts.Router.MetricMeter.Int64Histogram(
		metricNameRecalculationTime,
		metric.WithDescription("Entitlement recalculation time"),
	)
	if err != nil {
		return fmt.Errorf("failed to create recalculation time histogram: %w", err)
	}

	w.metricHighWatermarkCacheStats, err = w.opts.Router.MetricMeter.Int64Counter(
		metricNameHighWatermarkCacheStats,
		metric.WithDescription("High watermark cache stats"),
	)
	if err != nil {
		return fmt.Errorf("failed to create high watermark cache stats counter: %w", err)
	}

	w.metricEstimatorRequestsTotal, err = w.opts.Router.MetricMeter.Int64Counter(
		metricNameEstimatorRequestsTotal,
		metric.WithDescription("Estimator requests total"),
	)
	if err != nil {
		return fmt.Errorf("failed to create estimator requests total counter: %w", err)
	}

	w.metricEstimatorValidationErrorsTotal, err = w.opts.Router.MetricMeter.Int64Counter(
		metricNameEstimatorValidationErrorsTotal,
		metric.WithDescription("Estimator validation errors total"),
	)
	if err != nil {
		return fmt.Errorf("failed to create estimator validation errors total counter: %w", err)
	}

	w.metricEstimatorActionTotal, err = w.opts.Router.MetricMeter.Int64Counter(
		metricNameEstimatorActionTotal,
		metric.WithDescription("Estimator action total"),
	)
	if err != nil {
		return fmt.Errorf("failed to create estimator action total counter: %w", err)
	}
	return nil
}

func New(opts WorkerOptions) (*Worker, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate  options: %w", err)
	}

	highWatermarkCache, err := lru.New[string, highWatermarkCacheEntry](defaultHighWatermarkCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create high watermark cache: %w", err)
	}

	// Let's setup estimator
	var estCfg estimatorConfig
	if opts.Estimator.Enabled {
		estimatorInstance, err := estimator.New(estimator.Options{
			RedisURL:    opts.Estimator.RedisURL,
			LockTimeout: opts.Estimator.LockTimeout,
			CacheTTL:    opts.Estimator.CacheTTL,
			Logger:      opts.Logger,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create estimator: %w", err)
		}

		estCfg = estimatorConfig{
			Estimator: estimatorInstance,
			enabled:   true,

			thresholdProviders: opts.Estimator.ThresholdProviders,
			validationRate:     opts.Estimator.ValidationRate,
			lockTimeout:        opts.Estimator.LockTimeout,
		}
	} else {
		estCfg.enabled = false
	}

	worker := &Worker{
		opts:               opts,
		entitlement:        opts.Entitlement,
		repo:               opts.Repo,
		highWatermarkCache: highWatermarkCache,
		estimator:          estCfg,
		meter:              opts.MeterService,
	}

	if err := worker.initMetrics(); err != nil {
		return nil, fmt.Errorf("failed to init metrics: %w", err)
	}

	router, err := router.NewDefaultRouter(opts.Router)
	if err != nil {
		return nil, err
	}

	worker.router = router

	eventHandler, err := worker.eventHandler(opts.Router.MetricMeter)
	if err != nil {
		return nil, err
	}

	router.AddNoPublisherHandler(
		"balance_worker_system_events",
		opts.SystemEventsTopic,
		opts.Router.Subscriber,
		eventHandler,
	)

	if opts.SystemEventsTopic != opts.IngestEventsTopic {
		router.AddNoPublisherHandler(
			"balance_worker_ingest_events",
			opts.IngestEventsTopic,
			opts.Router.Subscriber,
			eventHandler,
		)
	}

	return worker, nil
}

// AddHandler adds an additional handler to the list of batched ingest event handlers.
// Handlers are called in the order they are added and run after the riginal balance worker handler.
// In the case of any handler returning an error, the event will be retried so it is important that all handlers are idempotent.
func (w *Worker) AddHandler(handler grouphandler.GroupEventHandler) {
	w.nonPublishingHandler.AddHandler(handler)
}

func (w *Worker) eventHandler(metricMeter metric.Meter) (message.NoPublishHandlerFunc, error) {
	publishingHandler, err := grouphandler.NewNoPublishingHandler(
		w.opts.EventBus.Marshaler(),
		metricMeter,

		// Entitlement created event
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *entitlement.EntitlementCreatedEvent) error {
			return w.opts.EventBus.
				WithContext(ctx).
				PublishIfNoError(w.handleEntitlementEvent(
					ctx,
					NamespacedID{Namespace: event.Namespace.ID, ID: event.ID},
					WithSource(metadata.ComposeResourcePath(event.Namespace.ID, metadata.EntityEntitlement, event.ID)),
					WithEventAt(event.CreatedAt),
				))
		}),

		// Entitlement deleted event
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *entitlement.EntitlementDeletedEvent) error {
			return w.opts.EventBus.
				WithContext(ctx).
				PublishIfNoError(w.handleEntitlementEvent(ctx,
					NamespacedID{Namespace: event.Namespace.ID, ID: event.ID},
					WithSource(metadata.ComposeResourcePath(event.Namespace.ID, metadata.EntityEntitlement, event.ID)),
					WithEventAt(lo.FromPtrOr(event.DeletedAt, time.Now())),
				))
		}),

		// Grant created event
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *grant.CreatedEvent) error {
			return w.opts.EventBus.
				WithContext(ctx).
				PublishIfNoError(w.handleEntitlementEvent(
					ctx,
					NamespacedID{Namespace: event.Namespace.ID, ID: event.OwnerID},
					WithSource(metadata.ComposeResourcePath(event.Namespace.ID, metadata.EntityEntitlement, event.OwnerID, metadata.EntityGrant, event.ID)),
					WithEventAt(event.CreatedAt),
				))
		}),

		// Grant voided event
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *grant.VoidedEvent) error {
			return w.opts.EventBus.
				WithContext(ctx).
				PublishIfNoError(w.handleEntitlementEvent(
					ctx,
					NamespacedID{Namespace: event.Namespace.ID, ID: event.OwnerID},
					WithSource(metadata.ComposeResourcePath(event.Namespace.ID, metadata.EntityEntitlement, event.OwnerID, metadata.EntityGrant, event.ID)),
					WithEventAt(event.UpdatedAt),
				))
		}),

		// Metered entitlement reset event
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *meteredentitlement.EntitlementResetEvent) error {
			return w.opts.EventBus.
				WithContext(ctx).
				PublishIfNoError(w.handleEntitlementEvent(
					ctx,
					NamespacedID{Namespace: event.Namespace.ID, ID: event.EntitlementID},
					WithSource(metadata.ComposeResourcePath(event.Namespace.ID, metadata.EntityEntitlement, event.EntitlementID)),
					WithEventAt(event.ResetRequestedAt),
				))
		}),

		// Ingest batched event
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *ingestevents.EventBatchedIngest) error {
			if event == nil {
				return errors.New("nil batched ingest event")
			}

			return w.handleBatchedIngestEvent(ctx, *event)
		}),

		// Edge Cache Miss Event
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *edge.EntitlementCacheMissEvent) error {
			return w.opts.EventBus.
				WithContext(ctx).
				PublishIfNoError(w.handleCacheMissEvent(ctx, event, metadata.ComposeResourcePath(event.Namespace.ID, metadata.EntitySubjectKey, event.SubjectKey)))
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create publishing handler: %w", err)
	}

	w.nonPublishingHandler = publishingHandler

	return publishingHandler.Handle, nil
}

func (w *Worker) Run(ctx context.Context) error {
	return w.router.Run(ctx)
}

func (w *Worker) Close() error {
	if err := w.router.Close(); err != nil {
		return err
	}

	return nil
}
