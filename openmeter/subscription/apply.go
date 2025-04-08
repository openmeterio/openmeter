package subscription

import (
	"fmt"
	"time"
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
