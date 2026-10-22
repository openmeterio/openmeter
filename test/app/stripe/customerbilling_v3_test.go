package appstripe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	customersbilling "github.com/openmeterio/openmeter/api/v3/handlers/customers/billing"
	"github.com/openmeterio/openmeter/api/v3/oasmiddleware"
	"github.com/openmeterio/openmeter/openmeter/app"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

// TestCustomerBillingV3 exercises the v3 customer billing GET and PUT endpoints through
// the production handler mounted behind the OpenAPI request-validation
// middleware, mirroring api/v3/server wiring.
func TestCustomerBillingV3(t *testing.T) {
	ctx := t.Context()

	env, err := NewTestEnv(t, ctx)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, env.Close())
	}()

	const namespace = "customer-billing-v3-test"

	stripeApp, err := env.Fixture().setupApp(ctx, namespace)
	require.NoError(t, err)

	defaultProfile, err := env.Billing().CreateProfile(ctx, newTestProfileInput(namespace, stripeApp.GetID(), "Default Profile", true))
	require.NoError(t, err)
	require.NotNil(t, defaultProfile)

	pinnedStripeApp, err := env.Fixture().setupApp(ctx, namespace)
	require.NoError(t, err)

	pinnedProfile, err := env.Billing().CreateProfile(ctx, newTestProfileInput(namespace, pinnedStripeApp.GetID(), "Pinned Profile", false))
	require.NoError(t, err)
	require.NotNil(t, pinnedProfile)

	router := newCustomerBillingRouter(t, env, namespace)

	getStripeData := func(ctx context.Context, cus *customer.Customer) (appstripe.CustomerData, error) {
		return env.AppStripe().GetStripeCustomerData(ctx, appstripe.GetStripeCustomerDataInput{
			AppID:      stripeApp.GetID(),
			CustomerID: cus.GetID(),
		})
	}

	// Stripe customer IDs are unique per app, so every seeding gets its own.
	seedStripeData := func(ctx context.Context, t *testing.T, cus *customer.Customer, stripeCustomerID string) {
		t.Helper()

		env.StripeAppClient().
			On("GetCustomer", stripeCustomerID).
			Return(stripeclient.StripeCustomer{StripeCustomerID: stripeCustomerID}, nil)
		defer env.StripeAppClient().Restore()

		require.NoError(t, stripeApp.UpsertCustomerData(ctx, app.UpsertAppInstanceCustomerDataInput{
			CustomerID: cus.GetID(),
			Data:       appstripe.CustomerData{StripeCustomerID: stripeCustomerID},
		}))
	}

	t.Run("Should set billing profile without app data", func(t *testing.T) {
		ctx := t.Context()

		// given a customer with no stripe app data
		cus, err := env.Fixture().setupCustomer(ctx, namespace)
		require.NoError(t, err)

		// when updating billing with only a billing profile reference
		rec := doJSONRequest(router, http.MethodPut, "/openmeter/customers/"+cus.ID+"/billing",
			`{"billing_profile":{"id":"`+pinnedProfile.ID+`"}}`)

		// then the profile is pinned and no app data is created
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

		var got api.BillingCustomerData
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		require.Equal(t, api.BillingProfileReference{Id: pinnedProfile.ID}, got.EffectiveBillingProfile)
		require.Equal(t, &api.BillingProfileReference{Id: pinnedProfile.ID}, got.BillingProfile)

		override, err := env.Billing().GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
			Customer: cus.GetID(),
		})
		require.NoError(t, err)
		require.Equal(t, pinnedProfile.ID, override.MergedProfile.ID)

		_, err = getStripeData(ctx, cus)
		require.Error(t, err, "no stripe customer data should have been created")
	})

	for _, tc := range []struct {
		name     string
		override *billing.UpsertCustomerOverrideInput
	}{
		{
			name: "Should return the effective default profile on update without a pin",
		},
		{
			name: "Should omit billing profile on update with only workflow overrides",
			override: &billing.UpsertCustomerOverrideInput{
				Invoicing: billing.InvoicingOverrideConfig{
					AutoAdvance: lo.ToPtr(false),
				},
			},
		},
		{
			name: "Should return an existing pin to the default profile when omitted on update",
			override: &billing.UpsertCustomerOverrideInput{
				ProfileID: defaultProfile.ID,
			},
		},
		{
			name: "Should retain an existing pin as the effective profile when omitted on update",
			override: &billing.UpsertCustomerOverrideInput{
				ProfileID: pinnedProfile.ID,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()

			// given a customer with the specified profile or workflow override
			cus, err := env.Fixture().setupCustomer(ctx, namespace)
			require.NoError(t, err)

			if tc.override != nil {
				input := *tc.override
				input.Namespace = namespace
				input.CustomerID = cus.ID
				_, err := env.Billing().UpsertCustomerOverride(ctx, input)
				require.NoError(t, err)
			}

			// when updating billing without specifying a profile
			rec := doJSONRequest(router, http.MethodPut, "/openmeter/customers/"+cus.ID+"/billing", `{}`)

			// then PUT reports the same explicit pin and effective profile as GET
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

			var got api.BillingCustomerData
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
			expectedProfileID := defaultProfile.ID
			if tc.override != nil && tc.override.ProfileID != "" {
				expectedProfileID = tc.override.ProfileID
				require.Equal(t, &api.BillingProfileReference{Id: expectedProfileID}, got.BillingProfile)
			} else {
				require.Nil(t, got.BillingProfile)

				var fields map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &fields))
				require.NotContains(t, fields, "billing_profile")
			}

			require.Equal(t, api.BillingProfileReference{Id: expectedProfileID}, got.EffectiveBillingProfile)

			getRec := doJSONRequest(router, http.MethodGet, "/openmeter/customers/"+cus.ID+"/billing", "")
			require.Equal(t, http.StatusOK, getRec.Code, "body: %s", getRec.Body.String())

			var getResponse api.BillingCustomerData
			require.NoError(t, json.Unmarshal(getRec.Body.Bytes(), &getResponse))
			require.Equal(t, getResponse.BillingProfile, got.BillingProfile)
			require.Equal(t, getResponse.EffectiveBillingProfile, got.EffectiveBillingProfile)

			override, err := env.Billing().GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
				Customer: cus.GetID(),
			})
			require.NoError(t, err)
			require.Equal(t, expectedProfileID, override.MergedProfile.ID)
			if tc.override == nil {
				require.Nil(t, override.CustomerOverride)
			}
		})
	}

	t.Run("Should update the pinned payment app when the billing profile is omitted", func(t *testing.T) {
		ctx := t.Context()

		// given a customer pinned to a different payment app with data in the default app
		cus, err := env.Fixture().setupCustomer(ctx, namespace)
		require.NoError(t, err)

		_, err = env.Billing().UpsertCustomerOverride(ctx, billing.UpsertCustomerOverrideInput{
			Namespace:  namespace,
			CustomerID: cus.ID,
			ProfileID:  pinnedProfile.ID,
		})
		require.NoError(t, err)

		seedStripeData(ctx, t, cus, "cus_default_unchanged")

		env.StripeAppClient().
			On("GetCustomer", "cus_pinned_update").
			Return(stripeclient.StripeCustomer{StripeCustomerID: "cus_pinned_update"}, nil)
		defer env.StripeAppClient().Restore()

		// when updating app data without specifying the billing profile
		rec := doJSONRequest(router, http.MethodPut, "/openmeter/customers/"+cus.ID+"/billing",
			`{"app_data":{"stripe":{"customer_id":"cus_pinned_update"}}}`)
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

		// then the write targets the pinned app and GET returns the updated data
		pinnedData, err := env.AppStripe().GetStripeCustomerData(ctx, appstripe.GetStripeCustomerDataInput{
			AppID:      pinnedStripeApp.GetID(),
			CustomerID: cus.GetID(),
		})
		require.NoError(t, err)
		require.Equal(t, "cus_pinned_update", pinnedData.StripeCustomerID)

		defaultData, err := getStripeData(ctx, cus)
		require.NoError(t, err)
		require.Equal(t, "cus_default_unchanged", defaultData.StripeCustomerID)

		getRec := doJSONRequest(router, http.MethodGet, "/openmeter/customers/"+cus.ID+"/billing", "")
		require.Equal(t, http.StatusOK, getRec.Code, "body: %s", getRec.Body.String())

		var got api.BillingCustomerData
		require.NoError(t, json.Unmarshal(getRec.Body.Bytes(), &got))
		require.Equal(t, &api.BillingProfileReference{Id: pinnedProfile.ID}, got.BillingProfile)
		require.Equal(t, api.BillingProfileReference{Id: pinnedProfile.ID}, got.EffectiveBillingProfile)
		require.NotNil(t, got.AppData)
		stripeData, err := got.AppData.Stripe.Get()
		require.NoError(t, err)
		require.Equal(t, lo.ToPtr("cus_pinned_update"), stripeData.CustomerId)
	})

	t.Run("Should reject stripe app data without customer id", func(t *testing.T) {
		ctx := t.Context()

		cus, err := env.Fixture().setupCustomer(ctx, namespace)
		require.NoError(t, err)

		rec := doJSONRequest(router, http.MethodPut, "/openmeter/customers/"+cus.ID+"/billing",
			`{"app_data":{"stripe":{}}}`)

		require.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())
		require.Contains(t, rec.Body.String(), "stripe customer id is required")
	})

	t.Run("Should upsert stripe app data", func(t *testing.T) {
		ctx := t.Context()

		cus, err := env.Fixture().setupCustomer(ctx, namespace)
		require.NoError(t, err)

		env.StripeAppClient().
			On("GetCustomer", "cus_upsert").
			Return(stripeclient.StripeCustomer{StripeCustomerID: "cus_upsert"}, nil)
		defer env.StripeAppClient().Restore()

		rec := doJSONRequest(router, http.MethodPut, "/openmeter/customers/"+cus.ID+"/billing",
			`{"app_data":{"stripe":{"customer_id":"cus_upsert"}}}`)

		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

		data, err := getStripeData(ctx, cus)
		require.NoError(t, err)
		require.Equal(t, "cus_upsert", data.StripeCustomerID)
	})

	t.Run("Should leave stripe app data unchanged when stripe is omitted", func(t *testing.T) {
		ctx := t.Context()

		// given a customer with existing stripe app data
		cus, err := env.Fixture().setupCustomer(ctx, namespace)
		require.NoError(t, err)

		seedStripeData(ctx, t, cus, "cus_keep_billing")

		// when updating billing with an app data object that omits stripe
		rec := doJSONRequest(router, http.MethodPut, "/openmeter/customers/"+cus.ID+"/billing",
			`{"app_data":{}}`)

		// then the request is a no-op and the stored stripe data stays
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

		data, err := getStripeData(ctx, cus)
		require.NoError(t, err)
		require.Equal(t, "cus_keep_billing", data.StripeCustomerID)
	})

	t.Run("Should leave stripe app data unchanged when stripe is omitted on app-data endpoint", func(t *testing.T) {
		ctx := t.Context()

		// given a customer with existing stripe app data
		cus, err := env.Fixture().setupCustomer(ctx, namespace)
		require.NoError(t, err)

		seedStripeData(ctx, t, cus, "cus_keep_app_data")

		// when updating app data with an empty object that omits stripe
		rec := doJSONRequest(router, http.MethodPut, "/openmeter/customers/"+cus.ID+"/billing/app-data",
			`{}`)

		// then the request is a no-op and the stored stripe data stays
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

		data, err := getStripeData(ctx, cus)
		require.NoError(t, err)
		require.Equal(t, "cus_keep_app_data", data.StripeCustomerID)
	})

	t.Run("Should wipe stripe app data with explicit null", func(t *testing.T) {
		ctx := t.Context()

		// given a customer with existing stripe app data
		cus, err := env.Fixture().setupCustomer(ctx, namespace)
		require.NoError(t, err)

		seedStripeData(ctx, t, cus, "cus_wipe_billing")

		// when updating billing with an explicit stripe null
		rec := doJSONRequest(router, http.MethodPut, "/openmeter/customers/"+cus.ID+"/billing",
			`{"app_data":{"stripe":null}}`)

		// then the stripe customer data is deleted
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

		_, err = getStripeData(ctx, cus)
		require.Error(t, err, "stripe customer data should have been wiped")
	})

	t.Run("Should wipe stripe app data with explicit null on app-data endpoint", func(t *testing.T) {
		ctx := t.Context()

		cus, err := env.Fixture().setupCustomer(ctx, namespace)
		require.NoError(t, err)

		seedStripeData(ctx, t, cus, "cus_wipe_app_data")

		rec := doJSONRequest(router, http.MethodPut, "/openmeter/customers/"+cus.ID+"/billing/app-data",
			`{"stripe":null}`)

		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

		_, err = getStripeData(ctx, cus)
		require.Error(t, err, "stripe customer data should have been wiped")
	})

	for _, tc := range []struct {
		name            string
		override        *billing.UpsertCustomerOverrideInput
		expectedProfile *api.BillingProfileReference
	}{
		{
			name: "Should omit billing profile on get without an override",
		},
		{
			name: "Should omit billing profile on get with only workflow overrides",
			override: &billing.UpsertCustomerOverrideInput{
				Invoicing: billing.InvoicingOverrideConfig{
					AutoAdvance: lo.ToPtr(false),
				},
			},
		},
		{
			name: "Should return an explicit pin to the default profile on get",
			override: &billing.UpsertCustomerOverrideInput{
				ProfileID: defaultProfile.ID,
			},
			expectedProfile: &api.BillingProfileReference{Id: defaultProfile.ID},
		},
		{
			name: "Should return an explicit pin to another profile on get",
			override: &billing.UpsertCustomerOverrideInput{
				ProfileID: pinnedProfile.ID,
			},
			expectedProfile: &api.BillingProfileReference{Id: pinnedProfile.ID},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()

			// given a customer with the specified profile or workflow override
			cus, err := env.Fixture().setupCustomer(ctx, namespace)
			require.NoError(t, err)

			if tc.override != nil {
				input := *tc.override
				input.Namespace = namespace
				input.CustomerID = cus.ID
				_, err := env.Billing().UpsertCustomerOverride(ctx, input)
				require.NoError(t, err)
			}

			// when reading the customer billing without app data configured
			rec := doJSONRequest(router, http.MethodGet, "/openmeter/customers/"+cus.ID+"/billing", "")

			// then only explicit pins are returned as billing_profile
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

			var got api.BillingCustomerData
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
			require.Equal(t, tc.expectedProfile, got.BillingProfile)

			expectedEffectiveProfile := api.BillingProfileReference{Id: defaultProfile.ID}
			if tc.expectedProfile != nil {
				expectedEffectiveProfile = *tc.expectedProfile
			}

			require.Equal(t, expectedEffectiveProfile, got.EffectiveBillingProfile)

			var fields map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &fields))
			require.Contains(t, fields, "effective_billing_profile")
			if tc.expectedProfile == nil {
				require.NotContains(t, fields, "billing_profile")
			}
		})
	}

	t.Run("Should return stripe app data on get", func(t *testing.T) {
		ctx := t.Context()

		// given a customer with existing stripe app data
		cus, err := env.Fixture().setupCustomer(ctx, namespace)
		require.NoError(t, err)

		seedStripeData(ctx, t, cus, "cus_get_billing")

		// when reading the customer billing
		rec := doJSONRequest(router, http.MethodGet, "/openmeter/customers/"+cus.ID+"/billing", "")

		// then the response returns the stripe customer id
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

		var got api.BillingCustomerData
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		require.NotNil(t, got.AppData)
		require.True(t, got.AppData.Stripe.IsSpecified())
		require.False(t, got.AppData.Stripe.IsNull())

		stripeData, err := got.AppData.Stripe.Get()
		require.NoError(t, err)
		require.NotNil(t, stripeData.CustomerId)
		require.Equal(t, "cus_get_billing", *stripeData.CustomerId)
	})

	t.Run("Should return null stripe app data on get after a wipe", func(t *testing.T) {
		ctx := t.Context()

		// given a customer whose stripe app data was wiped while the profile
		// still names stripe as the payment app
		cus, err := env.Fixture().setupCustomer(ctx, namespace)
		require.NoError(t, err)

		seedStripeData(ctx, t, cus, "cus_get_after_wipe")

		rec := doJSONRequest(router, http.MethodPut, "/openmeter/customers/"+cus.ID+"/billing",
			`{"app_data":{"stripe":null}}`)
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

		// when reading the customer billing
		rec = doJSONRequest(router, http.MethodGet, "/openmeter/customers/"+cus.ID+"/billing", "")

		// then the read succeeds and reports the stripe app data as explicit null
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

		var got api.BillingCustomerData
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		require.NotNil(t, got.AppData)
		require.True(t, got.AppData.Stripe.IsSpecified())
		require.True(t, got.AppData.Stripe.IsNull())
	})
}

func newCustomerBillingRouter(t *testing.T, env TestEnv, namespace string) http.Handler {
	t.Helper()

	doc, err := api.GetSpec()
	require.NoError(t, err)

	validationRouter, err := oasmiddleware.NewValidationRouter(t.Context(), doc, &oasmiddleware.ValidationRouterOpts{
		DeleteServers: true,
	})
	require.NoError(t, err)

	handler := customersbilling.New(
		func(ctx context.Context) (string, error) { return namespace, nil },
		env.Billing(),
		env.Customer(),
		env.AppStripe(),
	)

	router := chi.NewRouter()
	router.Use(oasmiddleware.ValidateRequest(validationRouter, oasmiddleware.ValidateRequestOption{
		RouteNotFoundHook: oasmiddleware.OasRouteNotFoundErrorHook,
		RouteValidationErrorHook: func(err error, w http.ResponseWriter, request *http.Request) bool {
			return oasmiddleware.OasValidationErrorHook(request.Context(), err, w, request)
		},
		FilterOptions: &openapi3filter.Options{
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
			MultiError:         true,
		},
	}))
	router.Put("/openmeter/customers/{customerId}/billing", func(w http.ResponseWriter, r *http.Request) {
		handler.UpdateCustomerBilling().With(chi.URLParam(r, "customerId")).ServeHTTP(w, r)
	})
	router.Put("/openmeter/customers/{customerId}/billing/app-data", func(w http.ResponseWriter, r *http.Request) {
		handler.UpdateCustomerBillingAppData().With(chi.URLParam(r, "customerId")).ServeHTTP(w, r)
	})
	router.Get("/openmeter/customers/{customerId}/billing", func(w http.ResponseWriter, r *http.Request) {
		handler.GetCustomerBilling().With(chi.URLParam(r, "customerId")).ServeHTTP(w, r)
	})

	return router
}

func doJSONRequest(router http.Handler, method, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}

func newTestProfileInput(namespace string, appID app.AppID, name string, isDefault bool) billing.CreateProfileInput {
	return billing.CreateProfileInput{
		Namespace: namespace,
		Name:      name,
		Default:   isDefault,

		WorkflowConfig: billing.WorkflowConfig{
			Collection: billing.CollectionConfig{
				Alignment: billing.AlignmentKindSubscription,
				Interval:  lo.Must(datetime.ISODurationString("PT0S").Parse()),
			},
			Invoicing: billing.InvoicingConfig{
				AutoAdvance:                  true,
				DraftPeriod:                  lo.Must(datetime.ISODurationString("P1D").Parse()),
				DueAfter:                     lo.Must(datetime.ISODurationString("P1W").Parse()),
				SubscriptionEndProrationMode: billing.SubscriptionEndProrationModeBillActualPeriod,
			},
			Payment: billing.PaymentConfig{
				CollectionMethod: billing.CollectionMethodChargeAutomatically,
			},
			Tax: billing.WorkflowTaxConfig{
				Enabled:  true,
				Enforced: false,
			},
		},

		Supplier: billing.SupplierContact{
			Name: "Awesome Supplier",
			Address: models.Address{
				Country: lo.ToPtr(models.CountryCode("US")),
			},
		},

		Apps: billing.CreateProfileAppsInput{
			Invoicing: appID,
			Payment:   appID,
			Tax:       appID,
		},
	}
}
