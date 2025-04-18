package models

import (
	"errors"
	"slices"
	"time"

	"github.com/samber/lo"

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

// GetOverlaps returns true if there is any overlap between the cadences in the timeline
func (t CadenceList[T]) GetOverlaps() [][2]int {
	overlaps := make(map[[2]int][2]int)

	addIfNew := func(a, b int) {
		tp := [2]int{a, b}
		if _, exists := overlaps[tp]; !exists {
			overlaps[tp] = tp
		}
	}

	for i := 0; i < len(t); i++ {
		if i == 0 {
			continue
		}

		if t[i-1].cadence().ActiveTo == nil {
			addIfNew(i-1, i)
			continue
		}

		if t[i-1].cadence().ActiveTo != nil && t[i].cadence().ActiveFrom.Before(*t[i-1].cadence().ActiveTo) {
			addIfNew(i-1, i)
		}
	}

	return lo.Values(overlaps)
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
