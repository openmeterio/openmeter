// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
