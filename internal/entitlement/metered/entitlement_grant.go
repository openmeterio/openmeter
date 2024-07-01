package meteredentitlement

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (e *connector) CreateGrant(ctx context.Context, entId models.NamespacedID, inputGrant CreateEntitlementGrantInputs) (EntitlementGrant, error) {
	ent, err := e.entitlementRepo.GetEntitlement(ctx, entId)
	if err != nil {
		return EntitlementGrant{}, err
	}
	_, err = ParseFromGenericEntitlement(ent)
	if err != nil {
		return EntitlementGrant{}, err
	}
	grant, error := e.grantConnector.CreateGrant(ctx, credit.NamespacedGrantOwner{
		Namespace: ent.Namespace,
		ID:        credit.GrantOwner(ent.ID),
	}, credit.CreateGrantInput{
		Amount:           inputGrant.Amount,
		Priority:         inputGrant.Priority,
		EffectiveAt:      inputGrant.EffectiveAt,
		Expiration:       inputGrant.Expiration,
		ResetMaxRollover: inputGrant.ResetMaxRollover,
		ResetMinRollover: inputGrant.ResetMinRollover,
		Recurrence:       inputGrant.Recurrence,
		Metadata:         inputGrant.Metadata,
	})
	if error != nil {
		if _, ok := error.(credit.OwnerNotFoundError); ok {
			return EntitlementGrant{}, &entitlement.NotFoundError{EntitlementID: entId}
		}

		return EntitlementGrant{}, error
	}

	g, err := GrantFromCreditGrant(*grant)
	return *g, err
}

func (e *connector) ListEntitlementGrants(ctx context.Context, entitlementID models.NamespacedID) ([]EntitlementGrant, error) {
	// check that we own the grant
	grants, err := e.grantConnector.ListGrants(ctx, credit.ListGrantsParams{
		Namespace:      entitlementID.Namespace,
		OwnerID:        convert.ToPointer(credit.GrantOwner(entitlementID.ID)),
		IncludeDeleted: false,
		OrderBy:        credit.GrantOrderByCreatedAt,
	})
	if err != nil {
		return nil, err
	}

	ents := make([]EntitlementGrant, 0, len(grants))
	for _, grant := range grants {
		g, err := GrantFromCreditGrant(grant)
		if err != nil {
			return nil, err
		}
		ents = append(ents, *g)
	}

	return ents, nil
}

type EntitlementGrant struct {
	credit.Grant

	// "removing" fields
	OwnerID          string  `json:"-"`
	ResetMaxRollover float64 `json:"-"`
	ResetMinRollover float64 `json:"-"`

	// "adding" fields
	EntitlementID     string     `json:"entitlementId"`
	NextRecurrence    *time.Time `json:"nextRecurrence,omitempty"`
	MaxRolloverAmount float64    `json:"maxRolloverAmount"`
	MinRolloverAmount float64    `json:"minRolloverAmount"`
}

func GrantFromCreditGrant(grant credit.Grant) (*EntitlementGrant, error) {
	g := &EntitlementGrant{}
	if grant.Recurrence != nil {
		next, err := grant.Recurrence.NextAfter(clock.Now())
		if err != nil {
			return nil, err
		}
		g.NextRecurrence = &next
	}
	g.Grant = grant
	g.EntitlementID = string(grant.OwnerID)
	g.MaxRolloverAmount = grant.ResetMaxRollover
	g.MinRolloverAmount = grant.ResetMinRollover
	return g, nil
}

type CreateEntitlementGrantInputs struct {
	credit.CreateGrantInput
}
