package namespacedriver

import (
	"context"
	"errors"
)

// TODO: move to the rigt place

type Wrapped[T any] struct {
	Request   T
	Namespace string
}

func Wrap[T any](ctx context.Context, request T, resolver NamespaceDecoder) (*Wrapped[T], error) {
	ns, found := resolver.GetNamespace(ctx)
	// TODO: return error instead?
	if !found {
		return nil, errors.New("TODO")
	}

	return &Wrapped[T]{
		Request:   request,
		Namespace: ns,
	}, nil
}
