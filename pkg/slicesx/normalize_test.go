package slicesx

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalize(t *testing.T) {
	require.Equal(t, []string{"api-calls", "storage"}, Normalize([]string{"storage", "api-calls", "storage"}))
	require.Equal(t, []int{1, 2, 3}, Normalize([]int{3, 1, 2, 1}))
	require.Nil(t, Normalize([]string(nil)))
}
