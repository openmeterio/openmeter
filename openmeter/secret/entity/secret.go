package secretentity

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

// SecretID represents a secret identifier.
type SecretID struct {
	models.NamespacedID

	Key string
}

func (i SecretID) Validate() error {
	if err := i.NamespacedID.Validate(); err != nil {
		return ValidationError{
			Err: fmt.Errorf("secret %w", err),
		}
	}

	if i.Key == "" {
		return ValidationError{
			Err: errors.New("secret key is required"),
		}
	}

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
