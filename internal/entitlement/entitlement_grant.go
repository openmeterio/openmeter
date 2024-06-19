package entitlement

import (
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
)

type EntitlementGrant struct {
	credit.Grant

	// "removing" fields
	OwnerID          string  `json:"-"`
	ResetMaxRollover float64 `json:"-"`

	// "adding" fields
	EntitlementID     string     `json:"entitlementId"`
	NextRecurrence    *time.Time `json:"nextRecurrence,omitempty"`
	MaxRolloverAmount float64    `json:"maxRolloverAmount"`
}

func GrantFromCreditGrant(grant credit.Grant) (*EntitlementGrant, error) {
	g := &EntitlementGrant{}
	if grant.Recurrence != nil {
		next, err := grant.Recurrence.NextAfter(time.Now())
		if err != nil {
			return nil, err
		}
		g.NextRecurrence = &next
	}
	g.Grant = grant
	g.EntitlementID = string(grant.OwnerID)
	g.MaxRolloverAmount = grant.ResetMaxRollover
	return g, nil
}

type CreateEntitlementGrantInputs struct {
	credit.CreateGrantInput
}
