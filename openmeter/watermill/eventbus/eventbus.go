package eventbus

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/config"
	ingestevents "github.com/openmeterio/openmeter/internal/sink/flushhandler/ingestnotification/events"
	"github.com/openmeterio/openmeter/openmeter/watermill/driver/noop"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

type Options struct {
	Publisher              message.Publisher
	Config                 config.EventsConfiguration
	Logger                 *slog.Logger
	MarshalerTransformFunc marshaler.TransformFunc
}

type Publisher interface {
	Publish(ctx context.Context, event marshaler.Event) error

	Marshaler() marshaler.Marshaler
}

type publisher struct {
	eventBus  *cqrs.EventBus
	marshaler marshaler.Marshaler
}

func (p publisher) Publish(ctx context.Context, event marshaler.Event) error {
	return p.eventBus.Publish(ctx, event)
}

func (p publisher) Marshaler() marshaler.Marshaler {
	return p.marshaler
}

func New(opts Options) (Publisher, error) {
	marshaler := marshaler.New(opts.MarshalerTransformFunc)

	ingestVersionSubsystemPrefix := ingestevents.EventVersionSubsystem + "."

	eventBus, err := cqrs.NewEventBusWithConfig(opts.Publisher, cqrs.EventBusConfig{
		GeneratePublishTopic: func(params cqrs.GenerateEventPublishTopicParams) (string, error) {
			switch {
			case strings.HasPrefix(params.EventName, ingestVersionSubsystemPrefix):
				return opts.Config.IngestEvents.Topic, nil
			default:
				return opts.Config.SystemEvents.Topic, nil
			}
		},

		Marshaler: marshaler,
		Logger:    watermill.NewSlogLogger(opts.Logger),
	})
	if err != nil {
		return nil, err
	}

	return publisher{
		eventBus:  eventBus,
		marshaler: marshaler,
	}, nil
}

func NewMock(t *testing.T) Publisher {
	eventBus, err := New(Options{
		Publisher: &noop.Publisher{},
		Config: config.EventsConfiguration{
			SystemEvents: config.EventSubsystemConfiguration{
				Topic: "test",
			},
			IngestEvents: config.EventSubsystemConfiguration{
				Topic: "test",
			},
		},
		Logger: slog.Default(),
	})

	assert.NoError(t, err)
	return eventBus
}
