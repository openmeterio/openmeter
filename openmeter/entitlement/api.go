package entitlement

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
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

// CustomerEntitlementAPIService is the API-facing facade for customer-scoped
// entitlement operations. Every operation resolves the customer, rejects deleted
// customers and reports an entitlement owned by another customer as not found.
type CustomerEntitlementAPIService interface {
	CreateCustomerEntitlement(ctx context.Context, input CreateCustomerEntitlementInput) (*Entitlement, error)
	GetCustomerEntitlementHistory(ctx context.Context, input GetCustomerEntitlementHistoryInput) (CustomerEntitlementHistory, error)
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

// BalanceHistoryWindow is the usage of a metered entitlement in a single window
// together with the balance the window started with.
type BalanceHistoryWindow struct {
	From           time.Time
	To             time.Time
	UsageInPeriod  float64
	BalanceAtStart float64
	OverageAtStart float64
}

type CustomerEntitlementHistory struct {
	Windows  []BalanceHistoryWindow
	Burndown engine.GrantBurnDownHistory
}

// GetCustomerEntitlementHistoryInput queries the history of a metered entitlement.
// From defaults to the last reset and To to the current time; TimeZone defaults to UTC.
type GetCustomerEntitlementHistoryInput struct {
	CustomerID    customer.CustomerID
	EntitlementID string
	From          *time.Time
	To            *time.Time
	WindowSize    meter.WindowSize
	TimeZone      *time.Location
}

func (i GetCustomerEntitlementHistoryInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer ID: %w", err))
	}

	if i.EntitlementID == "" {
		errs = append(errs, errors.New("entitlement ID is required"))
	}

	if i.WindowSize == "" {
		errs = append(errs, errors.New("window size is required"))
	}

	if i.From != nil && i.To != nil && !i.From.Before(*i.To) {
		errs = append(errs, errors.New("from must be before to"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
