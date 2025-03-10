package pagination

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestItem implements the Item interface
type TestItem struct {
	ItemID    string
	CreatedAt time.Time
}

// Time implements Item.Time
func (i TestItem) Time() time.Time {
	return i.CreatedAt
}

// ID implements Item.ID
func (i TestItem) ID() string {
	return i.ItemID
}

func TestCursorGeneration(t *testing.T) {
	items := []TestItem{
		{
			ItemID:    "1",
			CreatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			ItemID:    "2",
			CreatedAt: time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC),
		},
		{
			ItemID:    "3",
			CreatedAt: time.Date(2023, 1, 3, 12, 0, 0, 0, time.UTC),
		},
	}

	t.Run("Generate next cursor", func(t *testing.T) {
		result := NewResult(
			items,
			100, // Total count of 100
		)

		// Verify next cursor
		assert.NotNil(t, result.NextCursor)
		assert.Equal(t, int64(100), result.TotalCount)

		// Decode and verify next cursor
		nextCursor, err := DecodeCursor(*result.NextCursor)
		assert.NoError(t, err)
		assert.Equal(t, items[len(items)-1].CreatedAt.UTC(), nextCursor.Time)
		assert.Equal(t, items[len(items)-1].ItemID, nextCursor.ID)
	})

	t.Run("Empty results", func(t *testing.T) {
		emptyResult := NewResult(
			[]TestItem{},
			0, // No items total
		)

		// Verify cursor is not set
		assert.Nil(t, emptyResult.NextCursor)
		assert.Equal(t, int64(0), emptyResult.TotalCount)
	})
}
