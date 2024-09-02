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

// NewPage creates a new Page with the given pageNumber and pageSize.
func NewPage(pageNumber int, pageSize int) Page {
	return Page{
		PageSize:   pageSize,
		PageNumber: pageNumber,
	}
}

// NewPageFromRef creates a new Page from pointers to pageNumber and pageSize.
// Useful for handling query parameters.
func NewPageFromRef(pageNumber *int, pageSize *int) Page {
	pn := 0
	ps := 0

	if pageNumber != nil {
		pn = *pageNumber
	}

	if pageSize != nil {
		ps = *pageSize
	}

	return NewPage(pn, ps)
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

// MapPagedResponse creates a new PagedResponse with the given page, totalCount and items.
func MapPagedResponse[Out any, In any](resp PagedResponse[In], m func(in In) Out) PagedResponse[Out] {
	items := make([]Out, 0, len(resp.Items))
	for _, inItem := range resp.Items {
		items = append(items, m(inItem))
	}

	return PagedResponse[Out]{
		Page:       resp.Page,
		TotalCount: resp.TotalCount,
		Items:      items,
	}
}

// MapPagedResponseError is similar to MapPagedResponse
// but it allows the mapping function to return an error.
func MapPagedResponseError[Out any, In any](resp PagedResponse[In], m func(in In) (Out, error)) (PagedResponse[Out], error) {
	items := make([]Out, 0, len(resp.Items))
	for _, inItem := range resp.Items {
		item, err := m(inItem)
		if err != nil {
			return PagedResponse[Out]{}, err
		}

		items = append(items, item)
	}

	return PagedResponse[Out]{
		Page:       resp.Page,
		TotalCount: resp.TotalCount,
		Items:      items,
	}, nil
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
