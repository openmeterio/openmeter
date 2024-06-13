package credit

import (
	"sort"
	"time"

	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type Period struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

func (p Period) Duration() time.Duration {
	return p.To.Sub(p.From)
}

func (p Period) Contains(t time.Time) bool {
	return t.After(p.From) && t.Before(p.To)
}

// Returns a list of non-overlapping periods between the sorted times.
func PeriodsFromTimes(ts []time.Time) []Period {
	if len(ts) < 2 {
		return nil
	}

	// copy
	times := make([]time.Time, len(ts))
	copy(times, ts)

	// dedupe
	times = slicesx.Dedupe(times)

	// sort
	sort.Slice(times, func(i, j int) bool {
		return times[i].Before(times[j])
	})

	periods := make([]Period, 0, len(times)-1)
	for i := 1; i < len(times); i++ {
		periods = append(periods, Period{From: times[i-1], To: times[i]})
	}

	return periods
}
