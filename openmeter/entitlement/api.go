package entitlement

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/filter"
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

// CustomerEntitlementAPIService is the API-facing facade for customer-scoped
// entitlement operations. Every operation resolves the customer, rejects deleted
// customers and reports an entitlement owned by another customer as not found.
type CustomerEntitlementAPIService interface {
	CreateCustomerEntitlement(ctx context.Context, input CreateCustomerEntitlementInput) (*Entitlement, error)
	GetCustomerEntitlementHistory(ctx context.Context, input GetCustomerEntitlementHistoryInput) (CustomerEntitlementHistory, error)
	GetCustomerEntitlement(ctx context.Context, input GetCustomerEntitlementInput) (*Entitlement, error)
	ListCustomerEntitlements(ctx context.Context, input ListCustomerEntitlementsInput) (pagination.Result[Entitlement], error)
	ListCustomerEntitlementGrants(ctx context.Context, input ListCustomerEntitlementGrantsInput) (pagination.Result[grant.Grant], error)
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

	if i.From != nil {
		to := lo.FromPtrOr(i.To, clock.Now())

		if window := historyWindowDuration(i.WindowSize); window > 0 && to.Sub(*i.From) > maxHistoryWindows*window {
			errs = append(errs, fmt.Errorf("range must not span more than %d windows", maxHistoryWindows))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// FIXME: a flat window cap is a stopgap against expensive history queries; the
// limit should follow the actual cost of the calculation instead.
const maxHistoryWindows = 1000

func historyWindowDuration(size meter.WindowSize) time.Duration {
	switch size {
	case meter.WindowSizeHour:
		return time.Hour
	case meter.WindowSizeDay:
		return 24 * time.Hour
	default:
		return 0
	}
}

// GetCustomerEntitlementInput addresses an entitlement by ID within the customer
// referenced by ID. An entitlement that belongs to another customer is reported
// as not found.
type GetCustomerEntitlementInput struct {
	CustomerID    customer.CustomerID
	EntitlementID string
}

func (i GetCustomerEntitlementInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer ID: %w", err))
	}

	if i.EntitlementID == "" {
		errs = append(errs, errors.New("entitlement ID is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// ListCustomerEntitlementsInput lists the entitlements of the customer referenced
// by ID that are active at the time of the call. An unset OrderBy sorts by
// creation time so pagination is stable.
type ListCustomerEntitlementsInput struct {
	CustomerID customer.CustomerID

	FeatureID  *filter.FilterULID
	FeatureKey *filter.FilterString
	Type       *filter.FilterString

	OrderBy ListEntitlementsOrderBy
	Order   sortx.Order
	Page    pagination.Page
}

func (i ListCustomerEntitlementsInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer ID: %w", err))
	}

	if i.FeatureID != nil {
		if err := i.FeatureID.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("feature ID filter: %w", err))
		}
	}

	if i.FeatureKey != nil {
		if err := i.FeatureKey.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("feature key filter: %w", err))
		}
	}

	if i.Type != nil {
		if err := i.Type.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("type filter: %w", err))
		}

		// The column is free text in the database, so an unknown type would silently
		// match nothing instead of being reported.
		values := append(lo.FromPtr(i.Type.In), lo.FromPtr(i.Type.Eq), lo.FromPtr(i.Type.Ne))
		for _, value := range lo.Compact(values) {
			if !slices.Contains(EntitlementType(value).Values(), EntitlementType(value)) {
				errs = append(errs, fmt.Errorf("invalid entitlement type: %s", value))
			}
		}
	}

	if i.OrderBy != "" && !slices.Contains(i.OrderBy.Values(), i.OrderBy) {
		errs = append(errs, fmt.Errorf("invalid order by: %s, supported: %s", i.OrderBy, strings.Join(i.OrderBy.StrValues(), ", ")))
	}

	if !i.Page.IsZero() {
		if err := i.Page.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("page: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
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
