package pagination

import (
	"context"
)

type Paginator[T any] interface {
	Paginate(ctx context.Context, page Page) (Result[T], error)
}

type paginator[T any] struct {
	fn func(ctx context.Context, page Page) (Result[T], error)
}

var _ Paginator[any] = (*paginator[any])(nil)

func (p *paginator[T]) Paginate(ctx context.Context, page Page) (Result[T], error) {
	return p.fn(ctx, page)
}

func NewPaginator[T any](fn func(ctx context.Context, page Page) (Result[T], error)) Paginator[T] {
	return &paginator[T]{fn: fn}
}
