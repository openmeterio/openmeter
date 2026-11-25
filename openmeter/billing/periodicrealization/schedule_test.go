package periodicrealization_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/periodicrealization"
)

func TestPeriodTo(t *testing.T) {
	plusTwo := time.FixedZone("UTC+2", 2*60*60)

	tests := []struct {
		name string
		asOf time.Time
		want time.Time
	}{
		{
			name: "during the day",
			asOf: time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC),
			want: time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "a period end belongs to the following period",
			asOf: time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC),
			want: time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "just before a period end",
			asOf: time.Date(2026, 11, 23, 23, 59, 59, 999999999, time.UTC),
			want: time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "the day is taken in UTC, not in the time's own zone",
			asOf: time.Date(2026, 11, 24, 1, 30, 0, 0, plusTwo), // Nov 23 23:30 UTC
			want: time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "across a month end",
			asOf: time.Date(2026, 11, 30, 12, 0, 0, 0, time.UTC),
			want: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, periodicrealization.PeriodTo(tt.asOf))
		})
	}
}

func TestRealizeAfterAddsTheCollectionDelay(t *testing.T) {
	periodTo := time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC)

	require.Equal(t, time.Date(2026, 11, 24, 0, 1, 0, 0, time.UTC), periodicrealization.RealizeAfter(periodTo))
}
