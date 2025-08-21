package entitlement

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

type AlreadyExistsError struct {
	EntitlementID string
	FeatureID     string
	SubjectKey    string
}

func (e *AlreadyExistsError) Error() string {
	return fmt.Sprintf("entitlement with id %s already exists for feature %s and subject %s", e.EntitlementID, e.FeatureID, e.SubjectKey)
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

// SubjectCustomerConflictError indicates that a subject key resolves to multiple customers
// within the same namespace via usage attribution mapping.
// Uses the generic conflict error pattern for consistency with other domains.
type SubjectCustomerConflictError struct {
	err error
}

func (e *SubjectCustomerConflictError) Error() string {
	return e.err.Error()
}

func (e *SubjectCustomerConflictError) Unwrap() error {
	return e.err
}

// NewSubjectCustomerConflictError constructs a SubjectCustomerConflictError wrapped in a GenericConflictError.
func NewSubjectCustomerConflictError(namespace, subjectKey string) error {
	return &SubjectCustomerConflictError{
		err: models.NewGenericConflictError(fmt.Errorf("multiple customers reference subject key %s in namespace %s", subjectKey, namespace)),
	}
}

// IsSubjectCustomerConflictError checks whether the error chain contains a SubjectCustomerConflictError.
func IsSubjectCustomerConflictError(err error) bool {
	if err == nil {
		return false
	}
	var e *SubjectCustomerConflictError
	return errors.As(err, &e)
}
