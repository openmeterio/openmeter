package nopublisher

import (
	"errors"

	"github.com/ThreeDotsLabs/watermill/message"
)

var ErrMessagesProduced = errors.New("messages produced by no publisher handler")

func NoPublisherHandlerToHandlerFunc(h message.NoPublishHandlerFunc) message.HandlerFunc {
	return func(message *message.Message) ([]*message.Message, error) {
		return nil, h(message)
	}
}

func HandlerFuncToNoPublisherHandler(h message.HandlerFunc) message.NoPublishHandlerFunc {
	return func(message *message.Message) error {
		outMessages, err := h(message)
		if len(outMessages) > 0 {
			return ErrMessagesProduced
		}
		return err
	}
}
