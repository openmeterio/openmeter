package consumer

import (
	"context"
	"errors"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/grouphandler"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

var _ Handler = (*WatermillAdapter)(nil)

// WatermillAdapter converts the kafka message to a watermill message and invokes the provided watermill handler. Eventually we will get rid of
// this adapter so that we don't need to convert messages to watermill format just to format it to eventbus.
type WatermillAdapter struct {
	marshaler marshaler.Marshaler
	handler   *grouphandler.NoPublishingHandler
}

func NewWatermillAdapter(marshaler marshaler.Marshaler, h *grouphandler.NoPublishingHandler) (*WatermillAdapter, error) {
	if h == nil {
		return nil, errors.New("handler is required")
	}

	if marshaler == nil {
		return nil, errors.New("marshaler is required")
	}

	return &WatermillAdapter{
		marshaler: marshaler,
		handler:   h,
	}, nil
}

func (a *WatermillAdapter) Handle(ctx context.Context, msg *kafka.Message) error {
	return a.handler.Handle(a.convertKafkaMessageToWatermillMessage(msg))
}

func (a WatermillAdapter) convertKafkaMessageToWatermillMessage(msg *kafka.Message) *message.Message {
	watermillMsg := message.NewMessage(string(msg.Key), msg.Value)

	for _, header := range msg.Headers {
		watermillMsg.Metadata.Set(string(header.Key), string(header.Value))
	}
	return watermillMsg
}

func (a *WatermillAdapter) ExtractEventName(msg *kafka.Message) string {
	return a.marshaler.NameFromMessage(a.convertKafkaMessageToWatermillMessage(msg))
}

func (a *WatermillAdapter) AddHandler(handler grouphandler.GroupEventHandler) {
	a.handler.AddHandler(handler)
}
