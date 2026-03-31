package invoicesync

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/secret"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
)

var _ secret.Service = (*noopSecretService)(nil)

type noopSecretService struct{}

func (n noopSecretService) CreateAppSecret(ctx context.Context, input secretentity.CreateAppSecretInput) (secretentity.SecretID, error) {
	return secretentity.SecretID{}, nil
}

func (n noopSecretService) UpdateAppSecret(ctx context.Context, input secretentity.UpdateAppSecretInput) (secretentity.SecretID, error) {
	return secretentity.SecretID{}, nil
}

func (n noopSecretService) GetAppSecret(ctx context.Context, input secretentity.GetAppSecretInput) (secretentity.Secret, error) {
	return secretentity.Secret{Value: "sk_test_123"}, nil
}

func (n noopSecretService) DeleteAppSecret(ctx context.Context, input secretentity.DeleteAppSecretInput) error {
	return nil
}
