package models

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCadenceItem is a mock implementation of the Cadenced interface for testing.
type MockCadenceItem struct {
	ActiveFrom time.Time
	ActiveTo   *time.Time
}

func (m MockCadenceItem) GetCadence() CadencedModel {
	return CadencedModel(m)
}

func (m MockCadenceItem) cadence() CadencedModel {
	return CadencedModel(m)
}

// cadenced makes MockCadenceItem implement the Cadenced interface.
func (m MockCadenceItem) cadenced() cadencedMarker {
	return true // The actual value doesn't matter for these tests
}

var _ Cadenced = MockCadenceItem{} // Verify that MockCadenceItem implements Cadenced

func TestNewSortedCadenceListZeroLengthRevisions(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	replacedAt := start.Add(time.Hour)
	for _, activeTo := range []*time.Time{nil, lo.ToPtr(replacedAt.Add(time.Hour))} {
		previous := MockCadenceItem{ActiveFrom: start, ActiveTo: &replacedAt}
		empty := MockCadenceItem{ActiveFrom: replacedAt, ActiveTo: &replacedAt}
		current := MockCadenceItem{ActiveFrom: replacedAt, ActiveTo: activeTo}
		items := []MockCadenceItem{current, empty, previous, empty}

		timeline := NewSortedCadenceList(items)

		require.Equal(t, []MockCadenceItem{previous, empty, empty, current}, timeline.Cadences())
		require.Empty(t, timeline.GetOverlaps())
		require.True(t, timeline.IsContinuous())
		require.Equal(t, current, items[0], "sorting must not mutate the input")

		// A second active interval is still an overlap after the empty history.
		items = append(items, current)
		overlaps := NewSortedCadenceList(items).GetOverlaps()
		require.Len(t, overlaps, 1)
		require.Equal(t, current, overlaps[0].Item1)
		require.Equal(t, current, overlaps[0].Item2)
	}
}

func TestCadenceList_GetOverlaps(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		list := CadenceList[MockCadenceItem]{}
		expected := []OverlapDetail[MockCadenceItem]{}
		assert.ElementsMatch(t, expected, list.GetOverlaps(), "Elements should match in any order")
	})

	t.Run("no overlaps", func(t *testing.T) {
		list := CadenceList[MockCadenceItem]{
			{ActiveFrom: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC))},
			{ActiveFrom: time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC))},
		}
		expected := []OverlapDetail[MockCadenceItem]{}
		assert.ElementsMatch(t, expected, list.GetOverlaps(), "Elements should match in any order")
	})

	t.Run("overlap with nil ActiveTo", func(t *testing.T) {
		list := CadenceList[MockCadenceItem]{
			{ActiveFrom: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ActiveTo: nil},
			{ActiveFrom: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC))},
		}
		expected := []OverlapDetail[MockCadenceItem]{
			{
				Index1: 0,
				Index2: 1,
				Item1:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ActiveTo: nil},
				Item2:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC))},
			},
		}
		assert.ElementsMatch(t, expected, list.GetOverlaps(), "Elements should match in any order")
	})

	t.Run("overlap with non-nil ActiveTo", func(t *testing.T) {
		list := CadenceList[MockCadenceItem]{
			{ActiveFrom: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC))},
			{ActiveFrom: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC))},
		}
		expected := []OverlapDetail[MockCadenceItem]{
			{
				Index1: 0,
				Index2: 1,
				Item1:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC))},
				Item2:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC))},
			},
		}
		assert.ElementsMatch(t, expected, list.GetOverlaps(), "Elements should match in any order")
	})

	t.Run("multiple overlaps", func(t *testing.T) {
		list := CadenceList[MockCadenceItem]{
			{ActiveFrom: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC))},
			{ActiveFrom: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC))},
			{ActiveFrom: time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC), ActiveTo: nil},
			{ActiveFrom: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC))},
		}
		expected := []OverlapDetail[MockCadenceItem]{
			{
				Index1: 0,
				Index2: 1,
				Item1:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC))},
				Item2:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC))},
			},
			{
				Index1: 1,
				Index2: 2,
				Item1:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC))},
				Item2:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC), ActiveTo: nil},
			},
			{
				Index1: 2,
				Index2: 3,
				Item1:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC), ActiveTo: nil},
				Item2:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC))},
			},
		}
		assert.ElementsMatch(t, expected, list.GetOverlaps(), "Elements should match in any order")
	})

	t.Run("no overlap - adjacent", func(t *testing.T) {
		list := CadenceList[MockCadenceItem]{
			{ActiveFrom: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC))},
			{ActiveFrom: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC))},
		}
		expected := []OverlapDetail[MockCadenceItem]{}
		assert.ElementsMatch(t, expected, list.GetOverlaps(), "Elements should match in any order")
	})

	t.Run("single item", func(t *testing.T) {
		list := CadenceList[MockCadenceItem]{
			{ActiveFrom: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ActiveTo: lo.ToPtr(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC))},
		}
		expected := []OverlapDetail[MockCadenceItem]{}
		assert.ElementsMatch(t, expected, list.GetOverlaps(), "Elements should match in any order")
	})

	t.Run("all nil ActiveTo", func(t *testing.T) {
		list := CadenceList[MockCadenceItem]{
			{ActiveFrom: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ActiveTo: nil},
			{ActiveFrom: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), ActiveTo: nil},
			{ActiveFrom: time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC), ActiveTo: nil},
		}
		expected := []OverlapDetail[MockCadenceItem]{
			{
				Index1: 0,
				Index2: 1,
				Item1:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ActiveTo: nil},
				Item2:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), ActiveTo: nil},
			},
			{
				Index1: 1,
				Index2: 2,
				Item1:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), ActiveTo: nil},
				Item2:  MockCadenceItem{ActiveFrom: time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC), ActiveTo: nil},
			},
		}
		assert.ElementsMatch(t, expected, list.GetOverlaps(), "Elements should match in any order")
	})
}
