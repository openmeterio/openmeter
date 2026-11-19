package timeutil_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func mustRFC3339(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)

	return parsed
}

// The iterator returned by IterateFromNextAfter/IterateFromPrevBefore must keep
// its iteration index in sync with At. Callers advance it with Next and expect
// the following recurrence, not the one they already got.
func TestRecurrenceIteratorIndexStaysInSync(t *testing.T) {
	t.Run("next after with anchor in future", func(t *testing.T) {
		recurrence := timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodDaily,
			Anchor:   mustRFC3339(t, "2024-01-10T00:00:00Z"),
		}
		tm := mustRFC3339(t, "2024-01-06T12:00:00Z")

		it, err := recurrence.IterateFromNextAfter(tm, timeutil.Inclusive)
		require.NoError(t, err)
		assert.Equal(t, mustRFC3339(t, "2024-01-07T00:00:00Z"), it.At, "next occurrence at or after t")

		next, err := it.Next()
		require.NoError(t, err)
		assert.Equal(t, mustRFC3339(t, "2024-01-08T00:00:00Z"), next.At, "Next must advance one full interval from At")
	})

	t.Run("prev before with anchor in past", func(t *testing.T) {
		recurrence := timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodDaily,
			Anchor:   mustRFC3339(t, "2024-01-01T00:00:00Z"),
		}
		tm := mustRFC3339(t, "2024-01-06T12:00:00Z")

		it, err := recurrence.IterateFromPrevBefore(tm, timeutil.Inclusive)
		require.NoError(t, err)
		assert.Equal(t, mustRFC3339(t, "2024-01-06T00:00:00Z"), it.At, "last occurrence at or before t")

		next, err := it.Next()
		require.NoError(t, err)
		assert.Equal(t, mustRFC3339(t, "2024-01-07T00:00:00Z"), next.At, "Next must advance one full interval from At")
	})
}
