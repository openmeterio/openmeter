package productcatalog

import (
	"errors"
	"fmt"
	"time"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	PlanStatusDraft     PlanStatus = "draft"
	PlanStatusActive    PlanStatus = "active"
	PlanStatusArchived  PlanStatus = "archived"
	PlanStatusScheduled PlanStatus = "scheduled"
	PlanStatusInvalid   PlanStatus = "invalid"
)

type PlanStatus string

func (s PlanStatus) Values() []string {
	return []string{
		string(PlanStatusDraft),
		string(PlanStatusActive),
		string(PlanStatusArchived),
		string(PlanStatusScheduled),
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

func (p Plan) ValidateWith(validators ...models.ValidatorFunc[Plan]) error {
	return models.Validate(p, validators...)
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

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// FIXME: rename to publishable
// ValidForCreatingSubscriptions checks if the Plan is valid for creating Subscriptions, a stricter version of Validate
func (p Plan) ValidForCreatingSubscriptions() error {
	var errs []error

	if err := p.Validate(); err != nil {
		errs = append(errs, err)
	}

	// FIXME:

	if len(p.Phases) == 0 {
		return models.NewGenericValidationError(errors.New("invalid Plan: at least one PlanPhase is required"))
	}

	// Check if only the last phase has no duration
	for i, phase := range p.Phases {
		if phase.Duration == nil && i != len(p.Phases)-1 {
			errs = append(errs, models.NewGenericValidationError(
				fmt.Errorf("invalid Plan: the duration must be set for the phase %s (index %d)", phase.Name, i),
			))
		}

		if phase.Duration != nil && i == len(p.Phases)-1 {
			errs = append(errs, models.NewGenericValidationError(
				fmt.Errorf("invalid Plan: the duration must not be set for the last phase (index %d)", i),
			))
		}

		if len(phase.RateCards) < 1 {
			errs = append(errs, models.NewGenericValidationError(
				fmt.Errorf("invalid Plan: at least one RateCards in PlanPhase is required [phase_key=%s]", phase.Key),
			))
		}
	}

	// Let's check Alignment
	if p.Alignment.BillablesMustAlign {
		for i, phase := range p.Phases {
			periods := make(map[isodate.String]bool)

			// For each phase, all RateCards that have a price associated must align
			for _, rc := range phase.RateCards.Billables() {
				// 1 time prices are excluded
				if d := rc.GetBillingCadence(); d != nil {
					periods[d.Normalise(true).ISOString()] = true
				}
			}

			if len(periods) > 1 {
				errs = append(errs, models.NewGenericValidationError(
					fmt.Errorf("invalid Plan: all RateCards with prices in the phase %s (index %d) must have the same billing cadence, found: %v", phase.Name, i, lo.Keys(periods)),
				))
			}
		}
	}

	return errors.Join(errs...)
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
	_ models.Validator             = (*PlanMeta)(nil)
	_ models.CustomValidator[Plan] = (*Plan)(nil)
	_ models.Equaler[PlanMeta]     = (*PlanMeta)(nil)
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

	return models.NewNillableGenericValidationError(errors.Join(errs...))
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

// Status returns the current status of the Plan
func (p PlanMeta) Status() PlanStatus {
	return p.StatusAt(time.Now())
}

// StatusAt returns the plan status relative to time t.
func (p PlanMeta) StatusAt(t time.Time) PlanStatus {
	from := lo.FromPtrOr(p.EffectiveFrom, time.Time{})
	to := lo.FromPtrOr(p.EffectiveTo, time.Time{})

	// Plan has DraftStatus if neither the EffectiveFrom nor EffectiveTo are set
	if from.IsZero() && to.IsZero() {
		return PlanStatusDraft
	}

	// Plan has ArchivedStatus if EffectiveTo is in the past relative to time t.
	if from.Before(t) && (to.Before(t) && from.Before(to)) {
		return PlanStatusArchived
	}

	// Plan has ActiveStatus if EffectiveFrom is set in the past relative to time t and EffectiveTo is not set
	// or in the future relative to time t.
	if from.Before(t) && (to.IsZero() || to.After(t)) {
		return PlanStatusActive
	}

	// Plan is ScheduledForActiveStatus if EffectiveFrom is set in the future relative to time t and EffectiveTo is not set
	// or in the future relative to time t.
	if from.After(t) && (to.IsZero() || to.After(from)) {
		return PlanStatusScheduled
	}

	return PlanStatusInvalid
}

func PlanWithAllowedStatus(allowed ...PlanStatus) models.ValidatorFunc[Plan] {
	return func(p Plan) error {
		status := p.Status()
		if lo.Contains(allowed, status) {
			return nil
		}

		return fmt.Errorf("plan status %s is not valid, must be one of %+v", status, allowed)
	}
}
