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
			if !spec.Alignment.BillablesMustAlign {
				return def, models.NewGenericValidationError(fmt.Errorf("next_billing_cycle is not supported for non-aligned subscriptions"))
			}

			currentPhase, exists := spec.GetCurrentPhaseAt(clock.Now())
			if !exists {
				// If there isn't a current phase, the subscription hasn't started or has already ended
				return def, models.NewGenericValidationError(fmt.Errorf("billing isn't active for the subscription, there isn't a next_billing_cycle"))
			}

			if !currentPhase.HasBillables() {
				return def, models.NewGenericValidationError(fmt.Errorf("current phase has no billables, there isn't a next_billing_cycle"))
			}

			period, err := spec.GetAlignedBillingPeriodAt(currentPhase.PhaseKey, clock.Now())

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

		if subView.Subscription.Alignment.BillablesMustAlign {
			if c.Custom != nil {
				return models.NewGenericValidationError(fmt.Errorf("cannot cancel aligned subscription with custom timing"))
			}

			if c.Enum != nil && *c.Enum == TimingImmediate {
				// We only allow immediate cancels if the current phase has no billing period
				currentPhase, currentPhaseExists := subView.Spec.GetCurrentPhaseAt(clock.Now())

				if currentPhaseExists {
					_, err := subView.Spec.GetAlignedBillingPeriodAt(currentPhase.PhaseKey, clock.Now())
					if err == nil {
						return models.NewGenericValidationError(fmt.Errorf("cannot cancel aligned subscription immediately that has a billing period"))
					}
				}
			}
		}

		// We don't allow to cancel misaligned subscriptions with next_billing_cycle timing as it makes no sense
		if !subView.Subscription.Alignment.BillablesMustAlign && c.Enum != nil && *c.Enum == TimingNextBillingCycle {
			return models.NewGenericValidationError(fmt.Errorf("cannot cancel misaligned subscription with next_billing_cycle timing"))
		}

	default:
		slog.Warn("timing called with unsupported action", slog.Any("action", action), slog.String("stack", string(debug.Stack())))

		return nil
	}

	return nil
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
