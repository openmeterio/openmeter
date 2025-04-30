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
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/event/models"
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

	MetricMeter metric.Meter

	Logger *slog.Logger
}

func (o *WorkerOptions) Validate() error {
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

	if o.MetricMeter == nil {
		return errors.New("metric meter is required")
	}

	return nil
}

type highWatermarkCacheEntry struct {
	HighWatermark time.Time
	IsDeleted     bool
}

type Worker struct {
	opts        WorkerOptions
	entitlement *registry.Entitlement
	repo        BalanceWorkerRepository
	router      *message.Router

	highWatermarkCache *lru.Cache[string, highWatermarkCacheEntry]

	metricRecalculationTime       metric.Int64Histogram
	metricHighWatermarkCacheStats metric.Int64Counter

	// Handlers
	nonPublishingHandler *grouphandler.NoPublishingHandler
}

func (w *Worker) initMetrics() error {
	var err error

	w.metricRecalculationTime, err = w.opts.MetricMeter.Int64Histogram(
		metricNameRecalculationTime,
		metric.WithDescription("Entitlement recalculation time"),
	)
	if err != nil {
		return fmt.Errorf("failed to create recalculation time histogram: %w", err)
	}

	w.metricHighWatermarkCacheStats, err = w.opts.MetricMeter.Int64Counter(
		metricNameHighWatermarkCacheStats,
		metric.WithDescription("High watermark cache stats"),
	)
	if err != nil {
		return fmt.Errorf("failed to create high watermark cache stats counter: %w", err)
	}

	return nil
}

func New(opts WorkerOptions) (*Worker, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate worker options: %w", err)
	}

	highWatermarkCache, err := lru.New[string, highWatermarkCacheEntry](defaultHighWatermarkCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create high watermark cache: %w", err)
	}

	worker := &Worker{
		opts:               opts,
		entitlement:        opts.Entitlement,
		repo:               opts.Repo,
		highWatermarkCache: highWatermarkCache,
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
					WithEventAt(event.ResetAt),
					WithSourceOperation(snapshot.ValueOperationReset),
				))
		}),

		// Ingest batched event
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *ingestevents.EventBatchedIngest) error {
			if event == nil {
				return errors.New("nil batched ingest event")
			}

			return w.handleBatchedIngestEvent(ctx, *event)
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
