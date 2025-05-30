package productcatalog

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
)

type PlanAddonMeta struct {
	models.Metadata
	models.Annotations

	PlanAddonConfig
}

type PlanAddonConfig struct {
	// FromPlanPhase
	FromPlanPhase string `json:"fromPlanPhase"`

	// MaxQuantity
	MaxQuantity *int `json:"maxQuantity"`
}

var (
	_ models.Validator                  = (*PlanAddon)(nil)
	_ models.CustomValidator[PlanAddon] = (*PlanAddon)(nil)
)

type PlanAddon struct {
	PlanAddonMeta

	// Plan
	Plan Plan `json:"plan"`

	// Addon
	Addon Addon `json:"addon"`
}

func (c PlanAddon) ValidateWith(validators ...models.ValidatorFunc[PlanAddon]) error {
	return models.Validate(c, validators...)
}

// ValidationErrors returns a list of possible validation error(s) regarding to compatibility of the plan and add-on in the assignment.
// It returns nil if the plan and add-on are compatible.
func (c PlanAddon) ValidationErrors() (models.ValidationIssues, error) {
	err := c.Validate()
	if err == nil {
		return nil, nil
	}

	return models.AsValidationIssues(err)
}

func (c PlanAddon) Validate() error {
	var errs []error

	// Validate plan

	// Check plan status
	allowedPlanStatuses := []PlanStatus{PlanStatusDraft, PlanStatusActive, PlanStatusScheduled}
	if !lo.Contains(allowedPlanStatuses, c.Plan.Status()) {
		errs = append(errs, models.ErrorWithFieldPrefix(
			models.NewFieldSelectors(models.NewFieldSelector("plan")),
			ErrPlanAddonIncompatibleStatus,
		))
	}

	// Validate add-on

	addonPrefix := models.NewFieldSelectors(models.NewFieldSelector("addon"))

	// Add-on must be active and the effective period of add-on must be open-ended
	// as we do not support scheduled changes for add-ons.
	if c.Addon.Status() != AddonStatusActive || c.Addon.EffectiveTo != nil {
		errs = append(errs, models.ErrorWithFieldPrefix(addonPrefix, ErrPlanAddonIncompatibleStatus))
	}

	// Validate add-on assignment

	switch c.Addon.InstanceType {
	case AddonInstanceTypeMultiple:
		if c.MaxQuantity != nil && *c.MaxQuantity <= 0 {
			errs = append(errs, ErrPlanAddonMaxQuantityMustBeSet)
		}
	case AddonInstanceTypeSingle:
		if c.MaxQuantity != nil {
			errs = append(errs, ErrPlanAddonMaxQuantityMustNotBeSet)
		}
	}

	if c.Addon.Currency != c.Plan.Currency {
		errs = append(errs, models.ErrorWithFieldPrefix(addonPrefix, ErrPlanAddonCurrencyMismatch))
	}

	_, fromPhaseIdx, ok := lo.FindIndexOf(c.Plan.Phases, func(item Phase) bool {
		return item.Key == c.FromPlanPhase
	})

	if ok {
		// Validate ratecards from plan phases and addon.
		for _, phase := range c.Plan.Phases[fromPhaseIdx:] {
			if err := phase.ValidateWith(
				ValidatePlanPhaseAndAddonBillingCadenceAreAligned(c.Addon.RateCards),
				ValidatePlanPhaseAndAddonRateCardsAreCompatible(c.Addon.RateCards),
			); err != nil {
				errs = append(errs, models.ErrorWithFieldPrefix(addonPrefix, err))
			}
		}
	} else {
		errs = append(errs, ErrPlanAddonUnknownPlanPhaseKey)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func ValidatePlanPhaseAndAddonBillingCadenceAreAligned(ratecards RateCards) models.ValidatorFunc[Phase] {
	return func(p Phase) error {
		v := append(p.RateCards, ratecards...)

		if v.BillingCadenceAligned() {
			return nil
		}

		return models.ErrorWithFieldPrefix(
			models.NewFieldSelectors(models.NewFieldSelector("ratecards").WithExpression(models.WildCard)),
			ErrRateCardBillingCadenceUnaligned,
		)
	}
}

func ValidatePlanPhaseAndAddonRateCardsAreCompatible(addonRateCards RateCards) models.ValidatorFunc[Phase] {
	return func(p Phase) error {
		var errs []error

		phaseRateCardsByKey := lo.SliceToMap(p.RateCards, func(item RateCard) (string, RateCard) {
			return item.Key(), item
		})

		for _, addonRateCard := range addonRateCards {
			phaseRateCard, ok := phaseRateCardsByKey[addonRateCard.Key()]

			// Add-on ratecard is not present in plan phase ratecards, it is safe to skip.
			if !ok {
				continue
			}

			if err := NewRateCardWithOverlay(phaseRateCard, addonRateCard).Validate(); err != nil {
				errs = append(errs, fmt.Errorf("plan ratecard is not compatible with add-on ratecard: %w", err))
			}
		}

		return errors.Join(errs...)
	}
}
