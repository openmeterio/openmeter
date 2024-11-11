package plan

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

var _ Validator = (*Plan)(nil)

type Plan struct {
	models.NamespacedID
	models.ManagedModel
	EffectivePeriod

	// Key is the unique key for Plan.
	Key string `json:"key"`

	// Name
	Name string `json:"name"`

	// Description
	Description *string `json:"description,omitempty"`

	// Metadata
	Metadata map[string]string `json:"metadata,omitempty"`

	// Version
	Version int `json:"version"`

	// Currency
	Currency currency.Code `json:"currency"`

	// Phases
	Phases []Phase `json:"phases"`
}

func (p Plan) Validate() error {
	var errs []error

	if err := p.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid currency code: %s", err))
	}

	if p.Status() == InvalidStatus {
		errs = append(errs, fmt.Errorf("invalid effective time range: to is before from"))
	}

	// Check if there are multiple plan phase with the same startAfter which is not allowed.
	startAfters := make(map[datex.ISOString]Phase)
	for _, phase := range p.Phases {
		startAfter := phase.StartAfter.ISOString()

		if _, ok := startAfters[startAfter]; ok {
			errs = append(errs, fmt.Errorf("multiple plan phases have the same startAfter which is not allowed: %q", phase.Name))
		}

		if phase.Namespace != p.Namespace {
			errs = append(errs, fmt.Errorf("invalid phase %q: namespace mismatch %s", phase.Key, phase.Namespace))
		}

		if err := phase.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid phase %q: %s", phase.Name, err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

var _ Validator = (*EffectivePeriod)(nil)

type EffectivePeriod struct {
	// EffectiveFrom defines the time from the Plan becomes active.
	EffectiveFrom *time.Time `json:"effectiveFrom,omitempty"`

	// EffectiveTO defines the time from the Plan becomes archived.
	EffectiveTo *time.Time `json:"effectiveTo,omitempty"`
}

func (p EffectivePeriod) Validate() error {
	if p.Status() == InvalidStatus {
		return fmt.Errorf("invalid effective time range: to is before from")
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
