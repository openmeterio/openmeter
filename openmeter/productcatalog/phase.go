package productcatalog

import (
	"errors"

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
		errs = append(errs, ErrResourceKeyEmpty)
	}

	if p.Name == "" {
		errs = append(errs, ErrResourceNameEmpty)
	}

	if p.Duration != nil {
		if p.Duration.IsNegative() {
			errs = append(errs, ErrPlanPhaseWithNegativeDuration)
		}

		// The duration must be at least 1 hour.
		if per, err := p.Duration.Subtract(isodate.NewPeriod(0, 0, 0, 0, 1, 0, 0)); err == nil && per.Sign() == -1 {
			errs = append(errs, ErrPlanPhaseDurationLessThenAnHour)
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var (
	_ models.Validator              = (*Phase)(nil)
	_ models.Equaler[Phase]         = (*Phase)(nil)
	_ models.CustomValidator[Phase] = (*Phase)(nil)
)

type Phase struct {
	PhaseMeta

	// RateCards
	RateCards RateCards `json:"rateCards"`
}

func (p Phase) ValidateWith(v ...models.ValidatorFunc[Phase]) error {
	return models.Validate(p, v...)
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
	return p.ValidateWith(
		ValidatePhaseMeta(),
		ValidatePhaseRateCards(),
	)
}

// ValidatePhaseMeta returns a validation function can be passed to the object
// which implements models.CustomValidator interface. It validates attributes in PhaseMeta of Phase.
func ValidatePhaseMeta() models.ValidatorFunc[Phase] {
	return func(p Phase) error {
		return p.PhaseMeta.Validate()
	}
}

// ValidatePhaseRateCards returns a validation function can be passed to the object
// which implements models.CustomValidator interface.
// It checks for invalid and duplicated ratecards in Phase.
func ValidatePhaseRateCards() models.ValidatorFunc[Phase] {
	return func(p Phase) error {
		if len(p.RateCards) == 0 {
			return ErrPlanPhaseHasNoRateCards
		}

		return ValidateRateCards()(p.RateCards)
	}
}

func ValidatePhaseHasBillingCadenceAligned() models.ValidatorFunc[Phase] {
	return func(p Phase) error {
		if p.RateCards.BillingCadenceAligned() {
			return nil
		}

		return models.ErrorWithFieldPrefix(
			models.NewFieldSelectors(models.NewFieldSelector("ratecards").
				WithExpression(models.WildCard)),
			ErrRateCardBillingCadenceUnaligned,
		)
	}
}
