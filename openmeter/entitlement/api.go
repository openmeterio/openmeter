package entitlement

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
)

// CustomerEntitlementAccessAPIService is the API-facing facade for customer-scoped
// entitlement access. It resolves the customer, rejects deleted customers, and
// hides inactive entitlements so handlers only map the result.
type CustomerEntitlementAccessAPIService interface {
	GetCustomerEntitlementAccess(ctx context.Context, input GetCustomerEntitlementAccessInput) (CustomerEntitlementAccess, error)
	ListCustomerEntitlementAccess(ctx context.Context, input ListCustomerEntitlementAccessInput) ([]CustomerEntitlementAccess, error)
}

type CustomerEntitlementAccess struct {
	FeatureKey string
	Value      EntitlementValue
}

// GetCustomerEntitlementAccessInput addresses the entitlement by exactly one of
// FeatureKey or EntitlementID.
type GetCustomerEntitlementAccessInput struct {
	CustomerID    customer.CustomerID
	FeatureKey    string
	EntitlementID string
	At            time.Time
}

func (i GetCustomerEntitlementAccessInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer ID: %w", err))
	}

	if (i.FeatureKey == "") == (i.EntitlementID == "") {
		errs = append(errs, errors.New("exactly one of feature key or entitlement ID is required"))
	}

	if i.At.IsZero() {
		errs = append(errs, errors.New("at is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type ListCustomerEntitlementAccessInput struct {
	CustomerID customer.CustomerID
}

func (i ListCustomerEntitlementAccessInput) Validate() error {
	if err := i.CustomerID.Validate(); err != nil {
		return models.NewNillableGenericValidationError(fmt.Errorf("customer ID: %w", err))
	}

	return nil
}
