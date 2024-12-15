package billingworker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/grouphandler"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
)

type WorkerOptions struct {
	SystemEventsTopic string

	Router   router.Options
	EventBus eventbus.Publisher

	Logger *slog.Logger

	BillingService billing.Service
	// External connectors
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

	return nil
}

type Worker struct {
	router *message.Router

	billingService billing.Service
}

func New(opts WorkerOptions) (*Worker, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	worker := &Worker{
		billingService: opts.BillingService,
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

	router.AddNoPublisherHandler(
		"billing_worker_system_events",
		opts.SystemEventsTopic,
		opts.Router.Subscriber,
		eventHandler,
	)

	return worker, nil
}

func (w *Worker) eventHandler(opts WorkerOptions) (message.NoPublishHandlerFunc, error) {
	return grouphandler.NewNoPublishingHandler(
		opts.EventBus.Marshaler(),
		opts.Router.MetricMeter,

	/*	TODO:

		// Entitlement created event
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *entitlement.EntitlementCreatedEvent) error {
			return w.opts.EventBus.
				WithContext(ctx).
				PublishIfNoError(w.handleEntitlementEvent(
					ctx,
					NamespacedID{Namespace: event.Namespace.ID, ID: event.ID},
					metadata.ComposeResourcePath(event.Namespace.ID, metadata.EntityEntitlement, event.ID),
				))
		}), */
	)
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
