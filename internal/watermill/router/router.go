package router

import (
	"errors"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"

	"github.com/openmeterio/openmeter/config"
)

type Options struct {
	Subscriber message.Subscriber
	Publisher  message.Publisher
	Logger     *slog.Logger

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
	poisionQueue, err := middleware.PoisonQueue(opts.Publisher, opts.Config.DLQ.Topic)
	if err != nil {
		return nil, err
	}

	router.AddMiddleware(
		poisionQueue,
	)

	router.AddMiddleware(
		middleware.CorrelationID,
		middleware.Recoverer,
	)

	router.AddMiddleware(middleware.Retry{
		MaxRetries:      opts.Config.Retry.MaxRetries,
		InitialInterval: opts.Config.Retry.InitialInterval,
		MaxInterval:     opts.Config.Retry.MaxInterval,
		MaxElapsedTime:  opts.Config.Retry.MaxElapsedTime,

		Multiplier:          1.5,
		RandomizationFactor: 0.25,
		Logger:              watermill.NewSlogLogger(opts.Logger),
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

	return router, nil
}
