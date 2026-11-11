package worker

import (
	"context"
	"errors"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	"github.com/openmeterio/openmeter/openmeter/watermill/grouphandler"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Config struct {
	Service           rte.Service
	SystemEventsTopic string
	Marshaler         marshaler.Marshaler
	Router            router.Options
}

func (c Config) Validate() error {
	var errs []error
	if c.Service == nil {
		errs = append(errs, errors.New("service is required"))
	}

	if c.SystemEventsTopic == "" {
		errs = append(errs, errors.New("system events topic is required"))
	}

	if c.Marshaler == nil {
		errs = append(errs, errors.New("marshaler is required"))
	}

	if err := c.Router.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("router: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type worker struct {
	service rte.Service
	router  *message.Router
}

func New(config Config) (rte.Worker, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	r, err := router.NewDefaultRouter(config.Router)
	if err != nil {
		return nil, err
	}

	w := &worker{service: config.Service, router: r}
	handler, err := grouphandler.NewNoPublishingHandler(config.Marshaler, config.Router.MetricMeter,
		grouphandler.NewGroupEventHandler(w.handleRequested),
	)
	if err != nil {
		return nil, err
	}

	r.AddConsumerHandler("rte_calculation_requests", config.SystemEventsTopic, config.Router.Subscriber, handler.Handle)

	return w, nil
}

func (w *worker) handleRequested(ctx context.Context, event *rte.CalculationRequestedEvent) error {
	key, err := event.DecodeKey()
	if err != nil {
		return err
	}

	switch key := key.(type) {
	case creditbalance.Key:
		_, err = w.service.CreditBalances().Calculate(ctx, key)
	case meteredentitlement.Key:
		_, err = w.service.MeteredEntitlements().Calculate(ctx, key)
	}

	return err
}

func (w *worker) Run(ctx context.Context) error {
	return w.router.Run(ctx)
}

func (w *worker) Close() error {
	return w.router.Close()
}

func (w *worker) Running() <-chan struct{} {
	return w.router.Running()
}
