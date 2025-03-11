package pagination

import "github.com/samber/lo"

// Item is the interface that must be implemented by items used in cursor pagination.
// It provides access to the time and ID fields needed for cursor generation.
type Item interface {
	// Cursor returns the cursor used for cursor-based ordering
	Cursor() Cursor
}

// Result represents the response structure for cursor-based pagination
type Result[T any] struct {
	// The items returned
	Items []T `json:"items"`

	// The total count of items
	TotalCount int64 `json:"totalCount"`

	// Cursor for the next page
	NextCursor *string `json:"nextCursor"`
}

// NewResult creates a new pagination result
// T must implement the Item interface for cursor generation
func NewResult[T Item](
	items []T,
	totalCount int64,
) Result[T] {
	result := Result[T]{
		Items:      items,
		TotalCount: totalCount,
	}

	// Generate next cursor from the last item if there are any items
	if len(items) > 0 {
		lastItem := items[len(items)-1]
		result.NextCursor = lo.ToPtr(lastItem.Cursor().Encode())
	}

	return result
}
