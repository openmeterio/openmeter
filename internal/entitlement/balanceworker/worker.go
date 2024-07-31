package balanceworker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/cloudevents/sdk-go/v2/event"

	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/internal/event/publisher"
	"github.com/openmeterio/openmeter/internal/event/spec"
	"github.com/openmeterio/openmeter/internal/registry"
)

type WorkerOptions struct {
	SystemEventsTopic string
	IngestEventsTopic string
	Subscriber        message.Subscriber

	TargetTopic string
	PoisonQueue *WorkerPoisonQueueOptions
	Publisher   message.Publisher

	Marshaler publisher.CloudEventMarshaler

	Entitlement *registry.Entitlement

	Logger *slog.Logger
}

type WorkerPoisonQueueOptions struct {
	Topic            string
	Throttle         bool
	ThrottleDuration time.Duration
	ThrottleCount    int64
}

type Worker struct {
	opts   WorkerOptions
	router *message.Router
}

func New(opts WorkerOptions) (*Worker, error) {
	router, err := message.NewRouter(message.RouterConfig{}, watermill.NewSlogLogger(opts.Logger))
	if err != nil {
		return nil, err
	}

	worker := &Worker{
		opts: opts,
	}

	router.AddHandler(
		"balance_worker_system_events",
		opts.SystemEventsTopic,
		opts.Subscriber,
		opts.TargetTopic,
		opts.Publisher,
		worker.handleSystemEvent,
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

		poisionQueueProcessor := worker.handleSystemEvent
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

	return &Worker{
		opts:   opts,
		router: router,
	}, nil
}

func (w *Worker) Run(ctx context.Context) error {
	return w.router.Run(ctx)
}

func (w *Worker) Close() error {
	return w.router.Close()
}

func (w *Worker) handleSystemEvent(msg *message.Message) ([]*message.Message, error) {
	w.opts.Logger.Debug("received system event", w.messageToLogFields(msg)...)

	ceType, found := msg.Metadata[publisher.CloudEventsHeaderType]
	if !found {
		w.opts.Logger.Warn("missing CloudEvents type, ignoring message")
		return nil, nil
	}

	switch ceType {
	case entitlement.EntitlementCreatedEvent{}.Spec().Type():
		event, err := spec.ParseCloudEventFromBytes[entitlement.EntitlementCreatedEvent](msg.Payload)
		if err != nil {
			w.opts.Logger.Error("failed to parse entitlement created event", w.messageToLogFields(msg)...)
			return nil, err
		}

		return w.handleEntitlementCreatedEvent(event.Event, event.Payload)
	}
	// TODO[final-implementation]: use w.opts.Marshaler to create a new message

	return nil, nil
}

func (w *Worker) handleEntitlementCreatedEvent(_ event.Event, payload entitlement.EntitlementCreatedEvent) ([]*message.Message, error) {
	w.opts.Logger.Info("handling entitlement created event", slog.String("entitlement_id", payload.Entitlement.ID))

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
