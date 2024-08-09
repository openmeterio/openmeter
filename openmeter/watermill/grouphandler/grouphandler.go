package grouphandler

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/openmeterio/openmeter/internal/watermill/grouphandler"
)

type (
	GroupEventHandler = grouphandler.GroupEventHandler
)

func NewGroupEventHandler[T any](handleFunc func(ctx context.Context, event *T) error) GroupEventHandler {
	return grouphandler.NewGroupEventHandler(handleFunc)
}

func NewNoPublishingHandler(marshaler cqrs.CommandEventMarshaler, groupHandlers ...GroupEventHandler) message.NoPublishHandlerFunc {
	return grouphandler.NewNoPublishingHandler(marshaler, groupHandlers...)
}
