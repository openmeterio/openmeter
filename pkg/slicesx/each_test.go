package slicesx

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestForEachUntilWithErr(t *testing.T) {
	t.Run("Empty slice", func(t *testing.T) {
		callCount := 0
		err := ForEachUntilWithErr([]int{}, func(v int, i int) (breaks bool, err error) {
			callCount++
			return false, nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 0, callCount)
	})

	t.Run("Process all elements", func(t *testing.T) {
		callCount := 0
		err := ForEachUntilWithErr([]int{1, 2, 3}, func(v int, i int) (breaks bool, err error) {
			callCount++
			return false, nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 3, callCount)
	})

	t.Run("Break after first element", func(t *testing.T) {
		callCount := 0
		err := ForEachUntilWithErr([]int{1, 2, 3}, func(v int, i int) (breaks bool, err error) {
			callCount++
			return true, nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, callCount)
	})

	t.Run("Return error on second element", func(t *testing.T) {
		callCount := 0
		expectedErr := errors.New("test error")
		err := ForEachUntilWithErr([]int{1, 2, 3}, func(v int, i int) (breaks bool, err error) {
			callCount++
			if callCount == 2 {
				return false, expectedErr
			}
			return false, nil
		})
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 2, callCount)
	})
}

func TestASDF(t *testing.T) {
	asd := []int{1, 2, 3}

	cuc := func(v *[]int) {
		*v = append(*v, 4)
	}

	cuc(&asd)

	assert.Equal(t, []int{1, 2, 3, 4}, asd)
}
