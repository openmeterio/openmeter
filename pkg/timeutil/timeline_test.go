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
