package response

import (
	"github.com/oapi-codegen/nullable"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/pagination/v2"
)

// CursorMeta Pagination metadata.
type CursorMeta struct {
	Page CursorMetaPage `json:"page"`
}

// CursorMetaPage defines model for CursorMetaPage.
type CursorMetaPage struct {
	// First cursor
	First *string `json:"first,omitempty"`

	// Last cursor
	Last *string `json:"last,omitempty"`

	// Next URI to the next page
	Next nullable.Nullable[string] `json:"next"`

	// Previous URI to the previous page
	Previous nullable.Nullable[string] `json:"previous"`

	// Size of the requested page
	Size int `json:"size"`
}

// CursorPaginationResponse represents the response structure for cursor-based pagination
type CursorPaginationResponse[T any] struct {
	// The data returned
	Data []T `json:"data"`

	Meta CursorMeta `json:"meta"`
}

// NewCursorPaginationResponse creates a new pagination response from an ordered list of items.
// T must implement the Item interface for cursor generation.
func NewCursorPaginationResponse[T pagination.Item](items []T, pageSize int) CursorPaginationResponse[T] {
	result := CursorPaginationResponse[T]{
		Data: items,
		Meta: CursorMeta{
			Page: CursorMetaPage{
				Next:     nullable.NewNullNullable[string](),
				Previous: nullable.NewNullNullable[string](),
				Size:     pageSize,
			},
		},
	}

	// Generate first and last cursor from the first and last item if there are any items
	if len(items) > 0 {
		firstItem := items[0]
		lastItem := items[len(items)-1]
		result.Meta.Page.First = lo.ToPtr(firstItem.Cursor().Encode())
		result.Meta.Page.Last = lo.ToPtr(lastItem.Cursor().Encode())
	}

	return result
}
