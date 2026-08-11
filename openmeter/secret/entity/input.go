package secretentity

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CreateAppSecretInput struct {
	AppID app.AppID
	Key   string
	Value string
}

func (i CreateAppSecretInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return models.NewGenericValidationError(
			errors.New("app id is invalid"),
		)
	}

	if i.Key == "" {
		return models.NewGenericValidationError(
			errors.New("key is required"),
		)
	}

	if i.Value == "" {
		return models.NewGenericValidationError(
			errors.New("value is required"),
		)
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
	var errs []error

	if err := i.AppID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("appID: %w", err))
	}

	if err := i.SecretID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("secretID: %w", err))
	}

	if i.AppID != i.SecretID.AppID {
		errs = append(errs, errors.New("appID must match secretID appID"))
	}

	if i.Key == "" {
		errs = append(errs, errors.New("key is required"))
	}

	if i.Key != i.SecretID.Key {
		errs = append(errs, errors.New("key must match secretID key"))
	}

	if i.Value == "" {
		errs = append(errs, errors.New("value is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type GetAppSecretInput = SecretID

type DeleteAppSecretInput = SecretID
