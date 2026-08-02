package secretservice_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/secret"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
	secretservice "github.com/openmeterio/openmeter/openmeter/secret/service"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ secret.Adapter = (*countingAdapter)(nil)

type countingAdapter struct {
	getCalls atomic.Int64
	getErr   error
	value    string
}

func (a *countingAdapter) CreateAppSecret(_ context.Context, input secretentity.CreateAppSecretInput) (secretentity.SecretID, error) {
	return secretentity.NewSecretID(input.AppID, input.Value, input.Key), nil
}

func (a *countingAdapter) UpdateAppSecret(_ context.Context, input secretentity.UpdateAppSecretInput) (secretentity.SecretID, error) {
	return secretentity.NewSecretID(input.AppID, input.Value, input.Key), nil
}

func (a *countingAdapter) GetAppSecret(_ context.Context, input secretentity.GetAppSecretInput) (secretentity.Secret, error) {
	a.getCalls.Add(1)

	if a.getErr != nil {
		return secretentity.Secret{}, a.getErr
	}

	return secretentity.Secret{SecretID: input, Value: a.value}, nil
}

func (a *countingAdapter) DeleteAppSecret(_ context.Context, _ secretentity.DeleteAppSecretInput) error {
	return nil
}

func newTestSecretID() secretentity.SecretID {
	appID := app.AppID{Namespace: "test-namespace", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}

	return secretentity.NewSecretID(appID, "stripe_webhook_secret", "webhook_secret")
}

func TestGetAppSecretCaching(t *testing.T) {
	t.Run("repeated reads hit the secret store once", func(t *testing.T) {
		adapter := &countingAdapter{value: "whsec_1"}
		service, err := secretservice.New(secretservice.Config{Adapter: adapter})
		require.NoError(t, err)

		secretID := newTestSecretID()

		for range 5 {
			got, err := service.GetAppSecret(t.Context(), secretID)
			require.NoError(t, err)
			require.Equal(t, "whsec_1", got.Value)
		}

		require.Equal(t, int64(1), adapter.getCalls.Load())
	})

	t.Run("reads reach the secret store again once the ttl elapses", func(t *testing.T) {
		adapter := &countingAdapter{value: "whsec_1"}
		service, err := secretservice.New(secretservice.Config{
			Adapter:  adapter,
			CacheTTL: time.Minute,
		})
		require.NoError(t, err)

		secretID := newTestSecretID()

		now := time.Now()

		clock.FreezeTime(now)
		defer clock.UnFreeze()

		_, err = service.GetAppSecret(t.Context(), secretID)
		require.NoError(t, err)

		clock.FreezeTime(now.Add(2 * time.Minute))

		_, err = service.GetAppSecret(t.Context(), secretID)
		require.NoError(t, err)

		require.Equal(t, int64(2), adapter.getCalls.Load())
	})

	t.Run("a failing secret store is reported as a failed dependency and is not cached", func(t *testing.T) {
		adapter := &countingAdapter{getErr: errors.New("status-code=429 rate limit exceeded")}
		service, err := secretservice.New(secretservice.Config{Adapter: adapter})
		require.NoError(t, err)

		secretID := newTestSecretID()

		_, err = service.GetAppSecret(t.Context(), secretID)
		require.Error(t, err)
		require.True(t, models.IsGenericStatusFailedDependencyError(err))

		_, err = service.GetAppSecret(t.Context(), secretID)
		require.Error(t, err)

		require.Equal(t, int64(2), adapter.getCalls.Load())
	})

	t.Run("updating a secret evicts the cached value", func(t *testing.T) {
		adapter := &countingAdapter{value: "whsec_1"}
		service, err := secretservice.New(secretservice.Config{Adapter: adapter})
		require.NoError(t, err)

		secretID := newTestSecretID()

		_, err = service.GetAppSecret(t.Context(), secretID)
		require.NoError(t, err)

		_, err = service.UpdateAppSecret(t.Context(), secretentity.UpdateAppSecretInput{
			AppID:    secretID.AppID,
			SecretID: secretID,
			Key:      secretID.Key,
			Value:    "whsec_2",
		})
		require.NoError(t, err)

		adapter.value = "whsec_2"

		got, err := service.GetAppSecret(t.Context(), secretID)
		require.NoError(t, err)
		require.Equal(t, "whsec_2", got.Value)
		require.Equal(t, int64(2), adapter.getCalls.Load())
	})

	t.Run("deleting a secret evicts the cached value", func(t *testing.T) {
		adapter := &countingAdapter{value: "whsec_1"}
		service, err := secretservice.New(secretservice.Config{Adapter: adapter})
		require.NoError(t, err)

		secretID := newTestSecretID()

		_, err = service.GetAppSecret(t.Context(), secretID)
		require.NoError(t, err)

		require.NoError(t, service.DeleteAppSecret(t.Context(), secretID))

		_, err = service.GetAppSecret(t.Context(), secretID)
		require.NoError(t, err)
		require.Equal(t, int64(2), adapter.getCalls.Load())
	})
}
