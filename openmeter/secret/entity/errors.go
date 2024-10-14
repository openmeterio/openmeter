package secretentity

import (
	"fmt"
)

var _ error = (*SecretNotFoundError)(nil)

type SecretNotFoundError struct {
	SecretID
}

func (e SecretNotFoundError) Error() string {
	return fmt.Sprintf("app with id %s not found in %s namespace", e.ID, e.ID)
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
