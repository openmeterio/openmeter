package appstripe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppStripe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env, err := NewTestEnv(ctx)
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

		t.Run("CustomerCreate", func(t *testing.T) {
			testSuite.TestCustomerCreate(ctx, t)
		})

		t.Run("TestCustomerValidate", func(t *testing.T) {
			testSuite.TestCustomerValidate(ctx, t)
		})
	})
}
