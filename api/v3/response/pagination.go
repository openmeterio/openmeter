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
	Size float32 `json:"size"`
}

// CursorPaginationResponse represents the response structure for cursor-based pagination
type CursorPaginationResponse[T any] struct {
	// The data returned
	Data []T `json:"data"`

	Meta CursorMeta `json:"meta"`
}

type OffsetPaginationResponse[T any] struct {
	Data []T        `json:"data"`
	Meta OffsetMeta `json:"meta"`
}

type OffsetMeta struct {
	Page OffsetMetaPage `json:"page"`
}

type OffsetMetaPage struct {
	Size           int  `json:"size"`
	Number         int  `json:"number"`
	Total          *int `json:"total,omitempty"`
	EstimatedTotal *int `json:"estimatedTotal,omitempty"`
}

// NewCursorPaginationResponse creates a new pagination response from an ordered list of items.
// T must implement the Item interface for cursor generation.
func NewCursorPaginationResponse[T pagination.Item](items []T) CursorPaginationResponse[T] {
	result := CursorPaginationResponse[T]{
		Data: items,
		Meta: CursorMeta{
			Page: CursorMetaPage{
				Next:     nullable.NewNullNullable[string](),
				Previous: nullable.NewNullNullable[string](),
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

// // BuildRequestPathWithCursor generates the request path & querystring with new encoded cursors
// // to be return with the cursor pagination metadata in a paginated endpoint.
// func BuildRequestPathWithCursor(r *http.Request, result *CursorPaginationResponse[any]) (string, error) {
// 	out := ""

// 	rr, err := http.NewRequest(r.Method, r.URL.String(), nil)
// 	if err != nil {
// 		return out, err
// 	}

// 	q := rr.URL.Query()
// 	q.Del(request.PageAfterQuery)
// 	q.Del(request.PageBeforeQuery)
// 	if result.Meta.Page.First != nil {
// 		q.Set(request.PageAfterQuery, *result.Meta.Page.First)
// 	}
// 	if result.Meta.Page.Last != nil {
// 		q.Set(request.PageBeforeQuery, *result.Meta.Page.Last)
// 	}

// 	rr.URL.RawQuery = q.Encode()

// 	return fmt.Sprintf("%s?%s", rr.URL.Path, rr.URL.RawQuery), nil
// }

func NewOffsetPaginationResponse[T any](items []T, page OffsetMetaPage) OffsetPaginationResponse[T] {
	return OffsetPaginationResponse[T]{
		Data: items,
		Meta: OffsetMeta{
			Page: page,
		},
	}
}
