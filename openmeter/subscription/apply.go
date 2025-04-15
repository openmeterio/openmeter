package subscription

import (
	"fmt"
	"time"

	"github.com/samber/lo"
)

// The spec is the core object of Subscriptions being manipulated.
// Different resources like Patches, Addons, etc... can apply themselves to the spec.

type ApplyContext struct {
	CurrentTime time.Time
}

// Things can apply themselves to the spec
type AppliesToSpec interface {
	ApplyTo(spec *SubscriptionSpec, actx ApplyContext) error
}

// Some errors are allowed during applying individual things to the spec, but still mean the Spec as a whole is invalid
type AllowedDuringApplyingToSpecError struct {
	Inner error
}

func (e *AllowedDuringApplyingToSpecError) Error() string {
	return fmt.Sprintf("allowed during incremental validation failed: %s", e.Inner)
}

func (e *AllowedDuringApplyingToSpecError) Unwrap() error {
	return e.Inner
}

func NewAppliesToSpec(fn func(spec *SubscriptionSpec, actx ApplyContext) error) AppliesToSpec {
	return &someAppliesToSpec{
		Fn: fn,
	}
}

var _ AppliesToSpec = &someAppliesToSpec{}

type someAppliesToSpec struct {
	Fn func(spec *SubscriptionSpec, actx ApplyContext) error
}

func (s *someAppliesToSpec) ApplyTo(spec *SubscriptionSpec, actx ApplyContext) error {
	return s.Fn(spec, actx)
}

// NewAggregateAppliesToSpec aggregates multiple applies to spec into a single applies to spec, and also validates the spec after applying all the patches
func NewAggregateAppliesToSpec(applieses []AppliesToSpec) AppliesToSpec {
	return NewAppliesToSpec(func(spec *SubscriptionSpec, actx ApplyContext) error {
		for i, applies := range applieses {
			if err := spec.Apply(applies, actx); err != nil {
				if uw, ok := err.(interface{ Unwrap() []error }); ok {
					// If all returned errors are allowed during applying patches, we can continue
					if lo.EveryBy(uw.Unwrap(), func(e error) bool {
						_, ok := lo.ErrorsAs[*AllowedDuringApplyingToSpecError](e)
						return ok
					}) {
						continue
					}
				} else if uw, ok := err.(interface{ Unwrap() error }); ok {
					if _, ok := lo.ErrorsAs[*AllowedDuringApplyingToSpecError](uw.Unwrap()); ok {
						continue
					}
				}

				// Otherwise we return with the error
				return fmt.Errorf("appliesToSpec %d failed during validation: %w", i, err)
			}
		}

		return nil
	})
}
