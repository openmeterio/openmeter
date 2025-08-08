package router

import (
	"errors"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/config"
)

type Options struct {
	Subscriber  message.Subscriber
	Publisher   message.Publisher
	Logger      *slog.Logger
	MetricMeter metric.Meter
	Tracer      trace.Tracer

	Config config.ConsumerConfiguration
}

func (o *Options) Validate() error {
	if o.Subscriber == nil {
		return errors.New("subscriber is required")
	}

	if o.Publisher == nil {
		return errors.New("publisher is required")
	}

	if o.Logger == nil {
		return errors.New("logger is required")
	}

	if err := o.Config.Validate(); err != nil {
		return err
	}

	if o.MetricMeter == nil {
		return errors.New("metric meter is required")
	}

	if o.Tracer == nil {
		return errors.New("tracer is required")
	}

	return nil
}

// NewDefaultRouter creates a new router with the default middlewares, in case your consumer
// would mandate a different setup, feel free to create your own router
func NewDefaultRouter(opts Options) (*message.Router, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	router, err := message.NewRouter(message.RouterConfig{}, watermill.NewSlogLogger(opts.Logger))
	if err != nil {
		return nil, err
	}

	// This should be the outermost middleware, to catch failures including the ones caused by Recoverer

	// If retry queue is not enabled, we can directly push messages to the DLQ
	poisionQueue, err := middleware.PoisonQueueWithFilter(
		opts.Publisher,
		opts.Config.DLQ.Topic,
		func(err error) bool {
			// If the router is closed, we don't want to push to the DLQ as Close() will cancel the context almost
			// immediately.
			//
			// Propagating the error (skipping from poision queue) means that the message will be NAcked, meaning that
			// it will be retried at least for Kafka.
			if router.IsClosed() {
				return false
			}

			return true
		},
	)
	if err != nil {
		return nil, err
	}

	router.AddMiddleware(
		poisionQueue,
	)

	dlqMetrics, err := NewDLQTelemetryMiddleware(NewDLQTelemetryOptions{
		MetricMeter: opts.MetricMeter,
		Prefix:      "consumer",
		Logger:      opts.Logger,
		Router:      router,
		Tracer:      opts.Tracer,
	})
	if err != nil {
		return nil, err
	}

	router.AddMiddleware(dlqMetrics)

	router.AddMiddleware(
		middleware.CorrelationID,
		middleware.Recoverer,
	)

	// Note: The Retry middleware executes retries MaxRetries + 1 times, so let's fix it here
	maxRetries := opts.Config.Retry.MaxRetries
	if maxRetries > 0 {
		maxRetries = maxRetries - 1
	}

	router.AddMiddleware(middleware.Retry{
		MaxRetries:      maxRetries,
		InitialInterval: opts.Config.Retry.InitialInterval,
		MaxInterval:     opts.Config.Retry.MaxInterval,
		MaxElapsedTime:  opts.Config.Retry.MaxElapsedTime,

		Multiplier:          1.5,
		RandomizationFactor: 0.25,
		Logger:              newWarningOnlyLogger(opts.Logger),
	}.Middleware)

	// This should be after Retry, so that we can retry on timeouts before pushing to DLQ
	if opts.Config.ProcessingTimeout > 0 {
		router.AddMiddleware(
			// The Timeout middleware keeps the messages context overridden after returning, thus the retry will
			// also timeout, thus we need to save the context before applying the Timeout middleware
			//
			// Issue: https://github.com/ThreeDotsLabs/watermill/issues/467
			RestoreContext,
			middleware.Timeout(opts.Config.ProcessingTimeout),
		)
	}

	// This should be the last to report every message processing try
	handlerMetrics, err := HandlerMetrics(opts.MetricMeter, "consumer", opts.Logger)
	if err != nil {
		return nil, err
	}
	router.AddMiddleware(handlerMetrics)

	return router, nil
}
