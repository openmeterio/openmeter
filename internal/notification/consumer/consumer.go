package consumer

import (
	"context"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/openmeterio/openmeter/internal/entitlement/snapshot"
	"github.com/openmeterio/openmeter/internal/registry"
	"github.com/openmeterio/openmeter/internal/watermill/grouphandler"
	"github.com/openmeterio/openmeter/internal/watermill/router"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

type Options struct {
	SystemEventsTopic string

	Entitlement *registry.Entitlement
	Router      router.Options

	Marshaler marshaler.Marshaler

	Logger *slog.Logger
}

type Consumer struct {
	opts   Options
	router *message.Router
}

func New(opts Options) (*Consumer, error) {
	consumer := &Consumer{
		opts: opts,
	}

	router, err := router.NewDefaultRouter(opts.Router)
	if err != nil {
		return nil, err
	}

	_ = router.AddNoPublisherHandler(
		"balance_consumer_system_events",
		opts.SystemEventsTopic,
		opts.Router.Subscriber,
		grouphandler.NewNoPublishingHandler(opts.Marshaler,
			grouphandler.NewGroupEventHandler(func(ctx context.Context, event *snapshot.SnapshotEvent) error {
				if event == nil {
					return nil
				}

				return consumer.handleSnapshotEvent(ctx, *event)
			}),
		),
	)

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

func (w *Consumer) handleSnapshotEvent(_ context.Context, payload snapshot.SnapshotEvent) error {
	w.opts.Logger.Info("handling entitlement snapshot event", slog.String("entitlement_id", payload.Entitlement.ID))

	return nil
}
