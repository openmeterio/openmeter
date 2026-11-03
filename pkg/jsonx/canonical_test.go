package jsonx_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/jsonx"
)

func TestCanonical(t *testing.T) {
	canonical := func(doc string) string {
		t.Helper()

		out, err := jsonx.Canonical([]byte(doc))
		require.NoError(t, err)

		return string(out)
	}

	// Whitespace is dropped and object keys are sorted.
	require.Equal(t, `{"a":{"c":1,"d":2},"b":[1,2]}`, canonical(`{ "b": [1, 2], "a": {"d": 2, "c": 1} }`))

	// Array order and number text are kept, so values that only mean the same
	// thing stay different.
	require.Equal(t, `["y","x"]`, canonical(`["y", "x"]`))
	require.Equal(t, `1.0`, canonical(`1.0`))

	// Large integers keep their exact value.
	require.Equal(t, `9007199254740993`, canonical(`9007199254740993`))

	// HTML characters are not escaped, so "<" and "<" are the same text.
	require.Equal(t, `"<a>"`, canonical(`"<a>"`))

	for _, invalid := range []string{``, `{`, `{} {}`, `{"a":1}x`} {
		_, err := jsonx.Canonical([]byte(invalid))
		require.Error(t, err, invalid)
	}
}
