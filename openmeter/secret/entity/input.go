package secretentity

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/app"
)

type CreateAppSecretInput struct {
	AppID app.AppID
	Key   string
	Value string
}

func (i CreateAppSecretInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return &ValidationError{
			Err: errors.New("app id is invalid"),
		}
	}

	if i.Key == "" {
		return &ValidationError{
			Err: errors.New("key is required"),
		}
	}

	if i.Value == "" {
		return &ValidationError{
			Err: errors.New("value is required"),
		}
	}

	return nil
}

type UpdateAppSecretInput struct {
	AppID    app.AppID
	SecretID SecretID
	Key      string
	Value    string
}

func (i UpdateAppSecretInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return err
	}

	if err := i.SecretID.Validate(); err != nil {
		return err
	}

	if i.Key == "" {
		return &ValidationError{
			Err: errors.New("key is required"),
		}
	}

	if i.Value == "" {
		return &ValidationError{
			Err: errors.New("value is required"),
		}
	}

	return nil
}

type GetAppSecretInput = SecretID

type DeleteAppSecretInput = SecretID
