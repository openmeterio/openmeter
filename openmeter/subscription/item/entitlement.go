package subscriptionitem

import "github.com/openmeterio/openmeter/openmeter/entitlement"

type EntitlementCreator interface {
	// TODO: Arguments need to be defined later
	GetEntitlementSpec(args ...[]any) (entitlement.CreateEntitlementInputs, error)
}

type EntitlementUpdater interface{}

// You can update an entitlement by
// - adding one or more new grants to it
// - removing one or more grants from it
// - by resetting it??? is that a change here we want ot support?
