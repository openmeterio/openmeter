package pagination

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

// TestItem implements the Item interface
type TestItem struct {
	ID        string
	CreatedAt time.Time
}

// GetTime implements Item.GetTime
func (i TestItem) Cursor() Cursor {
	return NewCursor(i.CreatedAt, i.ID)
}

func TestCursorGeneration(t *testing.T) {
	items := []TestItem{
		{
			ID:        "1",
			CreatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			ID:        "2",
			CreatedAt: time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC),
		},
		{
			ID:        "3",
			CreatedAt: time.Date(2023, 1, 3, 12, 0, 0, 0, time.UTC),
		},
	}

	t.Run("Generate next cursor", func(t *testing.T) {
		result := NewResult(items)

		// Verify next cursor
		assert.NotNil(t, result.NextCursor)

		// Decode and verify next cursor
		nextCursor := lo.FromPtr(result.NextCursor)
		assert.NoError(t, nextCursor.Validate())
		assert.Equal(t, items[len(items)-1].CreatedAt.UTC(), nextCursor.Time)
		assert.Equal(t, items[len(items)-1].ID, nextCursor.ID)
	})

	t.Run("Empty results", func(t *testing.T) {
		emptyResult := NewResult([]TestItem{})

		// Verify cursor is not set
		assert.Nil(t, emptyResult.NextCursor)
	})
}
