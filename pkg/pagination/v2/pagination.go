package pagination

import (
	"context"
	"errors"
)

var (
	ErrCursorPaginationSizeInvalid   = errors.New("size must be greater than 0")
	ErrCursorPaginationRange         = errors.New("range pagination not supported, both before and after cursor were defined")
	ErrCursorPaginationAfterInvalid  = errors.New("after cursor is invalid")
	ErrCursorPaginationBeforeInvalid = errors.New("before cursor is invalid")
)

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

type CursorPagination struct {
	Size   int
	After  *Cursor
	Before *Cursor
}

func (p *CursorPagination) Validate() error {
	if p.Size < 1 {
		return ErrCursorPaginationSizeInvalid
	}

	if p.After != nil && p.Before != nil {
		return ErrCursorPaginationRange
	}

	if p.After != nil && p.After.Validate() != nil {
		return ErrCursorPaginationAfterInvalid
	}

	if p.Before != nil && p.Before.Validate() != nil {
		return ErrCursorPaginationBeforeInvalid
	}

	return nil
}
