package service

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/clock"
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

func (c *service) DeleteCustomerEntitlement(ctx context.Context, input entitlement.DeleteCustomerEntitlementInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	cus, err := c.getActiveCustomer(ctx, input.CustomerID)
	if err != nil {
		return err
	}

	entitlementID := models.NamespacedID{Namespace: cus.Namespace, ID: input.EntitlementID}

	ent, err := c.entitlementRepo.GetEntitlement(ctx, entitlementID)
	if err != nil {
		return err
	}

	// The entitlement is addressed through the customer, so one owned by another
	// customer must not be revealed.
	if ent.CustomerID != cus.ID {
		return &entitlement.NotFoundError{EntitlementID: entitlementID}
	}

	return c.DeleteEntitlement(ctx, cus.Namespace, ent.ID, clock.Now())
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
