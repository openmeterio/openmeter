package publisher

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/cloudevents/sdk-go/v2/event"
)

type TransformFunc func(watermillIn *message.Message, cloudEvent event.Event) (*message.Message, error)
