package router

import (
	"github.com/ThreeDotsLabs/watermill/message"
)

// RestoreContext ensures that the original context is restored after the handler is done processing the message.
//
// This helps with https://github.com/ThreeDotsLabs/watermill/issues/467
func RestoreContext(h message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		origCtx := msg.Context()
		defer func() {
			msg.SetContext(origCtx)
		}()

		return h(msg)
	}
}
