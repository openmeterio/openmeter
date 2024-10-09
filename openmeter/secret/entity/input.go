package secretentity

import (
	"errors"

	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
)

type CreateAppSecretInput struct {
	AppID appentitybase.AppID
	Key   string
	Value string
}

func (i CreateAppSecretInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return ValidationError{
			Err: errors.New("app id is invalid"),
		}
	}

	if i.Key == "" {
		return ValidationError{
			Err: errors.New("key is required"),
		}
	}

	if i.Value == "" {
		return ValidationError{
			Err: errors.New("value is required"),
		}
	}

	return nil
}

type GetAppSecretInput = SecretID

type DeleteAppSecretInput = SecretID
