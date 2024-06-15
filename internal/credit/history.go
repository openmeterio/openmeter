package credit

import (
	"fmt"
	"math"
	"sort"
	"time"
)

type SegmentTerminationReason struct {
	PriorityChange bool
	Recurrence     []string // Grant IDs
	UsageReset     bool
}

type GrantUsageTerminationReason string

const (
	GrantUsageTerminationReasonExhausted          GrantUsageTerminationReason = "GRANT_EXHAUSTED"     // Grant has been fully used
	GrantUsageTerminationReasonSegmentTermination GrantUsageTerminationReason = "SEGMENT_TERMINATION" // Segment has been terminated
)

func (GrantUsageTerminationReason) IsValid(reason GrantUsageTerminationReason) bool {
	for _, s := range []GrantUsageTerminationReason{
		GrantUsageTerminationReasonExhausted,
		GrantUsageTerminationReasonSegmentTermination,
	} {
		if s == reason {
			return true
		}
	}
	return false
}

type GrantUsage struct {
	GrantID           string
	Usage             float64
	TerminationReason GrantUsageTerminationReason
}

// GrantBurnDownHistorySegment represents the smallest segment of grant usage which we store and calcualte.
//
// A segment represents a period of time in which:
// 1) The grant priority does not change
// 2) Grants do not recurr
// 3) There was no usage reset
//
// It is not necessarily the largest such segment.
type GrantBurnDownHistorySegment struct {
	Period
	BalanceAtStart     GrantBalanceMap
	TerminationReasons SegmentTerminationReason // Reason why the segment was terminated (could be multiple taking effect at same time)
	TotalUsage         float64                  // Total usage of the feature in the Period
	Overage            float64                  // Usage beyond what culd be burnt down from the grants
	GrantUsages        []GrantUsage             // Grant usages in the segment order by grant priority
}

// Returns GrantBalanceMap at the end of the segment
func (s GrantBurnDownHistorySegment) ApplyUsage() GrantBalanceMap {
	balance := s.BalanceAtStart.Copy()
	for _, u := range s.GrantUsages {
		balance.Burn(u.GrantID, u.Usage)
	}
	return balance
}

func NewGrantBurnDownHistory(segments []GrantBurnDownHistorySegment) (*GrantBurnDownHistory, error) {
	s := make([]GrantBurnDownHistorySegment, len(segments))
	copy(s, segments)

	// sort segments by time
	sort.Slice(s, func(i, j int) bool {
		return s[i].Period.From.Before(s[j].Period.From)
	})

	// validate no two segments overlap
	for i := range s {
		if i == 0 {
			continue
		}

		if s[i-1].To.After(s[i].From) {
			return nil, fmt.Errorf("segments %d and %d overlap", i-1, i)
		}
	}

	return &GrantBurnDownHistory{segments: s}, nil
}

type GrantBurnDownHistory struct {
	segments []GrantBurnDownHistorySegment
}

func (g *GrantBurnDownHistory) Segments() []GrantBurnDownHistorySegment {
	return g.segments
}

func (g *GrantBurnDownHistory) TotalUsage() float64 {
	var total float64
	for _, s := range g.segments {
		total += s.TotalUsage
	}
	return total
}

func (g *GrantBurnDownHistory) Overage() float64 {
	lastSegment := g.segments[len(g.segments)-1]
	return lastSegment.Overage
}

// Returns the last segment that can be saved, taking the following into account:
//
//  1. We can save a segment if it is older than graceperiod.
//  2. At the end of a segment history changes: s1.endBalance <> s2.startBalance. This means only the
//     starting values can be saved credibly.
func (g *GrantBurnDownHistory) GetLastSaveableAt(at time.Time) (GrantBurnDownHistorySegment, error) {
	gracePeriod := time.Hour // TODO: make this configurable

	for i := len(g.segments) - 1; i >= 0; i-- {
		segment := g.segments[i]
		if segment.From.Add(gracePeriod).Before(at) {
			return segment, nil
		}
	}

	return GrantBurnDownHistorySegment{}, fmt.Errorf("no segment can be saved at %s with gracePeriod %s", at, gracePeriod)
}

// Creates a GrantBalanceSnapshot from the starting state of the segment
func (s *GrantBurnDownHistorySegment) ToSnapshot() GrantBalanceSnapshot {
	overageAtStart := math.Max(0, s.Overage-s.TotalUsage)
	return GrantBalanceSnapshot{
		Overage:  overageAtStart,
		Balances: s.BalanceAtStart,
		At:       s.From,
	}
}
