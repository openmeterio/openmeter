package customer

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ customer.RequestValidator = (*Validator)(nil)

func NewValidator(customerService customer.Service, entitlementRepo entitlement.EntitlementRepo) (*Validator, error) {
	if customerService == nil {
		return nil, fmt.Errorf("customer service is required")
	}
	if entitlementRepo == nil {
		return nil, fmt.Errorf("entitlement repository is required")
	}

	return &Validator{
		customerService: customerService,
		entitlementRepo: entitlementRepo,
	}, nil
}

type Validator struct {
	customer.NoopRequestValidator
	customerService customer.Service
	entitlementRepo entitlement.EntitlementRepo
}

func (v *Validator) ValidateDeleteCustomer(ctx context.Context, input customer.DeleteCustomerInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	// Get the customer first to check their usage attribution subjects
	cust, err := v.customerService.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerID: lo.ToPtr(input),
	})
	if err != nil {
		return err
	}

	// Check for active entitlements for each subject
	for _, subject := range cust.UsageAttribution.SubjectKeys {
		entitlements, err := v.entitlementRepo.GetActiveEntitlementsOfSubject(ctx, input.Namespace, subject, clock.Now())
		if err != nil {
			return fmt.Errorf("failed to list customer entitlements: %w", err)
		}

		if len(entitlements) > 0 {
			return models.NewGenericConflictError(fmt.Errorf("customer %s still has active entitlements, please remove them before deleting the customer", input.ID))
		}
	}

	return nil
}
