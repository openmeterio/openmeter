package pagination

import "context"

// Item is the interface that must be implemented by items used in cursor pagination.
// It provides access to the time and ID fields needed for cursor generation.
type Item interface {
	// Cursor returns the cursor used for cursor-based ordering
	Cursor() Cursor
}

type Paginator[T any] interface {
	Paginate(ctx context.Context, cursor *Cursor) (Result[T], error)
}

type paginator[T any] struct {
	fn func(ctx context.Context, cursor *Cursor) (Result[T], error)
}

var _ Paginator[any] = (*paginator[any])(nil)

func (p *paginator[T]) Paginate(ctx context.Context, cursor *Cursor) (Result[T], error) {
	return p.fn(ctx, cursor)
}

func NewPaginator[T any](fn func(ctx context.Context, cursor *Cursor) (Result[T], error)) Paginator[T] {
	return &paginator[T]{fn: fn}
}
