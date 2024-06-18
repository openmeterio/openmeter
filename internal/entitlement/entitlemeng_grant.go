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

// {
//     "amount": 100,
//     "priority": 1,
//     "effectiveAt": "2023-01-01T00:00:00Z",
//     "expiration": {
//       "duration": "HOUR",
//       "count": 12
//     },
//     "maxRolloverAmount": 100,
//     "metadata": {
//       "stripePaymentId": "pi_4OrAkhLvyihio9p51h9iiFnB"
//     },
//     "recurrence": {
//       "interval": "DAILY",
//       "anchor": "2024-06-18T11:18:06.752Z"
//     },
//     "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
//     "createdAt": "2023-01-01T00:00:00Z",
//     "updatedAt": "2023-01-01T00:00:00Z",
//     "deletedAt": null,
//     "entitlementId": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
//     "subjectKey": "customer-id",
//     "nextRecurrence": "2023-01-01T00:00:00Z",
//     "expiresAt": "2023-01-01T00:00:00Z"
// }
