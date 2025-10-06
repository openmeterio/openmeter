package models

import (
	"errors"
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

var _ CadenceComparable = CadencedModel{}

func (c CadencedModel) GetCadence() CadencedModel {
	return c
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
