package billing

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	ErrDefaultProfileAlreadyExists = errors.New("default profile already exists")
	ErrProfileWithKeyAlreadyExists = errors.New("a profile with the specified key already exists")
	ErrProfileNotFound             = errors.New("profile not found")
	ErrProfileAlreadyDeleted       = errors.New("profile already deleted")
	ErrProfileUpdateAfterDelete    = errors.New("profile cannot be updated after deletion")
)

var _ error = (*NotFoundError)(nil)

type NotFoundError struct {
	models.NamespacedID
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("resource with id %s not found in %s namespace", e.ID, e.Namespace)
}

type genericError struct {
	Err error
}

var _ error = (*ValidationError)(nil)

type ValidationError genericError

func (e ValidationError) Error() string {
	return e.Err.Error()
}

func (e ValidationError) Unwrap() error {
	return e.Err
}

var _ error = (*UpdateAfterDeleteError)(nil)

type UpdateAfterDeleteError genericError

func (e UpdateAfterDeleteError) Error() string {
	return e.Err.Error()
}

func (e UpdateAfterDeleteError) Unwrap() error {
	return e.Err
}
