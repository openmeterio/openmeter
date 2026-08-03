package secretservice_test

import (
	"context"
	"errors"
	"sync"
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
	onGet    func()
	onUpdate func()

	mu    sync.Mutex
	value string
}

func (a *countingAdapter) CreateAppSecret(_ context.Context, input secretentity.CreateAppSecretInput) (secretentity.SecretID, error) {
	return secretentity.NewSecretID(input.AppID, input.Value, input.Key), nil
}

func (a *countingAdapter) UpdateAppSecret(_ context.Context, input secretentity.UpdateAppSecretInput) (secretentity.SecretID, error) {
	if a.onUpdate != nil {
		a.onUpdate()
	}

	a.mu.Lock()
	a.value = input.Value
	a.mu.Unlock()

	return input.SecretID, nil
}

func (a *countingAdapter) GetAppSecret(_ context.Context, input secretentity.GetAppSecretInput) (secretentity.Secret, error) {
	a.getCalls.Add(1)

	a.mu.Lock()
	value := a.value
	a.mu.Unlock()

	if a.onGet != nil {
		a.onGet()
	}

	if a.getErr != nil {
		return secretentity.Secret{}, a.getErr
	}

	return secretentity.Secret{SecretID: input, Value: value}, nil
}

func (a *countingAdapter) DeleteAppSecret(_ context.Context, _ secretentity.DeleteAppSecretInput) error {
	return nil
}

func newTestSecretID() secretentity.SecretID {
	appID := app.AppID{Namespace: "test-namespace", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}

	return secretentity.NewSecretID(appID, "stripe_webhook_secret", "webhook_secret")
}

func TestConfigValidate(t *testing.T) {
	adapter := &countingAdapter{value: "whsec_1"}

	t.Run("a negative cache size is rejected", func(t *testing.T) {
		_, err := secretservice.New(secretservice.Config{Adapter: adapter, CacheSize: -1})
		require.Error(t, err)
		require.True(t, models.IsGenericValidationError(err))
	})

	t.Run("a negative cache ttl is rejected", func(t *testing.T) {
		_, err := secretservice.New(secretservice.Config{Adapter: adapter, CacheTTL: -time.Second})
		require.Error(t, err)
		require.True(t, models.IsGenericValidationError(err))
	})

	t.Run("every invalid field is reported", func(t *testing.T) {
		_, err := secretservice.New(secretservice.Config{CacheSize: -1, CacheTTL: -time.Second})
		require.Error(t, err)
		require.ErrorContains(t, err, "adapter is required")
		require.ErrorContains(t, err, "cache size must not be negative")
		require.ErrorContains(t, err, "cache ttl must not be negative")
	})
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

		got, err := service.GetAppSecret(t.Context(), secretID)
		require.NoError(t, err)
		require.Equal(t, "whsec_2", got.Value)
		require.Equal(t, int64(2), adapter.getCalls.Load())
	})

	t.Run("the cache size bounds how many secrets stay cached", func(t *testing.T) {
		adapter := &countingAdapter{value: "whsec_1"}
		service, err := secretservice.New(secretservice.Config{Adapter: adapter, CacheSize: 1})
		require.NoError(t, err)

		appID := app.AppID{Namespace: "test-namespace", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}
		first := secretentity.NewSecretID(appID, "stripe_webhook_secret", "webhook_secret")
		second := secretentity.NewSecretID(appID, "stripe_api_key", "api_key")

		_, err = service.GetAppSecret(t.Context(), first)
		require.NoError(t, err)

		_, err = service.GetAppSecret(t.Context(), second)
		require.NoError(t, err)

		_, err = service.GetAppSecret(t.Context(), first)
		require.NoError(t, err)

		require.Equal(t, int64(3), adapter.getCalls.Load())
	})

	t.Run("concurrent reads of the same secret reach the store once", func(t *testing.T) {
		release := make(chan struct{})
		started := make(chan struct{}, 1)

		adapter := &countingAdapter{value: "whsec_1"}
		adapter.onGet = func() {
			select {
			case started <- struct{}{}:
			default:
			}

			<-release
		}

		service, err := secretservice.New(secretservice.Config{Adapter: adapter})
		require.NoError(t, err)

		secretID := newTestSecretID()

		var wg sync.WaitGroup

		for range 8 {
			wg.Add(1)

			go func() {
				defer wg.Done()

				_, err := service.GetAppSecret(context.Background(), secretID)
				require.NoError(t, err)
			}()
		}

		<-started
		close(release)
		wg.Wait()

		require.Equal(t, int64(1), adapter.getCalls.Load())
	})

	t.Run("a read racing a mutation never leaves the superseded value cached", func(t *testing.T) {
		secretID := newTestSecretID()

		fetchStarted := make(chan struct{})
		releaseFetch := make(chan struct{})
		updateReached := make(chan struct{})

		adapter := &countingAdapter{value: "whsec_old"}
		adapter.onGet = func() {
			select {
			case <-fetchStarted:
			default:
				close(fetchStarted)
				<-releaseFetch
			}
		}
		adapter.onUpdate = func() {
			close(updateReached)
		}

		service, err := secretservice.New(secretservice.Config{Adapter: adapter})
		require.NoError(t, err)

		var wg sync.WaitGroup

		wg.Add(2)

		go func() {
			defer wg.Done()

			_, err := service.GetAppSecret(context.Background(), secretID)
			require.NoError(t, err)
		}()

		<-fetchStarted

		go func() {
			defer wg.Done()

			_, err := service.UpdateAppSecret(context.Background(), secretentity.UpdateAppSecretInput{
				AppID:    secretID.AppID,
				SecretID: secretID,
				Key:      secretID.Key,
				Value:    "whsec_new",
			})
			require.NoError(t, err)
		}()

		select {
		case <-updateReached:
		case <-time.After(200 * time.Millisecond):
		}

		close(releaseFetch)
		wg.Wait()

		got, err := service.GetAppSecret(context.Background(), secretID)
		require.NoError(t, err)
		require.Equal(t, "whsec_new", got.Value)
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
