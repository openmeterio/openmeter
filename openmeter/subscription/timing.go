package subscription

import (
	"fmt"
	"log/slog"
	"runtime/debug"
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
			return def, models.NewGenericValidationError(fmt.Errorf("unsupported enum value: %s", *c.Enum))
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
			if spec.BillingCadence.IsZero() {
				return def, models.NewGenericValidationError(fmt.Errorf("subscription does not have a billing cadence, there isn't a next_billing_cycle"))
			}

			period, err := spec.GetAlignedBillingPeriodAt(clock.Now())

			return period.To, err
		default:
			return def, models.NewGenericValidationError(fmt.Errorf("unsupported enum value: %s", *c.Enum))
		}
	}

	return def, fmt.Errorf("no logical branch entered")
}

func (c Timing) ValidateForAction(action SubscriptionAction, subView *SubscriptionView) error {
	if err := c.Validate(); err != nil {
		return err
	}

	switch action {
	case SubscriptionAction("any"):
		return nil
	case SubscriptionActionUpdate:
		if subView == nil {
			return fmt.Errorf("missing subscription view")
		}

		if c.Custom != nil {
			return models.NewGenericValidationError(fmt.Errorf("cannot edit running subscription with custom timing"))
		}

		currentTime := clock.Now()
		editTime, err := c.ResolveForSpec(subView.Spec)
		if err != nil {
			return fmt.Errorf("failed to resolve timing: %w", err)
		}

		// As we're possibly time-traveling, we need to set some constraints on what times we can travel to, otherwise, well, we know from sci-fi movies what happens
		if editTime.Before(currentTime) {
			return models.NewGenericValidationError(fmt.Errorf("cannot execute edits in the past"))
		}

		currentPhase, currentPhaseExists := subView.Spec.GetCurrentPhaseAt(currentTime)
		editPhase, editPhaseExists := subView.Spec.GetCurrentPhaseAt(editTime)

		if currentPhaseExists && editPhaseExists && currentPhase.PhaseKey != editPhase.PhaseKey {
			// Let's check if this happens due to a known cause. If so, we can return a more user-friendly error
			if c.Enum != nil && *c.Enum == TimingNextBillingCycle {
				return models.NewGenericValidationError(fmt.Errorf("cannot edit to the next billing cycle as it falls into a different phase"))
			}

			// If not, we return a generic error
			return models.NewGenericValidationError(fmt.Errorf("cannot time-travel to edit a different phase"))
		}

	case SubscriptionActionCreate:
		if c.Enum != nil && *c.Enum == TimingImmediate {
			return nil
		}

		tolerance := 2 * time.Minute

		if c.Custom != nil {
			if c.Custom.Before(clock.Now().Add(-tolerance)) {
				return models.NewGenericValidationError(fmt.Errorf("cannot create subscription in the past"))
			}

			return nil
		}

	case SubscriptionActionCancel:
		if subView == nil {
			return fmt.Errorf("missing subscription view")
		}

		if c.Custom != nil {
			if !c.isDateAlignedWithBillingCadence(subView.Spec, *c.Custom) {
				return models.NewGenericValidationError(fmt.Errorf("cannot cancel aligned subscription with custom misaligned timing"))
			}
		}

	case SubscriptionActionChangeAddons:
		if subView == nil {
			return fmt.Errorf("missing subscription view")
		}

		// Everything is possible
		return nil
	default:
		slog.Warn("timing called with unsupported action", slog.Any("action", action), slog.String("stack", string(debug.Stack())))

		return nil
	}

	return nil
}

func (c Timing) isDateAlignedWithBillingCadence(spec SubscriptionSpec, date time.Time) bool {
	period, err := spec.GetAlignedBillingPeriodAt(date)
	if err != nil {
		return false
	}

	switch {
	case period.From.Equal(date):
		return true
	case period.To.Equal(date):
		return true
	default:
		return false
	}
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
