package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (c *service) CreateCustomerEntitlement(ctx context.Context, input entitlement.CreateCustomerEntitlementInput) (*entitlement.Entitlement, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	cust, err := c.customerService.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerID: &input.CustomerID,
	})
	if err != nil {
		return nil, err
	}

	if cust.IsDeleted() {
		return nil, models.NewGenericPreConditionFailedError(
			fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cust.Namespace, cust.ID),
		)
	}

	createInput := input.Entitlement
	createInput.Namespace = cust.Namespace
	createInput.UsageAttribution = cust.GetUsageAttribution()

	return c.CreateEntitlement(ctx, createInput, input.Grants)
}
