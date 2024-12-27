package appstripe

import (
	"context"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v80"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	customerapp "github.com/openmeterio/openmeter/openmeter/customer/app"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
)

var TestStripeAPIKey = "test_stripe_api_key"

type AppHandlerTestSuite struct {
	Env TestEnv

	namespace string
}

// setupNamespace can be used to set up an independent namespace for testing, it contains a single
// feature and rule with a channel. For more complex scenarios, additional setup might be required.
func (s *AppHandlerTestSuite) setupNamespace(t *testing.T) {
	t.Helper()

	s.namespace = ulid.Make().String()
}

// TestCreate tests to create a new stripe app
func (s *AppHandlerTestSuite) TestCreate(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	app, err := s.Env.App().InstallMarketplaceListingWithAPIKey(ctx, appentity.InstallAppWithAPIKeyInput{
		MarketplaceListingID: appentity.MarketplaceListingID{
			Type: appentitybase.AppTypeStripe,
		},

		Namespace: s.namespace,
		APIKey:    TestStripeAPIKey,
	})

	require.NoError(t, err, "Create stripe app must not return error")
	require.NotNil(t, app, "Create stripe app must return app")
}

// TestGet tests getting a stripe app
func (s *AppHandlerTestSuite) TestGet(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	// Create a stripe app first
	createApp, err := s.Env.App().InstallMarketplaceListingWithAPIKey(ctx, appentity.InstallAppWithAPIKeyInput{
		MarketplaceListingID: appentity.MarketplaceListingID{
			Type: appentitybase.AppTypeStripe,
		},

		Namespace: s.namespace,
		APIKey:    TestStripeAPIKey,
	})

	require.NoError(t, err, "Create stripe app must not return error")
	require.NotNil(t, createApp, "Create stripe app must return app")

	// Get the app
	getApp, err := s.Env.App().GetApp(ctx, createApp.GetID())

	require.NoError(t, err, "Get stripe app must not return error")
	require.Equal(t, createApp.GetID(), getApp.GetID(), "apps must be equal")

	// Get should return 404
	appIdNotFound := appentitybase.AppID{
		Namespace: s.namespace,
		ID:        "not_found",
	}

	_, err = s.Env.App().GetApp(ctx, appIdNotFound)
	require.ErrorIs(t, err, app.AppNotFoundError{AppID: appIdNotFound}, "must return app not found error")
}

// TestGetDefault tests getting the default stripe app
func (s *AppHandlerTestSuite) TestGetDefault(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	// Create a stripe app first
	createApp1, err := s.Env.App().InstallMarketplaceListingWithAPIKey(ctx, appentity.InstallAppWithAPIKeyInput{
		MarketplaceListingID: appentity.MarketplaceListingID{
			Type: appentitybase.AppTypeStripe,
		},

		Namespace: s.namespace,
		APIKey:    TestStripeAPIKey,
	})

	require.NoError(t, err, "Create stripe app must not return error")
	require.NotNil(t, createApp1, "Create stripe app must return app")

	createApp2, err := s.Env.App().InstallMarketplaceListingWithAPIKey(ctx, appentity.InstallAppWithAPIKeyInput{
		MarketplaceListingID: appentity.MarketplaceListingID{
			Type: appentitybase.AppTypeStripe,
		},

		Namespace: s.namespace,
		APIKey:    TestStripeAPIKey,
	})

	require.NoError(t, err, "Create stripe app must not return error")
	require.NotNil(t, createApp2, "Create stripe app must return app")

	// Get the app
	getApp, err := s.Env.App().GetDefaultApp(ctx, appentity.GetDefaultAppInput{
		Namespace: s.namespace,
		Type:      appentitybase.AppTypeStripe,
	})

	require.NoError(t, err, "Get default stripe app must not return error")
	require.Equal(t, createApp1.GetID(), getApp.GetID(), "apps must be equal with first")
}

// TestUninstall tests uninstalling a stripe app
func (s *AppHandlerTestSuite) TestUninstall(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	// Create a stripe app first
	createApp, err := s.Env.App().InstallMarketplaceListingWithAPIKey(ctx, appentity.InstallAppWithAPIKeyInput{
		MarketplaceListingID: appentity.MarketplaceListingID{
			Type: appentitybase.AppTypeStripe,
		},

		Namespace: s.namespace,
		APIKey:    TestStripeAPIKey,
	})

	require.NoError(t, err, "Create stripe app must not return error")
	require.NotNil(t, createApp, "Create stripe app must return app")

	// Uninstall the app
	err = s.Env.App().UninstallApp(ctx, createApp.GetID())

	require.NoError(t, err, "Uninstall stripe app must not return error")

	// Get should return 404
	_, err = s.Env.App().GetApp(ctx, createApp.GetID())
	require.ErrorIs(t, err, app.AppNotFoundError{AppID: createApp.GetID()}, "get after uninstall must return app not found error")
}

// TestCustomerData tests stripe app behavior when adding customer data
func (s *AppHandlerTestSuite) TestCustomerData(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	testApp, customer, customerData, err := s.Env.Fixture().setupAppWithCustomer(ctx, s.namespace)
	require.NoError(t, err, "setup fixture must not return error")

	// Get customer data
	getCustomerData, err := testApp.GetCustomerData(ctx, appentity.GetAppInstanceCustomerDataInput{
		CustomerID: customer.GetID(),
	})

	require.NoError(t, err, "Get customer data must not return error")
	require.Equal(t, appstripeentity.CustomerData{
		StripeCustomerID: customerData.StripeCustomerID,
	}, getCustomerData, "Customer data must match")

	// Update customer data
	err = testApp.UpsertCustomerData(ctx, appentity.UpsertAppInstanceCustomerDataInput{
		CustomerID: customer.GetID(),
		Data: appstripeentity.CustomerData{
			StripeCustomerID: "cus_456",
		},
	})

	require.NoError(t, err, "Update customer data must not return error")

	// Updated customer data must match
	getCustomerData, err = testApp.GetCustomerData(ctx, appentity.GetAppInstanceCustomerDataInput{
		CustomerID: customer.GetID(),
	})

	require.NoError(t, err, "Get customer data must not return error")
	require.Equal(t, appstripeentity.CustomerData{
		StripeCustomerID: "cus_456",
	}, getCustomerData, "Customer data must match")

	// Delete customer data
	err = testApp.DeleteCustomerData(ctx, appentity.DeleteAppInstanceCustomerDataInput{
		CustomerID: customer.GetID(),
	})

	require.NoError(t, err, "Delete customer data must not return error")

	// Get customer data should return 404
	_, err = testApp.GetCustomerData(ctx, appentity.GetAppInstanceCustomerDataInput{
		CustomerID: customer.GetID(),
	})

	require.ErrorIs(t, err, app.CustomerPreConditionError{
		AppID:      testApp.GetID(),
		AppType:    appentitybase.AppTypeStripe,
		CustomerID: customer.GetID(),
		Condition:  "customer has no data for stripe app",
	})
}

// TestCustomerValidate tests stripe app behavior when validating a customer
func (s *AppHandlerTestSuite) TestCustomerValidate(ctx context.Context, t *testing.T) {
	app, customer, _, err := s.Env.Fixture().setupAppWithCustomer(ctx, s.namespace)
	require.NoError(t, err, "setup fixture must not return error")

	// Create customer without stripe data
	customerWithoutStripeData, err := s.Env.Customer().CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: s.namespace,
		CustomerMutate: customerentity.CustomerMutate{
			Name: "Test Customer Without Stripe",
		},
	})

	require.NoError(t, err, "Create customer must not return error")
	require.NotNil(t, customerWithoutStripeData, "Create customer must return customer")

	// Get App
	getApp, err := s.Env.App().GetApp(ctx, app.GetID())

	require.NoError(t, err, "Get app must not return error")

	// App should implement Customer App
	customerApp, err := customerapp.GetApp(getApp)

	require.NoError(t, err, "Get app must not return error")

	// App should validate the customer
	err = customerApp.ValidateCustomer(ctx, customer, []appentitybase.CapabilityType{appentitybase.CapabilityTypeCalculateTax})
	require.NoError(t, err, "Validate customer must not return error")

	// Validate the customer with an invalid capability
	err = customerApp.ValidateCustomer(ctx, customer, []appentitybase.CapabilityType{appentitybase.CapabilityTypeReportEvents})
	require.ErrorContains(t, err, "capability reportEvents is not supported", "Validate customer must return error")

	// Validate the customer without stripe data
	err = customerApp.ValidateCustomer(ctx, customerWithoutStripeData, []appentitybase.CapabilityType{appentitybase.CapabilityTypeCalculateTax})
	require.ErrorContains(t, err, "customer has no data", "Validate customer must return error")
}

// TestCreateCheckoutSession tests stripe app behavior when creating a new checkout session
func (s *AppHandlerTestSuite) TestCreateCheckoutSession(ctx context.Context, t *testing.T) {
	app, customer, _, err := s.Env.Fixture().setupAppWithCustomer(ctx, s.namespace)
	require.NoError(t, err, "setup fixture must not return error")

	// Create checkout session
	appID := app.GetID()
	customerID := customer.GetID()

	checkoutSession, err := s.Env.AppStripe().CreateCheckoutSession(ctx, appstripeentity.CreateCheckoutSessionInput{
		Namespace:  s.namespace,
		AppID:      &appID,
		CustomerID: &customerID,
		Options:    stripeclient.StripeCheckoutSessionOptions{},
	})

	require.NoError(t, err, "Create checkout session must not return error")

	require.Equal(t, appstripeentity.CreateCheckoutSessionOutput{
		CustomerID:       customer.GetID(),
		StripeCustomerID: "cus_123",
		SessionID:        "cs_123",
		SetupIntentID:    "seti_123",
		Mode:             stripe.CheckoutSessionModeSetup,
		URL:              "https://checkout.stripe.com/cs_123/test",
	}, checkoutSession, "Create checkout session must match")

	// Test app 404 error
	appIdNotFound := appentitybase.AppID{
		Namespace: s.namespace,
		ID:        "not_found",
	}

	_, err = s.Env.AppStripe().CreateCheckoutSession(ctx, appstripeentity.CreateCheckoutSessionInput{
		Namespace:  s.namespace,
		AppID:      &appIdNotFound,
		CustomerID: &customerID,
		Options:    stripeclient.StripeCheckoutSessionOptions{},
	})

	require.ErrorIs(t, err, appstripe.AppNotFoundError{AppID: appIdNotFound}, "Create checkout session must return app not found error")

	// Test customer 404 error
	customerIdNotFound := customerentity.CustomerID{
		Namespace: s.namespace,
		ID:        "not_found",
	}

	_, err = s.Env.AppStripe().CreateCheckoutSession(ctx, appstripeentity.CreateCheckoutSessionInput{
		Namespace:  s.namespace,
		AppID:      &appID,
		CustomerID: &customerIdNotFound,
		Options:    stripeclient.StripeCheckoutSessionOptions{},
	})

	require.ErrorIs(t, err, customerentity.NotFoundError{CustomerID: customerIdNotFound}, "Create checkout session must return customer not found error")
}

// TestUpdateAPIKey tests stripe app behavior when updating the API key
func (s *AppHandlerTestSuite) TestUpdateAPIKey(ctx context.Context, t *testing.T) {
	app, err := s.Env.Fixture().setupApp(ctx, s.namespace)
	require.NoError(t, err, "setup fixture must not return error")

	// Get stripe app
	stripeApp, err := s.Env.AppStripe().GetStripeAppData(ctx, appstripeentity.GetStripeAppDataInput{AppID: app.GetID()})
	require.NoError(t, err, "Get stripe app data must not return error")

	newAPIKey := "sk_test_abcde"

	// Should not allow to update test mode app with livemode key
	err = s.Env.AppStripe().UpdateAPIKey(ctx, appstripeentity.UpdateAPIKeyInput{
		AppID:  app.GetID(),
		APIKey: newAPIKey,
	})

	require.Error(t, err, "Update API key must return error")
	require.Equal(t, err.Error(), "new stripe api key is in test mode but the app is in live mode")

	// Mock the secret service
	s.Env.Secret().EnableMock()
	defer s.Env.Secret().DisableMock()

	newAPIKey = "sk_live_abcde"

	s.Env.Secret().
		On("GetAppSecret", secretentity.GetAppSecretInput{
			AppID: app.GetID(),
			Key:   appstripeentity.APIKeySecretKey,
		}).
		Return(stripeApp.APIKey, nil)

	s.Env.Secret().
		On("UpdateAppSecret", secretentity.UpdateAppSecretInput{
			ID:    stripeApp.APIKey,
			Key:   appstripeentity.APIKeySecretKey,
			Value: newAPIKey,
		}).
		Return(nil)

	// Should allow to update test mode app with test mode key
	err = s.Env.AppStripe().UpdateAPIKey(ctx, appstripeentity.UpdateAPIKeyInput{
		AppID:  app.GetID(),
		APIKey: newAPIKey,
	})

	require.NoError(t, err, "Update API key must not return error")
}
