package entitlement_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

func TestUsagePeriod(t *testing.T) {
	t1 := time.Now().Truncate(time.Minute)

	t.Run("Should be inclusive on period From and exclusive on period To", func(t *testing.T) {
		up := entitlement.UsagePeriod{
			Interval: recurrence.RecurrencePeriodDaily,
			Anchor:   t1,
		}

		period, err := up.GetCurrentPeriodAt(t1)
		assert.NoError(t, err)
		assert.Equal(t, t1, period.From)
		assert.Equal(t, t1.AddDate(0, 0, 1), period.To)
	})
}

func TestMeasureUsageFromInput(t *testing.T) {
	t.Run("Should return time from input", func(t *testing.T) {
		t1 := time.Now().Truncate(time.Minute)
		m := &entitlement.MeasureUsageFromInput{}
		m.FromTime(t1)
		assert.Equal(t, t1, m.Get())
	})

	t.Run("Should return time from CURRENT_PERIOD_START enum", func(t *testing.T) {
		t1 := time.Now().Truncate(time.Minute).Add(-time.Hour)
		up := entitlement.UsagePeriod{
			Interval: recurrence.RecurrencePeriodDaily,
			Anchor:   t1,
		}

		m := &entitlement.MeasureUsageFromInput{}
		m.FromEnum(entitlement.MeasureUsageFromCurrentPeriodStart, up)
		assert.Equal(t, t1, m.Get())
	})
}
