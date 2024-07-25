package noop

import "github.com/ThreeDotsLabs/watermill/message"

type Publisher struct{}

var _ message.Publisher = (*Publisher)(nil)

func (Publisher) Publish(topic string, messages ...*message.Message) error {
	return nil
}

// Close should flush unsent messages, if publisher is async.
func (Publisher) Close() error {
	return nil
}
