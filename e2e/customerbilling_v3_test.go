package e2e

import (
	"net/http"
	"slices"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

func TestV3CustomerBillingUpdateDropsOmittedProfileOverride(t *testing.T) {
	c := newV3Client(t)

	prefix := uniqueKey("customer_billing_replace")

	customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
		Key:  prefix + "_customer",
		Name: "Customer Billing Replace Test Customer",
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, customer)

	profiles, err := c.Billing.ListProfiles(t.Context(), v3sdk.ProfileListParams{
		Page: &v3sdk.PageParams{Size: lo.ToPtr(100)},
	})
	c.requireStatus(http.StatusOK, err)
	require.NotNil(t, profiles)

	defaultIdx := slices.IndexFunc(profiles.Data, func(profile v3sdk.Profile) bool {
		return profile.Default
	})
	require.NotEqual(t, -1, defaultIdx, "default billing profile is required")
	defaultProfileID := profiles.Data[defaultIdx].ID

	pinned := createNewBillingProfileFromDefault(t, c, prefix, nil)
	require.NotEqual(t, defaultProfileID, pinned.ID)

	t.Run("Should pin the customer to a named profile", func(t *testing.T) {
		updated, err := c.Customers.Billing.Update(t.Context(), customer.ID, v3sdk.UpsertCustomerBillingDataRequest{
			BillingProfile: &v3sdk.ProfileReference{ID: pinned.ID},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, updated)
		require.NotNil(t, updated.BillingProfile)
		assert.Equal(t, pinned.ID, updated.BillingProfile.ID)

		fetched, err := c.Customers.Billing.Get(t.Context(), customer.ID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, fetched)
		require.NotNil(t, fetched.BillingProfile)
		assert.Equal(t, pinned.ID, fetched.BillingProfile.ID)
	})

	t.Run("Should drop the pin when billing_profile is omitted", func(t *testing.T) {
		updated, err := c.Customers.Billing.Update(t.Context(), customer.ID, v3sdk.UpsertCustomerBillingDataRequest{})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, updated)
		require.NotNil(t, updated.BillingProfile)
		assert.Equal(t, defaultProfileID, updated.BillingProfile.ID)

		fetched, err := c.Customers.Billing.Get(t.Context(), customer.ID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, fetched)
		require.NotNil(t, fetched.BillingProfile)
		assert.Equal(t, defaultProfileID, fetched.BillingProfile.ID,
			"an omitted billing_profile must unpin the customer, not merely answer with the default")
	})

	t.Run("Should tolerate a repeated omission once unpinned", func(t *testing.T) {
		updated, err := c.Customers.Billing.Update(t.Context(), customer.ID, v3sdk.UpsertCustomerBillingDataRequest{})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, updated)
		require.NotNil(t, updated.BillingProfile)
		assert.Equal(t, defaultProfileID, updated.BillingProfile.ID)
	})
}
