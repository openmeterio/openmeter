package topicresolver

import (
	"context"
	"errors"
	"fmt"
)

var _ Resolver = (*NamespacedTopicResolver)(nil)

type NamespacedTopicResolver struct {
	// template needs to contain at least one string parameter passed to fmt.Sprintf.
	// For example: "om_%s_events"
	template string
}

func (r NamespacedTopicResolver) Resolve(_ context.Context, namespace string) (string, error) {
	return fmt.Sprintf(r.template, namespace), nil
}

func NewNamespacedTopicResolver(template string) (*NamespacedTopicResolver, error) {
	if template == "" {
		return nil, errors.New("topic name template cannot be empty")
	}
	return &NamespacedTopicResolver{
		template: template,
	}, nil
}
