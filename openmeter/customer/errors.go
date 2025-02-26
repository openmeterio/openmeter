package customer

import (
	"errors"
	"fmt"
	"strings"

	"github.com/openmeterio/openmeter/pkg/models"
)

func NewNotFoundError(id CustomerID) *NotFoundError {
	return &NotFoundError{
		err: models.NewGenericNotFoundError(
			fmt.Errorf("customer with id %s not found in %s namespace", id.ID, id.Namespace),
		),
	}
}

// NotFoundError represents an error when a resource is not found
var _ models.GenericError = &NotFoundError{}

type NotFoundError struct {
	err error
}

func (e NotFoundError) Error() string {
	return e.err.Error()
}

func (e NotFoundError) Unwrap() error {
	return e.err
}

func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var e *NotFoundError

	return errors.As(err, &e)
}

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

// KeyConflictError represents an error when a subject key is already associated with a customer
type KeyConflictError struct {
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
}

func (e KeyConflictError) Error() string {
	return fmt.Sprintf("key \"%s\" is already used by an another customer in the namespace %s", e.Key, e.Namespace)
}

// SubjectKeyConflictError represents an error when a subject key is already associated with a customer
type SubjectKeyConflictError struct {
	Namespace   string   `json:"namespace"`
	SubjectKeys []string `json:"subjectKeys"`
}

func (e SubjectKeyConflictError) Error() string {
	return fmt.Sprintf("one or multiple subject keys of [%s] are already associated with an different customer in the namespace %s", strings.Join(e.SubjectKeys, ", "), e.Namespace)
}
