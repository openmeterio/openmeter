package balanceworker

import (
	"context"
	"errors"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/openmeterio/openmeter/openmeter/consumer"
	"github.com/openmeterio/openmeter/openmeter/watermill/grouphandler"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
)

type Consumer interface {
	Run(ctx context.Context) error
	Close() error
	AddHandler(handler grouphandler.GroupEventHandler)
}

type WatermillConsumerOptions struct {
	SystemEventsTopic        string
	IngestEventsTopic        string
	BalanceWorkerEventsTopic string

	Worker *Worker
	Router router.Options
}

func (o WatermillConsumerOptions) Validate() error {
	var errs []error

	if o.SystemEventsTopic == "" {
		errs = append(errs, errors.New("system events topic is required"))
	}

	if o.IngestEventsTopic == "" {
		errs = append(errs, errors.New("ingest events topic is required"))
	}

	if o.BalanceWorkerEventsTopic == "" {
		errs = append(errs, errors.New("balance worker events topic is required"))
	}

	if o.Worker == nil {
		errs = append(errs, errors.New("worker is required"))
	}

	if err := o.Router.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("router: %w", err))
	}

	return errors.Join(errs...)
}

var _ Consumer = (*WatermillConsumer)(nil)

type WatermillConsumer struct {
	router               *message.Router
	nonPublishingHandler *grouphandler.NoPublishingHandler
}

func NewWatermillConsumer(opts WatermillConsumerOptions) (*WatermillConsumer, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate watermill consumer options: %w", err)
	}

	r, err := router.NewDefaultRouter(opts.Router)
	if err != nil {
		return nil, err
	}

	eventHandler, err := opts.Worker.eventHandler(opts.Router.MetricMeter)
	if err != nil {
		return nil, err
	}

	r.AddConsumerHandler(
		"balance_worker_system_events",
		opts.SystemEventsTopic,
		opts.Router.Subscriber,
		eventHandler.Handle,
	)

	r.AddConsumerHandler(
		"balance_worker_ingest_events",
		opts.IngestEventsTopic,
		opts.Router.Subscriber,
		eventHandler.Handle,
	)

	r.AddConsumerHandler(
		"balance_worker_balance_worker_events",
		opts.BalanceWorkerEventsTopic,
		opts.Router.Subscriber,
		eventHandler.Handle,
	)

	return &WatermillConsumer{
		router:               r,
		nonPublishingHandler: eventHandler,
	}, nil
}

// AddHandler adds an additional handler to the list of batched ingest event handlers.
// Handlers are called in the order they are added and run after the riginal balance worker handler.
// In the case of any handler returning an error, the event will be retried so it is important that all handlers are idempotent.
func (c *WatermillConsumer) AddHandler(handler grouphandler.GroupEventHandler) {
	c.nonPublishingHandler.AddHandler(handler)
}

func (w *WatermillConsumer) Run(ctx context.Context) error {
	return w.router.Run(ctx)
}

func (w *WatermillConsumer) Close() error {
	if err := w.router.Close(); err != nil {
		return err
	}

	return nil
}

type RDKafkaConsumerOptions struct {
	SystemEventsTopic        string
	IngestEventsTopic        string
	BalanceWorkerEventsTopic string

	EventBusMarshaler   marshaler.Marshaler
	ConsumerEnvironment consumer.EnvironmentConfig

	Worker *Worker
}

func (o RDKafkaConsumerOptions) Validate() error {
	var errs []error

	if o.Worker == nil {
		errs = append(errs, errors.New("worker is required"))
	}

	if err := o.ConsumerEnvironment.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("consumer environment: %w", err))
	}

	if o.EventBusMarshaler == nil {
		errs = append(errs, errors.New("event bus marshaler is required"))
	}

	if o.SystemEventsTopic == "" {
		errs = append(errs, errors.New("system events topic is required"))
	}

	if o.IngestEventsTopic == "" {
		errs = append(errs, errors.New("ingest events topic is required"))
	}

	if o.BalanceWorkerEventsTopic == "" {
		errs = append(errs, errors.New("balance worker events topic is required"))
	}

	return errors.Join(errs...)
}

var _ Consumer = (*RDKafkaConsumer)(nil)

// RDKafkaConsumer is a consumer that uses librdkafka to consume messages from Kafka topics.
type RDKafkaConsumer struct {
	consumer.Consumer
	worker  *Worker
	adapter *consumer.WatermillAdapter
}

func NewRDKafkaConsumer(opts RDKafkaConsumerOptions) (*RDKafkaConsumer, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate rdkafka consumer options: %w", err)
	}

	eventHandler, err := opts.Worker.eventHandler(opts.ConsumerEnvironment.MetricMeter)
	if err != nil {
		return nil, fmt.Errorf("failed to create event handler: %w", err)
	}

	wmAdapter, err := consumer.NewWatermillAdapter(opts.EventBusMarshaler, eventHandler)
	if err != nil {
		return nil, fmt.Errorf("failed to create watermill adapter: %w", err)
	}

	c, err := consumer.New(consumer.Config{
		Environment: opts.ConsumerEnvironment,
		Topics: []string{
			opts.SystemEventsTopic,
			opts.IngestEventsTopic,
			opts.BalanceWorkerEventsTopic,
		},
		Handler: wmAdapter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create rdkafka consumer: %w", err)
	}

	return &RDKafkaConsumer{
		Consumer: c,
		adapter:  wmAdapter,
		worker:   opts.Worker,
	}, nil
}

func (c *RDKafkaConsumer) AddHandler(handler grouphandler.GroupEventHandler) {
	c.adapter.AddHandler(handler)
}
