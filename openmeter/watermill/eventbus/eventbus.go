package eventbus

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"

	balanceworkerevents "github.com/openmeterio/openmeter/openmeter/entitlement/balanceworker/events"
	ingestevents "github.com/openmeterio/openmeter/openmeter/sink/flushhandler/ingestnotification/events"
	"github.com/openmeterio/openmeter/openmeter/watermill/driver/noop"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

type TopicMapping struct {
	IngestEventsTopic        string
	SystemEventsTopic        string
	BalanceWorkerEventsTopic string
}

func (t TopicMapping) Validate() error {
	if t.IngestEventsTopic == "" {
		return errors.New("ingest events topic is required")
	}

	if t.SystemEventsTopic == "" {
		return errors.New("system events topic is required")
	}

	if t.BalanceWorkerEventsTopic == "" {
		return errors.New("balance worker events topic is required")
	}

	return nil
}

type Options struct {
	Publisher              message.Publisher
	TopicMapping           TopicMapping
	Logger                 *slog.Logger
	MarshalerTransformFunc marshaler.TransformFunc
}

func (o Options) Validate() error {
	if o.Publisher == nil {
		return errors.New("publisher is required")
	}

	if err := o.TopicMapping.Validate(); err != nil {
		return fmt.Errorf("topic mapping: %w", err)
	}

	if o.Logger == nil {
		return errors.New("logger is required")
	}

	return nil
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
	if event == nil {
		// nil events are always ignored as the handler signifies that it doesn't want to publish anything
		return nil
	}

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
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	marshaler := marshaler.New(opts.MarshalerTransformFunc)

	ingestVersionSubsystemPrefix := ingestevents.EventVersionSubsystem + "."
	balanceWorkerVersionSubsystemPrefix := balanceworkerevents.EventVersionSubsystem + "."

	eventBus, err := cqrs.NewEventBusWithConfig(opts.Publisher, cqrs.EventBusConfig{
		GeneratePublishTopic: func(params cqrs.GenerateEventPublishTopicParams) (string, error) {
			switch {
			case strings.HasPrefix(params.EventName, ingestVersionSubsystemPrefix):
				return opts.TopicMapping.IngestEventsTopic, nil
			case strings.HasPrefix(params.EventName, balanceWorkerVersionSubsystemPrefix):
				return opts.TopicMapping.BalanceWorkerEventsTopic, nil
			default:
				return opts.TopicMapping.SystemEventsTopic, nil
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
		TopicMapping: TopicMapping{
			IngestEventsTopic:        "test-ingest-events",
			SystemEventsTopic:        "test-system-events",
			BalanceWorkerEventsTopic: "test-balance-worker-events",
		},
		Logger: slog.Default(),
	})

	assert.NoError(t, err)
	return eventBus
}
