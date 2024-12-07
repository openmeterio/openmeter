package appstripe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppStripe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env, err := NewTestEnv(t, ctx)
	require.NoError(t, err, "AppStripeTestEnv() failed")
	require.NotNil(t, env.App())
	require.NotNil(t, env.AppStripe())
	require.NotNil(t, env.Customer())
	require.NotNil(t, env.Secret())

	defer func() {
		if err := env.Close(); err != nil {
			t.Errorf("failed to close environment: %v", err)
		}
	}()

	// Test suite covering the stripe app
	t.Run("AppStripe", func(t *testing.T) {
		testSuite := AppHandlerTestSuite{
			Env: env,
		}

		t.Run("Create", func(t *testing.T) {
			testSuite.TestCreate(ctx, t)
		})

		t.Run("Get", func(t *testing.T) {
			testSuite.TestGet(ctx, t)
		})

		t.Run("GetDefault", func(t *testing.T) {
			testSuite.TestGetDefault(ctx, t)
		})

		t.Run("Uninstall", func(t *testing.T) {
			testSuite.TestUninstall(ctx, t)
		})

		t.Run("CustomerData", func(t *testing.T) {
			testSuite.TestCustomerData(ctx, t)
		})

		t.Run("TestCustomerValidate", func(t *testing.T) {
			testSuite.TestCustomerValidate(ctx, t)
		})

		t.Run("TestCreateCheckoutSession", func(t *testing.T) {
			testSuite.TestCreateCheckoutSession(ctx, t)
		})
	})
}
