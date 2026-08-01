package timeutil_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestTimelineGetClosedPeriods(t *testing.T) {
	t.Run("Empty timeline", func(t *testing.T) {
		timeline := timeutil.NewSimpleTimeline([]time.Time{})
		periods := timeline.GetClosedPeriods()
		assert.Empty(t, periods)
	})

	t.Run("Single time", func(t *testing.T) {
		time1 := testutils.GetRFC3339Time(t, "2021-01-01T01:00:00Z")

		timeline := timeutil.NewSimpleTimeline([]time.Time{time1})
		periods := timeline.GetClosedPeriods()

		assert.Len(t, periods, 1)
		assert.Equal(t, time1, periods[0].From)
		assert.Equal(t, time1, periods[0].To)
	})

	t.Run("Multiple times", func(t *testing.T) {
		time1 := testutils.GetRFC3339Time(t, "2021-01-01T01:00:00Z")
		time2 := testutils.GetRFC3339Time(t, "2021-01-01T02:00:00Z")
		time3 := testutils.GetRFC3339Time(t, "2021-01-01T03:00:00Z")

		timeline := timeutil.NewSimpleTimeline([]time.Time{time1, time2, time3})
		periods := timeline.GetClosedPeriods()

		assert.Len(t, periods, 2)

		// First period: time1 to time2
		assert.Equal(t, time1, periods[0].From)
		assert.Equal(t, time2, periods[0].To)

		// Second period: time2 to time3
		assert.Equal(t, time2, periods[1].From)
		assert.Equal(t, time3, periods[1].To)
	})

	t.Run("Unsorted times", func(t *testing.T) {
		time1 := testutils.GetRFC3339Time(t, "2021-01-01T01:00:00Z")
		time2 := testutils.GetRFC3339Time(t, "2021-01-01T02:00:00Z")
		time3 := testutils.GetRFC3339Time(t, "2021-01-01T03:00:00Z")

		// Insert times in non-sequential order
		timeline := timeutil.NewSimpleTimeline([]time.Time{time3, time1, time2})
		periods := timeline.GetClosedPeriods()

		assert.Len(t, periods, 2)

		// First period: time1 to time2
		assert.Equal(t, time1, periods[0].From)
		assert.Equal(t, time2, periods[0].To)

		// Second period: time2 to time3
		assert.Equal(t, time2, periods[1].From)
		assert.Equal(t, time3, periods[1].To)
	})
}

func TestTimelineGetOpenPeriods(t *testing.T) {
	t.Run("Empty timeline", func(t *testing.T) {
		timeline := timeutil.NewSimpleTimeline([]time.Time{})
		periods := timeline.GetOpenPeriods()
		assert.Empty(t, periods)
	})

	t.Run("Single time", func(t *testing.T) {
		time1 := testutils.GetRFC3339Time(t, "2021-01-01T01:00:00Z")

		timeline := timeutil.NewSimpleTimeline([]time.Time{time1})
		periods := timeline.GetOpenPeriods()

		assert.Len(t, periods, 2)

		// First period: open start to time1
		assert.Nil(t, periods[0].From)
		assert.NotNil(t, periods[0].To)
		assert.Equal(t, time1, *periods[0].To)

		// Second period: time1 to open end
		assert.NotNil(t, periods[1].From)
		assert.Nil(t, periods[1].To)
		assert.Equal(t, time1, *periods[1].From)
	})

	t.Run("Multiple times", func(t *testing.T) {
		time1 := testutils.GetRFC3339Time(t, "2021-01-01T01:00:00Z")
		time2 := testutils.GetRFC3339Time(t, "2021-01-01T02:00:00Z")
		time3 := testutils.GetRFC3339Time(t, "2021-01-01T03:00:00Z")

		timeline := timeutil.NewSimpleTimeline([]time.Time{time1, time2, time3})
		periods := timeline.GetOpenPeriods()

		assert.Len(t, periods, 4)

		// First period: open start to time1
		assert.Nil(t, periods[0].From)
		assert.NotNil(t, periods[0].To)
		assert.Equal(t, time1, *periods[0].To)

		// Second period: time1 to time2
		assert.NotNil(t, periods[1].From)
		assert.NotNil(t, periods[1].To)
		assert.Equal(t, time1, *periods[1].From)
		assert.Equal(t, time2, *periods[1].To)

		// Third period: time2 to time3
		assert.NotNil(t, periods[2].From)
		assert.NotNil(t, periods[2].To)
		assert.Equal(t, time2, *periods[2].From)
		assert.Equal(t, time3, *periods[2].To)

		// Fourth period: time3 to open end
		assert.NotNil(t, periods[3].From)
		assert.Nil(t, periods[3].To)
		assert.Equal(t, time3, *periods[3].From)
	})

	t.Run("Unsorted times", func(t *testing.T) {
		time1 := testutils.GetRFC3339Time(t, "2021-01-01T01:00:00Z")
		time2 := testutils.GetRFC3339Time(t, "2021-01-01T02:00:00Z")
		time3 := testutils.GetRFC3339Time(t, "2021-01-01T03:00:00Z")

		// Insert times in non-sequential order
		timeline := timeutil.NewSimpleTimeline([]time.Time{time3, time1, time2})
		periods := timeline.GetOpenPeriods()

		assert.Len(t, periods, 4)

		// First period: open start to time1
		assert.Nil(t, periods[0].From)
		assert.NotNil(t, periods[0].To)
		assert.Equal(t, time1, *periods[0].To)

		// Second period: time1 to time2
		assert.NotNil(t, periods[1].From)
		assert.NotNil(t, periods[1].To)
		assert.Equal(t, time1, *periods[1].From)
		assert.Equal(t, time2, *periods[1].To)

		// Third period: time2 to time3
		assert.NotNil(t, periods[2].From)
		assert.NotNil(t, periods[2].To)
		assert.Equal(t, time2, *periods[2].From)
		assert.Equal(t, time3, *periods[2].To)

		// Fourth period: time3 to open end
		assert.NotNil(t, periods[3].From)
		assert.Nil(t, periods[3].To)
		assert.Equal(t, time3, *periods[3].From)
	})
}

// lastIndexNotAfterLinear is the reverse linear scan that LastIndexNotAfter replaced.
// It is kept as the differential oracle: the binary search must agree with it for every
// input, including timelines carrying duplicate timestamps.
func lastIndexNotAfterLinear[T any](tl timeutil.Timeline[T], at time.Time) int {
	for i := tl.Len() - 1; i >= 0; i-- {
		if !tl.GetAt(i).GetTime().After(at) {
			return i
		}
	}

	return -1
}

func TestTimelineLastIndexNotAfter(t *testing.T) {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("Empty timeline", func(t *testing.T) {
		assert.Equal(t, -1, timeutil.NewSimpleTimeline([]time.Time{}).LastIndexNotAfter(base))
	})

	t.Run("All entries after at", func(t *testing.T) {
		timeline := timeutil.NewSimpleTimeline([]time.Time{base.Add(time.Hour), base.Add(2 * time.Hour)})
		assert.Equal(t, -1, timeline.LastIndexNotAfter(base))
	})

	t.Run("Boundary is inclusive", func(t *testing.T) {
		timeline := timeutil.NewSimpleTimeline([]time.Time{base, base.Add(time.Hour)})
		assert.Equal(t, 0, timeline.LastIndexNotAfter(base))
	})

	t.Run("Duplicate timestamps resolve to the last of the tied group", func(t *testing.T) {
		timeline := timeutil.NewSimpleTimeline([]time.Time{base, base, base, base.Add(time.Hour)})
		assert.Equal(t, 2, timeline.LastIndexNotAfter(base))
	})

	// Differential sweep: every probe against every timeline must match the linear oracle.
	// Durations deliberately repeat so the generated timelines contain ties.
	t.Run("Matches the linear scan", func(t *testing.T) {
		offsets := []time.Duration{0, 0, time.Hour, time.Hour, 2 * time.Hour, 5 * time.Hour, 5 * time.Hour}

		for size := 0; size <= len(offsets); size++ {
			times := make([]time.Time, 0, size)
			for _, offset := range offsets[:size] {
				times = append(times, base.Add(offset))
			}

			timeline := timeutil.NewSimpleTimeline(times)

			for probe := -time.Hour; probe <= 6*time.Hour; probe += 30 * time.Minute {
				at := base.Add(probe)
				assert.Equal(t, lastIndexNotAfterLinear(timeline, at), timeline.LastIndexNotAfter(at),
					"size=%d at=%s", size, at)
			}
		}
	})
}
