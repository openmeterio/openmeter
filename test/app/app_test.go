package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApp(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env, err := NewTestEnv(t, ctx)
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

		t.Run("TestGetMarketplaceListing", func(t *testing.T) {
			testSuite.TestGetMarketplaceListing(ctx, t)
		})

		t.Run("TestListMarketplaceListings", func(t *testing.T) {
			testSuite.TestListMarketplaceListings(ctx, t)
		})
	})
}
