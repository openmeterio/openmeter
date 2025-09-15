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

func NewValidator(entitlementRepo entitlement.EntitlementRepo) (*Validator, error) {
	if entitlementRepo == nil {
		return nil, fmt.Errorf("entitlement repository is required")
	}

	return &Validator{
		entitlementRepo: entitlementRepo,
	}, nil
}

type Validator struct {
	customer.NoopRequestValidator
	entitlementRepo entitlement.EntitlementRepo
}

func (v *Validator) ValidateDeleteCustomer(ctx context.Context, input customer.DeleteCustomerInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	now := clock.Now()

	// Check for active entitlements for each subject
	entitlements, err := v.entitlementRepo.ListEntitlements(ctx, entitlement.ListEntitlementsParams{
		CustomerIDs:         []string{input.ID},
		Namespaces:          []string{input.Namespace},
		ActiveAt:            lo.ToPtr(now),
		IncludeDeleted:      true,
		IncludeDeletedAfter: now,
	})
	if err != nil {
		return fmt.Errorf("failed to list customer entitlements: %w", err)
	}

	if len(entitlements.Items) > 0 {
		return models.NewGenericConflictError(fmt.Errorf("customer %s still has active entitlements, please remove them before deleting the customer", input.ID))
	}

	return nil
}
