package balanceworker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/registry"
	ingestevents "github.com/openmeterio/openmeter/openmeter/sink/flushhandler/ingestnotification/events"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/grouphandler"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
	pkgmodels "github.com/openmeterio/openmeter/pkg/models"
)

const (
	defaultHighWatermarkCacheSize = 100_000
)

type BatchedIngestEventHandler = func(ctx context.Context, event ingestevents.EventBatchedIngest) error

type WorkerOptions struct {
	SystemEventsTopic string
	IngestEventsTopic string

	Router   router.Options
	EventBus eventbus.Publisher

	Entitlement *registry.Entitlement
	Repo        BalanceWorkerRepository

	// External connectors
	NotificationService notification.Service
	Subject             subject.Service
	Customer            customer.Service

	MetricMeter metric.Meter

	Logger *slog.Logger

	FilterStateStorage FilterStateStorage
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

	if o.NotificationService == nil {
		return errors.New("notification service is required")
	}

	if o.Customer == nil {
		return errors.New("customer service is required")
	}

	if o.Subject == nil {
		return errors.New("subject service is required")
	}

	if err := o.FilterStateStorage.Validate(); err != nil {
		return fmt.Errorf("filter state storage: %w", err)
	}

	return nil
}

type Worker struct {
	opts   WorkerOptions
	router *message.Router

	filters *EntitlementFilters

	metricRecalculationTime metric.Int64Histogram

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

	return nil
}

func New(opts WorkerOptions) (*Worker, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate worker options: %w", err)
	}

	filters, err := NewEntitlementFilters(EntitlementFiltersConfig{
		NotificationService: opts.NotificationService,
		MetricMeter:         opts.MetricMeter,
		StateStorage:        opts.FilterStateStorage,
		Logger:              opts.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create entitlement filters: %w", err)
	}

	worker := &Worker{
		opts:    opts,
		filters: filters,
	}

	if err := worker.initMetrics(); err != nil {
		return nil, fmt.Errorf("failed to init metrics: %w", err)
	}

	r, err := router.NewDefaultRouter(opts.Router)
	if err != nil {
		return nil, err
	}

	worker.router = r

	eventHandler, err := worker.eventHandler(opts.Router.MetricMeter)
	if err != nil {
		return nil, err
	}

	r.AddNoPublisherHandler(
		"balance_worker_system_events",
		opts.SystemEventsTopic,
		opts.Router.Subscriber,
		eventHandler,
	)

	if opts.SystemEventsTopic != opts.IngestEventsTopic {
		r.AddNoPublisherHandler(
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
					pkgmodels.NamespacedID{Namespace: event.Namespace.ID, ID: event.ID},
					WithSource(metadata.ComposeResourcePath(event.Namespace.ID, metadata.EntityEntitlement, event.ID)),
					WithEventAt(event.CreatedAt),
				))
		}),

		// Entitlement created event v2
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *entitlement.EntitlementCreatedEventV2) error {
			return w.opts.EventBus.
				WithContext(ctx).
				PublishIfNoError(w.handleEntitlementEvent(
					ctx,
					pkgmodels.NamespacedID{Namespace: event.Namespace.ID, ID: event.Entitlement.ID},
					WithSource(metadata.ComposeResourcePath(event.Namespace.ID, metadata.EntityEntitlement, event.Entitlement.ID)),
					WithEventAt(event.Entitlement.ManagedModel.CreatedAt),
				))
		}),

		// Entitlement deleted event
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *entitlement.EntitlementDeletedEvent) error {
			return w.opts.EventBus.
				WithContext(ctx).
				PublishIfNoError(w.handleEntitlementEvent(ctx,
					pkgmodels.NamespacedID{Namespace: event.Namespace.ID, ID: event.ID},
					WithSource(metadata.ComposeResourcePath(event.Namespace.ID, metadata.EntityEntitlement, event.ID)),
					WithEventAt(lo.FromPtrOr(event.DeletedAt, time.Now())),
				))
		}),

		// Entitlement deleted event v2
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *entitlement.EntitlementDeletedEventV2) error {
			return w.opts.EventBus.
				WithContext(ctx).
				PublishIfNoError(w.handleEntitlementEvent(
					ctx,
					pkgmodels.NamespacedID{Namespace: event.Namespace.ID, ID: event.Entitlement.ID},
					WithSource(metadata.ComposeResourcePath(event.Namespace.ID, metadata.EntityEntitlement, event.Entitlement.ID)),
					WithEventAt(lo.FromPtrOr(event.Entitlement.ManagedModel.DeletedAt, time.Now())),
				))
		}),

		// Grant created event
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *grant.CreatedEvent) error {
			return w.opts.EventBus.
				WithContext(ctx).
				PublishIfNoError(w.handleEntitlementEvent(
					ctx,
					pkgmodels.NamespacedID{Namespace: event.Namespace.ID, ID: event.OwnerID},
					WithSource(metadata.ComposeResourcePath(event.Namespace.ID, metadata.EntityEntitlement, event.OwnerID, metadata.EntityGrant, event.ID)),
					WithEventAt(event.CreatedAt),
				))
		}),

		// Grant created event v2
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *grant.CreatedEventV2) error {
			return w.opts.EventBus.
				WithContext(ctx).
				PublishIfNoError(w.handleEntitlementEvent(
					ctx,
					pkgmodels.NamespacedID{Namespace: event.Namespace.ID, ID: event.Grant.OwnerID},
					WithSource(metadata.ComposeResourcePath(event.Namespace.ID, metadata.EntityEntitlement, event.Grant.OwnerID, metadata.EntityGrant, event.Grant.ID)),
					WithEventAt(event.Grant.ManagedModel.CreatedAt),
				))
		}),

		// Grant voided event
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *grant.VoidedEvent) error {
			return w.opts.EventBus.
				WithContext(ctx).
				PublishIfNoError(w.handleEntitlementEvent(
					ctx,
					pkgmodels.NamespacedID{Namespace: event.Namespace.ID, ID: event.OwnerID},
					WithSource(metadata.ComposeResourcePath(event.Namespace.ID, metadata.EntityEntitlement, event.OwnerID, metadata.EntityGrant, event.ID)),
					WithEventAt(event.UpdatedAt),
				))
		}),

		// Grant voided event v2
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *grant.VoidedEventV2) error {
			return w.opts.EventBus.
				WithContext(ctx).
				PublishIfNoError(w.handleEntitlementEvent(
					ctx,
					pkgmodels.NamespacedID{Namespace: event.Namespace.ID, ID: event.Grant.OwnerID},
					WithSource(metadata.ComposeResourcePath(event.Namespace.ID, metadata.EntityEntitlement, event.Grant.OwnerID, metadata.EntityGrant, event.Grant.ID)),
					WithEventAt(event.Grant.ManagedModel.UpdatedAt),
				))
		}),

		// Metered entitlement reset event
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *meteredentitlement.EntitlementResetEvent) error {
			return w.opts.EventBus.
				WithContext(ctx).
				PublishIfNoError(w.handleEntitlementEvent(
					ctx,
					pkgmodels.NamespacedID{Namespace: event.Namespace.ID, ID: event.EntitlementID},
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
