package billingworker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/asyncadvance"
	billingworkersubscription "github.com/openmeterio/openmeter/openmeter/billing/worker/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/grouphandler"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
)

type WorkerOptions struct {
	SystemEventsTopic string

	Router   router.Options
	EventBus eventbus.Publisher

	Logger *slog.Logger

	BillingAdapter                 billing.Adapter
	BillingService                 billing.Service
	BillingSubscriptionSyncHandler *billingworkersubscription.Handler
	// External connectors

	SubscriptionService subscription.Service
}

func (w WorkerOptions) Validate() error {
	if w.SystemEventsTopic == "" {
		return fmt.Errorf("system events topic is required")
	}

	if err := w.Router.Validate(); err != nil {
		return fmt.Errorf("router: %w", err)
	}

	if w.EventBus == nil {
		return fmt.Errorf("event bus is required")
	}

	if w.Logger == nil {
		return fmt.Errorf("logger is required")
	}

	if w.BillingService == nil {
		return fmt.Errorf("billing service is required")
	}

	if w.BillingAdapter == nil {
		return fmt.Errorf("billing adapter is required")
	}

	if w.SubscriptionService == nil {
		return fmt.Errorf("subscription service is required")
	}

	if w.BillingSubscriptionSyncHandler == nil {
		return fmt.Errorf("billing subscription sync handler is required")
	}

	return nil
}

type Worker struct {
	router *message.Router

	billingService          billing.Service
	subscriptionSyncHandler *billingworkersubscription.Handler
	asyncAdvanceHandler     *asyncadvance.Handler

	nonPublishingHandler *grouphandler.NoPublishingHandler
}

func New(opts WorkerOptions) (*Worker, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	// asyncAdvancer
	asyncAdvancer, err := asyncadvance.New(asyncadvance.Config{
		Logger:         opts.Logger,
		BillingService: opts.BillingService,
	})
	if err != nil {
		return nil, err
	}

	worker := &Worker{
		billingService:          opts.BillingService,
		subscriptionSyncHandler: opts.BillingSubscriptionSyncHandler,
		asyncAdvanceHandler:     asyncAdvancer,
	}

	router, err := router.NewDefaultRouter(opts.Router)
	if err != nil {
		return nil, err
	}

	worker.router = router

	eventHandler, err := worker.eventHandler(opts)
	if err != nil {
		return nil, err
	}

	worker.nonPublishingHandler = eventHandler

	router.AddConsumerHandler(
		"billing_worker_system_events",
		opts.SystemEventsTopic,
		opts.Router.Subscriber,
		worker.nonPublishingHandler.Handle,
	)

	return worker, nil
}

// AddHandler adds an additional handler to the list of event handlers.
// Handlers are called in the order they are added and run after the built in handlers.
// In the case of any handler returning an error, the event will be retried so it is important that all handlers are idempotent.
func (w *Worker) AddHandler(handler grouphandler.GroupEventHandler) {
	w.nonPublishingHandler.AddHandler(handler)
}

func (w *Worker) eventHandler(opts WorkerOptions) (*grouphandler.NoPublishingHandler, error) {
	handler, err := grouphandler.NewNoPublishingHandler(
		opts.EventBus.Marshaler(),
		opts.Router.MetricMeter,

		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *subscription.CreatedEvent) error {
			return w.subscriptionSyncHandler.SyncronizeSubscriptionAndInvoiceCustomer(ctx, event.SubscriptionView, time.Now())
		}),
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *subscription.CancelledEvent) error {
			return w.subscriptionSyncHandler.HandleCancelledEvent(ctx, event)
		}),
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *subscription.ContinuedEvent) error {
			return w.subscriptionSyncHandler.SyncronizeSubscriptionAndInvoiceCustomer(ctx, event.SubscriptionView, time.Now())
		}),
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *subscription.UpdatedEvent) error {
			return w.subscriptionSyncHandler.SyncronizeSubscriptionAndInvoiceCustomer(ctx, event.UpdatedView, time.Now())
		}),
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *billing.AdvanceInvoiceEvent) error {
			return w.asyncAdvanceHandler.Handle(ctx, event)
		}),
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *billing.InvoiceCreatedEvent) error {
			return w.subscriptionSyncHandler.HandleInvoiceCreation(ctx, event.EventInvoice)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create event handler: %w", err)
	}

	return handler, nil
}

func (w *Worker) Run(ctx context.Context) error {
	return w.router.Run(ctx)
}

func (w *Worker) Close() error {
	return w.router.Close()
}
