package eventbus

import (
	"testing"

	"github.com/openmeterio/openmeter/internal/watermill/eventbus"
)

type (
	Publisher        = eventbus.Publisher
	ContextPublisher = eventbus.ContextPublisher
	Options          = eventbus.Options
)

func New(options Options) (Publisher, error) {
	return eventbus.New(options)
}

func NewMock(t *testing.T) Publisher {
	return eventbus.NewMock(t)
}
