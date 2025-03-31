package productcatalog

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	_ models.Validator          = (*PhaseMeta)(nil)
	_ models.Equaler[PhaseMeta] = (*PhaseMeta)(nil)
)

type PhaseMeta struct {
	// Key is the unique key for Phase.
	Key string `json:"key"`

	// Name is the name of the Phase.
	Name string `json:"name"`

	// Description is the detailed description of the Phase.
	Description *string `json:"description,omitempty"`

	// Metadata stores user defined metadata for Phase.
	Metadata models.Metadata `json:"metadata,omitempty"`

	// Duration is the duration of the Phase.
	Duration *isodate.Period `json:"duration"`
}

// Equal returns true if the two PhaseMetas are equal.
func (p PhaseMeta) Equal(v PhaseMeta) bool {
	if p.Key != v.Key {
		return false
	}

	if p.Name != v.Name {
		return false
	}

	if lo.FromPtr(p.Description) != lo.FromPtr(v.Description) {
		return false
	}

	if !p.Metadata.Equal(v.Metadata) {
		return false
	}

	if !p.Duration.Equal(v.Duration) {
		return false
	}

	return true
}

// Validate validates the PhaseMeta.
func (p PhaseMeta) Validate() error {
	var errs []error

	if p.Key == "" {
		errs = append(errs, errors.New("missing Key"))
	}

	if p.Name == "" {
		errs = append(errs, errors.New("missing Name"))
	}

	if p.Duration != nil {
		if p.Duration.IsNegative() {
			errs = append(errs, fmt.Errorf("the Duration period must not be negative"))
		}

		// The duration must be at least 1 hour.
		if per, err := p.Duration.Subtract(isodate.NewPeriod(0, 0, 0, 0, 1, 0, 0)); err == nil && per.Sign() == -1 {
			errs = append(errs, fmt.Errorf("the Duration period must be at least 1 hour"))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var (
	_ models.Validator      = (*Phase)(nil)
	_ models.Equaler[Phase] = (*Phase)(nil)
)

type Phase struct {
	PhaseMeta

	// RateCards
	RateCards RateCards `json:"rateCards"`
}

// Equal returns true if the two Phases are equal.
func (p Phase) Equal(v Phase) bool {
	if !p.PhaseMeta.Equal(v.PhaseMeta) {
		return false
	}

	return p.RateCards.Equal(v.RateCards)
}

// Validate validates the Phase.
func (p Phase) Validate() error {
	var errs []error

	if err := p.PhaseMeta.Validate(); err != nil {
		errs = append(errs, err)
	}

	// Check for
	// * duplicated rate card keys
	// * namespace mismatch
	// * invalid RateCards
	rateCardKeys := make(map[string]RateCard)
	for _, rateCard := range p.RateCards {
		if _, ok := rateCardKeys[rateCard.Key()]; ok {
			errs = append(errs, fmt.Errorf("duplicated RateCard: %s", rateCard.Key()))
		} else {
			rateCardKeys[rateCard.Key()] = rateCard
		}

		if err := rateCard.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid RateCard: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
