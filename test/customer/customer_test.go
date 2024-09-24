package customer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCustomer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env, err := NewTestEnv(ctx)
	require.NoError(t, err, "CustomerTestEnv() failed")
	require.NotNil(t, env.Customer())
	require.NotNil(t, env.CustomerRepo())

	defer func() {
		if err := env.Close(); err != nil {
			t.Errorf("failed to close environment: %v", err)
		}
	}()

	// Test suite covering the customer
	t.Run("Customer", func(t *testing.T) {
		testSuite := CustomerHandlerTestSuite{
			Env: env,
		}

		t.Run("TestCreate", func(t *testing.T) {
			testSuite.TestCreate(ctx, t)
		})

		t.Run("TestList", func(t *testing.T) {
			testSuite.TestList(ctx, t)
		})

		t.Run("TestGet", func(t *testing.T) {
			testSuite.TestGet(ctx, t)
		})

		t.Run("TestUpdate", func(t *testing.T) {
			testSuite.TestUpdate(ctx, t)
		})

		t.Run("TestDelete", func(t *testing.T) {
			testSuite.TestDelete(ctx, t)
		})
	})
}
