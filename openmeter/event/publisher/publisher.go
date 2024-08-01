package publisher

import "github.com/openmeterio/openmeter/internal/event/publisher"

type (
	Publisher           = publisher.Publisher
	PublisherOptions    = publisher.PublisherOptions
	TopicPublisher      = publisher.TopicPublisher
	CloudEventMarshaler = publisher.CloudEventMarshaler
)

type (
	TransformFunc = publisher.TransformFunc
)

func NewPublisher(options PublisherOptions) (Publisher, error) {
	return publisher.NewPublisher(options)
}

func NewCloudEventMarshaler(transform TransformFunc) CloudEventMarshaler {
	return publisher.NewCloudEventMarshaler(transform)
}
