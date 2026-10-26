package appstripe

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/app"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	"github.com/openmeterio/openmeter/openmeter/billing"
)

func TestCustomerBillingProfileV3(t *testing.T) {
	ctx := t.Context()
	env, err := NewTestEnv(t, ctx)
	require.NoError(t, err)
	defer func() { require.NoError(t, env.Close()) }()

	const namespace = "customer-billing-profile-v3-test"
	defaultApp, err := env.Fixture().setupApp(ctx, namespace)
	require.NoError(t, err)
	defaultProfile, err := env.Billing().CreateProfile(ctx, newTestProfileInput(namespace, defaultApp.GetID(), "Default", true))
	require.NoError(t, err)
	pinnedApp, err := env.Fixture().setupApp(ctx, namespace)
	require.NoError(t, err)
	pinnedProfile, err := env.Billing().CreateProfile(ctx, newTestProfileInput(namespace, pinnedApp.GetID(), "Pinned", false))
	require.NoError(t, err)
	router := newCustomerBillingRouter(t, env, namespace)

	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "omission clears the pin", body: `{}`},
		{name: "null clears the pin", body: `{"billing_profile":null}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given a customer explicitly pinned to a different payment app
			cus, err := env.Fixture().setupCustomer(t.Context(), namespace)
			require.NoError(t, err)
			path := "/openmeter/customers/" + cus.ID + "/billing"
			rec := doJSONRequest(router, http.MethodPut, path+"/profile", `{"billing_profile":{"id":"`+pinnedProfile.ID+`"}}`)
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			var pinned api.BillingCustomerProfile
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &pinned))
			require.Equal(t, nullable.NewNullableWithValue(api.BillingProfileReference{Id: pinnedProfile.ID}), pinned.BillingProfile)
			require.Equal(t, pinnedProfile.ID, pinned.EffectiveBillingProfile.Id)

			// when clearing the selection twice
			for range 2 {
				rec = doJSONRequest(router, http.MethodPut, path+"/profile", tc.body)

				// then the operation is idempotent and GET agrees with its response
				require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
				var cleared api.BillingCustomerProfile
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cleared))
				require.False(t, cleared.BillingProfile.IsSpecified())
				require.Equal(t, defaultProfile.ID, cleared.EffectiveBillingProfile.Id)
				get := doJSONRequest(router, http.MethodGet, path, "")
				require.Equal(t, http.StatusOK, get.Code, get.Body.String())
				var got api.BillingCustomerData
				require.NoError(t, json.Unmarshal(get.Body.Bytes(), &got))
				require.Equal(t, cleared.BillingProfile, got.BillingProfile)
				require.Equal(t, cleared.EffectiveBillingProfile, got.EffectiveBillingProfile)
			}
		})
	}

	t.Run("pinning and unpinning preserve data in both apps", func(t *testing.T) {
		// given customer data in two distinct payment apps
		cus, err := env.Fixture().setupCustomer(t.Context(), namespace)
		require.NoError(t, err)
		for _, fixture := range []struct {
			application app.App
			externalID  string
		}{
			{application: defaultApp, externalID: "cus_profile_default"},
			{application: pinnedApp, externalID: "cus_profile_pinned"},
		} {
			env.StripeAppClient().On("GetCustomer", fixture.externalID).
				Return(stripeclient.StripeCustomer{StripeCustomerID: fixture.externalID}, nil)
			defer env.StripeAppClient().Restore()
			require.NoError(t, fixture.application.UpsertCustomerData(t.Context(), app.UpsertAppInstanceCustomerDataInput{
				CustomerID: cus.GetID(),
				Data:       appstripe.CustomerData{StripeCustomerID: fixture.externalID},
			}))
		}

		// when switching the selected app and returning to the dynamic default
		path := "/openmeter/customers/" + cus.ID + "/billing/profile"
		for _, body := range []string{`{"billing_profile":{"id":"` + pinnedProfile.ID + `"}}`, `{"billing_profile":null}`} {
			rec := doJSONRequest(router, http.MethodPut, path, body)
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		}

		// then neither app's customer data is changed or removed
		defaultData, err := env.AppStripe().GetStripeCustomerData(t.Context(), appstripe.GetStripeCustomerDataInput{
			AppID:      defaultApp.GetID(),
			CustomerID: cus.GetID(),
		})
		require.NoError(t, err)
		require.Equal(t, "cus_profile_default", defaultData.StripeCustomerID)
		pinnedData, err := env.AppStripe().GetStripeCustomerData(t.Context(), appstripe.GetStripeCustomerDataInput{
			AppID:      pinnedApp.GetID(),
			CustomerID: cus.GetID(),
		})
		require.NoError(t, err)
		require.Equal(t, "cus_profile_pinned", pinnedData.StripeCustomerID)
	})

	t.Run("invalid references are rejected without changing the pin", func(t *testing.T) {
		// given a pinned customer and profiles that cannot be selected
		cus, err := env.Fixture().setupCustomer(t.Context(), namespace)
		require.NoError(t, err)
		_, err = env.Billing().SetCustomerBillingProfile(t.Context(), billing.SetCustomerBillingProfileInput{
			Customer:  cus.GetID(),
			ProfileID: pinnedProfile.ID,
		})
		require.NoError(t, err)
		foreignApp, err := env.Fixture().setupApp(t.Context(), namespace+"-foreign")
		require.NoError(t, err)
		foreignProfile, err := env.Billing().CreateProfile(t.Context(), newTestProfileInput(namespace+"-foreign", foreignApp.GetID(), "Foreign", false))
		require.NoError(t, err)
		deletedProfile, err := env.Billing().CreateProfile(t.Context(), newTestProfileInput(namespace, defaultApp.GetID(), "Deleted", false))
		require.NoError(t, err)
		require.NoError(t, env.Billing().DeleteProfile(t.Context(), deletedProfile.ProfileID()))

		for _, tc := range []struct {
			name   string
			body   string
			status int
		}{
			{name: "empty reference", body: `{"billing_profile":{}}`, status: http.StatusBadRequest},
			{name: "empty id", body: `{"billing_profile":{"id":""}}`, status: http.StatusBadRequest},
			{name: "malformed id", body: `{"billing_profile":{"id":"invalid"}}`, status: http.StatusBadRequest},
			{name: "missing profile", body: `{"billing_profile":{"id":"01ARZ3NDEKTSV4RRFFQ69G5FAV"}}`, status: http.StatusNotFound},
			{name: "foreign profile", body: `{"billing_profile":{"id":"` + foreignProfile.ID + `"}}`, status: http.StatusNotFound},
			{name: "deleted profile", body: `{"billing_profile":{"id":"` + deletedProfile.ID + `"}}`, status: http.StatusNotFound},
			{name: "read-only field", body: `{"effective_billing_profile":{"id":"` + defaultProfile.ID + `"}}`, status: http.StatusBadRequest},
			{name: "app data", body: `{"app_data":{}}`, status: http.StatusBadRequest},
			{name: "empty body", body: ``, status: http.StatusBadRequest},
		} {
			t.Run(tc.name, func(t *testing.T) {
				// when attempting an invalid selection
				rec := doJSONRequest(router, http.MethodPut, "/openmeter/customers/"+cus.ID+"/billing/profile", tc.body)

				// then validation rejects it and the stored selection survives
				require.Equal(t, tc.status, rec.Code, rec.Body.String())
				override, err := env.Billing().GetCustomerBillingProfile(t.Context(), billing.GetCustomerBillingProfileInput{Customer: cus.GetID()})
				require.NoError(t, err)
				require.Equal(t, pinnedProfile.ID, override.AssignedProfile.ProfileID)
			})
		}
	})

	t.Run("missing and deleted customers cannot be changed", func(t *testing.T) {
		// given a deleted customer
		cus, err := env.Fixture().setupCustomer(t.Context(), namespace)
		require.NoError(t, err)
		require.NoError(t, env.Customer().DeleteCustomer(t.Context(), cus.GetID()))

		// when selecting profiles for missing or deleted customers
		missing := doJSONRequest(router, http.MethodPut, "/openmeter/customers/01ARZ3NDEKTSV4RRFFQ69G5FAV/billing/profile", `{}`)
		deleted := doJSONRequest(router, http.MethodPut, "/openmeter/customers/"+cus.ID+"/billing/profile", `{}`)

		// then neither request succeeds
		require.Equal(t, http.StatusNotFound, missing.Code, missing.Body.String())
		require.Equal(t, http.StatusGone, deleted.Code, deleted.Body.String())
	})

	t.Run("clearing a customer without a pin is idempotent", func(t *testing.T) {
		// given a customer that has never been pinned
		cus, err := env.Fixture().setupCustomer(t.Context(), namespace)
		require.NoError(t, err)

		// when both supported clear forms are sent
		for _, body := range []string{`{}`, `{"billing_profile":null}`} {
			rec := doJSONRequest(router, http.MethodPut, "/openmeter/customers/"+cus.ID+"/billing/profile", body)

			// then no assignment is created and the effective default is returned
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			var cleared api.BillingCustomerProfile
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cleared))
			require.False(t, cleared.BillingProfile.IsSpecified())
			require.Equal(t, defaultProfile.ID, cleared.EffectiveBillingProfile.Id)
			profile, err := env.Billing().GetCustomerBillingProfile(t.Context(), billing.GetCustomerBillingProfileInput{Customer: cus.GetID()})
			require.NoError(t, err)
			require.Nil(t, profile.AssignedProfile)
		}
	})

	t.Run("a missing default rolls back clearing the pin", func(t *testing.T) {
		// given a pinned customer in a namespace with no default profile
		const noDefaultNamespace = namespace + "-no-default"
		application, err := env.Fixture().setupApp(t.Context(), noDefaultNamespace)
		require.NoError(t, err)
		profile, err := env.Billing().CreateProfile(t.Context(), newTestProfileInput(noDefaultNamespace, application.GetID(), "Only Profile", false))
		require.NoError(t, err)
		cus, err := env.Fixture().setupCustomer(t.Context(), noDefaultNamespace)
		require.NoError(t, err)
		noDefaultRouter := newCustomerBillingRouter(t, env, noDefaultNamespace)
		path := "/openmeter/customers/" + cus.ID + "/billing/profile"
		rec := doJSONRequest(noDefaultRouter, http.MethodPut, path, `{"billing_profile":{"id":"`+profile.ID+`"}}`)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		// when clearing would leave the effective profile unresolved
		rec = doJSONRequest(noDefaultRouter, http.MethodPut, path, `{}`)

		// then the failed response leaves the previous assignment intact
		require.GreaterOrEqual(t, rec.Code, http.StatusBadRequest, rec.Body.String())
		stored, err := env.Billing().GetCustomerBillingProfile(t.Context(), billing.GetCustomerBillingProfileInput{Customer: cus.GetID()})
		require.NoError(t, err)
		require.NotNil(t, stored.AssignedProfile)
		require.Equal(t, profile.ID, stored.AssignedProfile.ProfileID)
	})

	t.Run("an explicit default pin survives a default change until cleared", func(t *testing.T) {
		// given a customer explicitly pinned to the current default
		cus, err := env.Fixture().setupCustomer(t.Context(), namespace)
		require.NoError(t, err)
		path := "/openmeter/customers/" + cus.ID + "/billing"
		rec := doJSONRequest(router, http.MethodPut, path+"/profile", `{"billing_profile":{"id":"`+defaultProfile.ID+`"}}`)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		// when replacing the default with another profile using the same app type
		nextDefault, err := env.Billing().CreateProfile(t.Context(), newTestProfileInput(namespace, defaultApp.GetID(), "Next Default", true))
		require.NoError(t, err)
		get := doJSONRequest(router, http.MethodGet, path, "")
		require.Equal(t, http.StatusOK, get.Code, get.Body.String())
		var pinned api.BillingCustomerData
		require.NoError(t, json.Unmarshal(get.Body.Bytes(), &pinned))
		require.Equal(t, defaultProfile.ID, pinned.EffectiveBillingProfile.Id)

		// then clearing uses the new default and subsequent default changes are dynamic
		rec = doJSONRequest(router, http.MethodPut, path+"/profile", `{}`)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var cleared api.BillingCustomerProfile
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cleared))
		require.Equal(t, nextDefault.ID, cleared.EffectiveBillingProfile.Id)
		latestDefault, err := env.Billing().CreateProfile(t.Context(), newTestProfileInput(namespace, defaultApp.GetID(), "Latest Default", true))
		require.NoError(t, err)
		get = doJSONRequest(router, http.MethodGet, path, "")
		require.Equal(t, http.StatusOK, get.Code, get.Body.String())
		var dynamic api.BillingCustomerData
		require.NoError(t, json.Unmarshal(get.Body.Bytes(), &dynamic))
		require.False(t, dynamic.BillingProfile.IsSpecified())
		require.Equal(t, latestDefault.ID, dynamic.EffectiveBillingProfile.Id)
	})
}
