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
func (c PlanAddon) ValidationErrors() []InvalidResourceError {
	err := c.Validate()
	if err == nil {
		return nil
	}

	return UnwrapErrors[InvalidResourceError](err)
}

func (c PlanAddon) Validate() error {
	var errs []error

	// Validate plan

	planResource := Resource{
		Key:  c.Plan.Key,
		Kind: "plan",
	}

	// Check plan status
	allowedPlanStatuses := []PlanStatus{PlanStatusDraft, PlanStatusActive, PlanStatusScheduled}
	if !lo.Contains(allowedPlanStatuses, c.Plan.Status()) {
		errs = append(errs, InvalidResourceError{
			Resource: planResource,
			Field:    "status",
			Detail:   fmt.Sprintf("invalid status %s allowed statuses: %+v", c.Plan.Status(), allowedPlanStatuses),
		})
	}

	// Validate add-on

	addonResource := Resource{
		Key:  c.Addon.Key,
		Kind: "addon",
		Attributes: map[string]any{
			"version": c.Addon.Version,
		},
	}

	// Add-on must be active and the effective period of add-on must be open-ended
	// as we do not support scheduled changes for add-ons.
	if c.Addon.Status() != AddonStatusActive || c.Addon.EffectiveTo != nil {
		errs = append(errs, InvalidResourceError{
			Resource: addonResource,
			Field:    "status",
			Detail:   fmt.Sprintf("invalid status %s, add-on must be active", c.Addon.Status()),
		})
	}

	// Validate add-on assignment

	switch c.Addon.InstanceType {
	case AddonInstanceTypeMultiple:
		if c.MaxQuantity != nil && *c.MaxQuantity <= 0 {
			errs = append(errs, InvalidResourceError{
				Resource: addonResource,
				Field:    "maxQuantity",
				Detail:   "must be set to positive number for add-on with multiple instance type",
			})
		}
	case AddonInstanceTypeSingle:
		if c.MaxQuantity != nil {
			errs = append(errs, InvalidResourceError{
				Resource: addonResource,
				Field:    "maxQuantity",
				Detail:   "must not be set for add-on with single instance type",
			})
		}
	}

	if c.Addon.Currency != c.Plan.Currency {
		errs = append(errs, InvalidResourceError{
			Resource: addonResource,
			Field:    "currency",
			Detail:   "currency mismatch",
		})
	}

	_, fromPhaseIdx, ok := lo.FindIndexOf(c.Plan.Phases, func(item Phase) bool {
		return item.Key == c.FromPlanPhase
	})

	if ok {
		// Validate ratecards from plan phases and addon.
		for _, phase := range c.Plan.Phases[fromPhaseIdx:] {
			if err := c.validateRateCardsInPhase(phase.RateCards, c.Addon.RateCards); err != nil {
				errs = append(errs, fmt.Errorf("invalid phase [phase.key=%s]: ratecards are not compatible: %w", phase.Key, err))
			}
		}
	} else {
		errs = append(errs, InvalidResourceError{
			Resource: addonResource,
			Field:    "fromPlanPhase",
			Detail:   fmt.Sprintf("plan does not have phase with key: %s", c.FromPlanPhase),
		})
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
				phaseRateCard.Key(), err))
		}
	}

	return errors.Join(errs...)
}
