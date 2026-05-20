package chargeadapter_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func requireLedgerBookedAtEqual(t *testing.T, expected, actual time.Time) {
	t.Helper()

	require.True(t, actual.UTC().Equal(expected.UTC().Truncate(time.Microsecond)))
}

func requireLedgerBookedAtNotEqual(t *testing.T, expected, actual time.Time) {
	t.Helper()

	require.False(t, actual.UTC().Equal(expected.UTC().Truncate(time.Microsecond)))
}
