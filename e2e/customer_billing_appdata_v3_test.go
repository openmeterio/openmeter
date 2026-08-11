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

// TestV3CustomerBillingAppDataReplace exercises the replacement semantics of
// PUT /customers/{id}/billing and /billing/app-data: a provided value replaces
// the stored state, while omitting a field (or setting it to null, which is
// equivalent) removes it — the billing profile override is unpinned and app
// data is deleted. Stripe is not exercisable here (installing a Stripe app
// requires a live API key), so the external invoicing app covers the
// data-bearing path and the sandbox default profile covers the no-data path.
func TestV3CustomerBillingAppDataReplace(t *testing.T) {
	c := newV3Client(t)

	prefix := uniqueKey("billing_appdata")
	labels := map[string]string{"team": "billing", "env": "e2e"}
	replacedLabels := map[string]string{"team": "platform"}

	var (
		defaultProfileID string
		profile          v3sdk.Profile
		customerID       string
	)

	pinnedProfile := func() v3sdk.Nullable[v3sdk.ProfileReference] {
		return v3sdk.NullableValue(v3sdk.ProfileReference{ID: profile.ID})
	}

	runRequired(t, "Setup external invoicing app, profile, and customer", func(t *testing.T) {
		// given: an installed external invoicing app, a billing profile using
		// it for all app roles, and a fresh customer
		// when: the customer is pinned to that profile
		// then: the update reports the pinned profile as effective
		profiles, err := c.Billing.ListProfiles(t.Context(), v3sdk.ProfileListParams{
			Page: &v3sdk.PageParams{Size: lo.ToPtr(100)},
		})
		c.requireStatus(http.StatusOK, err)
		defaultIdx := slices.IndexFunc(profiles.Data, func(profile v3sdk.Profile) bool {
			return profile.Default
		})
		require.NotEqual(t, -1, defaultIdx, "default billing profile is required")
		defaultProfileID = profiles.Data[defaultIdx].ID

		installReq, err := v3sdk.InstallAppRequestFromInstallAppExternalInvoicing(v3sdk.InstallAppExternalInvoicing{
			Type:                 v3sdk.AppTypeExternalInvoicing,
			Name:                 "AppData Replace " + prefix,
			CreateBillingProfile: false,
		})
		require.NoError(t, err)
		installResp, err := c.Apps.Install(t.Context(), installReq)
		require.NoError(t, err)
		installedApp, err := installResp.App.AsAppExternalInvoicing()
		require.NoError(t, err)

		appRef := v3sdk.AppReference{ID: installedApp.ID}
		profile = createNewBillingProfileFromDefault(t, c, prefix, func(request *v3sdk.CreateBillingProfileRequest) {
			request.Name = "AppData Replace Profile " + prefix
			request.Apps = v3sdk.ProfileAppReferences{
				Tax:       appRef,
				Invoicing: appRef,
				Payment:   appRef,
			}
		})

		createdCustomer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:      prefix + "_customer",
			Name:     "AppData Replace Customer " + prefix,
			Currency: lo.ToPtr("USD"),
		})
		c.requireStatus(http.StatusCreated, err)
		customerID = createdCustomer.ID

		updated, err := c.Customers.Billing.Update(t.Context(), customerID, v3sdk.UpsertCustomerBillingDataRequest{
			BillingProfile: pinnedProfile(),
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, updated.BillingProfile)
		assert.Equal(t, profile.ID, updated.BillingProfile.ID)
	})

	runRequired(t, "Value sets external invoicing labels", func(t *testing.T) {
		updated, err := c.Customers.Billing.UpdateAppData(t.Context(), customerID, v3sdk.UpsertAppCustomerDataRequest{
			ExternalInvoicing: v3sdk.NullableValue(v3sdk.AppCustomerDataExternalInvoicingInput{
				Labels: &labels,
			}),
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, updated.ExternalInvoicing)
		assert.Equal(t, labels, updated.ExternalInvoicing.Labels)

		current, err := c.Customers.Billing.Get(t.Context(), customerID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, current.AppData)
		require.NotNil(t, current.AppData.ExternalInvoicing)
		assert.Equal(t, labels, current.AppData.ExternalInvoicing.Labels)
	})

	runRequired(t, "Value replaces labels through the billing endpoint", func(t *testing.T) {
		updated, err := c.Customers.Billing.Update(t.Context(), customerID, v3sdk.UpsertCustomerBillingDataRequest{
			BillingProfile: pinnedProfile(),
			AppData: &v3sdk.UpsertAppCustomerDataRequest{
				ExternalInvoicing: v3sdk.NullableValue(v3sdk.AppCustomerDataExternalInvoicingInput{
					Labels: &replacedLabels,
				}),
			},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, updated.AppData)
		require.NotNil(t, updated.AppData.ExternalInvoicing)
		assert.Equal(t, replacedLabels, updated.AppData.ExternalInvoicing.Labels)
	})

	runRequired(t, "Omitted app data deletes it", func(t *testing.T) {
		_, err := c.Customers.Billing.Update(t.Context(), customerID, v3sdk.UpsertCustomerBillingDataRequest{
			BillingProfile: pinnedProfile(),
		})
		c.requireStatus(http.StatusOK, err)

		current, err := c.Customers.Billing.Get(t.Context(), customerID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, current.AppData)
		require.NotNil(t, current.AppData.ExternalInvoicing)
		assert.Empty(t, current.AppData.ExternalInvoicing.Labels)
	})

	runRequired(t, "Null deletes external invoicing data", func(t *testing.T) {
		_, err := c.Customers.Billing.UpdateAppData(t.Context(), customerID, v3sdk.UpsertAppCustomerDataRequest{
			ExternalInvoicing: v3sdk.NullableValue(v3sdk.AppCustomerDataExternalInvoicingInput{
				Labels: &labels,
			}),
		})
		c.requireStatus(http.StatusOK, err)

		updated, err := c.Customers.Billing.UpdateAppData(t.Context(), customerID, v3sdk.UpsertAppCustomerDataRequest{
			ExternalInvoicing: v3sdk.Null[v3sdk.AppCustomerDataExternalInvoicingInput](),
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, updated.ExternalInvoicing)
		assert.Empty(t, updated.ExternalInvoicing.Labels)

		_, err = c.Customers.Billing.UpdateAppData(t.Context(), customerID, v3sdk.UpsertAppCustomerDataRequest{
			ExternalInvoicing: v3sdk.Null[v3sdk.AppCustomerDataExternalInvoicingInput](),
		})
		c.requireStatus(http.StatusOK, err)

		_, err = c.Customers.Billing.UpdateAppData(t.Context(), customerID, v3sdk.UpsertAppCustomerDataRequest{})
		c.requireStatus(http.StatusOK, err)
	})

	runRequired(t, "Invalid stripe value is rejected even on a non-stripe profile", func(t *testing.T) {
		_, err := c.Customers.Billing.UpdateAppData(t.Context(), customerID, v3sdk.UpsertAppCustomerDataRequest{
			Stripe: v3sdk.NullableValue(v3sdk.AppCustomerDataStripeInput{}),
		})
		requireProblem(t, err, http.StatusBadRequest)

		_, err = c.Customers.Billing.Update(t.Context(), customerID, v3sdk.UpsertCustomerBillingDataRequest{
			BillingProfile: pinnedProfile(),
			AppData: &v3sdk.UpsertAppCustomerDataRequest{
				Stripe: v3sdk.NullableValue(v3sdk.AppCustomerDataStripeInput{}),
			},
		})
		requireProblem(t, err, http.StatusBadRequest)
	})

	runRequired(t, "Omitted billing profile removes the override", func(t *testing.T) {
		updated, err := c.Customers.Billing.Update(t.Context(), customerID, v3sdk.UpsertCustomerBillingDataRequest{})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, updated.BillingProfile)
		assert.Equal(t, defaultProfileID, updated.BillingProfile.ID)

		current, err := c.Customers.Billing.Get(t.Context(), customerID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, current.BillingProfile)
		assert.Equal(t, defaultProfileID, current.BillingProfile.ID)

		// Null is equivalent to omission and removing an absent override is a no-op
		_, err = c.Customers.Billing.Update(t.Context(), customerID, v3sdk.UpsertCustomerBillingDataRequest{
			BillingProfile: v3sdk.Null[v3sdk.ProfileReference](),
		})
		c.requireStatus(http.StatusOK, err)
	})

	runRequired(t, "Sandbox default profile accepts empty updates", func(t *testing.T) {
		_, err := c.Customers.Billing.Update(t.Context(), customerID, v3sdk.UpsertCustomerBillingDataRequest{})
		c.requireStatus(http.StatusOK, err)

		_, err = c.Customers.Billing.UpdateAppData(t.Context(), customerID, v3sdk.UpsertAppCustomerDataRequest{})
		c.requireStatus(http.StatusOK, err)
	})
}
