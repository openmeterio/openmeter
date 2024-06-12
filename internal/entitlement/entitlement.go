package entitlement

import "github.com/openmeterio/openmeter/pkg/models"

type EntitlementID string
type NamespacedEntitlementID struct {
	Namespace string
	ID        EntitlementID
}

type Entitlement struct {
	models.NamespacedModel
	models.ManagedModel
	// ID is the readonly identifies of a entitlement.
	ID EntitlementID `json:"id,omitempty"`
}

// What an entitlement does
// It has balances => not balance directly but a dynamic field, whether its active, entitles (boolean type)
//      Something else calculates the balance it just uses the calcualted balance
// It has references...
