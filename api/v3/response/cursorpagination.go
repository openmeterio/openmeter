package response

import (
	"github.com/oapi-codegen/nullable"
	"github.com/samber/lo"

	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
)

// CursorPaginationResponse represents the response structure for cursor-based pagination
type CursorPaginationResponse[T any] struct {
	// The data returned
	Data []T `json:"data"`

	Meta apiv3models.CursorMeta `json:"meta"`
}

// CursorPageEdges holds the encoded cursors of a returned page. Next and
// Previous are set only when that page exists; First and Last only when the
// page has items.
type CursorPageEdges struct {
	First    *string
	Last     *string
	Next     *string
	Previous *string
}

// NewCursorMeta builds the cursor pagination metadata. The cursors are opaque
// tokens that clients send back in page[after] or page[before], not URIs, even
// though the shared schema annotates them as URIs. A missing next or previous
// cursor is returned as null.
func NewCursorMeta(size int, edges CursorPageEdges) apiv3models.CursorMeta {
	meta := apiv3models.CursorMeta{
		Page: apiv3models.CursorMetaPage{
			First:    edges.First,
			Last:     edges.Last,
			Next:     nullable.NewNullNullable[string](),
			Previous: nullable.NewNullNullable[string](),
			Size:     float32(size),
		},
	}

	if edges.Next != nil {
		meta.Page.Next = nullable.NewNullableWithValue(*edges.Next)
	}

	if edges.Previous != nil {
		meta.Page.Previous = nullable.NewNullableWithValue(*edges.Previous)
	}

	return meta
}

// NewCursorPaginationResponse creates a forward-only pagination response from an
// ordered list of items. There is a next page only while a full page was
// returned.
func NewCursorPaginationResponse[T pagination.Item](items []T, pageSize int) CursorPaginationResponse[T] {
	var edges CursorPageEdges

	if len(items) > 0 {
		edges.First = lo.ToPtr(items[0].Cursor().Encode())
		edges.Last = lo.ToPtr(items[len(items)-1].Cursor().Encode())

		if len(items) == pageSize {
			edges.Next = edges.Last
		}
	}

	return CursorPaginationResponse[T]{
		Data: items,
		Meta: NewCursorMeta(pageSize, edges),
	}
}
