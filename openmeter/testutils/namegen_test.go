package testutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NameGen(t *testing.T) {
	t.Run("Generate", func(t *testing.T) {
		names := NameGenerator.Generate()
		t.Logf("Generated name: %v", names)

		require.NotEmpty(t, names)
		assert.NotEmpty(t, names.Key)
		assert.NotEmpty(t, names.Name)
	})
}
