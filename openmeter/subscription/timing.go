package subscription

import (
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Timing represents the timing of a change in a subscription, such as a plan change or a cancellation.
type Timing struct {
	Custom *time.Time
	Enum   *TimingEnum
}

func (c Timing) Validate() error {
	if c.Custom == nil && c.Enum == nil {
		return fmt.Errorf("change timing must have either a custom time or an enum")
	}

	if c.Custom != nil && c.Enum != nil {
		return fmt.Errorf("change timing cannot have both a custom time and an enum")
	}

	if c.Enum != nil {
		return c.Enum.Validate()
	}

	return nil
}

func (c Timing) Resolve() (time.Time, error) {
	var def time.Time

	if err := c.Validate(); err != nil {
		return def, err
	}

	if c.Custom != nil {
		return *c.Custom, nil
	}

	if c.Enum != nil {
		switch *c.Enum {
		case TimingImmediate:
			return clock.Now(), nil
		default:
			return def, &models.GenericUserError{Inner: fmt.Errorf("unsupported enum value: %s", *c.Enum)}
		}
	}

	return def, fmt.Errorf("no logical branch entered")
}

func (c Timing) ResolveForSpec(spec SubscriptionSpec) (time.Time, error) {
	var def time.Time

	if err := c.Validate(); err != nil {
		return def, err
	}

	if c.Custom != nil {
		return *c.Custom, nil
	}

	if c.Enum != nil {
		switch *c.Enum {
		case TimingImmediate:
			return clock.Now(), nil
		case TimingNextBillingCycle:
			if !spec.Alignment.BillablesMustAlign {
				return def, &models.GenericUserError{Inner: fmt.Errorf("next_billing_cycle is not supported for non-aligned subscriptions")}
			}

			currentPhase, exists := spec.GetCurrentPhaseAt(clock.Now())
			if !exists {
				// If there isn't a current phase, the subscription hasn't started or has already ended
				return def, &models.GenericUserError{Inner: fmt.Errorf("billing isn't active for the subscription, there isn't a next_billing_cycle")}
			}

			period, err := spec.GetAlignedBillingPeriodAt(currentPhase.PhaseKey, clock.Now())

			return period.To, err
		default:
			return def, &models.GenericUserError{Inner: fmt.Errorf("unsupported enum value: %s", *c.Enum)}
		}
	}

	return def, fmt.Errorf("no logical branch entered")
}

type TimingEnum string

const (
	// Immediate means the change will take effect immediately.
	TimingImmediate TimingEnum = "immediate"
	// NextBillingCycle means the change will take effect at the start of the next billing cycle.
	// This value is only supported for aligned subscriptions.
	TimingNextBillingCycle TimingEnum = "next_billing_cycle"
)

func (c TimingEnum) Validate() error {
	switch c {
	case TimingImmediate, TimingNextBillingCycle:
		return nil
	default:
		return fmt.Errorf("invalid change timing: %s", c)
	}
}
