package entitlement

import (
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

type AlreadyExistsError struct {
	EntitlementID string
	FeatureID     string
	CustomerID    string
}

func (e *AlreadyExistsError) Error() string {
	return fmt.Sprintf("entitlement with id %s already exists for feature %s and customer %s", e.EntitlementID, e.FeatureID, e.CustomerID)
}

type AlreadyDeletedError struct {
	EntitlementID string
}

func (e *AlreadyDeletedError) Error() string {
	return fmt.Sprintf("entitlement with id %s already deleted", e.EntitlementID)
}

type NotFoundError struct {
	EntitlementID models.NamespacedID
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("entitlement not found %s in namespace %s", e.EntitlementID.ID, e.EntitlementID.Namespace)
}

type WrongTypeError struct {
	Expected EntitlementType
	Actual   EntitlementType
}

func (e *WrongTypeError) Error() string {
	return fmt.Sprintf("expected entitlement type %s but got %s", e.Expected, e.Actual)
}

type InvalidValueError struct {
	Message string
	Type    EntitlementType
}

func (e *InvalidValueError) Error() string {
	return fmt.Sprintf("invalid entitlement value for type %s: %s", e.Type, e.Message)
}

type InvalidFeatureError struct {
	FeatureID string
	Message   string
}

func (e *InvalidFeatureError) Error() string {
	return fmt.Sprintf("invalid feature %s: %s", e.FeatureID, e.Message)
}

type ForbiddenError struct {
	Message string
}

func (e *ForbiddenError) Error() string {
	return fmt.Sprintf("forbidden: %s", e.Message)
}

const ErrCodeEntitlementCreatePropertyMismatch models.ErrorCode = "entitlement_create_property_mismatch"

var ErrEntitlementCreatePropertyMismatch = models.NewValidationIssue(
	ErrCodeEntitlementCreatePropertyMismatch,
	"entitlement create property mismatch",
)

const ErrCodeEntitlementGrantsOnlySupportedForMeteredEntitlements models.ErrorCode = "entitlement_grants_only_supported_for_metered_entitlements"

var ErrEntitlementGrantsOnlySupportedForMeteredEntitlements = models.NewValidationIssue(
	ErrCodeEntitlementGrantsOnlySupportedForMeteredEntitlements,
	"grants are only supported for metered entitlements",
)
