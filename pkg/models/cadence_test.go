package models

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

// MockCadenceItem is a mock implementation of the Cadenced interface for testing.
type MockCadenceItem struct {
	ActiveFrom time.Time
	ActiveTo   *time.Time
}

func (m MockCadenceItem) cadence() CadencedModel {
	return CadencedModel(m)
}

// cadenced makes MockCadenceItem implement the Cadenced interface.
func (m MockCadenceItem) cadenced() cadencedMarker {
	return true // The actual value doesn't matter for these tests
}

var _ Cadenced = MockCadenceItem{} // Verify that MockCadenceItem implements Cadenced

func TestCadenceList_GetOverlaps(t *testing.T) {
	tests := []struct {
		name     string
		list     CadenceList[MockCadenceItem]
		expected []OverlapDetail[MockCadenceItem]
	}{
		{
			name:     "empty list",
			list:     CadenceList[MockCadenceItem]{},
			expected: []OverlapDetail[MockCadenceItem]{},
		},
		{
			name: "no overlaps",
			list: CadenceList[MockCadenceItem]{
				{ActiveFrom: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC))},
				{ActiveFrom: time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC))},
			},
			expected: []OverlapDetail[MockCadenceItem]{},
		},
		{
			name: "overlap with nil ActiveTo",
			list: CadenceList[MockCadenceItem]{
				{ActiveFrom: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ActiveTo: nil},
				{ActiveFrom: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC))},
			},
			expected: []OverlapDetail[MockCadenceItem]{
				{
					Index1: 0,
					Index2: 1,
					Item1:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ActiveTo: nil},
					Item2:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC))},
					Reason: OverlapReasonActiveToNil,
				},
			},
		},
		{
			name: "overlap with non-nil ActiveTo",
			list: CadenceList[MockCadenceItem]{
				{ActiveFrom: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC))},
				{ActiveFrom: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC))},
			},
			expected: []OverlapDetail[MockCadenceItem]{
				{
					Index1: 0,
					Index2: 1,
					Item1:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC))},
					Item2:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC))},
					Reason: OverlapReasonActiveToAfterActiveFrom,
				},
			},
		},
		{
			name: "multiple overlaps",
			list: CadenceList[MockCadenceItem]{
				{ActiveFrom: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC))},
				{ActiveFrom: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC))},
				{ActiveFrom: time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC), ActiveTo: nil},
				{ActiveFrom: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC))},
			},
			expected: []OverlapDetail[MockCadenceItem]{
				{
					Index1: 0,
					Index2: 1,
					Item1:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC))},
					Item2:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC))},
					Reason: OverlapReasonActiveToAfterActiveFrom,
				},
				{
					Index1: 1,
					Index2: 2,
					Item1:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC))},
					Item2:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC), ActiveTo: nil},
					Reason: OverlapReasonActiveToAfterActiveFrom,
				},
				{
					Index1: 2,
					Index2: 3,
					Item1:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC), ActiveTo: nil},
					Item2:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC))},
					Reason: OverlapReasonActiveToNil,
				},
			},
		},
		{
			name: "no overlap - adjacent",
			list: CadenceList[MockCadenceItem]{
				{ActiveFrom: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC))},
				{ActiveFrom: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC))},
			},
			expected: []OverlapDetail[MockCadenceItem]{},
		},
		{
			name: "single item",
			list: CadenceList[MockCadenceItem]{
				{ActiveFrom: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC))},
			},
			expected: []OverlapDetail[MockCadenceItem]{},
		},
		{
			name: "all nil ActiveTo",
			list: CadenceList[MockCadenceItem]{
				{ActiveFrom: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ActiveTo: nil},
				{ActiveFrom: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), ActiveTo: nil},
				{ActiveFrom: time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC), ActiveTo: nil},
			},
			expected: []OverlapDetail[MockCadenceItem]{
				{
					Index1: 0,
					Index2: 1,
					Item1:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ActiveTo: nil},
					Item2:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), ActiveTo: nil},
					Reason: OverlapReasonActiveToNil,
				},
				{
					Index1: 1,
					Index2: 2,
					Item1:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), ActiveTo: nil},
					Item2:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC), ActiveTo: nil},
					Reason: OverlapReasonActiveToNil,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ElementsMatch(t, tt.expected, tt.list.GetOverlaps(), "Elements should match in any order")
		})
	}
}
