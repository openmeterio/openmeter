package e2e

import (
	"net/http"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

func TestV3SubscriptionMigrateInPlace(t *testing.T) {
	c := newV3Client(t)

	// given an active subscription and two later versions of the same plan
	body := validPlanRequest("migrate")
	before := createMigrationSubscription(t, c, body)
	added := validFlatRateCard("added")
	body.Phases[0].RateCards = append(body.Phases[0].RateCards, added)
	v2 := publishMigrationPlan(t, c, body)
	v3 := publishMigrationPlan(t, c, body)

	// when a specific version is requested, it takes precedence over latest
	result, err := c.Subscriptions.Migrate(t.Context(), before.ID, v3sdk.SubscriptionMigrate{TargetVersion: &v2.Version})
	c.requireStatus(http.StatusOK, err)
	require.NotNil(t, result)

	// then the original snapshot survives and the same subscription gains the item
	require.Equal(t, before.ID, result.Current.ID)
	require.Equal(t, before.ID, result.Next.ID)
	require.EqualValues(t, 1, result.Current.Plan.Version)
	require.Equal(t, v2.Version, result.Next.Plan.Version)
	require.Equal(t, before.BillingAnchor, result.Next.BillingAnchor)
	require.Equal(t, before.ActiveFrom, result.Next.ActiveFrom)
	require.True(t, subscriptionHasItem(&result.Next, body.Phases[0].Key, added.Key))
	unchanged, ok := lo.Find(result.Next.Phases[0].Items, func(item v3sdk.SubscriptionItem) bool {
		return item.RateCard.Key == before.Phases[0].Items[0].RateCard.Key
	})
	require.True(t, ok)
	require.Equal(t, before.Phases[0].Items[0], unchanged)

	t.Run("omitted version selects latest", func(t *testing.T) {
		latest, err := c.Subscriptions.Migrate(t.Context(), before.ID, v3sdk.SubscriptionMigrate{})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, latest)
		require.Equal(t, v2.Version, latest.Current.Plan.Version)
		require.Equal(t, v3.Version, latest.Next.Plan.Version)
	})

	t.Run("same version is rejected", func(t *testing.T) {
		_, err := c.Subscriptions.Migrate(t.Context(), before.ID, v3sdk.SubscriptionMigrate{TargetVersion: &v3.Version})
		requireProblem(t, err, http.StatusBadRequest)
	})

	t.Run("older version is rejected", func(t *testing.T) {
		_, err := c.Subscriptions.Migrate(t.Context(), before.ID, v3sdk.SubscriptionMigrate{TargetVersion: &v2.Version})
		requireProblem(t, err, http.StatusBadRequest)
	})

	t.Run("invalid version is rejected by the schema", func(t *testing.T) {
		_, err := c.Subscriptions.Migrate(t.Context(), before.ID, v3sdk.SubscriptionMigrate{TargetVersion: lo.ToPtr(int64(0))})
		requireProblem(t, err, http.StatusBadRequest)
	})

	t.Run("missing subscription is not found", func(t *testing.T) {
		_, err := c.Subscriptions.Migrate(t.Context(), ulid.Make().String(), v3sdk.SubscriptionMigrate{})
		requireProblem(t, err, http.StatusNotFound)
	})
}

func TestV3SubscriptionMigrateNextCycle(t *testing.T) {
	c := newV3Client(t)

	// given a new rate card in the target plan
	body := validPlanRequest("migrate_next")
	before := createMigrationSubscription(t, c, body)
	body.Phases[0].RateCards = append(body.Phases[0].RateCards, validFlatRateCard("future"))
	target := publishMigrationPlan(t, c, body)
	timing := lo.Must(v3sdk.SubscriptionEditTimingFromEnum(v3sdk.SubscriptionEditTimingEnumNextBillingCycle))

	// when migration is scheduled for the next cycle
	result, err := c.Subscriptions.Migrate(t.Context(), before.ID, v3sdk.SubscriptionMigrate{Timing: &timing})
	c.requireStatus(http.StatusOK, err)
	require.NotNil(t, result)

	// then the reference advances now, but the current phase still exposes its original item
	require.Equal(t, before.ID, result.Next.ID)
	require.Equal(t, target.Version, result.Next.Plan.Version)
	require.Equal(t, before.CurrentPeriod, result.Next.CurrentPeriod)
	require.Equal(t, before.Phases[0].Items, result.Next.Phases[0].Items)
}

func TestV3SubscriptionMigrateCustomTiming(t *testing.T) {
	c := newV3Client(t)

	// given a plan update and the subscription's next billing boundary
	body := validPlanRequest("migrate_custom")
	before := createMigrationSubscription(t, c, body)
	require.NotNil(t, before.CurrentPeriod)
	publishMigrationPlan(t, c, body)
	timing := lo.Must(v3sdk.SubscriptionEditTimingFromCustom(before.CurrentPeriod.To))

	// when a billing-aligned timestamp is supplied
	result, err := c.Subscriptions.Migrate(t.Context(), before.ID, v3sdk.SubscriptionMigrate{Timing: &timing})
	c.requireStatus(http.StatusOK, err)
	require.NotNil(t, result)

	// then migration succeeds without replacing the subscription
	require.Equal(t, before.ID, result.Next.ID)
	require.EqualValues(t, 2, result.Next.Plan.Version)
}

func TestV3SubscriptionMigrateStartingPhase(t *testing.T) {
	c := newV3Client(t)

	// given incompatible phase keys between the subscription and target plan
	body := validPlanRequest("migrate_phase")
	before := createMigrationSubscription(t, c, body)
	body.Phases[0].Key = uniqueKey("replacement_phase")
	publishMigrationPlan(t, c, body)

	// when replacement is not explicitly requested, the incompatibility is rejected
	_, err := c.Subscriptions.Migrate(t.Context(), before.ID, v3sdk.SubscriptionMigrate{})
	requireProblem(t, err, http.StatusBadRequest)

	// then supplying the target phase permits an explicit replacement
	result, err := c.Subscriptions.Migrate(t.Context(), before.ID, v3sdk.SubscriptionMigrate{StartingPhase: &body.Phases[0].Key})
	c.requireStatus(http.StatusOK, err)
	require.NotNil(t, result)
	require.NotEqual(t, before.ID, result.Next.ID)
	require.Equal(t, before.ID, result.Current.ID)
	require.Equal(t, &result.Next.ActiveFrom, result.Current.ActiveTo)
	require.Equal(t, body.Phases[0].Key, result.Next.Phases[0].Key)
}

func TestV3SubscriptionMigrateBillingAnchor(t *testing.T) {
	c := newV3Client(t)

	// given matching plans and a different requested anchor
	body := validPlanRequest("migrate_anchor")
	before := createMigrationSubscription(t, c, body)
	publishMigrationPlan(t, c, body)
	anchor := before.BillingAnchor.AddDate(0, 0, 1)

	// when only the anchor requests replacement
	result, err := c.Subscriptions.Migrate(t.Context(), before.ID, v3sdk.SubscriptionMigrate{BillingAnchor: &anchor})
	c.requireStatus(http.StatusOK, err)
	require.NotNil(t, result)

	// then the new subscription retains the supplied anchor
	require.NotEqual(t, before.ID, result.Next.ID)
	require.Equal(t, anchor, result.Next.BillingAnchor)
	require.Equal(t, &result.Next.ActiveFrom, result.Current.ActiveTo)
}

func publishMigrationPlan(t *testing.T, c *v3Client, body v3sdk.CreatePlanRequest) *v3sdk.Plan {
	t.Helper()

	plan, err := c.Plans.Create(t.Context(), body)
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, plan)
	published, err := c.Plans.Publish(t.Context(), plan.ID)
	c.requireStatus(http.StatusOK, err)
	require.NotNil(t, published)

	return published
}

func createMigrationSubscription(t *testing.T, c *v3Client, body v3sdk.CreatePlanRequest) *v3sdk.BillingSubscription {
	t.Helper()

	plan := publishMigrationPlan(t, c, body)
	customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
		Key: uniqueKey("migration_customer"), Name: "Migration test", Currency: lo.ToPtr("USD"),
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, customer)

	sub, err := c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
		Customer: v3sdk.SubscriptionChangeCustomer{ID: &customer.ID},
		Plan:     &v3sdk.SubscriptionChangePlan{ID: &plan.ID},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, sub)

	return sub
}
