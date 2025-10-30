package engine

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/pkg/timeutil"
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

// GrantBurnDownHistorySegment represents the smallest segment of grant usage which we store and calculate.
//
// A segment represents a period of time in which:
// 1) The grant priority does not change
// 2) Grants do not recurr
// 3) There was no usage reset
//
// It is not necessarily the largest such segment.
type GrantBurnDownHistorySegment struct {
	timeutil.ClosedPeriod
	BalanceAtStart     balance.Map
	TerminationReasons SegmentTerminationReason // Reason why the segment was terminated (could be multiple taking effect at same time)
	TotalUsage         float64                  // Total usage of the feature in the Period
	OverageAtStart     float64                  // Usage beyond what could be burnt down from the grants in the previous segment (if any)
	Overage            float64                  // Usage beyond what cloud be burnt down from the grants
	GrantUsages        []GrantUsage             // Grant usages in the segment order by grant priority
}

// Returns GrantBalanceMap at the end of the segment
func (s GrantBurnDownHistorySegment) ApplyUsage() balance.Map {
	balance := s.BalanceAtStart.Clone()
	for _, u := range s.GrantUsages {
		balance.Burn(u.GrantID, u.Usage)
	}
	return balance
}

func NewGrantBurnDownHistory(segments []GrantBurnDownHistorySegment, usageAtStart balance.SnapshottedUsage) (GrantBurnDownHistory, error) {
	s := make([]GrantBurnDownHistorySegment, len(segments))
	copy(s, segments)

	// sort segments by time
	sort.Slice(s, func(i, j int) bool {
		return s[i].ClosedPeriod.From.Before(s[j].ClosedPeriod.From)
	})

	// validate no two segments overlap
	for i := range s {
		if i == 0 {
			continue
		}

		if s[i-1].To.After(s[i].From) {
			return GrantBurnDownHistory{}, fmt.Errorf("segments %d and %d overlap", i-1, i)
		}
	}

	return GrantBurnDownHistory{segments: s, usageAtStart: usageAtStart}, nil
}

type GrantBurnDownHistory struct {
	segments     []GrantBurnDownHistorySegment
	usageAtStart balance.SnapshottedUsage
}

func (g GrantBurnDownHistory) MarshalJSON() ([]byte, error) {
	return json.Marshal(g.segments)
}

func (g *GrantBurnDownHistory) GetSnapshotAtStartOfSegment(segmentIndex int) (balance.Snapshot, error) {
	// Let's validate the segment index
	if segmentIndex < 0 || segmentIndex >= len(g.segments) {
		return balance.Snapshot{}, fmt.Errorf("segment index %d out of bounds", segmentIndex)
	}

	// Let's get the segment
	segment := g.segments[segmentIndex]

	// Let's get the usage in the period until the start of the segment
	usage, err := g.GetUsageInPeriodUntilSegment(segmentIndex)
	if err != nil {
		return balance.Snapshot{}, fmt.Errorf("failed to get usage in period until segment: %w", err)
	}

	return balance.Snapshot{
		Usage:    usage,
		Overage:  segment.OverageAtStart,
		Balances: segment.BalanceAtStart,
		At:       segment.From,
	}, nil
}

// GetUsageInPeriodUntilSegment returns the SnapshottedUsage at the start of the given segment
func (g *GrantBurnDownHistory) GetUsageInPeriodUntilSegment(segmentIndex int) (balance.SnapshottedUsage, error) {
	// Let's validate the segment index
	if segmentIndex < 0 || segmentIndex >= len(g.segments) {
		return balance.SnapshottedUsage{}, fmt.Errorf("segment index %d out of bounds", segmentIndex)
	}

	// Let's find the segment of the last reset before the provided segment
	lastResetSegmentIndex := -1
	for i := 0; i < segmentIndex; i++ {
		if g.segments[i].TerminationReasons.UsageReset {
			lastResetSegmentIndex = i
		}
	}

	// Now let's build a starting SnapshottedUsage
	usage := g.usageAtStart

	if lastResetSegmentIndex != -1 {
		// We need the segment right after the last reset
		if lastResetSegmentIndex+1 < len(g.segments) {
			firstSeg := g.segments[lastResetSegmentIndex+1]
			usage = balance.SnapshottedUsage{
				Since: firstSeg.From,
				Usage: 0.0,
			}
		}
	}

	// Now we need to add up the usage in all segments between the starting usage and the provided segment
	for i := lastResetSegmentIndex + 1; i < segmentIndex; i++ {
		usage.Usage += g.segments[i].TotalUsage
	}

	return usage, nil
}

func (g *GrantBurnDownHistory) Segments() []GrantBurnDownHistorySegment {
	return g.segments
}

func (g *GrantBurnDownHistory) TotalUsageInHistory() float64 {
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

func (g *GrantBurnDownHistory) GetPeriods() []timeutil.ClosedPeriod {
	periods := make([]timeutil.ClosedPeriod, len(g.segments))
	for i, seg := range g.segments {
		periods[i] = seg.ClosedPeriod
	}

	return periods
}
