package service

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (c *service) GetCustomerEntitlementAccess(ctx context.Context, input entitlement.GetCustomerEntitlementAccessInput) (entitlement.CustomerEntitlementAccess, error) {
	if err := input.Validate(); err != nil {
		return entitlement.CustomerEntitlementAccess{}, err
	}

	cus, err := c.getActiveCustomer(ctx, input.CustomerID)
	if err != nil {
		return entitlement.CustomerEntitlementAccess{}, err
	}

	ent, err := c.getCustomerEntitlement(ctx, cus, input)
	if err != nil {
		return entitlement.CustomerEntitlementAccess{}, err
	}

	if ent == nil {
		return entitlement.CustomerEntitlementAccess{
			FeatureKey: input.FeatureKey,
			Value:      &entitlement.NoAccessValue{},
		}, nil
	}

	value, err := c.getEntitlementValueAt(ctx, ent, input.At)
	if err != nil {
		return entitlement.CustomerEntitlementAccess{}, err
	}

	return entitlement.CustomerEntitlementAccess{
		FeatureKey: ent.FeatureKey,
		Value:      value,
	}, nil
}

// getCustomerEntitlement returns nil without an error when the feature key has no
// active entitlement, since missing access is a valid answer for a feature lookup
// but not for an entitlement ID.
func (c *service) getCustomerEntitlement(ctx context.Context, cus *customer.Customer, input entitlement.GetCustomerEntitlementAccessInput) (*entitlement.Entitlement, error) {
	if input.EntitlementID == "" {
		ent, err := c.entitlementRepo.GetActiveEntitlementOfCustomerAt(ctx, cus.Namespace, cus.ID, input.FeatureKey, input.At)
		if _, ok := lo.ErrorsAs[*entitlement.NotFoundError](err); ok {
			return nil, nil
		}

		return ent, err
	}

	ent, err := c.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: cus.Namespace, ID: input.EntitlementID})
	if err != nil {
		if _, ok := lo.ErrorsAs[*entitlement.NotFoundError](err); ok {
			return nil, models.NewGenericNotFoundError(err)
		}

		return nil, err
	}

	if ent.CustomerID != cus.ID {
		return nil, models.NewGenericNotFoundError(
			fmt.Errorf("entitlement not found %s for customer %s in namespace %s", input.EntitlementID, cus.ID, cus.Namespace),
		)
	}

	return ent, nil
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
