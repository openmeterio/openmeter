package balanceworker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/internal/entitlement/metered"
	"github.com/openmeterio/openmeter/internal/event/publisher"
	"github.com/openmeterio/openmeter/internal/event/spec"
	"github.com/openmeterio/openmeter/internal/registry"
	"github.com/openmeterio/openmeter/internal/sink/flushhandler/ingestnotification"
	pkgmodels "github.com/openmeterio/openmeter/pkg/models"
)

const (
	defaultHighWatermarkCacheSize = 100_000

	// defaultClockDrift specifies how much clock drift is allowed when calculating the current time between the worker nodes.
	// with AWS, Google Cloud 1ms is guaranteed, this should work well for any NTP based setup.
	defaultClockDrift = time.Millisecond
)

type NamespacedID = pkgmodels.NamespacedID

type SubjectIDResolver interface {
	GetSubjectIDByKey(ctx context.Context, namespace, key string) (string, error)
}

type WorkerOptions struct {
	SystemEventsTopic string
	IngestEventsTopic string
	Subscriber        message.Subscriber

	TargetTopic string
	PoisonQueue *WorkerPoisonQueueOptions
	Publisher   message.Publisher
	Marshaler   publisher.CloudEventMarshaler

	Entitlement *registry.Entitlement
	// External connectors
	SubjectIDResolver SubjectIDResolver

	Logger *slog.Logger
}

type WorkerPoisonQueueOptions struct {
	Topic            string
	Throttle         bool
	ThrottleDuration time.Duration
	ThrottleCount    int64
}

type highWatermarkCacheEntry struct {
	HighWatermark time.Time
	IsDeleted     bool
}

type Worker struct {
	ctx        context.Context
	ctxCancel  context.CancelFunc
	opts       WorkerOptions
	connectors *registry.Entitlement
	router     *message.Router

	highWatermarkCache *lru.Cache[string, highWatermarkCacheEntry]
}

func New(opts WorkerOptions) (*Worker, error) {
	router, err := message.NewRouter(message.RouterConfig{}, watermill.NewSlogLogger(opts.Logger))
	if err != nil {
		return nil, err
	}

	highWatermarkCache, err := lru.New[string, highWatermarkCacheEntry](defaultHighWatermarkCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create high watermark cache: %w", err)
	}

	worker := &Worker{
		opts:               opts,
		router:             router,
		connectors:         opts.Entitlement,
		highWatermarkCache: highWatermarkCache,
	}

	router.AddHandler(
		"balance_worker_system_events",
		opts.SystemEventsTopic,
		opts.Subscriber,
		opts.TargetTopic,
		opts.Publisher,
		worker.handleEvent,
	)

	router.AddHandler(
		"balance_worker_ingest_events",
		opts.IngestEventsTopic,
		opts.Subscriber,
		opts.TargetTopic,
		opts.Publisher,
		worker.handleEvent,
	)

	router.AddMiddleware(
		middleware.CorrelationID,

		middleware.Retry{
			MaxRetries:      5,
			InitialInterval: 100 * time.Millisecond,
			Logger:          watermill.NewSlogLogger(opts.Logger),
		}.Middleware,

		middleware.Recoverer,
	)

	if opts.PoisonQueue != nil {
		poisionQueue, err := middleware.PoisonQueue(opts.Publisher, opts.PoisonQueue.Topic)
		if err != nil {
			return nil, err
		}

		router.AddMiddleware(
			poisionQueue,
		)

		poisionQueueProcessor := worker.handleEvent
		if opts.PoisonQueue.Throttle {
			poisionQueueProcessor = middleware.NewThrottle(
				opts.PoisonQueue.ThrottleCount,
				opts.PoisonQueue.ThrottleDuration,
			).Middleware(poisionQueueProcessor)
		}
		router.AddHandler(
			"balance_worker_process_poison_queue",
			opts.PoisonQueue.Topic,
			opts.Subscriber,
			opts.TargetTopic,
			opts.Publisher,
			poisionQueueProcessor,
		)
	}

	return worker, nil
}

func (w *Worker) Run(ctx context.Context) error {
	w.ctx, w.ctxCancel = context.WithCancel(ctx)
	return w.router.Run(ctx)
}

func (w *Worker) Close() error {
	if err := w.router.Close(); err != nil {
		return err
	}

	w.ctxCancel()
	return nil
}

func (w *Worker) handleEvent(msg *message.Message) ([]*message.Message, error) {
	w.opts.Logger.Debug("received system event", w.messageToLogFields(msg)...)

	ceType, found := msg.Metadata[publisher.CloudEventsHeaderType]
	if !found {
		w.opts.Logger.Warn("missing CloudEvents type, ignoring message")
		return nil, nil
	}

	switch ceType {
	// Entitlement events
	case entitlement.EntitlementCreatedEvent{}.Spec().Type():
		event, err := spec.ParseCloudEventFromBytes[entitlement.EntitlementCreatedEvent](msg.Payload)
		if err != nil {
			w.opts.Logger.Error("failed to parse entitlement created event", w.messageToLogFields(msg)...)
			return nil, err
		}
		return w.handleEntitlementUpdateEvent(
			w.ctx,
			NamespacedID{Namespace: event.Payload.Namespace.ID, ID: event.Payload.ID},
			spec.ComposeResourcePath(event.Payload.Namespace.ID, spec.EntityEntitlement, event.Payload.ID),
		)
	case entitlement.EntitlementDeletedEvent{}.Spec().Type():
		event, err := spec.ParseCloudEventFromBytes[entitlement.EntitlementDeletedEvent](msg.Payload)
		if err != nil {
			w.opts.Logger.Error("failed to parse entitlement deleted event", w.messageToLogFields(msg)...)
			return nil, err
		}

		return w.handleEntitlementDeleteEvent(w.ctx, event.Payload)

	// Grant events
	case credit.GrantCreatedEvent{}.Spec().Type():
		event, err := spec.ParseCloudEventFromBytes[credit.GrantCreatedEvent](msg.Payload)
		if err != nil {
			return nil, fmt.Errorf("failed to parse grant created event: %w", err)
		}

		return w.handleEntitlementUpdateEvent(
			w.ctx,
			NamespacedID{Namespace: event.Payload.Namespace.ID, ID: string(event.Payload.OwnerID)},
			spec.ComposeResourcePath(event.Payload.Namespace.ID, spec.EntityEntitlement, string(event.Payload.OwnerID), spec.EntityGrant, event.Payload.ID),
		)
	case credit.GrantVoidedEvent{}.Spec().Type():
		event, err := spec.ParseCloudEventFromBytes[credit.GrantVoidedEvent](msg.Payload)
		if err != nil {
			return nil, fmt.Errorf("failed to parse grant voided event: %w", err)
		}

		return w.handleEntitlementUpdateEvent(
			w.ctx,
			NamespacedID{Namespace: event.Payload.Namespace.ID, ID: string(event.Payload.OwnerID)},
			spec.ComposeResourcePath(event.Payload.Namespace.ID, spec.EntityEntitlement, string(event.Payload.OwnerID), spec.EntityGrant, event.Payload.ID),
		)

	// Metered entitlement events
	case meteredentitlement.ResetEntitlementEvent{}.Spec().Type():
		event, err := spec.ParseCloudEventFromBytes[meteredentitlement.ResetEntitlementEvent](msg.Payload)
		if err != nil {
			return nil, fmt.Errorf("failed to parse reset entitlement event: %w", err)
		}

		return w.handleEntitlementUpdateEvent(
			w.ctx,
			NamespacedID{Namespace: event.Payload.Namespace.ID, ID: event.Payload.EntitlementID},
			spec.ComposeResourcePath(event.Payload.Namespace.ID, spec.EntityEntitlement, event.Payload.EntitlementID),
		)

	// Metered entitlement events
	case meteredentitlement.ResetEntitlementEvent{}.Spec().Type():
		event, err := spec.ParseCloudEventFromBytes[meteredentitlement.ResetEntitlementEvent](msg.Payload)
		if err != nil {
			return nil, fmt.Errorf("failed to parse reset entitlement event: %w", err)
		}

		return w.handleEntitlementUpdateEvent(
			msg.Context(),
			NamespacedID{Namespace: event.Payload.Namespace.ID, ID: event.Payload.EntitlementID},
			spec.ComposeResourcePath(event.Payload.Namespace.ID, spec.EntityEntitlement, event.Payload.EntitlementID),
		)

	// Ingest event
	case ingestnotification.IngestEvent{}.Spec().Type():
		event, err := spec.ParseCloudEventFromBytes[ingestnotification.IngestEvent](msg.Payload)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ingest event: %w", err)
		}

		return w.handleIngestEvent(w.ctx, event.Payload)
	}

	return nil, nil
}

func (w *Worker) messageToLogFields(msg *message.Message) []any {
	out := make([]any, 0, 3)
	out = append(out, slog.String("message_uuid", msg.UUID))
	out = append(out, slog.String("message_payload", string(msg.Payload)))

	meta, err := json.Marshal(msg.Metadata)
	if err != nil {
		return out
	}

	out = append(out, slog.String("message_metadata", string(meta)))
	return out
}
