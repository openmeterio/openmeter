package consumer

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"

	"github.com/openmeterio/openmeter/internal/entitlement/snapshot"
	"github.com/openmeterio/openmeter/internal/event/publisher"
	"github.com/openmeterio/openmeter/internal/event/spec"
	"github.com/openmeterio/openmeter/internal/registry"
	"github.com/openmeterio/openmeter/internal/watermill/nopublisher"
)

type Options struct {
	SystemEventsTopic string
	Subscriber        message.Subscriber

	Publisher message.Publisher

	DLQ *DLQOptions

	Entitlement *registry.Entitlement

	Logger *slog.Logger
}

type DLQOptions struct {
	Topic            string
	Throttle         bool
	ThrottleDuration time.Duration
	ThrottleCount    int64
}

type Consumer struct {
	opts   Options
	router *message.Router
}

func New(opts Options) (*Consumer, error) {
	router, err := message.NewRouter(message.RouterConfig{}, watermill.NewSlogLogger(opts.Logger))
	if err != nil {
		return nil, err
	}

	consumer := &Consumer{
		opts: opts,
	}

	router.AddNoPublisherHandler(
		"balance_consumer_system_events",
		opts.SystemEventsTopic,
		opts.Subscriber,
		consumer.handleSystemEvent,
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

	if opts.DLQ != nil {
		poisionQueue, err := middleware.PoisonQueue(opts.Publisher, opts.DLQ.Topic)
		if err != nil {
			return nil, err
		}

		router.AddMiddleware(
			poisionQueue,
		)

		poisionQueueProcessor := nopublisher.NoPublisherHandlerToHandlerFunc(consumer.handleSystemEvent)
		if opts.DLQ.Throttle {
			poisionQueueProcessor = middleware.NewThrottle(
				opts.DLQ.ThrottleCount,
				opts.DLQ.ThrottleDuration,
			).Middleware(poisionQueueProcessor)
		}
		router.AddNoPublisherHandler(
			"balance_consumer_process_poison_queue",
			opts.DLQ.Topic,
			opts.Subscriber,
			nopublisher.HandlerFuncToNoPublisherHandler(poisionQueueProcessor),
		)
	}

	return &Consumer{
		opts:   opts,
		router: router,
	}, nil
}

func (w *Consumer) Run(ctx context.Context) error {
	return w.router.Run(ctx)
}

func (w *Consumer) Close() error {
	return w.router.Close()
}

func (w *Consumer) handleSystemEvent(msg *message.Message) error {
	w.opts.Logger.Debug("received system event", w.messageToLogFields(msg)...)

	ceType, found := msg.Metadata[publisher.CloudEventsHeaderType]
	if !found {
		w.opts.Logger.Warn("missing CloudEvents type, ignoring message")
		return nil
	}

	switch ceType {
	case snapshot.SnapshotEvent{}.Spec().Type():
		event, err := spec.ParseCloudEventFromBytes[snapshot.SnapshotEvent](msg.Payload)
		if err != nil {
			w.opts.Logger.Error("failed to parse entitlement created event", w.messageToLogFields(msg)...)
			return err
		}

		return w.handleSnapshotEvent(msg.Context(), event.Payload)
	}
	return nil
}

func (w *Consumer) handleSnapshotEvent(_ context.Context, payload snapshot.SnapshotEvent) error {
	w.opts.Logger.Info("handling entitlement snapshot event", slog.String("entitlement_id", payload.Entitlement.ID))

	return nil
}

func (w *Consumer) messageToLogFields(msg *message.Message) []any {
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
