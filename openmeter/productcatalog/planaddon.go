package productcatalog

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
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

func (c PlanAddon) Validate() error {
	var errs []error

	// Validate config

	switch c.Addon.InstanceType {
	case AddonInstanceTypeMultiple:
		if c.MaxQuantity != nil && *c.MaxQuantity <= 0 {
			errs = append(errs,
				fmt.Errorf("maxQuantity must be set to positive number for add-on with multiple instance type [addon.key=%s addon.version=%d]",
					c.Addon.Key, c.Addon.Version),
			)
		}
	case AddonInstanceTypeSingle:
		if c.MaxQuantity != nil {
			errs = append(errs,
				fmt.Errorf("maxQuantity must not be set for add-on with single instance type [addon.key=%s addon.version=%d]",
					c.Addon.Key, c.Addon.Version),
			)
		}
	}

	// Validate plan

	// Check plan status
	allowedPlanStatuses := []PlanStatus{PlanStatusDraft, PlanStatusActive, PlanStatusScheduled}
	if !lo.Contains(allowedPlanStatuses, c.Plan.Status()) {
		errs = append(errs,
			fmt.Errorf("invalid plan [plan.key=%s plan.version=%d]: invalid %s status, allowed statuses: %+v",
				c.Plan.Key, c.Plan.Version, c.Plan.Status(), allowedPlanStatuses),
		)
	}

	// Validate add-on

	// Add-on must be active and the effective period of add-on must be open-ended
	// as we do not support scheduled changes for add-ons.
	if c.Addon.Status() != AddonStatusActive || c.Addon.EffectiveTo != nil {
		errs = append(errs,
			fmt.Errorf("invalid add-on [addon.key=%s addon.version=%d]: status must be active",
				c.Addon.Key, c.Addon.Version),
		)
	}

	// validate plan with add-on

	// Currency must match.
	if c.Addon.Currency != c.Plan.Currency {
		errs = append(errs, errors.New("currency mismatch"))
	}

	if len(c.Plan.Phases) > 0 {
		phaseIdx := -1
		for i, phase := range c.Plan.Phases {
			if phase.Key == c.FromPlanPhase {
				phaseIdx = i
				break
			}
		}

		if phaseIdx == -1 {
			errs = append(errs, fmt.Errorf("plan does not have phase %q", c.FromPlanPhase))
		} else {
			// Validate ratecards from plan phases and addon.
			for _, phase := range c.Plan.Phases[phaseIdx:] {
				if err := c.validateRateCardsInPhase(phase.RateCards, c.Addon.RateCards); err != nil {
					errs = append(errs, fmt.Errorf("invalid phase [phase.key=%s]: ratecards are not compatible: %w", phase.Key, err))
				}
			}
		}
	} else {
		errs = append(errs, errors.New("invalid plan: has no phases"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (c PlanAddon) validateRateCardsInPhase(phaseRateCards, addonRateCards RateCards) error {
	var errs []error

	// We'll assume phaseRateCards and addonRateCards are otherwise valid by unique constraints and formal contents...
	for _, phaseRateCard := range phaseRateCards {
		affectingRateCards := lo.Filter(addonRateCards, slicesx.AsFilterIteratee(AddonRateCardMatcherForAGivenPlanRateCard(phaseRateCard)))

		// No RateCards affect this plan RateCard
		if len(affectingRateCards) == 0 {
			continue
		}

		// For now we only support a single RateCard per addon effecting a single plan RateCard.
		if len(affectingRateCards) > 1 {
			errs = append(errs, fmt.Errorf("multiple add-on ratecards affect plan ratecard [plan.ratecard.key=%s]: affecting add-on ratecard keys: %+v", phaseRateCard.Key(), lo.Map(affectingRateCards, func(item RateCard, index int) string {
				return item.Key()
			})))

			continue
		}

		// Finally, let's check that they are compatible
		if err := rateCardsCompatible(phaseRateCard, affectingRateCards[0]); err != nil {
			errs = append(errs, fmt.Errorf("plan ratecard is not compatible with add-on ratecard [plan.ratecard.key=%s add-on.ratecard.key=%s]: %w", phaseRateCard.Key(), affectingRateCards[0].Key(), err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
