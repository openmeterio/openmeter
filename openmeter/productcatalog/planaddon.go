package productcatalog

import (
	"errors"
	"fmt"

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

	// Plan must be active.
	if c.Plan.Status() != PlanStatusActive {
		errs = append(errs,
			fmt.Errorf("invalid plan: status must be active [plan.key=%s plan.version=%d]",
				c.Plan.Key, c.Plan.Version),
		)
	}

	// Validate add-on

	// Add-on must be active and the effective period of add-on must be open-ended
	// as we do not support scheduled changes for add-ons.
	if c.Addon.Status() != AddonStatusActive || c.Addon.EffectiveTo != nil {
		errs = append(errs,
			fmt.Errorf("invalid add-on: status must be active [addon.key=%s addon.version=%d]",
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
				// If ratecards can be merged then they are compatible.
				if err := phase.RateCards.Compatible(c.Addon.RateCards); err != nil {
					errs = append(errs,
						fmt.Errorf("invalid phase [phase.key=%s]: ratecards are not compatible: %w", phase.Key, err),
					)
				}
			}
		}
	} else {
		errs = append(errs, errors.New("invalid plan: has no phases"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
