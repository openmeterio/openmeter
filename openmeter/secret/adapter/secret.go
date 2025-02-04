package adapter

import (
	"context"

	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
)

// CreateAppSecret creates a new secret for an app.
// In this plaintext implementation the ID is the same as the value
// In the real implementation, this method would create a secret in a secret store.
func (a adapter) CreateAppSecret(ctx context.Context, input secretentity.CreateAppSecretInput) (secretentity.SecretID, error) {
	return secretentity.NewSecretID(input.AppID, input.Value, input.Key), nil
}

// UpdateAppSecret updates the secret for an app.
// In this plaintext implementation the ID is the same as the value.
// In the real implementation, this method would create a secret in a secret store.
func (a adapter) UpdateAppSecret(ctx context.Context, input secretentity.UpdateAppSecretInput) (secretentity.SecretID, error) {
	return secretentity.NewSecretID(input.AppID, input.Value, input.Key), nil
}

// GetAppSecret retrieves a secret for an app.
// In this plaintext implementation the ID is the same as the value.
// In the real implementation, this method would retrieve a secret from a secret store.
func (a adapter) GetAppSecret(ctx context.Context, input secretentity.GetAppSecretInput) (secretentity.Secret, error) {
	value := input.ID

	return secretentity.Secret{
		SecretID: secretentity.NewSecretID(input.AppID, value, input.Key),
		Value:    value,
	}, nil
}

// DeleteAppSecret deletes a secret for an app.
// In the real implementation, this method would delete a secret from a secret store.
func (a adapter) DeleteAppSecret(ctx context.Context, input secretentity.DeleteAppSecretInput) error {
	return nil
}
