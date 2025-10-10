package subscription

import (
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
)

// The spec is the core object of Subscriptions being manipulated.
// Different resources like Patches, Addons, etc... can apply themselves to the spec.

type ApplyContext struct {
	CurrentTime time.Time
}

// Things can apply themselves to the spec
type AppliesToSpec interface {
	// This method should only ever be invoked by SubscriptionSpec (spec.ApplyX), so subsequent validations and logic are always guaranteed to run
	// FIXME(galexi): this can be enforced by making it private, but that will require rewriting all patches & addons
	ApplyTo(spec *SubscriptionSpec, actx ApplyContext) error
}

const (
	// Some errors are allowed during applying individual things to the spec, but still mean the Spec as a whole is invalid
	subscriptionPatchErrAttrNameAllowedDuringApplyingToSpecError = "allowed_during_applying_to_spec_error"
)

func AllowedDuringApplyingToSpecError() models.ValidationIssueOption {
	return models.WithAttribute(subscriptionPatchErrAttrNameAllowedDuringApplyingToSpecError, true)
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
				wrapError := func(err error) error {
					return models.ErrorWithComponent(models.ComponentName(fmt.Sprintf("patch at idx %d", i)), err)
				}

				issues, err := models.AsValidationIssues(err)
				if err != nil {
					return wrapError(err)
				}

				if lo.EveryBy(issues, func(issue models.ValidationIssue) bool {
					return IsValidationIssueWithBoolAttr(issue, subscriptionPatchErrAttrNameAllowedDuringApplyingToSpecError)
				}) {
					continue
				}

				// Otherwise we return with the error
				return wrapError(err)
			}
		}

		return nil
	})
}
