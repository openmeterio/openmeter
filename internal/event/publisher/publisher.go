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

const (
	CloudEventsHeaderType    = "ce_type"
	CloudEventsHeaderTime    = "ce_time"
	CloudEventsHeaderSource  = "ce_source"
	CloudEventsHeaderSubject = "ce_subject"
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
	marshaler CloudEventMarshaler
}

func NewPublisher(opts PublisherOptions) (Publisher, error) {
	if opts.Publisher == nil {
		return nil, errors.New("publisher is required")
	}

	return &publisher{
		publisher: opts.Publisher,
		marshaler: NewCloudEventMarshaler(opts.Transform),
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
		marshaler: p.marshaler,
	}
}

type TopicPublisher interface {
	Publish(event event.Event) error
}

type topicPublisher struct {
	publisher message.Publisher
	topic     string
	marshaler CloudEventMarshaler
}

func (p *topicPublisher) Publish(event event.Event) error {
	msg, err := p.marshaler.MarshalEvent(event)
	if err != nil {
		return err
	}

	return p.publisher.Publish(p.topic, msg)
}

type CloudEventMarshaler interface {
	MarshalEvent(event.Event) (*message.Message, error)
}

type cloudEventMarshaler struct {
	transform TransformFunc
}

func NewCloudEventMarshaler(transform TransformFunc) CloudEventMarshaler {
	return &cloudEventMarshaler{
		transform: transform,
	}
}

func (m *cloudEventMarshaler) MarshalEvent(event event.Event) (*message.Message, error) {
	payload, err := event.MarshalJSON()
	if err != nil {
		return nil, err
	}

	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.Metadata.Set(CloudEventsHeaderType, event.Type())
	msg.Metadata.Set(CloudEventsHeaderTime, event.Time().In(time.UTC).Format(time.RFC3339))
	msg.Metadata.Set(CloudEventsHeaderSource, event.Source())
	if event.Subject() != "" {
		msg.Metadata.Set(CloudEventsHeaderSubject, event.Subject())
	}

	if m.transform != nil {
		msg, err = m.transform(msg, event)
		if err != nil {
			return nil, err
		}
	}

	return msg, nil
}
