package credit

type SegmentTerminationReason struct {
	PriorityChange bool
	Recurrence     []GrantID
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
	GrantID           GrantID
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

type GrantBurnDownHistory struct {
	Segments []GrantBurnDownHistorySegment
}

func (g *GrantBurnDownHistory) Append(segment GrantBurnDownHistorySegment) {
	g.Segments = append(g.Segments, segment)
}
