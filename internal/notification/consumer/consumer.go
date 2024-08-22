package consumer

import (
	"context"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/openmeterio/openmeter/internal/entitlement/snapshot"
	"github.com/openmeterio/openmeter/internal/notification"
	"github.com/openmeterio/openmeter/internal/watermill/grouphandler"
	"github.com/openmeterio/openmeter/internal/watermill/router"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

type Options struct {
	SystemEventsTopic string

	Router router.Options

	Notification notification.Service

	Marshaler marshaler.Marshaler

	Logger *slog.Logger
}

type Consumer struct {
	opts   Options
	router *message.Router

	balanceThresholdHandler *BalanceThresholdEventHandler
}

func New(opts Options) (*Consumer, error) {
	balanceThresholdEventHandler := &BalanceThresholdEventHandler{
		Notification: opts.Notification,
		Logger:       opts.Logger.WithGroup("balance_threshold_event_handler"),
	}

	consumer := &Consumer{
		opts: opts,

		balanceThresholdHandler: balanceThresholdEventHandler,
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

				return consumer.balanceThresholdHandler.Handle(ctx, *event)
			}),
		),
	)

	return consumer, nil
}

func (c *Consumer) Handle(ctx context.Context, event snapshot.SnapshotEvent) error {
	return c.balanceThresholdHandler.Handle(ctx, event)
}

func (c *Consumer) Run(ctx context.Context) error {
	return c.router.Run(ctx)
}

func (c *Consumer) Close() error {
	return c.router.Close()
}
