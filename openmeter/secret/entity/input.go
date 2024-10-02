package secretentity

import "errors"

type CreateAppSecretInput struct {
	Namespace string
	Key       string
	Value     string
}

func (i CreateAppSecretInput) Validate() error {
	if i.Namespace == "" {
		return ValidationError{
			Err: errors.New("namespace is required"),
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
