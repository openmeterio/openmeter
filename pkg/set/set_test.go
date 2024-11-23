package set_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/set"
)

func TestSet(t *testing.T) {
	t.Run("Union", func(t *testing.T) {
		res := set.Union(set.New(1, 2), set.New(2, 3))

		require.ElementsMatch(t, res.AsSlice(), []int{1, 2, 3})
	})

	t.Run("Union (empty)", func(t *testing.T) {
		res := set.Union(set.New(1, 2))

		require.ElementsMatch(t, res.AsSlice(), []int{1, 2})
	})

	t.Run("Subtract", func(t *testing.T) {
		res := set.Subtract(set.New(1, 2, 3), set.New(2, 3))

		require.ElementsMatch(t, res.AsSlice(), []int{1})
	})
}
