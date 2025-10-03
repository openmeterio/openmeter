package models

import "slices"

// TODO[galexi]: Get rid of these types and use period.Period instead:
// - Add Intersection method to period.Period so all types implement it
// - Write helpers using period.Period
// CadenceList is a simple abstraction for a list of cadenced models.
// It is useful to validate the relationship between the cadences of the models, like their ordering, overlaps, continuity, etc.
type CadenceList[T CadenceComparable] []T

type CadenceComparable interface {
	GetCadence() CadencedModel
}

type Overlap[T any] struct {
	This  T `json:"this"`
	Other T `json:"other"`
}

type OverlapDetail[T CadenceComparable] struct {
	Index1 int
	Index2 int
	Item1  T
	Item2  T
}

func NewSortedCadenceList[T CadenceComparable](cadences []T) CadenceList[T] {
	local := make([]T, len(cadences))
	copy(local, cadences)

	t := CadenceList[T](local)
	t.sort()

	return t
}

// Cadences returns the cadences in the timeline
func (t CadenceList[T]) Cadences() []T {
	return t
}

// TODO: rewrite CadenceList helpers to use timeutil.OpenPeriod instead

// GetOverlaps returns details about any overlaps between the cadences in the timeline.
func (t CadenceList[T]) GetOverlaps() []OverlapDetail[T] {
	var overlaps []OverlapDetail[T]

	for i := 0; i < len(t); i++ {
		if i == 0 {
			continue
		}

		item1 := t[i-1]
		item2 := t[i]
		cadence1 := item1.GetCadence()
		cadence2 := item2.GetCadence()

		if cadence1.ActiveTo == nil {
			overlaps = append(overlaps, OverlapDetail[T]{
				Index1: i - 1,
				Index2: i,
				Item1:  item1,
				Item2:  item2,
			})
			continue
		}

		if cadence1.ActiveTo != nil && cadence2.ActiveFrom.Before(*cadence1.ActiveTo) {
			overlaps = append(overlaps, OverlapDetail[T]{
				Index1: i - 1,
				Index2: i,
				Item1:  item1,
				Item2:  item2,
			})
		}
	}

	return overlaps
}

func (t CadenceList[T]) IsSorted() bool {
	for i := 1; i < len(t); i++ {
		if t[i-1].GetCadence().ActiveFrom.After(t[i].GetCadence().ActiveFrom) {
			return false
		}
	}

	return true
}

func (t CadenceList[T]) IsContinuous() bool {
	for i := 1; i < len(t); i++ {
		if t[i-1].GetCadence().ActiveTo == nil || !t[i-1].GetCadence().ActiveTo.Equal(t[i].GetCadence().ActiveFrom) {
			return false
		}
	}

	return true
}

func (t CadenceList[T]) sort() {
	slices.SortStableFunc(t, func(a, b T) int {
		aC := a.GetCadence()
		bC := b.GetCadence()

		switch {
		case aC.ActiveFrom.Before(bC.ActiveFrom):
			return -1
		case aC.ActiveFrom.After(bC.ActiveFrom):
			return 1
		default:
			return 0
		}
	})
}
