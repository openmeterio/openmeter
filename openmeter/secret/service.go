package secret

import (
	"context"

	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
)

type Service interface {
	SecretService
}

type SecretService interface {
	CreateAppSecret(ctx context.Context, input secretentity.CreateAppSecretInput) (secretentity.SecretID, error)
	GetAppSecret(ctx context.Context, input secretentity.GetAppSecretInput) (secretentity.Secret, error)
	DeleteAppSecret(ctx context.Context, input secretentity.DeleteAppSecretInput) error
}
