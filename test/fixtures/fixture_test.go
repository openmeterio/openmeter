package fixtures_test

import (
	"testing"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/test/fixtures"
	"github.com/samber/do"
	"github.com/stretchr/testify/require"
)

func TestFixtures(t *testing.T) {
	registry := do.New()

	fixtures.NamespaceFixture{}.Register(registry)
	fixtures.CustomerFixture{
		Deps: fixtures.NoopDeps{},
	}.Register(registry)
	fixtures.SubscriptionFixture{}.Register(registry)

	t.Run("Should get a subscription", func(t *testing.T) {
		sub, err := do.InvokeNamed[subscription.Subscription](registry, fixtures.SubscriptionFixtureName)
		if err != nil {
			t.Fatal(err)
		}

		require.NotNil(t, sub)
	})

	t.Run("Should get 2 different subscriptions", func(t *testing.T) {
		getSub, err := do.InvokeNamed[fixtures.InstanceProvider[subscription.Subscription]](registry, fixtures.SubscriptionFixtureInstanceName)
		if err != nil {
			t.Fatal(err)
		}

		sub1, err := getSub(registry)
		require.NoError(t, err)

		sub2, err := getSub(registry)
		require.NoError(t, err)

		require.NotEqual(t, sub1.ID, sub2.ID)
	})

	t.Run("Should get the same subscription twice", func(t *testing.T) {
		sub1, err := do.InvokeNamed[subscription.Subscription](registry, fixtures.SubscriptionFixtureName)
		require.NoError(t, err)

		sub2, err := do.InvokeNamed[subscription.Subscription](registry, fixtures.SubscriptionFixtureName)
		require.NoError(t, err)

		require.Equal(t, sub1.ID, sub2.ID)
	})
}
