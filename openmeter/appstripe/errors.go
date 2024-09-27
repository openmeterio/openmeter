package appstripe

import (
	"fmt"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
)

var _ error = (*AppNotFoundError)(nil)

type AppNotFoundError struct {
	appentity.AppID
}

func (e AppNotFoundError) Error() string {
	return fmt.Sprintf("app with id %s not found in %s namespace", e.ID, e.Namespace)
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
