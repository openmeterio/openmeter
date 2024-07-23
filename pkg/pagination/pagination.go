package pagination

import "context"

type Page struct {
	PageSize   int
	PageNumber int
}

func (p Page) Offset() int {
	return p.PageSize * (p.PageNumber - 1)
}

func (p Page) Limit() int {
	return p.PageSize
}

type PagedResponse[T any] struct {
	Items      []T
	TotalCount int
	Page       Page
}

type Paginator[T any] interface {
	Paginate(ctx context.Context, page Page) (PagedResponse[T], error)
}
