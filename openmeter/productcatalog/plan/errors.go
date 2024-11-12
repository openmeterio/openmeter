package plan

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

var _ error = (*NotFoundError)(nil)

type NotFoundError struct {
	models.NamespacedModel
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("resource not found in %s namespace", e.Namespace)
}

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var e NotFoundError
	return errors.As(err, &e)
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
