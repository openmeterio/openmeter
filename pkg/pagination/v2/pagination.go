package pagination

import (
	"time"
)

// Default values for pagination
const (
	DefaultLimit = 100
	MaxLimit     = 1000
)

// Item is the interface that must be implemented by items used in cursor pagination.
// It provides access to the time and ID fields needed for cursor generation.
type Item interface {
	// Time returns the timestamp used for cursor-based ordering
	Time() time.Time

	// ID returns the unique identifier for this item
	ID() string
}

// CursorParams represents the parameters for cursor-based pagination
type CursorParams struct {
	// Cursor for pagination
	Cursor *string

	// Number of items to return
	Limit int
}

// Validate checks if the parameters are valid
func (p *CursorParams) Validate() error {
	// Validate limit
	if p.Limit <= 0 {
		p.Limit = DefaultLimit
	} else if p.Limit > MaxLimit {
		p.Limit = MaxLimit
	}

	return nil
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
) *Result[T] {
	result := &Result[T]{
		Items:      items,
		TotalCount: totalCount,
	}

	// Generate next cursor from the last item if there are any items
	if len(items) > 0 {
		lastItem := items[len(items)-1]
		cursor := NewCursor(lastItem.Time(), lastItem.ID()).Encode()
		result.NextCursor = &cursor
	}

	return result
}
