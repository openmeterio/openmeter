package topicresolver

import "context"

// Resolver use provided namespace to return a topic name belongs to that namespace.
type Resolver interface {
	Resolve(ctx context.Context, namespace string) (string, error)
}
