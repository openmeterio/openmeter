package plan

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

var _ error = (*NotFoundError)(nil)

type NotFoundErrorParams struct {
	Namespace string
	ID        string
	Key       string
	Version   int
}

func NewNotFoundError(e NotFoundErrorParams) *NotFoundError {
	var m string

	if e.Namespace != "" {
		m += fmt.Sprintf(" namespace=%s", e.Namespace)
	}

	if e.ID != "" {
		m += fmt.Sprintf(" id=%s", e.ID)
	}

	if e.Key != "" {
		m += fmt.Sprintf(" key=%s", e.Key)
	}

	if e.Version != 0 {
		m += fmt.Sprintf(" version=%d", e.Version)
	}

	if len(m) > 0 {
		m = fmt.Sprintf("plan not found. [%s]", m[1:])
	} else {
		m = "plan not found"
	}

	return &NotFoundError{
		err: models.NewGenericNotFoundError(
			errors.New(m),
		),
	}
}

var _ models.GenericError = &NotFoundError{}

type NotFoundError struct {
	err error
}

func (e *NotFoundError) Error() string {
	return e.err.Error()
}

func (e *NotFoundError) Unwrap() error {
	return e.err
}

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var e *NotFoundError

	return errors.As(err, &e)
}
