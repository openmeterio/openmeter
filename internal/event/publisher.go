package event

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

type Publisher interface {
	Publish(ctx context.Context, event marshaler.Event) error
}
