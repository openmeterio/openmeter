package appstripe

import (
	"fmt"

	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
)

var _ error = (*AppNotFoundError)(nil)

type AppNotFoundError struct {
	appentitybase.AppID
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
