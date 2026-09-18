package service

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func (c *service) GetCustomerEntitlementAccess(ctx context.Context, input entitlement.GetCustomerEntitlementAccessInput) (entitlement.CustomerEntitlementAccess, error) {
	if err := input.Validate(); err != nil {
		return entitlement.CustomerEntitlementAccess{}, err
	}

	cus, err := c.getActiveCustomer(ctx, input.CustomerID)
	if err != nil {
		return entitlement.CustomerEntitlementAccess{}, err
	}

	value, err := c.GetEntitlementValue(ctx, cus.Namespace, cus.ID, input.FeatureKey, clock.Now())
	if err != nil {
		if _, ok := lo.ErrorsAs[*entitlement.NotFoundError](err); ok {
			return entitlement.CustomerEntitlementAccess{
				FeatureKey: input.FeatureKey,
				Value:      &entitlement.NoAccessValue{},
			}, nil
		}

		return entitlement.CustomerEntitlementAccess{}, err
	}

	return entitlement.CustomerEntitlementAccess{
		FeatureKey: input.FeatureKey,
		Value:      value,
	}, nil
}

func (c *service) ListCustomerEntitlementAccess(ctx context.Context, input entitlement.ListCustomerEntitlementAccessInput) ([]entitlement.CustomerEntitlementAccess, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	cus, err := c.getActiveCustomer(ctx, input.CustomerID)
	if err != nil {
		return nil, err
	}

	access, err := c.GetAccess(ctx, cus.Namespace, cus.ID)
	if err != nil {
		return nil, err
	}

	items := make([]entitlement.CustomerEntitlementAccess, 0, len(access.Entitlements))
	for featureKey, ent := range access.Entitlements {
		if _, ok := ent.Value.(*entitlement.NoAccessValue); ok {
			continue
		}

		items = append(items, entitlement.CustomerEntitlementAccess{
			FeatureKey: featureKey,
			Value:      ent.Value,
		})
	}

	slices.SortFunc(items, func(a, b entitlement.CustomerEntitlementAccess) int {
		return strings.Compare(a.FeatureKey, b.FeatureKey)
	})

	return items, nil
}

func (c *service) ListCustomerEntitlementGrants(ctx context.Context, input entitlement.ListCustomerEntitlementGrantsInput) (pagination.Result[grant.Grant], error) {
	if err := input.Validate(); err != nil {
		return pagination.Result[grant.Grant]{}, err
	}

	ent, err := c.getCustomerEntitlement(ctx, input.CustomerID, input.EntitlementID)
	if err != nil {
		return pagination.Result[grant.Grant]{}, err
	}

	// The entitlement is already resolved, so the grant list is addressed by ID; the
	// list itself does not depend on the entitlement type and is empty for the
	// non-metered ones, as grants can only be issued for metered entitlements.
	grants, err := c.meteredEntitlementConnector.ListEntitlementGrants(ctx, ent.Namespace, meteredentitlement.ListEntitlementGrantsParams{
		CustomerID:                ent.CustomerID,
		EntitlementIDOrFeatureKey: ent.ID,
		IncludeDeleted:            input.IncludeDeleted,
		OrderBy:                   input.OrderBy,
		Order:                     input.Order,
		Page:                      input.Page,
	})
	if err != nil {
		return pagination.Result[grant.Grant]{}, err
	}

	return pagination.MapResult(grants, func(g meteredentitlement.EntitlementGrant) grant.Grant {
		return g.Grant
	}), nil
}

// getCustomerEntitlement resolves an entitlement addressed through the customer
// that owns it. The customer has to exist and not be deleted.
func (c *service) getCustomerEntitlement(ctx context.Context, customerID customer.CustomerID, entitlementID string) (*entitlement.Entitlement, error) {
	cus, err := c.getActiveCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}

	id := models.NamespacedID{Namespace: cus.Namespace, ID: entitlementID}

	ent, err := c.entitlementRepo.GetEntitlement(ctx, id)
	if err != nil {
		return nil, err
	}

	// The entitlement is addressed through the customer, so one owned by another
	// customer must not be revealed.
	if ent.CustomerID != cus.ID {
		return nil, &entitlement.NotFoundError{EntitlementID: id}
	}

	return ent, nil
}

func (c *service) getActiveCustomer(ctx context.Context, customerID customer.CustomerID) (*customer.Customer, error) {
	cus, err := c.customerService.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerID: &customerID,
	})
	if err != nil {
		return nil, err
	}

	if cus.IsDeleted() {
		return nil, models.NewGenericPreConditionFailedError(
			fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
		)
	}

	return cus, nil
}
