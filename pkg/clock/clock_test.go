package clock_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func TestClock(t *testing.T) {
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-30T15:39:00Z"))
	defer clock.ResetTime()

	now := clock.Now()
	diff := now.Sub(testutils.GetRFC3339Time(t, "2024-06-30T15:39:00Z"))
	if diff < 0 {
		diff = -diff
	}
	assert.True(t, diff < time.Second)
}
