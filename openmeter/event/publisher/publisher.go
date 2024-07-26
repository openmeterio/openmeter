package publisher

import "github.com/openmeterio/openmeter/internal/event/publisher"

type (
	Publisher        = publisher.Publisher
	PublisherOptions = publisher.PublisherOptions
	TopicPublisher   = publisher.TopicPublisher
)

func NewPublisher(options PublisherOptions) (Publisher, error) {
	return publisher.NewPublisher(options)
}
