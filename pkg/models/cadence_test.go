package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Simple implementation of Cadenced interface for testing
type TestCadence struct {
	From *time.Time
	To   *time.Time
}

func (tc TestCadence) cadenced() cadencedMarker {
	return true
}

func (tc TestCadence) cadence() CadencedModel {
	if tc.From == nil {
		panic("From cannot be nil")
	}
	return CadencedModel{
		ActiveFrom: *tc.From,
		ActiveTo:   tc.To,
	}
}

func parseTime(t *testing.T, timeStr string) *time.Time {
	if timeStr == "" {
		return nil
	}

	parsed, err := time.Parse(time.RFC3339Nano, timeStr)
	if err != nil {
		t.Fatalf("Failed to parse time %s: %v", timeStr, err)
	}
	return &parsed
}

func TestGetOverlaps(t *testing.T) {
	tests := []struct {
		name     string
		cadences CadenceList[TestCadence]
		want     [][2]int
	}{
		{
			name:     "no cadences",
			cadences: CadenceList[TestCadence]{},
			want:     [][2]int{},
		},
		{
			name: "single cadence",
			cadences: CadenceList[TestCadence]{
				{
					From: parseTime(t, "2025-01-01T00:00:00Z"),
					To:   parseTime(t, "2025-01-02T00:00:00Z"),
				},
			},
			want: [][2]int{},
		},
		{
			name: "non-overlapping cadences",
			cadences: CadenceList[TestCadence]{
				{
					From: parseTime(t, "2025-01-01T00:00:00Z"),
					To:   parseTime(t, "2025-01-02T00:00:00Z"),
				},
				{
					From: parseTime(t, "2025-01-02T00:00:00Z"),
					To:   parseTime(t, "2025-01-03T00:00:00Z"),
				},
				{
					From: parseTime(t, "2025-01-03T00:00:00Z"),
					To:   parseTime(t, "2025-01-04T00:00:00Z"),
				},
			},
			want: [][2]int{},
		},
		{
			name: "overlapping cadences",
			cadences: CadenceList[TestCadence]{
				{
					From: parseTime(t, "2025-01-01T00:00:00Z"),
					To:   parseTime(t, "2025-01-03T00:00:00Z"),
				},
				{
					From: parseTime(t, "2025-01-02T00:00:00Z"),
					To:   parseTime(t, "2025-01-04T00:00:00Z"),
				},
			},
			want: [][2]int{{0, 1}},
		},
		{
			name: "exact boundary case (should not overlap)",
			cadences: CadenceList[TestCadence]{
				{
					From: parseTime(t, "2025-04-30T09:33:12.456401Z"),
					To:   parseTime(t, "2025-04-30T09:33:26.346Z"),
				},
				{
					From: parseTime(t, "2025-04-30T09:33:26.346Z"),
					To:   parseTime(t, "2025-04-30T09:33:40.000Z"),
				},
			},
			want: [][2]int{},
		},
		{
			name: "production error case (should NOT be an overlap with exact boundary)",
			cadences: CadenceList[TestCadence]{
				{
					From: parseTime(t, "2025-04-30T09:33:12.456401Z"),
					To:   parseTime(t, "2025-04-30T09:33:26.346Z"),
				},
				{
					From: parseTime(t, "2025-04-30T09:33:26.346Z"),
					To:   nil,
				},
			},
			want: [][2]int{},
		},
		{
			name: "nanosecond precision overlap",
			cadences: CadenceList[TestCadence]{
				{
					From: parseTime(t, "2025-04-30T09:33:12.456401Z"),
					To:   parseTime(t, "2025-04-30T09:33:26.347Z"),
				},
				{
					From: parseTime(t, "2025-04-30T09:33:26.346Z"),
					To:   parseTime(t, "2025-04-30T09:33:40.000Z"),
				},
			},
			want: [][2]int{{0, 1}},
		},
		{
			name: "nil ActiveTo (should be considered overlap)",
			cadences: CadenceList[TestCadence]{
				{
					From: parseTime(t, "2025-01-01T00:00:00Z"),
					To:   nil,
				},
				{
					From: parseTime(t, "2025-01-02T00:00:00Z"),
					To:   parseTime(t, "2025-01-03T00:00:00Z"),
				},
			},
			want: [][2]int{{0, 1}},
		},
		{
			name: "multiple overlaps",
			cadences: CadenceList[TestCadence]{
				{
					From: parseTime(t, "2025-01-01T00:00:00Z"),
					To:   parseTime(t, "2025-01-03T00:00:00Z"),
				},
				{
					From: parseTime(t, "2025-01-02T00:00:00Z"),
					To:   parseTime(t, "2025-01-04T00:00:00Z"),
				},
				{
					From: parseTime(t, "2025-01-03T12:00:00Z"),
					To:   parseTime(t, "2025-01-05T00:00:00Z"),
				},
			},
			want: [][2]int{{0, 1}, {1, 2}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cadences.GetOverlaps()
			// Use len comparison instead of direct equality to handle nil vs empty slice
			if len(got) == 0 && len(tt.want) == 0 {
				// Both are empty, test passes
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
