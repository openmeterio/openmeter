package timeutil_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestClosedPeriod(t *testing.T) {
	startTime := testutils.GetRFC3339Time(t, "2021-01-01T01:00:00Z")
	endTime := testutils.GetRFC3339Time(t, "2021-01-01T02:00:00Z")

	period := timeutil.ClosedPeriod{
		From: startTime,
		To:   endTime,
	}

	t.Run("ContainsInclusive", func(t *testing.T) {
		t.Run("Should be inclusive of start time", func(t *testing.T) {
			assert.True(t, period.ContainsInclusive(startTime))
		})

		t.Run("Should be inclusive of end time", func(t *testing.T) {
			assert.True(t, period.ContainsInclusive(endTime))
		})

		t.Run("Should be true for value in between", func(t *testing.T) {
			assert.True(t, period.ContainsInclusive(startTime.Add(time.Second)))
		})

		t.Run("Should be false for earlier time", func(t *testing.T) {
			assert.False(t, period.ContainsInclusive(startTime.Add(-time.Second)))
		})

		t.Run("Should be false for later time", func(t *testing.T) {
			assert.False(t, period.ContainsInclusive(endTime.Add(time.Second)))
		})

		t.Run("Should be true for 0 length period", func(t *testing.T) {
			period := timeutil.ClosedPeriod{
				From: startTime,
				To:   startTime,
			}

			assert.True(t, period.ContainsInclusive(startTime))
		})
	})

	t.Run("Contains", func(t *testing.T) {
		t.Run("Should be inclusive of start time", func(t *testing.T) {
			assert.True(t, period.Contains(startTime))
		})

		t.Run("Should be exclusive of end time", func(t *testing.T) {
			assert.False(t, period.Contains(endTime))
		})

		t.Run("Should be true for value in between", func(t *testing.T) {
			assert.True(t, period.Contains(startTime.Add(time.Second)))
		})

		t.Run("Should be false for earlier time", func(t *testing.T) {
			assert.False(t, period.Contains(startTime.Add(-time.Second)))
		})

		t.Run("Should be false for later time", func(t *testing.T) {
			assert.False(t, period.Contains(endTime.Add(time.Second)))
		})

		t.Run("Should be false for 0 length period", func(t *testing.T) {
			period := timeutil.ClosedPeriod{
				From: startTime,
				To:   startTime,
			}

			assert.False(t, period.Contains(startTime))
		})
	})

	t.Run("Overlaps", func(t *testing.T) {
		t.Run("Should be false for exactly sequential periods", func(t *testing.T) {
			assert.False(t, period.Overlaps(timeutil.ClosedPeriod{From: endTime, To: endTime.Add(time.Second)}))
			assert.False(t, period.Overlaps(timeutil.ClosedPeriod{From: startTime.Add(-1 * time.Second), To: startTime}))
		})

		t.Run("Should be false for distant periods", func(t *testing.T) {
			assert.False(t, period.Overlaps(timeutil.ClosedPeriod{From: endTime.Add(time.Second), To: endTime.Add(time.Second * 2)}))
			assert.False(t, period.Overlaps(timeutil.ClosedPeriod{From: startTime.Add(-2 * time.Second), To: startTime.Add(-time.Second)}))
		})

		t.Run("Should be true for overlapping periods", func(t *testing.T) {
			assert.True(t, period.Overlaps(timeutil.ClosedPeriod{From: startTime.Add(-time.Second), To: endTime.Add(-time.Second)}))
			assert.True(t, period.Overlaps(timeutil.ClosedPeriod{From: startTime.Add(time.Second), To: endTime.Add(time.Second)}))
		})

		t.Run("Should be true for containing periods", func(t *testing.T) {
			assert.True(t, period.Overlaps(timeutil.ClosedPeriod{From: startTime.Add(-time.Second), To: endTime.Add(time.Second)}))
			assert.True(t, period.Overlaps(timeutil.ClosedPeriod{From: startTime.Add(time.Second), To: endTime.Add(-time.Second)}))
		})
	})
	t.Run("OverlapsInclusive", func(t *testing.T) {
		t.Run("Should be true for exactly sequential periods", func(t *testing.T) {
			assert.True(t, period.OverlapsInclusive(timeutil.ClosedPeriod{From: endTime, To: endTime.Add(time.Second)}))
		})

		t.Run("Should be false for distant periods", func(t *testing.T) {
			assert.False(t, period.OverlapsInclusive(timeutil.ClosedPeriod{From: endTime.Add(time.Second), To: endTime.Add(time.Second * 2)}))
			assert.False(t, period.OverlapsInclusive(timeutil.ClosedPeriod{From: startTime.Add(-2 * time.Second), To: startTime.Add(-time.Second)}))
		})

		t.Run("Should be true for overlapping periods", func(t *testing.T) {
			assert.True(t, period.OverlapsInclusive(timeutil.ClosedPeriod{From: startTime.Add(-time.Second), To: endTime.Add(-time.Second)}))
			assert.True(t, period.OverlapsInclusive(timeutil.ClosedPeriod{From: startTime.Add(time.Second), To: endTime.Add(time.Second)}))
		})

		t.Run("Should be true for containing periods", func(t *testing.T) {
			assert.True(t, period.OverlapsInclusive(timeutil.ClosedPeriod{From: startTime.Add(-time.Second), To: endTime.Add(time.Second)}))
			assert.True(t, period.OverlapsInclusive(timeutil.ClosedPeriod{From: startTime.Add(time.Second), To: endTime.Add(-time.Second)}))
		})
	})

	t.Run("Intersection", func(t *testing.T) {
		t.Run("Should return nil for non-overlapping periods", func(t *testing.T) {
			// Distant periods
			other := timeutil.ClosedPeriod{From: endTime.Add(time.Second), To: endTime.Add(2 * time.Second)}
			assert.Nil(t, period.Intersection(other))

			// Sequential periods (touching at boundary)
			other = timeutil.ClosedPeriod{From: endTime, To: endTime.Add(time.Second)}
			assert.Nil(t, period.Intersection(other))
		})

		t.Run("Should return intersection for overlapping periods", func(t *testing.T) {
			// Partial overlap from the left
			other := timeutil.ClosedPeriod{From: startTime.Add(-time.Second), To: startTime.Add(30 * time.Second)}
			intersection := period.Intersection(other)
			assert.NotNil(t, intersection)
			assert.Equal(t, startTime, intersection.From)
			assert.Equal(t, startTime.Add(30*time.Second), intersection.To)

			// Partial overlap from the right
			other = timeutil.ClosedPeriod{From: startTime.Add(30 * time.Second), To: endTime.Add(time.Second)}
			intersection = period.Intersection(other)
			assert.NotNil(t, intersection)
			assert.Equal(t, startTime.Add(30*time.Second), intersection.From)
			assert.Equal(t, endTime, intersection.To)
		})

		t.Run("Should return contained period when one period is inside another", func(t *testing.T) {
			// Other period is contained within this period
			other := timeutil.ClosedPeriod{From: startTime.Add(15 * time.Second), To: startTime.Add(45 * time.Second)}
			intersection := period.Intersection(other)
			assert.NotNil(t, intersection)
			assert.Equal(t, other.From, intersection.From)
			assert.Equal(t, other.To, intersection.To)

			// This period is contained within other period
			other = timeutil.ClosedPeriod{From: startTime.Add(-time.Second), To: endTime.Add(time.Second)}
			intersection = period.Intersection(other)
			assert.NotNil(t, intersection)
			assert.Equal(t, period.From, intersection.From)
			assert.Equal(t, period.To, intersection.To)
		})

		t.Run("Should return exact period when periods are identical", func(t *testing.T) {
			other := timeutil.ClosedPeriod{From: startTime, To: endTime}
			intersection := period.Intersection(other)
			assert.NotNil(t, intersection)
			assert.Equal(t, startTime, intersection.From)
			assert.Equal(t, endTime, intersection.To)
		})

		t.Run("Should handle zero-length periods", func(t *testing.T) {
			// Zero-length period at start boundary
			other := timeutil.ClosedPeriod{From: startTime, To: startTime}
			intersection := period.Intersection(other)
			assert.Nil(t, intersection) // No valid intersection since end is not after start

			// Zero-length period inside
			other = timeutil.ClosedPeriod{From: startTime.Add(30 * time.Second), To: startTime.Add(30 * time.Second)}
			intersection = period.Intersection(other)
			assert.Nil(t, intersection) // No valid intersection since end is not after start
		})
	})
}
