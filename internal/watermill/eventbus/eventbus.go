package eventbus

import (
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

type EventBusOptions struct {
	EventsConfig config.EventsConfiguration
	Logger       *slog.Logger
	Publisher    message.Publisher
}

func NewEventBus(opts EventBusOptions) (*cqrs.EventBus, error) {
	return cqrs.NewEventBusWithConfig(opts.Publisher, cqrs.EventBusConfig{
		GeneratePublishTopic: func(params cqrs.GenerateEventPublishTopicParams) (string, error) {
			// TODO: make it generic between sink / server
			return opts.EventsConfig.SystemEvents.Topic, nil
		},

		Marshaler: marshaler.New(),
		Logger:    watermill.NewSlogLogger(opts.Logger),
	})
}
