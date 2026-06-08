package governance

import (
	"errors"
	"time"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
	pagination "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

// ReasonCode is the machine-readable reason a customer does not have access to a feature.
type ReasonCode string

const (
	ReasonCodeUnknown            ReasonCode = "unknown"
	ReasonCodeUsageLimitReached  ReasonCode = "usage_limit_reached"
	ReasonCodeFeatureUnavailable ReasonCode = "feature_unavailable"
	ReasonCodeFeatureNotFound    ReasonCode = "feature_not_found"
	ReasonCodeNoCreditAvailable  ReasonCode = "no_credit_available"
)

var AccessReasonUsageLimitReached = &AccessReason{
	Code:    ReasonCodeUsageLimitReached,
	Message: "usage limit for feature reached",
}

var AccessReasonFeatureUnavailable = &AccessReason{
	Code:    ReasonCodeFeatureUnavailable,
	Message: "feature is not available for customer",
}

var AccessReasonFeatureNotFound = &AccessReason{
	Code:    ReasonCodeFeatureNotFound,
	Message: "feature is not found",
}

// AccessReason explains why a feature is not accessible.
type AccessReason struct {
	Code    ReasonCode
	Message string
}

// FeatureAccess is the access status for a single feature.
type FeatureAccess struct {
	HasAccess bool
	// Reason is set when HasAccess is false.
	Reason *AccessReason
}

// CustomerAccess is the access evaluation for a single resolved customer.
type CustomerAccess struct {
	// Customer the matched identifiers resolved to.
	Customer customer.Customer
	// Matched lists the request identifiers (customer key or usage-attribution subject
	// key) that resolved to this customer.
	Matched []string
	// Features maps feature key to its access status.
	Features map[string]FeatureAccess
	// UpdatedAt is the time the access state was evaluated.
	UpdatedAt time.Time
}

// QueryErrorCode is the machine-readable code for a per-customer query error.
type QueryErrorCode string

const (
	QueryErrorUnknown          QueryErrorCode = "unknown"
	QueryErrorCustomerNotFound QueryErrorCode = "customer_not_found"
)

// QueryError is a partial error for a single input identifier.
type QueryError struct {
	// CustomerKey is the request identifier that produced this error.
	CustomerKey string
	Code        QueryErrorCode
	Message     string
}

var _ models.Validator = (*QueryAccessInput)(nil)

// QueryAccessInput is the input for evaluating governance access.
type QueryAccessInput struct {
	Namespace string
	// CustomerKeys are arbitrary identifiers — each a customer key or a usage-attribution
	// subject key. Identifiers that cannot be resolved are reported in QueryResult.Errors.
	CustomerKeys []string
	// FeatureKeys, when non-empty, restricts evaluation to those feature keys. When empty,
	// every non-archived feature in the namespace is evaluated.
	FeatureKeys []string
	// IncludeCredits requests credit-balance evaluation. Not yet implemented.
	IncludeCredits bool

	// Pagination over the resolved customers (sorted by CreatedAt, ID). At most one of
	// After/Before may be set.
	PageSize int
	After    *pagination.Cursor
	Before   *pagination.Cursor
}

func (i QueryAccessInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if len(i.CustomerKeys) == 0 {
		errs = append(errs, errors.New("at least one customer key is required"))
	}

	if i.PageSize < 1 {
		errs = append(errs, errors.New("page size must be positive"))
	}

	if i.After != nil && i.Before != nil {
		errs = append(errs, errors.New("after and before cursors are mutually exclusive"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// QueryResult is the paged result of a governance access query.
type QueryResult struct {
	// Customers are the access evaluations for the current page, ordered by (CreatedAt, ID).
	Customers []CustomerAccess
	// Errors are partial errors for unresolved input identifiers.
	Errors []QueryError

	// HasPrev/HasNext indicate adjacent pages relative to the current one.
	HasPrev bool
	HasNext bool
	// First/Last are the cursors of the first and last item on the current page.
	First *pagination.Cursor
	Last  *pagination.Cursor
}
