package secretservice

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/secret"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
)

var _ secret.SecretService = (*Service)(nil)

func (s *Service) CreateAppSecret(ctx context.Context, input secretentity.CreateAppSecretInput) (secretentity.SecretID, error) {
	if err := input.Validate(); err != nil {
		return secretentity.SecretID{}, secretentity.ValidationError{
			Err: fmt.Errorf("error create app secret: %w", err),
		}
	}

	return s.adapter.CreateAppSecret(ctx, input)
}

func (s *Service) GetAppSecret(ctx context.Context, input secretentity.GetAppSecretInput) (secretentity.Secret, error) {
	if err := input.Validate(); err != nil {
		return secretentity.Secret{}, secretentity.ValidationError{
			Err: fmt.Errorf("error get app secret: %w", err),
		}
	}

	return s.adapter.GetAppSecret(ctx, input)
}

func (s *Service) DeleteAppSecret(ctx context.Context, input secretentity.DeleteAppSecretInput) error {
	if err := input.Validate(); err != nil {
		return secretentity.ValidationError{
			Err: fmt.Errorf("error delete app secret: %w", err),
		}
	}

	return s.adapter.DeleteAppSecret(ctx, input)
}
