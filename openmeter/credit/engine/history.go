package engine

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type SegmentTerminationReason struct {
	PriorityChange bool
	Recurrence     []string // Grant IDs
	UsageReset     bool
	// Rollover marks grant balance rollover followed by settlement of preserved overage.
	Rollover bool
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
// A non-rollover segment represents a period of time in which:
// 1) The grant priority does not change
// 2) Grants do not recurr
// 3) There was no usage reset
//
// A rollover segment is an instantaneous reset transition with no metered usage.
// Its starting balance is the rolled-over grant balance, and its grant usages
// capture settlement of overage preserved from the previous usage period.
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

// NewGrantBurnDownHistory creates a history anchored to startingSnapshot.
// Segments must continuously cover time beginning at the snapshot.
func NewGrantBurnDownHistory(segments []GrantBurnDownHistorySegment, startingSnapshot balance.Snapshot) (GrantBurnDownHistory, error) {
	s := make([]GrantBurnDownHistorySegment, len(segments))
	copy(s, segments)

	for i, segment := range s {
		if segment.From.After(segment.To) {
			return GrantBurnDownHistory{}, fmt.Errorf("segment %d starts after it ends", i)
		}
	}

	// Sort segments by time. Rollover transitions precede regular segments at
	// the same timestamp so the reset transition is applied before new-period
	// usage.
	sort.SliceStable(s, func(i, j int) bool {
		if s[i].ClosedPeriod.From.Equal(s[j].ClosedPeriod.From) {
			return s[i].TerminationReasons.Rollover && !s[j].TerminationReasons.Rollover
		}

		return s[i].ClosedPeriod.From.Before(s[j].ClosedPeriod.From)
	})

	if len(s) > 0 {
		if !s[0].From.Equal(startingSnapshot.At) {
			return GrantBurnDownHistory{}, fmt.Errorf(
				"first segment starts at %s, expected starting snapshot at %s",
				s[0].From,
				startingSnapshot.At,
			)
		}

		if s[0].OverageAtStart != startingSnapshot.Overage {
			return GrantBurnDownHistory{}, fmt.Errorf(
				"first segment starts with overage %f, expected starting snapshot overage %f",
				s[0].OverageAtStart,
				startingSnapshot.Overage,
			)
		}
	}

	for i := 1; i < len(s); i++ {
		if !s[i-1].To.Equal(s[i].From) {
			return GrantBurnDownHistory{}, fmt.Errorf(
				"segments %d and %d are not contiguous: %s != %s",
				i-1,
				i,
				s[i-1].To,
				s[i].From,
			)
		}
	}

	return GrantBurnDownHistory{
		segments:         s,
		startingSnapshot: startingSnapshot.Clone(),
	}, nil
}

type GrantBurnDownHistory struct {
	segments         []GrantBurnDownHistorySegment
	startingSnapshot balance.Snapshot
}

func (g GrantBurnDownHistory) MarshalJSON() ([]byte, error) {
	return json.Marshal(g.segments)
}

func (g *GrantBurnDownHistory) GetSnapshotAtStartOfSegment(segmentIndex int) (balance.Snapshot, error) {
	if segmentIndex < 0 || segmentIndex >= len(g.segments) {
		return balance.Snapshot{}, fmt.Errorf("segment index %d out of bounds", segmentIndex)
	}

	return g.getSnapshotAtStartOfSegment(segmentIndex), nil
}

func (g *GrantBurnDownHistory) getSnapshotAtStartOfSegment(segmentIndex int) balance.Snapshot {
	segment := g.segments[segmentIndex]
	snapshot := g.startingSnapshot.Clone()
	snapshot.Usage = g.getUsageInPeriodUntilSegment(segmentIndex)
	snapshot.Overage = segment.OverageAtStart
	snapshot.Balances = segment.BalanceAtStart.Clone()
	snapshot.At = segment.From

	return snapshot
}

// GetUsageInPeriodUntilSegment returns the SnapshottedUsage at the start of the given segment
func (g *GrantBurnDownHistory) GetUsageInPeriodUntilSegment(segmentIndex int) (balance.SnapshottedUsage, error) {
	if segmentIndex < 0 || segmentIndex >= len(g.segments) {
		return balance.SnapshottedUsage{}, fmt.Errorf("segment index %d out of bounds", segmentIndex)
	}

	return g.getUsageInPeriodUntilSegment(segmentIndex), nil
}

func (g *GrantBurnDownHistory) getUsageInPeriodUntilSegment(segmentIndex int) balance.SnapshottedUsage {
	// Let's find the segment of the last reset before the provided segment
	lastResetSegmentIndex := -1
	for i := 0; i < segmentIndex; i++ {
		if g.segments[i].TerminationReasons.UsageReset {
			lastResetSegmentIndex = i
		}
	}

	// Now let's build a starting SnapshottedUsage
	usage := g.startingSnapshot.Usage

	if lastResetSegmentIndex != -1 {
		// We need the segment right after the last reset
		if lastResetSegmentIndex+1 < len(g.segments) {
			firstSeg := g.segments[lastResetSegmentIndex+1]
			usage = usageAtReset(firstSeg.From)
		}
	}

	// Now we need to add up the usage in all segments between the starting usage and the provided segment
	for i := lastResetSegmentIndex + 1; i < segmentIndex; i++ {
		usage.Usage += g.segments[i].TotalUsage
	}

	return usage
}

func (g *GrantBurnDownHistory) Segments() []GrantBurnDownHistorySegment {
	return g.segments
}

func (g GrantBurnDownHistory) ChunkByResets() []GrantBurnDownHistory {
	if len(g.segments) == 0 {
		return nil
	}

	chunks := make([]GrantBurnDownHistory, 0, 1)
	current := GrantBurnDownHistory{
		startingSnapshot: g.startingSnapshot.Clone(),
		segments:         make([]GrantBurnDownHistorySegment, 0, len(g.segments)),
	}

	for i, seg := range g.segments {
		current.segments = append(current.segments, seg)
		if seg.TerminationReasons.UsageReset {
			chunks = append(chunks, current)

			var startingSnapshot balance.Snapshot
			if i+1 < len(g.segments) {
				startingSnapshot = g.getSnapshotAtStartOfSegment(i + 1)
			}

			current = GrantBurnDownHistory{
				startingSnapshot: startingSnapshot,
				segments:         make([]GrantBurnDownHistorySegment, 0, len(g.segments)),
			}
		}
	}

	if len(current.segments) > 0 {
		chunks = append(chunks, current)
	}

	return chunks
}

func (g GrantBurnDownHistory) TotalGrantUsage() alpacadecimal.Decimal {
	total := alpacadecimal.NewFromFloat(0)

	for _, seg := range g.segments {
		for _, usage := range seg.GrantUsages {
			total = total.Add(alpacadecimal.NewFromFloat(usage.Usage))
		}
	}

	return total
}

func usageAtReset(at time.Time) balance.SnapshottedUsage {
	return balance.SnapshottedUsage{
		Since: at,
		Usage: 0.0,
	}
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
