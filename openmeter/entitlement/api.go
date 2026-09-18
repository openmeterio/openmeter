package entitlement

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
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

type GetCustomerEntitlementAccessInput struct {
	CustomerID customer.CustomerID
	FeatureKey string
}

func (i GetCustomerEntitlementAccessInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer ID: %w", err))
	}

	if i.FeatureKey == "" {
		errs = append(errs, errors.New("feature key is required"))
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

// CustomerEntitlementAPIService is the API-facing facade for customer-scoped entitlement operations.
type CustomerEntitlementAPIService interface {
	ListCustomerEntitlementGrants(ctx context.Context, input ListCustomerEntitlementGrantsInput) (pagination.Result[grant.Grant], error)
}

// ListCustomerEntitlementGrantsInput lists the grants of the entitlement referenced
// by ID within the customer referenced by ID. Grants only exist for metered
// entitlements, so the list of any other entitlement type is empty. Deleted grants
// are excluded unless IncludeDeleted is set; voided and expired grants are always
// part of the list as they remain part of the balance history.
type ListCustomerEntitlementGrantsInput struct {
	CustomerID    customer.CustomerID
	EntitlementID string

	IncludeDeleted bool

	OrderBy grant.OrderBy
	Order   sortx.Order
	Page    pagination.Page
}

func (i ListCustomerEntitlementGrantsInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer ID: %w", err))
	}

	if i.EntitlementID == "" {
		errs = append(errs, errors.New("entitlement ID is required"))
	}

	if i.OrderBy != "" && !slices.Contains(i.OrderBy.Values(), i.OrderBy) {
		errs = append(errs, fmt.Errorf("invalid order by: %s", i.OrderBy))
	}

	// Grants are only listed page by page; the limit/offset mode of the grant list is
	// not exposed here, so a page is always required.
	if i.Page.IsZero() {
		errs = append(errs, errors.New("page is required"))
	} else if err := i.Page.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("page: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
