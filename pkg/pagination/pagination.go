package pagination

import (
	"context"
	"encoding/json"
	"fmt"
)

type InvalidError struct {
	p   Page
	msg string
}

func (e InvalidError) Error() string {
	return fmt.Sprintf("invalid page: %+v, %s", e.p, e.msg)
}

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

func (p Page) Validate() error {
	if p.PageSize < 0 {
		return &InvalidError{p, "pagesize cannot be negative"}
	}

	if p.PageNumber < 1 {
		return &InvalidError{p, "page has to be at least 1"}
	}

	return nil
}

func (p Page) IsZero() bool {
	return p.PageSize == 0 && p.PageNumber == 0
}

type PagedResponse[T any] struct {
	Page       Page `json:"-"`
	TotalCount int  `json:"totalCount"`
	Items      []T  `json:"items"`
}

// Implement json.Marshaler interface to flatten the Page struct
func (p PagedResponse[T]) MarshalJSON() ([]byte, error) {
	type Alias PagedResponse[T]
	return json.Marshal(&struct {
		PageSize   int `json:"pageSize"`
		PageNumber int `json:"page"`
		*Alias
	}{
		PageSize:   p.Page.PageSize,
		PageNumber: p.Page.PageNumber,
		Alias:      (*Alias)(&p),
	})
}

type Paginator[T any] interface {
	Paginate(ctx context.Context, page Page) (PagedResponse[T], error)
}
