package chargeadapter_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func requireLedgerBookedAtEqual(t *testing.T, expected, actual time.Time) {
	t.Helper()

	expectedAt := expected.UTC().Truncate(time.Microsecond)
	actualAt := actual.UTC().Truncate(time.Microsecond)
	require.True(t, actualAt.Equal(expectedAt))
}

func requireLedgerBookedAtNotEqual(t *testing.T, expected, actual time.Time) {
	t.Helper()

	expectedAt := expected.UTC().Truncate(time.Microsecond)
	actualAt := actual.UTC().Truncate(time.Microsecond)
	require.False(t, actualAt.Equal(expectedAt))
}
