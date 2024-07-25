package publisher

import (
	"errors"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/watermill/driver/noop"
)

type Publisher interface {
	ForTopic(topic string) TopicPublisher
}

type PublisherOptions struct {
	// Publisher is the underlying watermill publisher object
	Publisher message.Publisher

	// Transform is a function that can be used to transform the message before it is published, mainly used
	// for driver specific tweaks. If more are required, we should add a chain function.
	Transform TransformFunc
}

type publisher struct {
	publisher message.Publisher
	transform TransformFunc
}

func NewPublisher(opts PublisherOptions) (Publisher, error) {
	if opts.Publisher == nil {
		return nil, errors.New("publisher is required")
	}

	return &publisher{
		publisher: opts.Publisher,
		transform: opts.Transform,
	}, nil
}

func NewMockTopicPublisher(t *testing.T) TopicPublisher {
	pub, err := NewPublisher(PublisherOptions{
		Publisher: noop.Publisher{},
	})

	assert.NoError(t, err)
	return pub.ForTopic("test")
}

func (p *publisher) ForTopic(topic string) TopicPublisher {
	return &topicPublisher{
		publisher: p.publisher,
		topic:     topic,
		transform: p.transform,
	}
}

type TopicPublisher interface {
	Publish(event event.Event) error
}

type topicPublisher struct {
	publisher message.Publisher
	topic     string
	transform TransformFunc
}

func (p *topicPublisher) Publish(event event.Event) error {
	payload, err := event.MarshalJSON()
	if err != nil {
		return err
	}

	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.Metadata.Set("ce_type", event.Type())
	msg.Metadata.Set("ce_time", event.Time().In(time.UTC).Format(time.RFC3339))
	msg.Metadata.Set("ce_source", event.Source())
	if event.Subject() != "" {
		msg.Metadata.Set("ce_subject", event.Subject())
	}

	if p.transform != nil {
		msg, err = p.transform(msg, event)
		if err != nil {
			return err
		}
	}

	return p.publisher.Publish(p.topic, msg)
}
