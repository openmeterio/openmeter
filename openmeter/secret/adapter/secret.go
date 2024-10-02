package adapter

import (
	"context"

	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (a adapter) CreateAppSecret(ctx context.Context, input secretentity.CreateAppSecretInput) (secretentity.SecretID, error) {
	// In the real implementation, this method would create a secret in a secret store.
	// In this example the ID is the same as the value.
	return secretentity.SecretID{
		NamespacedID: models.NamespacedID{
			Namespace: input.Namespace,
			ID:        input.Value,
		},
		Key: input.Key,
	}, nil
}

func (a adapter) GetAppSecret(ctx context.Context, input secretentity.GetAppSecretInput) (secretentity.Secret, error) {
	// In the real implementation, this method would retrieve a secret from a secret store.
	// In this example the ID is the same as the value.
	return secretentity.Secret{
		SecretID: secretentity.SecretID{
			NamespacedID: models.NamespacedID{
				Namespace: input.Namespace,
				ID:        input.ID,
			},
			Key: input.Key,
		},
		Value: input.ID,
	}, nil
}

func (a adapter) DeleteAppSecret(ctx context.Context, input secretentity.DeleteAppSecretInput) error {
	return nil
}
