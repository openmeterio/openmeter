package secretentity

import (
	"errors"
	"fmt"

	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/pkg/models"
)

// SecretID represents a secret identifier.
type SecretID struct {
	models.NamespacedID
	// AppID appentitybase.AppID
	// Key string
}

func NewSecretID(appID appentitybase.AppID, id string, key string) SecretID {
	return SecretID{
		NamespacedID: models.NamespacedID{
			Namespace: appID.Namespace,
			ID:        id,
		},
		// AppID: appID,
		// Key: key,
	}
}

func (i SecretID) Validate() error {
	if err := i.NamespacedID.Validate(); err != nil {
		return ValidationError{
			Err: fmt.Errorf("secret %w", err),
		}
	}

	// if err := i.AppID.Validate(); err != nil {
	// 	return ValidationError{
	// 		Err: fmt.Errorf("secret app id %w", err),
	// 	}
	// }

	// if i.Key == "" {
	// 	return ValidationError{
	// 		Err: errors.New("secret key is required"),
	// 	}
	// }

	return nil
}

// Secret represents a secret with a value.
type Secret struct {
	SecretID SecretID
	Value    string
}

func (s Secret) Validate() error {
	if err := s.SecretID.Validate(); err != nil {
		return ValidationError{
			Err: fmt.Errorf("secret %w", err),
		}
	}

	if s.Value == "" {
		return ValidationError{
			Err: errors.New("secret value is required"),
		}
	}

	return nil
}
