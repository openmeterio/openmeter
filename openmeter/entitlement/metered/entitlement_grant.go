package meteredentitlement

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type ListEntitlementGrantsOrderBy grant.OrderBy

type ListEntitlementGrantsParams struct {
	CustomerID                string
	EntitlementIDOrFeatureKey string
	OrderBy                   grant.OrderBy
	Order                     sortx.Order
	Page                      pagination.Page
}

func (p ListEntitlementGrantsParams) Validate() error {
	if err := p.Page.Validate(); err != nil {
		return err
	}

	if p.CustomerID == "" {
		return fmt.Errorf("customerID is required")
	}

	if p.EntitlementIDOrFeatureKey == "" {
		return fmt.Errorf("entitlementIDOrFeatureKey is required")
	}

	return nil
}

// CreateGrant creates a grant for a given entitlement
//
// You can issue grants for inactive entitlements by passing the entitlement ID
func (e *connector) CreateGrant(ctx context.Context, namespace string, customerID string, entitlementIdOrFeatureKey string, inputGrant CreateEntitlementGrantInputs) (EntitlementGrant, error) {
	// First we attempt to find the entitlement by ID, then by featureKey
	ent, err := e.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: entitlementIdOrFeatureKey})
	if _, ok := lo.ErrorsAs[*entitlement.NotFoundError](err); ok {
		ent, err = e.entitlementRepo.GetActiveEntitlementOfCustomerAt(ctx, namespace, customerID, entitlementIdOrFeatureKey, clock.Now())
	}
	if err != nil {
		return EntitlementGrant{}, err
	}
	metered, err := ParseFromGenericEntitlement(ent)
	if err != nil {
		return EntitlementGrant{}, err
	}

	if err := e.hooks.PreUpdate(ctx, metered); err != nil {
		return EntitlementGrant{}, err
	}

	g, err := e.grantConnector.CreateGrant(ctx, models.NamespacedID{
		Namespace: ent.Namespace,
		ID:        ent.ID,
	}, credit.CreateGrantInput{
		Amount:           inputGrant.Amount,
		Priority:         inputGrant.Priority,
		EffectiveAt:      inputGrant.EffectiveAt,
		Expiration:       inputGrant.Expiration,
		ResetMaxRollover: inputGrant.ResetMaxRollover,
		ResetMinRollover: inputGrant.ResetMinRollover,
		Recurrence:       inputGrant.Recurrence,
		Annotations:      inputGrant.Annotations,
		Metadata:         inputGrant.Metadata,
	})
	if err != nil {
		if _, ok := lo.ErrorsAs[*grant.OwnerNotFoundError](err); ok {
			return EntitlementGrant{}, &entitlement.NotFoundError{EntitlementID: models.NamespacedID{Namespace: namespace, ID: ent.ID}}
		}

		return EntitlementGrant{}, err
	}

	eg, err := GrantFromCreditGrant(*g, clock.Now())
	return *eg, err
}

// ListEntitlementGrants lists all grants for a given entitlement
func (e *connector) ListEntitlementGrants(ctx context.Context, namespace string, params ListEntitlementGrantsParams) (pagination.Result[EntitlementGrant], error) {
	var def pagination.Result[EntitlementGrant]

	if err := params.Validate(); err != nil {
		return def, err
	}

	// Find the matching entitlement, first by ID, then by feature key
	ent, err := e.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: params.EntitlementIDOrFeatureKey})
	if _, ok := lo.ErrorsAs[*entitlement.NotFoundError](err); ok {
		ent, err = e.entitlementRepo.GetActiveEntitlementOfCustomerAt(ctx, namespace, params.CustomerID, params.EntitlementIDOrFeatureKey, clock.Now())
	}
	if err != nil {
		return def, err
	}

	grants, err := e.grantRepo.ListGrants(ctx, grant.ListParams{
		Namespace:      ent.Namespace,
		OwnerID:        convert.ToPointer(ent.ID),
		IncludeDeleted: false,
		OrderBy:        params.OrderBy,
		Order:          params.Order,
		Page:           params.Page,
	})
	if err != nil {
		return def, err
	}

	return pagination.MapResultErr(grants, func(grant grant.Grant) (EntitlementGrant, error) {
		g, err := GrantFromCreditGrant(grant, clock.Now())
		if err != nil {
			return EntitlementGrant{}, err
		}
		return *g, nil
	})
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

func GrantFromCreditGrant(grant grant.Grant, now time.Time) (*EntitlementGrant, error) {
	g := &EntitlementGrant{}
	if grant.Recurrence != nil {
		next, err := grant.Recurrence.NextAfter(now, timeutil.Inclusive)
		if err != nil {
			return nil, err
		}
		g.NextRecurrence = &next
	}
	g.Grant = grant
	g.EntitlementID = grant.OwnerID
	g.MaxRolloverAmount = grant.ResetMaxRollover
	g.MinRolloverAmount = grant.ResetMinRollover
	return g, nil
}

type CreateEntitlementGrantInputs = entitlement.CreateEntitlementGrantInputs
