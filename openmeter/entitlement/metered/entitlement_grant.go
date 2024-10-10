package meteredentitlement

import (
	"context"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
)

// CreateGrant creates a grant for a given entitlement
//
// You can issue grants for inactive entitlements by passing the entitlement ID
func (e *connector) CreateGrant(ctx context.Context, namespace string, subjectKey string, entitlementIdOrFeatureKey string, inputGrant CreateEntitlementGrantInputs) (EntitlementGrant, error) {
	// First we attempt to find the entitlement by ID, then by featureKey
	ent, err := e.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: entitlementIdOrFeatureKey})
	if _, ok := lo.ErrorsAs[*entitlement.NotFoundError](err); ok {
		ent, err = e.entitlementRepo.GetActiveEntitlementOfSubjectAt(ctx, namespace, subjectKey, entitlementIdOrFeatureKey, clock.Now())
	}
	if err != nil {
		return EntitlementGrant{}, err
	}
	_, err = ParseFromGenericEntitlement(ent)
	if err != nil {
		return EntitlementGrant{}, err
	}
	g, err := e.grantConnector.CreateGrant(ctx, grant.NamespacedOwner{
		Namespace: ent.Namespace,
		ID:        grant.Owner(ent.ID),
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
	if err != nil {
		if _, ok := err.(grant.OwnerNotFoundError); ok {
			return EntitlementGrant{}, &entitlement.NotFoundError{EntitlementID: models.NamespacedID{Namespace: namespace, ID: ent.ID}}
		}

		return EntitlementGrant{}, err
	}

	eg, err := GrantFromCreditGrant(*g)
	return *eg, err
}

// ListEntitlementGrants lists all grants for a given entitlement
//
// You can list grants for inactive entitlements by passing the entitlement ID
func (e *connector) ListEntitlementGrants(ctx context.Context, namespace string, subjectKey string, entitlementIdOrFeatureKey string) ([]EntitlementGrant, error) {
	// Find the matching entitlement, first by ID, then by feature key
	ent, err := e.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: entitlementIdOrFeatureKey})
	if _, ok := lo.ErrorsAs[*entitlement.NotFoundError](err); ok {
		ent, err = e.entitlementRepo.GetActiveEntitlementOfSubjectAt(ctx, namespace, subjectKey, entitlementIdOrFeatureKey, clock.Now())
	}
	if err != nil {
		return nil, err
	}

	// check that we own the grant
	grants, err := e.grantRepo.ListGrants(ctx, grant.ListParams{
		Namespace:      ent.Namespace,
		OwnerID:        convert.ToPointer(grant.Owner(ent.ID)),
		IncludeDeleted: false,
		OrderBy:        grant.OrderByCreatedAt,
	})
	if err != nil {
		return nil, err
	}

	ents := make([]EntitlementGrant, 0, len(grants.Items))
	for _, grant := range grants.Items {
		g, err := GrantFromCreditGrant(grant)
		if err != nil {
			return nil, err
		}
		ents = append(ents, *g)
	}

	return ents, nil
}

type EntitlementGrant struct {
	grant.Grant

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

func GrantFromCreditGrant(grant grant.Grant) (*EntitlementGrant, error) {
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
