package secretentity

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

var _ models.GenericError = (*SecretNotFoundError)(nil)

func NewSecretNotFoundError(id SecretID) *SecretNotFoundError {
	return &SecretNotFoundError{
		err: models.NewGenericNotFoundError(
			fmt.Errorf("app with id %s not found in %s namespace", id.ID, id.Namespace),
		),
	}
}

type SecretNotFoundError struct {
	err error
}

func (e *SecretNotFoundError) Error() string {
	return e.err.Error()
}

func (e *SecretNotFoundError) Unwrap() error {
	return e.err
}

// IsSecretNotFoundError returns true if the error is a SecretNotFoundError.
func IsSecretNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var e *SecretNotFoundError

	return errors.As(err, &e)
}
