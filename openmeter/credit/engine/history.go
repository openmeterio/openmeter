// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package engine

import (
	"fmt"
	"sort"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/pkg/recurrence"
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
	recurrence.Period
	BalanceAtStart     balance.Map
	TerminationReasons SegmentTerminationReason // Reason why the segment was terminated (could be multiple taking effect at same time)
	TotalUsage         float64                  // Total usage of the feature in the Period
	OverageAtStart     float64                  // Usage beyond what could be burnt down from the grants in the previous segment (if any)
	Overage            float64                  // Usage beyond what cloud be burnt down from the grants
	GrantUsages        []GrantUsage             // Grant usages in the segment order by grant priority
}

// Returns GrantBalanceMap at the end of the segment
func (s GrantBurnDownHistorySegment) ApplyUsage() balance.Map {
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

// Creates a GrantBalanceSnapshot from the starting state of the segment
func (s *GrantBurnDownHistorySegment) ToSnapshot() balance.Snapshot {
	return balance.Snapshot{
		Overage:  s.OverageAtStart,
		Balances: s.BalanceAtStart,
		At:       s.From,
	}
}
