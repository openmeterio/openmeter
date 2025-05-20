package models

import (
	"errors"
	"slices"
	"time"

	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// Cadenced represents a model with active from and to dates.
// The interval described is inclusive on the from side and exclusive on the to side.
type (
	cadencedMarker bool // marker is used so only CadencedModel can implement Cadenced
	Cadenced       interface {
		cadenced() cadencedMarker
		cadence() CadencedModel
	}
)

type CadencedModel struct {
	ActiveFrom time.Time `json:"activeFrom"`
	// ActiveTo CANNOT be BEFORE ActiveFrom (it can be the same, which would mean the entity is never active)
	ActiveTo *time.Time `json:"activeTo"`
}

func (c CadencedModel) Equal(other CadencedModel) bool {
	if !c.ActiveFrom.Equal(other.ActiveFrom) {
		return false
	}

	if (c.ActiveTo == nil) != (other.ActiveTo == nil) {
		return false
	}

	if c.ActiveTo != nil && other.ActiveTo != nil {
		if !c.ActiveTo.Equal(*other.ActiveTo) {
			return false
		}
	}

	return true
}

func (c CadencedModel) AsPeriod() timeutil.OpenPeriod {
	return timeutil.OpenPeriod{
		From: &c.ActiveFrom,
		To:   c.ActiveTo,
	}
}

func NewCadencedModelFromPeriod(period timeutil.OpenPeriod) (CadencedModel, error) {
	if period.From == nil {
		return CadencedModel{}, errors.New("from date is required")
	}

	return CadencedModel{
		ActiveFrom: *period.From,
		ActiveTo:   period.To,
	}, nil
}

var _ Cadenced = CadencedModel{}

func (c CadencedModel) cadenced() cadencedMarker {
	return true
}

func (c CadencedModel) cadence() CadencedModel {
	return c
}

func (c CadencedModel) IsActiveAt(t time.Time) bool {
	if c.ActiveFrom.After(t) {
		return false
	}

	if c.ActiveTo != nil && !c.ActiveTo.After(t) {
		return false
	}

	return true
}

func (c CadencedModel) IsZero() bool {
	return c.ActiveFrom.IsZero() && c.ActiveTo == nil
}

// CadenceList is a simple abstraction for a list of cadenced models.
// It is useful to validate the relationship between the cadences of the models, like their ordering, overlaps, continuity, etc.
type CadenceList[T Cadenced] []T

// OverlapReason defines the reason for an overlap.
type OverlapReason string

const (
	// OverlapReasonActiveToNil indicates an overlap because the first item's ActiveTo is nil.
	OverlapReasonActiveToNil OverlapReason = "ActiveTo is nil"
	// OverlapReasonActiveToAfterActiveFrom indicates an overlap because the first item's ActiveTo is after the second item's ActiveFrom.
	OverlapReasonActiveToAfterActiveFrom OverlapReason = "ActiveTo is after next ActiveFrom"
)

// OverlapDetail provides detailed information about an overlap between two cadenced items.
type OverlapDetail[T Cadenced] struct {
	Index1 int           // Index of the first item in the original list
	Index2 int           // Index of the second item in the original list
	Item1  T             // The first item involved in the overlap
	Item2  T             // The second item involved in the overlap
	Reason OverlapReason // The reason for the overlap
}

func NewSortedCadenceList[T Cadenced](cadences []T) CadenceList[T] {
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
		cadence1 := item1.cadence()
		cadence2 := item2.cadence()

		if cadence1.ActiveTo == nil {
			overlaps = append(overlaps, OverlapDetail[T]{
				Index1: i - 1,
				Index2: i,
				Item1:  item1,
				Item2:  item2,
				Reason: OverlapReasonActiveToNil,
			})
			continue
		}

		if cadence1.ActiveTo != nil && cadence2.ActiveFrom.Before(*cadence1.ActiveTo) {
			overlaps = append(overlaps, OverlapDetail[T]{
				Index1: i - 1,
				Index2: i,
				Item1:  item1,
				Item2:  item2,
				Reason: OverlapReasonActiveToAfterActiveFrom,
			})
		}
	}

	// The original implementation used a map to ensure uniqueness of [2]int pairs.
	// Since we are now returning richer details, and the loop structure inherently processes pairs (i-1, i) sequentially,
	// direct appends should be fine. If specific de-duplication logic for OverlapDetail is needed based on new criteria,
	// it would need to be implemented here. For now, this matches the previous logic's intent of finding overlaps between adjacent elements.
	return overlaps
}

func (t CadenceList[T]) IsSorted() bool {
	for i := 1; i < len(t); i++ {
		if t[i-1].cadence().ActiveFrom.After(t[i].cadence().ActiveFrom) {
			return false
		}
	}

	return true
}

func (t CadenceList[T]) IsContinuous() bool {
	for i := 1; i < len(t); i++ {
		if t[i-1].cadence().ActiveTo == nil || !t[i-1].cadence().ActiveTo.Equal(t[i].cadence().ActiveFrom) {
			return false
		}
	}

	return true
}

func (t CadenceList[T]) sort() {
	slices.SortStableFunc(t, func(a, b T) int {
		aC := a.cadence()
		bC := b.cadence()

		return int(aC.ActiveFrom.Sub(bC.ActiveFrom).Milliseconds())
	})
}
