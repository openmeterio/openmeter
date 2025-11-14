package customer

import (
	"errors"
	"fmt"
	"strings"

	"github.com/openmeterio/openmeter/pkg/models"
)

// UpdateAfterDeleteError represents an error when an update is attempted after a delete
var _ error = (*UpdateAfterDeleteError)(nil)

func NewUpdateAfterDeleteError() *UpdateAfterDeleteError {
	return &UpdateAfterDeleteError{
		err: models.NewGenericConflictError(fmt.Errorf("update after delete")),
	}
}

var _ models.GenericError = &UpdateAfterDeleteError{}

type UpdateAfterDeleteError struct {
	err error
}

func (e UpdateAfterDeleteError) Error() string {
	return e.err.Error()
}

func (e UpdateAfterDeleteError) Unwrap() error {
	return e.err
}

// NewKeyConflictError creates a new KeyConflictError
func NewKeyConflictError(namespace, key string) *KeyConflictError {
	return &KeyConflictError{
		err: models.NewGenericConflictError(fmt.Errorf("key \"%s\" is already used by an another customer in the namespace %s", key, namespace)),
	}
}

var _ models.GenericError = &KeyConflictError{}

// KeyConflictError represents an error when a subject key is already associated with a customer
type KeyConflictError struct {
	err error
}

func (e KeyConflictError) Error() string {
	return e.err.Error()
}

func (e KeyConflictError) Unwrap() error {
	return e.err
}

func IsKeyConflictError(err error) bool {
	if err == nil {
		return false
	}

	var e *KeyConflictError

	return errors.As(err, &e)
}

// NewSubjectKeyConflictError creates a new SubjectKeyConflictError
func NewSubjectKeyConflictError(namespace string, subjectKeys []string) *SubjectKeyConflictError {
	return &SubjectKeyConflictError{
		err: models.NewGenericConflictError(fmt.Errorf("one or multiple subject keys of [%s] are already associated with an different customer in the namespace %s", strings.Join(subjectKeys, ", "), namespace)),
	}
}

var _ models.GenericError = &SubjectKeyConflictError{}

// SubjectKeyConflictError represents an error when a subject key is already associated with a customer
type SubjectKeyConflictError struct {
	err error
}

func (e SubjectKeyConflictError) Error() string {
	return e.err.Error()
}

func (e SubjectKeyConflictError) Unwrap() error {
	return e.err
}

func IsSubjectKeyConflictError(err error) bool {
	if err == nil {
		return false
	}

	var e *SubjectKeyConflictError

	return errors.As(err, &e)
}

const ErrCodeDeletingCustomerWithActiveSubscriptions models.ErrorCode = "deleting_customer_with_active_subscriptions"

var ErrDeletingCustomerWithActiveSubscriptions = models.NewValidationIssue(
	ErrCodeDeletingCustomerWithActiveSubscriptions,
	"cannot delete customer with active subscriptions",
)

func NewErrDeletingCustomerWithActiveSubscriptions(subscriptionIDs []string) error {
	return ErrDeletingCustomerWithActiveSubscriptions.WithAttr("subscriptions", subscriptionIDs)
}
