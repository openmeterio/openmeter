package set

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSet(t *testing.T) {
	t.Run("Union", func(t *testing.T) {
		res := Union(New(1, 2), New(2, 3))

		assert.ElementsMatch(t, res.AsSlice(), []int{1, 2, 3})
	})

	t.Run("Union (empty)", func(t *testing.T) {
		res := Union(New(1, 2))

		assert.ElementsMatch(t, res.AsSlice(), []int{1, 2})
	})

	t.Run("Subtract", func(t *testing.T) {
		res := Subtract(New(1, 2, 3), New(2, 3))

		assert.ElementsMatch(t, res.AsSlice(), []int{1})
	})
}

func TestSet_IsEmpty(t *testing.T) {
	t.Run("new set is empty", func(t *testing.T) {
		// Create a new empty set
		s := New[string]()

		// Check that it's empty
		assert.True(t, s.IsEmpty(), "A newly created set with no items should be empty")
	})

	t.Run("set with items is not empty", func(t *testing.T) {
		// Create a set with items
		s := New("item1", "item2", "item3")

		// Check that it's not empty
		assert.False(t, s.IsEmpty(), "A set with items should not be empty")
	})

	t.Run("set becomes empty after removing all items", func(t *testing.T) {
		// Create a set with items
		s := New("item1", "item2")

		// Remove the items
		s.Remove("item1", "item2")

		// Check that it's now empty
		assert.True(t, s.IsEmpty(), "A set should be empty after removing all items")
	})

	t.Run("empty set becomes non-empty after adding an item", func(t *testing.T) {
		// Create an empty set
		s := New[string]()

		// Check that it starts empty
		assert.True(t, s.IsEmpty(), "A newly created set with no items should be empty")

		// Add an item
		s.Add("item1")

		// Check that it's no longer empty
		assert.False(t, s.IsEmpty(), "A set should not be empty after adding an item")
	})

	t.Run("concurrency safety test", func(t *testing.T) {
		// This test doesn't really verify concurrency safety directly,
		// but it serves as a smoke test for the locking mechanism
		s := New[int]()

		// Add and remove in succession to exercise the locks
		for i := 0; i < 100; i++ {
			s.Add(i)
			assert.False(t, s.IsEmpty(), "Set should not be empty after adding an item")
			s.Remove(i)
			assert.True(t, s.IsEmpty(), "Set should be empty after removing all items")
		}
	})
}
