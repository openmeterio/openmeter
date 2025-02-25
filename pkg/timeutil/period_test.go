package timeutil_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestPeriod(t *testing.T) {
	startTime := testutils.GetRFC3339Time(t, "2021-01-01T01:00:00Z")
	endTime := testutils.GetRFC3339Time(t, "2021-01-01T02:00:00Z")

	period := timeutil.Period{
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
			period := timeutil.Period{
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
			period := timeutil.Period{
				From: startTime,
				To:   startTime,
			}

			assert.False(t, period.Contains(startTime))
		})
	})

	t.Run("Overlaps", func(t *testing.T) {
		t.Run("Should be false for exactly sequential periods", func(t *testing.T) {
			assert.False(t, period.Overlaps(timeutil.Period{From: endTime, To: endTime.Add(time.Second)}))
		})

		t.Run("Should be false for distant periods", func(t *testing.T) {
			assert.False(t, period.Overlaps(timeutil.Period{From: endTime.Add(time.Second), To: endTime.Add(time.Second * 2)}))
			assert.False(t, period.Overlaps(timeutil.Period{From: startTime.Add(-2 * time.Second), To: startTime.Add(-time.Second)}))
		})

		t.Run("Should be true for overlapping periods", func(t *testing.T) {
			assert.True(t, period.Overlaps(timeutil.Period{From: startTime.Add(-time.Second), To: endTime.Add(-time.Second)}))
			assert.True(t, period.Overlaps(timeutil.Period{From: startTime.Add(time.Second), To: endTime.Add(time.Second)}))
		})

		t.Run("Should be true for containing periods", func(t *testing.T) {
			assert.True(t, period.Overlaps(timeutil.Period{From: startTime.Add(-time.Second), To: endTime.Add(time.Second)}))
			assert.True(t, period.Overlaps(timeutil.Period{From: startTime.Add(time.Second), To: endTime.Add(-time.Second)}))
		})
	})
	t.Run("OverlapsInclusive", func(t *testing.T) {
		t.Run("Should be true for exactly sequential periods", func(t *testing.T) {
			assert.True(t, period.OverlapsInclusive(timeutil.Period{From: endTime, To: endTime.Add(time.Second)}))
		})

		t.Run("Should be false for distant periods", func(t *testing.T) {
			assert.False(t, period.OverlapsInclusive(timeutil.Period{From: endTime.Add(time.Second), To: endTime.Add(time.Second * 2)}))
			assert.False(t, period.OverlapsInclusive(timeutil.Period{From: startTime.Add(-2 * time.Second), To: startTime.Add(-time.Second)}))
		})

		t.Run("Should be true for overlapping periods", func(t *testing.T) {
			assert.True(t, period.OverlapsInclusive(timeutil.Period{From: startTime.Add(-time.Second), To: endTime.Add(-time.Second)}))
			assert.True(t, period.OverlapsInclusive(timeutil.Period{From: startTime.Add(time.Second), To: endTime.Add(time.Second)}))
		})

		t.Run("Should be true for containing periods", func(t *testing.T) {
			assert.True(t, period.OverlapsInclusive(timeutil.Period{From: startTime.Add(-time.Second), To: endTime.Add(time.Second)}))
			assert.True(t, period.OverlapsInclusive(timeutil.Period{From: startTime.Add(time.Second), To: endTime.Add(-time.Second)}))
		})
	})
}
