package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCustomer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env, err := NewTestEnv(ctx)
	require.NoError(t, err, "MarketplaceTestEnv() failed")
	require.NotNil(t, env.App())
	require.NotNil(t, env.Adapter())

	defer func() {
		if err := env.Close(); err != nil {
			t.Errorf("failed to close environment: %v", err)
		}
	}()

	// Test suite covering the marketplace
	t.Run("Marketplace", func(t *testing.T) {
		testSuite := AppHandlerTestSuite{
			Env: env,
		}

		t.Run("TestGet", func(t *testing.T) {
			testSuite.TestGetMarketplaceListing(ctx, t)
		})
	})
}
