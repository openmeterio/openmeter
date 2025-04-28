package consumer

import (
	"context"
	"errors"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/watermill/grouphandler"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
)

type Options struct {
	SystemEventsTopic string

	Router router.Options

	Notification notification.Service

	Marshaler marshaler.Marshaler

	Logger *slog.Logger
}

func (o *Options) Validate() error {
	if o.Notification == nil {
		return errors.New("notification is required")
	}

	if o.Logger == nil {
		return errors.New("logger is required")
	}

	if o.SystemEventsTopic == "" {
		return errors.New("system events topic is required")
	}

	if o.Marshaler == nil {
		return errors.New("marshaler is required")
	}

	return nil
}

type Consumer struct {
	opts   Options
	router *message.Router

	balanceThresholdHandler *BalanceThresholdEventHandler
}

type BalanceThresholdEventHandlerOptions struct {
	Notification notification.Service
	Logger       *slog.Logger
}

func (o *BalanceThresholdEventHandlerOptions) Validate() error {
	if o.Notification == nil {
		return errors.New("notification is required")
	}

	if o.Logger == nil {
		return errors.New("logger is required")
	}

	return nil
}

func NewBalanceThresholdEventHandler(opts BalanceThresholdEventHandlerOptions) (*BalanceThresholdEventHandler, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	return &BalanceThresholdEventHandler{
		Notification: opts.Notification,
		Logger:       opts.Logger.WithGroup("balance_threshold_event_handler"),
	}, nil
}

func New(opts Options) (*Consumer, error) {
	balanceThresholdEventHandler, err := NewBalanceThresholdEventHandler(BalanceThresholdEventHandlerOptions{
		Notification: opts.Notification,
		Logger:       opts.Logger.WithGroup("balance_threshold_event_handler"),
	})
	if err != nil {
		return nil, err
	}

	router, err := router.NewDefaultRouter(opts.Router)
	if err != nil {
		return nil, err
	}

	consumer := &Consumer{
		opts:   opts,
		router: router,

		balanceThresholdHandler: balanceThresholdEventHandler,
	}

	handler, err := grouphandler.NewNoPublishingHandler(opts.Marshaler, opts.Router.MetricMeter,
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *snapshot.SnapshotEvent) error {
			if event == nil {
				return nil
			}

			return consumer.balanceThresholdHandler.Handle(ctx, *event)
		}),
	)
	if err != nil {
		return nil, err
	}

	_ = router.AddNoPublisherHandler(
		"balance_consumer_system_events",
		opts.SystemEventsTopic,
		opts.Router.Subscriber,
		handler.Handle,
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
