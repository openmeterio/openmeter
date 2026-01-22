package consumer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/openmeterio/openmeter/openmeter/billing"
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

func (o Options) Validate() error {
	var errs []error

	if o.SystemEventsTopic == "" {
		errs = append(errs, errors.New("system events topic is required"))
	}

	if o.Notification == nil {
		errs = append(errs, errors.New("notification service is required"))
	}

	if o.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	return errors.Join(errs...)
}

type Consumer struct {
	opts   Options
	router *message.Router

	entitlementSnapshotHandler *EntitlementSnapshotHandler
	invoiceHandler             *InvoiceEventHandler
}

func New(opts Options) (*Consumer, error) {
	entitlementSnapshotHandler := &EntitlementSnapshotHandler{
		Notification: opts.Notification,
		Logger:       opts.Logger.WithGroup("entitlement_snapshot_handler"),
	}

	invoiceEventHandler := &InvoiceEventHandler{
		Notification: opts.Notification,
		Logger:       opts.Logger.WithGroup("invoice_event_handler"),
	}

	r, err := router.NewDefaultRouter(opts.Router)
	if err != nil {
		return nil, err
	}

	consumer := &Consumer{
		opts:   opts,
		router: r,

		entitlementSnapshotHandler: entitlementSnapshotHandler,
		invoiceHandler:             invoiceEventHandler,
	}

	handler, err := grouphandler.NewNoPublishingHandler(opts.Marshaler, opts.Router.MetricMeter,
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *snapshot.SnapshotEvent) error {
			if event == nil {
				return nil
			}

			return consumer.entitlementSnapshotHandler.Handle(ctx, *event)
		}),
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *billing.StandardInvoiceCreatedEvent) error {
			if event == nil {
				return nil
			}

			return consumer.invoiceHandler.Handle(ctx, event.EventStandardInvoice, notification.EventTypeInvoiceCreated)
		}),
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *billing.StandardInvoiceUpdatedEvent) error {
			if event == nil {
				return nil
			}

			return consumer.invoiceHandler.Handle(ctx, event.New, notification.EventTypeInvoiceUpdated)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize notification event handler: %w", err)
	}

	_ = r.AddConsumerHandler(
		"notification_consumer_system_events",
		opts.SystemEventsTopic,
		opts.Router.Subscriber,
		handler.Handle,
	)

	return consumer, nil
}

func (c *Consumer) Handle(ctx context.Context, event snapshot.SnapshotEvent) error {
	return c.entitlementSnapshotHandler.Handle(ctx, event)
}

func (c *Consumer) Run(ctx context.Context) error {
	return c.router.Run(ctx)
}

func (c *Consumer) Close() error {
	return c.router.Close()
}
