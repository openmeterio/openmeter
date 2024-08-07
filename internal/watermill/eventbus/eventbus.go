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
	"github.com/openmeterio/openmeter/internal/watermill/driver/noop"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

type Options struct {
	Publisher              message.Publisher
	Config                 config.EventsConfiguration
	Logger                 *slog.Logger
	MarshalerTransformFunc marshaler.TransformFunc
}

type Publisher interface {
	// Publish publishes an event to the event bus
	Publish(ctx context.Context, event marshaler.Event) error

	// WithContext is a convinience method to publish from the router. Usually if we need
	// to publish from the router, a function returns a marshaler.Event and an error. Using this
	// method we can inline the publish call and avoid the need to check for errors:
	//
	//    return p.WithContext(ctx).PublishIfNoError(worker.handleEvent(ctx, event))
	WithContext(ctx context.Context) ContextPublisher

	Marshaler() marshaler.Marshaler
}

type ContextPublisher interface {
	// PublishIfNoError publishes an event if the error is nil or returns the error
	PublishIfNoError(event marshaler.Event, err error) error
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

type contextPublisher struct {
	publisher *publisher
	ctx       context.Context
}

func (p publisher) WithContext(ctx context.Context) ContextPublisher {
	return contextPublisher{
		publisher: &p,
		ctx:       ctx,
	}
}

func (p contextPublisher) PublishIfNoError(event marshaler.Event, err error) error {
	if err != nil {
		return err
	}

	return p.publisher.Publish(p.ctx, event)
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
