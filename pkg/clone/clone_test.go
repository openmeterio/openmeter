package clone_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/clone"
)

type testEnum string

func TestCloneEnumPtr(t *testing.T) {
	value := testEnum("a")

	tests := []struct {
		name string
		in   *testEnum
	}{
		{name: "nil stays nil", in: nil},
		{name: "value is copied", in: &value},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := clone.CloneEnumPtr(tt.in)

			if tt.in == nil {
				assert.Nil(t, out)
				return
			}

			require.NotNil(t, out)
			assert.Equal(t, *tt.in, *out)
			assert.NotSame(t, tt.in, out)
		})
	}
}
