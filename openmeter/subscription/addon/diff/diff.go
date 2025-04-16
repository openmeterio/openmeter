package addondiff

import (
	"github.com/openmeterio/openmeter/openmeter/subscription"
)

// Diffable is a type that can be both applied to a spec and removed from a spec
type Diffable interface {
	// GetApplies returns the applies that will be applied to the spec
	GetApplies() subscription.AppliesToSpec
	// GetRestores returns the restores that will be applied to the spec
	GetRestores() subscription.AppliesToSpec
}

var _ Diffable = &someDiffable{}

type someDiffable struct {
	ApplyFn   func(spec *subscription.SubscriptionSpec, actx subscription.ApplyContext) error
	RestoreFn func(spec *subscription.SubscriptionSpec, actx subscription.ApplyContext) error
}

func (s *someDiffable) GetApplies() subscription.AppliesToSpec {
	return subscription.NewAppliesToSpec(s.ApplyFn)
}

func (s *someDiffable) GetRestores() subscription.AppliesToSpec {
	return subscription.NewAppliesToSpec(s.RestoreFn)
}
