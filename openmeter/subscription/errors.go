package subscription

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

//
// Business Errors
// (TODO(galexi): probably should not use ValidationIssue for these)
//

const ErrCodeSubscriptionBillingPeriodQueriedBeforeSubscriptionStart models.ErrorCode = "subscription_billing_period_queried_before_subscription_start"

var ErrSubscriptionBillingPeriodQueriedBeforeSubscriptionStart = models.NewValidationIssue(
	ErrCodeSubscriptionBillingPeriodQueriedBeforeSubscriptionStart,
	"billing period queried before subscription start",
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

func NewErrSubscriptionBillingPeriodQueriedBeforeSubscriptionStart(queriedAt, subscriptionStart time.Time) error {
	return ErrSubscriptionBillingPeriodQueriedBeforeSubscriptionStart.WithAttr("queried_at", queriedAt).WithAttr("subscription_start", subscriptionStart)
}

func IsErrSubscriptionBillingPeriodQueriedBeforeSubscriptionStart(err error) bool {
	return IsValidationIssueWithCode(err, ErrCodeSubscriptionBillingPeriodQueriedBeforeSubscriptionStart)
}

const ErrCodeOnlySingleSubscriptionAllowed models.ErrorCode = "only_single_subscription_allowed_per_customer_at_a_time"

var ErrOnlySingleSubscriptionAllowed = models.NewValidationIssue(
	ErrCodeOnlySingleSubscriptionAllowed,
	"only single subscription is allowed per customer at a time",
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusConflict),
)

var ErrRestoreSubscriptionNotAllowedForMultiSubscription = models.NewGenericForbiddenError(errors.New("restore subscription is not allowed for multi-subscription"))

const ErrCodeOnlySingleSubscriptionItemAllowedAtATime models.ErrorCode = "only_single_subscription_item_allowed_at_a_time"

var ErrOnlySingleSubscriptionItemAllowedAtATime = models.NewValidationIssue(
	ErrCodeOnlySingleSubscriptionItemAllowedAtATime,
	"for any given feature, only one subscription item with entitlements or billable prices can exist at a time",
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusConflict),
)

// TODO(galexi): "ValidationIssue" is not the right concept here. We should have a different kind of error with all this capability. It's used here as a hack to localize things for the time being.

func IsValidationIssueWithCode(err error, code models.ErrorCode) bool {
	issues, err := models.AsValidationIssues(err)
	if err != nil {
		return false
	}

	if len(issues) != 1 {
		return false
	}

	return issues[0].Code() == code
}

func IsValidationIssueWithBoolAttr(err error, attrName string) bool {
	issues, err := models.AsValidationIssues(err)
	if err != nil {
		return false
	}

	if len(issues) != 1 {
		return false
	}

	return issues[0].Attributes()[attrName] == true
}

//
// Validation Issues
//

// Subscription

const ErrCodeSubscriptionBillingAnchorIsRequired models.ErrorCode = "subscription_billing_anchor_is_required"

var ErrSubscriptionBillingAnchorIsRequired = models.NewValidationIssue(
	ErrCodeSubscriptionBillingAnchorIsRequired,
	"billing anchor is required",
	models.WithFieldString("billingAnchor"),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

// Phase

const ErrCodeSubscriptionPhaseStartAfterIsNegative models.ErrorCode = "subscription_phase_start_after_is_negative"

var ErrSubscriptionPhaseStartAfterIsNegative = models.NewValidationIssue(
	ErrCodeSubscriptionPhaseStartAfterIsNegative,
	"subscription phase start after cannot be negative",
	models.WithFieldString("startAfter"),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeSubscriptionPhaseHasNoItems models.ErrorCode = "subscription_phase_has_no_items"

var ErrSubscriptionPhaseHasNoItems = models.NewValidationIssue(
	ErrCodeSubscriptionPhaseHasNoItems,
	"subscription phase must have at least one item",
	models.WithFieldString("items"),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeSubscriptionPhaseItemHistoryKeyMismatch models.ErrorCode = "subscription_phase_item_history_key_mismatch"

var ErrSubscriptionPhaseItemHistoryKeyMismatch = models.NewValidationIssue(
	ErrCodeSubscriptionPhaseItemHistoryKeyMismatch,
	"subscription phase item history key mismatch",
	models.WithFieldString("itemsByKey"),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeSubscriptionPhaseItemKeyMismatchWithPhaseKey models.ErrorCode = "subscription_phase_item_key_mismatch_with_phase_key"

var ErrSubscriptionPhaseItemKeyMismatchWithPhaseKey = models.NewValidationIssue(
	ErrCodeSubscriptionPhaseItemKeyMismatchWithPhaseKey,
	"subscription phase item key mismatch with phase key",
	models.WithFieldString("itemKey"),
	models.WithFieldString("phaseKey"),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

// Item

const ErrCodeSubscriptionItemBillingOverrideIsOnlyAllowedForBillableItems models.ErrorCode = "subscription_item_billing_override_is_only_allowed_for_billable_items"

var ErrSubscriptionItemBillingOverrideIsOnlyAllowedForBillableItems = models.NewValidationIssue(
	ErrCodeSubscriptionItemBillingOverrideIsOnlyAllowedForBillableItems,
	"billing override is only allowed for billable items",
	models.WithFieldString("billingBehaviorOverride"),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeSubscriptionItemActiveFromOverrideRelativeToPhaseStartIsNegative models.ErrorCode = "subscription_item_active_from_override_relative_to_phase_start_is_negative"

var ErrSubscriptionItemActiveFromOverrideRelativeToPhaseStartIsNegative = models.NewValidationIssue(
	ErrCodeSubscriptionItemActiveFromOverrideRelativeToPhaseStartIsNegative,
	"active from override relative to phase start cannot be negative",
	models.WithFieldString("activeFromOverrideRelativeToPhaseStart"),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeSubscriptionItemActiveToOverrideRelativeToPhaseStartIsNegative models.ErrorCode = "subscription_item_active_to_override_relative_to_phase_start_is_negative"

var ErrSubscriptionItemActiveToOverrideRelativeToPhaseStartIsNegative = models.NewValidationIssue(
	ErrCodeSubscriptionItemActiveToOverrideRelativeToPhaseStartIsNegative,
	"active to override relative to phase start cannot be negative",
	models.WithFieldString("activeToOverrideRelativeToPhaseStart"),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeSubscriptionItemHistoryOverlap models.ErrorCode = "subscription_item_history_overlap"

var ErrSubscriptionItemHistoryOverlap = models.NewValidationIssue(
	ErrCodeSubscriptionItemHistoryOverlap,
	"subscription item history overlap",
	models.WithFieldString("itemsByKey"),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

//
// NotFound errors
//

// NewSubscriptionNotFoundError returns a new SubscriptionNotFoundError.
func NewSubscriptionNotFoundError(id string) error {
	return &SubscriptionNotFoundError{
		err: models.NewGenericNotFoundError(
			fmt.Errorf("subscription %s not found", id),
		),
	}
}

var _ models.GenericError = &SubscriptionNotFoundError{}

// SubscriptionNotFoundError is returned when a meter is not found.
type SubscriptionNotFoundError struct {
	err error
}

// Error returns the error message.
func (e *SubscriptionNotFoundError) Error() string {
	return e.err.Error()
}

// Unwrap returns the wrapped error.
func (e *SubscriptionNotFoundError) Unwrap() error {
	return e.err
}

// IsSubscriptionNotFoundError returns true if the error is a SubscriptionNotFoundError.
func IsSubscriptionNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var e *SubscriptionNotFoundError

	return errors.As(err, &e)
}

// NewPhaseNotFoundError returns a new PhaseNotFoundError.
func NewPhaseNotFoundError(phaseId string) error {
	return &PhaseNotFoundError{
		err: models.NewGenericNotFoundError(
			fmt.Errorf("subscription phase %s not found", phaseId),
		),
	}
}

var _ models.GenericError = &PhaseNotFoundError{}

// PhaseNotFoundError is returned when a meter is not found.
type PhaseNotFoundError struct {
	err error
}

// Error returns the error message.
func (e *PhaseNotFoundError) Error() string {
	return e.err.Error()
}

// Unwrap returns the wrapped error.
func (e *PhaseNotFoundError) Unwrap() error {
	return e.err
}

// IsPhaseNotFoundError returns true if the error is a PhaseNotFoundError.
func IsPhaseNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var e *PhaseNotFoundError

	return errors.As(err, &e)
}

// NewItemNotFoundError returns a new ItemNotFoundError.
func NewItemNotFoundError(itemId string) error {
	return &ItemNotFoundError{
		err: models.NewGenericNotFoundError(
			fmt.Errorf("subscription item %s not found", itemId),
		),
	}
}

var _ models.GenericError = &ItemNotFoundError{}

// ItemNotFoundError is returned when a meter is not found.
type ItemNotFoundError struct {
	err error
}

// Error returns the error message.
func (e *ItemNotFoundError) Error() string {
	return e.err.Error()
}

// Unwrap returns the wrapped error.
func (e *ItemNotFoundError) Unwrap() error {
	return e.err
}

// IsItemNotFoundError returns true if the error is a ItemNotFoundError.
func IsItemNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var e *ItemNotFoundError

	return errors.As(err, &e)
}
