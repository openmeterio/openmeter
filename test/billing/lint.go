package billing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type testStruct struct {
	ID string
}

func TestLint(t *testing.T) {
	in := []testStruct{
		{ID: "1"},
		{ID: "2"},
		{ID: "3"},
	}

	for _, item := range in {
		item.ID = "4"
	}
	require.Equal(t, in, []testStruct{
		{ID: "4"},
		{ID: "4"},
		{ID: "4"},
	})
}
