package entitlement

type EntitlementID string

type Entitlement struct {
	// ID is the readonly identifies of a entitlement.
	ID EntitlementID `json:"id,omitempty"`
}

// What an entitlement does
// It has balances => not balance directly but a dynamic field, whether its active, entitles (boolean type)
//      Something else calculates the balance it just uses the calcualted balance
// It has references...
