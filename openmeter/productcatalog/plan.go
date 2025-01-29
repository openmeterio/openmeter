package productcatalog

import (
	"errors"
	"fmt"
	"time"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	DraftStatus     PlanStatus = "draft"
	ActiveStatus    PlanStatus = "active"
	ArchivedStatus  PlanStatus = "archived"
	ScheduledStatus PlanStatus = "scheduled"
	InvalidStatus   PlanStatus = "invalid"
)

type PlanStatus string

func (s PlanStatus) Values() []string {
	return []string{
		string(DraftStatus),
		string(ActiveStatus),
		string(ArchivedStatus),
		string(ScheduledStatus),
	}
}

var (
	_ models.Validator     = (*Plan)(nil)
	_ models.Equaler[Plan] = (*Plan)(nil)
)

type Plan struct {
	PlanMeta

	// Phases
	Phases []Phase `json:"phases"`
}

func (p Plan) Validate() error {
	var errs []error

	if err := p.PlanMeta.Validate(); err != nil {
		errs = append(errs, err)
	}

	for _, phase := range p.Phases {
		if err := phase.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid PlanPhase %q: %s", phase.Name, err))
		}
	}

	return NewValidationError(errors.Join(errs...))
}

// ValidForCreatingSubscriptions checks if the Plan is valid for creating Subscriptions, a stricter version of Validate
func (p Plan) ValidForCreatingSubscriptions() error {
	var errs []error

	if err := p.Validate(); err != nil {
		errs = append(errs, err)
	}

	if len(p.Phases) == 0 {
		return NewValidationError(fmt.Errorf("invalid Plan: at least one PlanPhase is required"))
	}

	// Check if only the last phase has no duration
	for i, phase := range p.Phases {
		if phase.Duration == nil && i != len(p.Phases)-1 {
			errs = append(errs, NewValidationError(
				fmt.Errorf("invalid Plan: the duration must be set for the phase %s (index %d)", phase.Name, i),
			))
		}

		if phase.Duration != nil && i == len(p.Phases)-1 {
			errs = append(errs, NewValidationError(
				fmt.Errorf("invalid Plan: the duration must not be set for the last phase (index %d)", i),
			))
		}
	}

	// Let's check Alignment
	if p.Alignment.BillablesMustAlign {
		for i, phase := range p.Phases {
			periods := make(map[datex.ISOString]bool)

			// For each phase, all RateCards that have a price associated must align
			for _, rc := range phase.RateCards.Billables() {
				// 1 time prices are excluded
				if d := rc.GetBillingCadence(); d != nil {
					periods[d.Normalise(true).ISOString()] = true
				}
			}

			if len(periods) > 1 {
				errs = append(errs, NewValidationError(
					fmt.Errorf("invalid Plan: all RateCards with prices in the phase %s (index %d) must have the same billing cadence, found: %v", phase.Name, i, lo.Keys(periods)),
				))
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// Equal returns true if the two Plans are equal.
func (p Plan) Equal(o Plan) bool {
	if !p.PlanMeta.Equal(o.PlanMeta) {
		return false
	}

	if len(p.Phases) != len(o.Phases) {
		return false
	}

	for i, phase := range p.Phases {
		if !phase.Equal(o.Phases[i]) {
			return false
		}
	}

	return true
}

var (
	_ models.Validator         = (*PlanMeta)(nil)
	_ models.Equaler[PlanMeta] = (*PlanMeta)(nil)
)

type PlanMeta struct {
	EffectivePeriod
	Alignment

	// Key is the unique key for Plan.
	Key string `json:"key"`

	// Version
	Version int `json:"version"`

	// Name
	Name string `json:"name"`

	// Description
	Description *string `json:"description,omitempty"`

	// Currency
	Currency currency.Code `json:"currency"`

	// Metadata
	Metadata models.Metadata `json:"metadata,omitempty"`
}

// Validate validates the PlanMeta.
func (p PlanMeta) Validate() error {
	var errs []error

	if err := p.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid Currency: %s", err))
	}

	if err := p.EffectivePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid EffectivePeriod: %s", err))
	}

	if p.Key == "" {
		errs = append(errs, fmt.Errorf("invalid Key: must not be empty"))
	}

	if p.Name == "" {
		errs = append(errs, fmt.Errorf("invalid Name: must not be empty"))
	}

	return NewValidationError(errors.Join(errs...))
}

// Equal returns true if the two PlanMetas are equal.
func (p PlanMeta) Equal(o PlanMeta) bool {
	if p.Key != o.Key {
		return false
	}

	if p.Version != o.Version {
		return false
	}

	if p.Name != o.Name {
		return false
	}

	if p.Description != o.Description {
		return false
	}

	if p.Currency != o.Currency {
		return false
	}

	if !p.EffectivePeriod.Equal(o.EffectivePeriod) {
		return false
	}

	if !p.Metadata.Equal(o.Metadata) {
		return false
	}

	return true
}

var (
	_ models.Validator                = (*EffectivePeriod)(nil)
	_ models.Equaler[EffectivePeriod] = (*EffectivePeriod)(nil)
)

type EffectivePeriod struct {
	// EffectiveFrom defines the time from the Plan becomes active.
	EffectiveFrom *time.Time `json:"effectiveFrom,omitempty"`

	// EffectiveTO defines the time from the Plan becomes archived.
	EffectiveTo *time.Time `json:"effectiveTo,omitempty"`
}

func (p EffectivePeriod) Validate() error {
	if p.Status() == InvalidStatus {
		return NewValidationError(fmt.Errorf("invalid effective time range: to is before from"))
	}

	return nil
}

// Status returns the current status of the Plan
func (p EffectivePeriod) Status() PlanStatus {
	return p.StatusAt(time.Now())
}

// StatusAt returns the plan status relative to time t.
func (p EffectivePeriod) StatusAt(t time.Time) PlanStatus {
	from := lo.FromPtrOr(p.EffectiveFrom, time.Time{})
	to := lo.FromPtrOr(p.EffectiveTo, time.Time{})

	// Plan has DraftStatus if neither the EffectiveFrom nor EffectiveTo are set
	if from.IsZero() && to.IsZero() {
		return DraftStatus
	}

	// Plan has ArchivedStatus if EffectiveTo is in the past relative to time t.
	if from.Before(t) && (to.Before(t) && from.Before(to)) {
		return ArchivedStatus
	}

	// Plan has ActiveStatus if EffectiveFrom is set in the past relative to time t and EffectiveTo is not set
	// or in the future relative to time t.
	if from.Before(t) && (to.IsZero() || to.After(t)) {
		return ActiveStatus
	}

	// Plan is ScheduledForActiveStatus if EffectiveFrom is set in the future relative to time t and EffectiveTo is not set
	// or in the future relative to time t.
	if from.After(t) && (to.IsZero() || to.After(from)) {
		return ScheduledStatus
	}

	return InvalidStatus
}

// Equal returns true if the two EffectivePeriods are equal.
func (p EffectivePeriod) Equal(o EffectivePeriod) bool {
	return lo.FromPtrOr(p.EffectiveFrom, time.Time{}).Equal(lo.FromPtrOr(o.EffectiveFrom, time.Time{})) &&
		lo.FromPtrOr(p.EffectiveTo, time.Time{}).Equal(lo.FromPtrOr(o.EffectiveTo, time.Time{}))
}
