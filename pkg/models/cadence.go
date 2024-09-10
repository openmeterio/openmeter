package models

import (
	"time"
)

// Cadenced represents a model with active from and to dates.
// The interval described is inclusive on the from side and exclusive on the to side.
type (
	cadencedMarker bool // marker is used so only CadencedModel can implement Cadenced
	Cadenced       interface {
		cadenced() cadencedMarker
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

var _ Cadenced = CadencedModel{}

func (c CadencedModel) cadenced() cadencedMarker {
	return true
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
