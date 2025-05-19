package productcatalog

import (
	"errors"
	"fmt"
	"strconv"

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
	planPrefix := models.FieldPathFromParts("plans", c.Plan.Key, "versions", strconv.Itoa(c.Plan.Version))

	// Check plan status
	allowedPlanStatuses := []PlanStatus{PlanStatusDraft, PlanStatusActive, PlanStatusScheduled}
	if !lo.Contains(allowedPlanStatuses, c.Plan.Status()) {
		errs = append(errs, models.ErrorWithFieldPrefix(planPrefix,
			ErrPlanAddonIncompatibleStatus,
		))
	}

	// Validate add-on

	addonPrefix := models.FieldPathFromParts("addons", c.Addon.Key, "versions", strconv.Itoa(c.Addon.Version))

	// Add-on must be active and the effective period of add-on must be open-ended
	// as we do not support scheduled changes for add-ons.
	if c.Addon.Status() != AddonStatusActive || c.Addon.EffectiveTo != nil {
		errs = append(errs, models.ErrorWithFieldPrefix(addonPrefix,
			ErrPlanAddonIncompatibleStatus,
		))
	}

	// Validate add-on assignment

	switch c.Addon.InstanceType {
	case AddonInstanceTypeMultiple:
		if c.MaxQuantity != nil && *c.MaxQuantity <= 0 {
			errs = append(errs, models.ErrorWithFieldPrefix(addonPrefix,
				ErrPlanAddonMaxQuantityMustBeSet,
			))
		}
	case AddonInstanceTypeSingle:
		if c.MaxQuantity != nil {
			errs = append(errs, models.ErrorWithFieldPrefix(addonPrefix,
				ErrPlanAddonMaxQuantityMustNotBeSet,
			))
		}
	}

	if c.Addon.Currency != c.Plan.Currency {
		errs = append(errs, models.ErrorWithFieldPrefix(addonPrefix,
			ErrPlanAddonCurrencyMismatch,
		))
	}

	_, fromPhaseIdx, ok := lo.FindIndexOf(c.Plan.Phases, func(item Phase) bool {
		return item.Key == c.FromPlanPhase
	})

	if ok {
		// Validate ratecards from plan phases and addon.
		for _, phase := range c.Plan.Phases[fromPhaseIdx:] {
			if err := c.validateRateCardsInPhase(phase.RateCards, c.Addon.RateCards); err != nil {
				errs = append(errs, models.ErrorWithFieldPrefix(models.FieldPathFromParts(planPrefix, "phases", phase.Key),
					err),
				)
			}
		}
	} else {
		errs = append(errs, models.ErrorWithFieldPrefix(models.FieldPathFromParts(addonPrefix),
			ErrPlanAddonUnknownPlanPhaseKey,
		))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (c PlanAddon) validateRateCardsInPhase(phaseRateCards, addonRateCards RateCards) error {
	var errs []error

	phaseRateCardsByKey := lo.SliceToMap(phaseRateCards, func(item RateCard) (string, RateCard) {
		return item.Key(), item
	})

	for _, addonRateCard := range addonRateCards {
		phaseRateCard, ok := phaseRateCardsByKey[addonRateCard.Key()]

		// Add-on ratecard is not present in plan phase ratecards, it is safe to skip.
		if !ok {
			continue
		}

		if err := NewRateCardWithOverlay(phaseRateCard, addonRateCard).Validate(); err != nil {
			errs = append(errs, fmt.Errorf("plan ratecard is not compatible with add-on ratecard [key=%s]: %w",
				phaseRateCard.Key(), err),
			)
		}
	}

	return errors.Join(errs...)
}
