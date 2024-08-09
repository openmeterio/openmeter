package router

import (
	"errors"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/internal/watermill/nopublisher"
)

type Options struct {
	Subscriber message.Subscriber
	Publisher  message.Publisher
	Logger     *slog.Logger

	DLQ config.DLQConfiguration
}

func (o *Options) Validate() error {
	if o.Subscriber == nil {
		return errors.New("subscriber is required")
	}

	if o.DLQ.Enabled && o.Publisher == nil {
		return errors.New("publisher is required")
	}

	if o.Logger == nil {
		return errors.New("logger is required")
	}

	if o.DLQ.Enabled {
		if err := o.DLQ.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// NewDefaultRouter creates a new router with the default middlewares, in case your consumer
// would mandate a different setup, feel free to create your own router
//
// dlqHandler is the handler that will be called when a message is consumed from the DLQ,
// this is specified separately as the options struct is initialized externally from the consumer
// and the handler is initialized internally
func NewDefaultRouter(opts Options, dlqHandler message.NoPublishHandlerFunc) (*message.Router, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	if opts.DLQ.Enabled {
		if dlqHandler == nil {
			return nil, errors.New("dlq handler is required")
		}
	}

	router, err := message.NewRouter(message.RouterConfig{}, watermill.NewSlogLogger(opts.Logger))
	if err != nil {
		return nil, err
	}

	router.AddMiddleware(
		middleware.CorrelationID,

		middleware.Retry{
			MaxRetries:      5,
			InitialInterval: 100 * time.Millisecond,
			Logger:          watermill.NewSlogLogger(opts.Logger),
		}.Middleware,

		middleware.Recoverer,
	)

	if opts.DLQ.Enabled {
		poisionQueue, err := middleware.PoisonQueue(opts.Publisher, opts.DLQ.Topic)
		if err != nil {
			return nil, err
		}

		router.AddMiddleware(
			poisionQueue,
		)

		poisionQueueProcessor := nopublisher.NoPublisherHandlerToHandlerFunc(dlqHandler)
		if opts.DLQ.Throttle.Enabled {
			poisionQueueProcessor = middleware.NewThrottle(
				opts.DLQ.Throttle.Count,
				opts.DLQ.Throttle.Duration,
			).Middleware(poisionQueueProcessor)
		}
		router.AddNoPublisherHandler(
			"process_dlq",
			opts.DLQ.Topic,
			opts.Subscriber,
			nopublisher.HandlerFuncToNoPublisherHandler(poisionQueueProcessor),
		)
	}
	return router, nil
}
