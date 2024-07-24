package pagination

import (
	"context"
	"encoding/json"
)

type Page struct {
	PageSize   int `json:"pageSize"`
	PageNumber int `json:"page"`
}

func (p Page) Offset() int {
	return p.PageSize * (p.PageNumber - 1)
}

func (p Page) Limit() int {
	return p.PageSize
}

func (p Page) IsZero() bool {
	return p.PageSize == 0 && p.PageNumber == 0
}

type PagedResponse[T any] struct {
	Items      []T `json:"items"`
	TotalCount int `json:"totalCount"`
	// flattens the page struct
	Page Page `json:"-"`
}

// Implement json.Marshaler interface to flatten the Page struct
func (p PagedResponse[T]) MarshalJSON() ([]byte, error) {
	type Alias PagedResponse[T]
	return json.Marshal(&struct {
		*Alias
		PageSize   int `json:"pageSize"`
		PageNumber int `json:"page"`
	}{
		Alias:      (*Alias)(&p),
		PageSize:   p.Page.PageSize,
		PageNumber: p.Page.PageNumber,
	})
}

type Paginator[T any] interface {
	Paginate(ctx context.Context, page Page) (PagedResponse[T], error)
}
