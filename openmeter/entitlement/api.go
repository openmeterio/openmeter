package entitlement

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
)

// CustomerEntitlementAPIService is the API-facing facade for customer-scoped entitlement operations.
type CustomerEntitlementAPIService interface {
	CreateCustomerEntitlement(ctx context.Context, input CreateCustomerEntitlementInput) (*Entitlement, error)
}

// CreateCustomerEntitlementInput creates an entitlement for the customer referenced by ID.
// The namespace and usage attribution of the entitlement are taken from the resolved
// customer, so callers leave them unset.
type CreateCustomerEntitlementInput struct {
	CustomerID  customer.CustomerID
	Entitlement CreateEntitlementInputs
	Grants      []CreateEntitlementGrantInputs
}

func (i CreateCustomerEntitlementInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer ID: %w", err))
	}

	if i.Entitlement.FeatureID == nil && i.Entitlement.FeatureKey == nil {
		errs = append(errs, errors.New("feature is required"))
	}

	if i.Entitlement.IssueAfterReset != nil && len(i.Grants) > 0 {
		errs = append(errs, errors.New("issue after reset and grants cannot be used together"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
