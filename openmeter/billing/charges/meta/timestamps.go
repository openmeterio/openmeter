package meta

import (
	"time"

	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func NormalizeTimestamp(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}

	return t.UTC().Truncate(streaming.MinimumWindowSizeDuration)
}

func NormalizeOptionalTimestamp(t *time.Time) *time.Time {
	if t == nil || t.IsZero() {
		return nil
	}

	normalized := NormalizeTimestamp(*t)
	return &normalized
}

func NormalizeClosedPeriod(period timeutil.ClosedPeriod) timeutil.ClosedPeriod {
	return timeutil.ClosedPeriod{
		From: NormalizeTimestamp(period.From),
		To:   NormalizeTimestamp(period.To),
	}
}

func (i Intent) Normalized() Intent {
	i.ServicePeriod = NormalizeClosedPeriod(i.ServicePeriod)
	i.FullServicePeriod = NormalizeClosedPeriod(i.FullServicePeriod)
	i.BillingPeriod = NormalizeClosedPeriod(i.BillingPeriod)

	return i
}
