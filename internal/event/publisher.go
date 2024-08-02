package event

import "context"

type Publisher interface {
	// TODO: can we constraint it to accept only events?
	Publish(ctx context.Context, event any) error
}
