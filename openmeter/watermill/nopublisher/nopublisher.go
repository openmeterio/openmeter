package nopublisher

import (
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/openmeterio/openmeter/internal/watermill/nopublisher"
)

var ErrMessagesProduced = nopublisher.ErrMessagesProduced

func NoPublisherHandlerToHandlerFunc(h message.NoPublishHandlerFunc) message.HandlerFunc {
	return nopublisher.NoPublisherHandlerToHandlerFunc(h)
}

func HandlerFuncToNoPublisherHandler(h message.HandlerFunc) message.NoPublishHandlerFunc {
	return nopublisher.HandlerFuncToNoPublisherHandler(h)
}
