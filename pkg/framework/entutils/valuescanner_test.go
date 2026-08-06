package entutils

import (
	"database/sql"
	"testing"

	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
)

type testStringArray []string

type testStringArrayOption struct {
	mo.Option[testStringArray]
}

func newTestStringArrayOption(option mo.Option[testStringArray]) testStringArrayOption {
	return testStringArrayOption{Option: option}
}

func TestJSONStringArrayOptionValueScanner(t *testing.T) {
	t.Parallel()

	scanner := JSONStringArrayOptionValueScanner(newTestStringArrayOption)

	t.Run("value", func(t *testing.T) {
		t.Parallel()

		value, err := scanner.Value(newTestStringArrayOption(mo.None[testStringArray]()))
		require.NoError(t, err)
		require.Nil(t, value)

		value, err = scanner.Value(newTestStringArrayOption(mo.Some(testStringArray(nil))))
		require.NoError(t, err)
		require.Equal(t, []byte("[]"), value)

		value, err = scanner.Value(newTestStringArrayOption(mo.Some(testStringArray{})))
		require.NoError(t, err)
		require.Equal(t, []byte("[]"), value)

		value, err = scanner.Value(newTestStringArrayOption(mo.Some(testStringArray{"one", "two"})))
		require.NoError(t, err)
		require.Equal(t, []byte(`["one","two"]`), value)
	})

	t.Run("scan", func(t *testing.T) {
		t.Parallel()

		value, err := scanner.FromValue(&sql.NullString{})
		require.NoError(t, err)
		require.True(t, value.IsAbsent())

		value, err = scanner.FromValue(&sql.NullString{Valid: true, String: "null"})
		require.NoError(t, err)
		array, ok := value.Get()
		require.True(t, ok)
		require.NotNil(t, array)
		require.Empty(t, array)

		value, err = scanner.FromValue(&sql.NullString{Valid: true, String: "[]"})
		require.NoError(t, err)
		array, ok = value.Get()
		require.True(t, ok)
		require.NotNil(t, array)
		require.Empty(t, array)

		value, err = scanner.FromValue(&sql.NullString{Valid: true, String: `["one","two"]`})
		require.NoError(t, err)
		array, ok = value.Get()
		require.True(t, ok)
		require.Equal(t, testStringArray{"one", "two"}, array)

		_, err = scanner.FromValue(&sql.NullString{Valid: true, String: "{}"})
		require.Error(t, err)
	})
}
