package grouphandler

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

type GroupEventHandler = cqrs.GroupEventHandler

func NewGroupEventHandler[T any](handleFunc func(ctx context.Context, event *T) error) GroupEventHandler {
	return cqrs.NewGroupEventHandler(handleFunc)
}

// NewNoPublishingHandler creates a NoPublishHandlerFunc that will handle events with the provided GroupEventHandlers.
func NewNoPublishingHandler(marshaler cqrs.CommandEventMarshaler, groupHandlers ...GroupEventHandler) message.NoPublishHandlerFunc {
	typeHandlerMap := make(map[string]cqrs.GroupEventHandler)
	for _, groupHandler := range groupHandlers {
		event := groupHandler.NewEvent()
		typeHandlerMap[marshaler.Name(event)] = groupHandler
	}

	return func(msg *message.Message) error {
		eventName := marshaler.NameFromMessage(msg)

		groupHandler, ok := typeHandlerMap[eventName]
		if !ok {
			return nil
		}

		event := groupHandler.NewEvent()

		if err := marshaler.Unmarshal(msg, event); err != nil {
			return err
		}

		return groupHandler.Handle(msg.Context(), event)
	}
}
