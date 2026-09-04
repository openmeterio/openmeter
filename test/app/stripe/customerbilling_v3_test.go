package appstripe

import (
	"context"
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

// TestCustomerBillingV3 exercises the v3 customer billing PUT endpoints through
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

	pinnedProfile, err := env.Billing().CreateProfile(ctx, newTestProfileInput(namespace, stripeApp.GetID(), "Pinned Profile", false))
	require.NoError(t, err)
	require.NotNil(t, pinnedProfile)

	router := newCustomerBillingRouter(t, env, namespace)

	getStripeData := func(cus *customer.Customer) (appstripe.CustomerData, error) {
		return env.AppStripe().GetStripeCustomerData(ctx, appstripe.GetStripeCustomerDataInput{
			AppID:      stripeApp.GetID(),
			CustomerID: cus.GetID(),
		})
	}

	// Stripe customer IDs are unique per app, so every seeding gets its own.
	seedStripeData := func(t *testing.T, cus *customer.Customer, stripeCustomerID string) {
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
		// given a customer with no stripe app data
		cus, err := env.Fixture().setupCustomer(ctx, namespace)
		require.NoError(t, err)

		// when updating billing with only a billing profile reference
		rec := doJSONRequest(router, http.MethodPut, "/openmeter/customers/"+cus.ID+"/billing",
			`{"billing_profile":{"id":"`+pinnedProfile.ID+`"}}`)

		// then the profile is pinned and no app data is created
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

		override, err := env.Billing().GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
			Customer: cus.GetID(),
		})
		require.NoError(t, err)
		require.Equal(t, pinnedProfile.ID, override.MergedProfile.ID)

		_, err = getStripeData(cus)
		require.Error(t, err, "no stripe customer data should have been created")
	})

	t.Run("Should reject stripe app data without customer id", func(t *testing.T) {
		cus, err := env.Fixture().setupCustomer(ctx, namespace)
		require.NoError(t, err)

		rec := doJSONRequest(router, http.MethodPut, "/openmeter/customers/"+cus.ID+"/billing",
			`{"app_data":{"stripe":{}}}`)

		require.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())
		require.Contains(t, rec.Body.String(), "stripe customer id is required")
	})

	t.Run("Should upsert stripe app data", func(t *testing.T) {
		cus, err := env.Fixture().setupCustomer(ctx, namespace)
		require.NoError(t, err)

		env.StripeAppClient().
			On("GetCustomer", "cus_upsert").
			Return(stripeclient.StripeCustomer{StripeCustomerID: "cus_upsert"}, nil)
		defer env.StripeAppClient().Restore()

		rec := doJSONRequest(router, http.MethodPut, "/openmeter/customers/"+cus.ID+"/billing",
			`{"app_data":{"stripe":{"customer_id":"cus_upsert"}}}`)

		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

		data, err := getStripeData(cus)
		require.NoError(t, err)
		require.Equal(t, "cus_upsert", data.StripeCustomerID)
	})

	t.Run("Should leave stripe app data unchanged when stripe is omitted", func(t *testing.T) {
		// given a customer with existing stripe app data
		cus, err := env.Fixture().setupCustomer(ctx, namespace)
		require.NoError(t, err)

		seedStripeData(t, cus, "cus_keep_billing")

		// when updating billing with an app data object that omits stripe
		rec := doJSONRequest(router, http.MethodPut, "/openmeter/customers/"+cus.ID+"/billing",
			`{"app_data":{}}`)

		// then the request is a no-op and the stored stripe data stays
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

		data, err := getStripeData(cus)
		require.NoError(t, err)
		require.Equal(t, "cus_keep_billing", data.StripeCustomerID)
	})

	t.Run("Should leave stripe app data unchanged when stripe is omitted on app-data endpoint", func(t *testing.T) {
		// given a customer with existing stripe app data
		cus, err := env.Fixture().setupCustomer(ctx, namespace)
		require.NoError(t, err)

		seedStripeData(t, cus, "cus_keep_app_data")

		// when updating app data with an empty object that omits stripe
		rec := doJSONRequest(router, http.MethodPut, "/openmeter/customers/"+cus.ID+"/billing/app-data",
			`{}`)

		// then the request is a no-op and the stored stripe data stays
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

		data, err := getStripeData(cus)
		require.NoError(t, err)
		require.Equal(t, "cus_keep_app_data", data.StripeCustomerID)
	})

	t.Run("Should wipe stripe app data with explicit null", func(t *testing.T) {
		// given a customer with existing stripe app data
		cus, err := env.Fixture().setupCustomer(ctx, namespace)
		require.NoError(t, err)

		seedStripeData(t, cus, "cus_wipe_billing")

		// when updating billing with an explicit stripe null
		rec := doJSONRequest(router, http.MethodPut, "/openmeter/customers/"+cus.ID+"/billing",
			`{"app_data":{"stripe":null}}`)

		// then the stripe customer data is deleted
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

		_, err = getStripeData(cus)
		require.Error(t, err, "stripe customer data should have been wiped")
	})

	t.Run("Should wipe stripe app data with explicit null on app-data endpoint", func(t *testing.T) {
		cus, err := env.Fixture().setupCustomer(ctx, namespace)
		require.NoError(t, err)

		seedStripeData(t, cus, "cus_wipe_app_data")

		rec := doJSONRequest(router, http.MethodPut, "/openmeter/customers/"+cus.ID+"/billing/app-data",
			`{"stripe":null}`)

		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

		_, err = getStripeData(cus)
		require.Error(t, err, "stripe customer data should have been wiped")
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
