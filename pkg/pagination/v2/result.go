package pagination

import "github.com/samber/lo"

// Result represents the response structure for cursor-based pagination
type Result[T any] struct {
	// The items returned
	Items []T `json:"items"`

	// Cursor for the next page
	NextCursor *Cursor `json:"nextCursor"`
}

// NewResult creates a new pagination result from an ordered list of items.
// T must implement the Item interface for cursor generation.
func NewResult[T Item](items []T) Result[T] {
	result := Result[T]{
		Items: items,
	}

	// Generate next cursor from the last item if there are any items
	if len(items) > 0 {
		lastItem := items[len(items)-1]
		result.NextCursor = lo.ToPtr(lastItem.Cursor())
	}

	return result
}
