package appstripe

import (
	"context"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v80"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	stripeclient "github.com/openmeterio/openmeter/openmeter/appstripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
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

	app, err := s.Env.App().InstallAppWithAPIKey(ctx, appentity.InstallAppWithAPIKeyInput{
		MarketplaceListingID: appentity.MarketplaceListingID{
			Type: appentitybase.AppTypeStripe,
		},

		Namespace: s.namespace,
		APIKey:    TestStripeAPIKey,
	})

	require.NoError(t, err, "Create stripe app must not return error")
	require.NotNil(t, app, "Create stripe app must return app")
}

// TestCustomerCreate tests stripe app behavior when creating a new customer
func (s *AppHandlerTestSuite) TestCustomerCreate(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	// Create a stripe app first
	app, err := s.Env.App().InstallAppWithAPIKey(ctx, appentity.InstallAppWithAPIKeyInput{
		MarketplaceListingID: appentity.MarketplaceListingID{
			Type: appentitybase.AppTypeStripe,
		},

		Namespace: s.namespace,
		APIKey:    TestStripeAPIKey,
	})

	require.NoError(t, err, "Create stripe app must not return error")
	require.NotNil(t, app, "Create stripe app must return app")

	// Create a customer
	customer, err := s.Env.Customer().CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: s.namespace,
		Customer: customerentity.Customer{
			Name: "Test Customer",
			Apps: []customerentity.CustomerApp{
				{
					Type: appentitybase.AppTypeStripe,
					Data: appstripeentity.CustomerAppData{
						StripeCustomerID: "cus_123",
					},
				},
			},
		},
	})

	require.NoError(t, err, "Create customer must not return error")
	require.NotNil(t, customer, "Create customer must return customer")
}

// TestCustomerValidate tests stripe app behavior when validating a customer
func (s *AppHandlerTestSuite) TestCustomerValidate(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	// Create a stripe app first
	app, err := s.Env.App().InstallAppWithAPIKey(ctx, appentity.InstallAppWithAPIKeyInput{
		MarketplaceListingID: appentity.MarketplaceListingID{
			Type: appentitybase.AppTypeStripe,
		},

		Namespace: s.namespace,
		APIKey:    TestStripeAPIKey,
	})

	require.NoError(t, err, "Create stripe app must not return error")
	require.NotNil(t, app, "Create stripe app must return app")

	// Create test customers
	customer, err := s.Env.Customer().CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: s.namespace,
		Customer: customerentity.Customer{
			Name: "Test Customer",
			Apps: []customerentity.CustomerApp{
				{
					Type: appentitybase.AppTypeStripe,
					Data: appstripeentity.CustomerAppData{
						StripeCustomerID: "cus_123",
					},
				},
			},
		},
	})

	require.NoError(t, err, "Create customer must not return error")
	require.NotNil(t, customer, "Create customer must return customer")

	customerWithoutStripeData, err := s.Env.Customer().CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: s.namespace,
		Customer: customerentity.Customer{
			Name: "Test Customer Without Stripe",
		},
	})

	require.NoError(t, err, "Create customer must not return error")
	require.NotNil(t, customerWithoutStripeData, "Create customer must return customer")

	// Get App
	getApp, err := s.Env.App().GetApp(ctx, app.GetID())

	require.NoError(t, err, "Get app must not return error")

	// Generic app should validate the customer
	err = getApp.ValidateCustomer(ctx, customer, []appentitybase.CapabilityType{appentitybase.CapabilityTypeCalculateTax})
	require.NoError(t, err, "Validate customer must not return error")

	// Stripe app should validate the customer
	err = app.ValidateCustomer(ctx, customer, []appentitybase.CapabilityType{appentitybase.CapabilityTypeCalculateTax})
	require.NoError(t, err, "Validate customer must not return error")

	// Validate the customer with an invalid capability
	err = getApp.ValidateCustomer(ctx, customer, []appentitybase.CapabilityType{appentitybase.CapabilityTypeReportEvents})
	require.ErrorContains(t, err, "capability reportEvents is not supported", "Validate customer must return error")

	// Validate the customer without stripe data
	err = getApp.ValidateCustomer(ctx, customerWithoutStripeData, []appentitybase.CapabilityType{appentitybase.CapabilityTypeCalculateTax})
	require.ErrorContains(t, err, "customer has no data", "Validate customer must return error")
}

// TestCreateCheckoutSession tests stripe app behavior when creating a new checkout session
func (s *AppHandlerTestSuite) TestCreateCheckoutSession(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	// Create a stripe app first
	app, err := s.Env.App().InstallAppWithAPIKey(ctx, appentity.InstallAppWithAPIKeyInput{
		MarketplaceListingID: appentity.MarketplaceListingID{
			Type: appentitybase.AppTypeStripe,
		},

		Namespace: s.namespace,
		APIKey:    TestStripeAPIKey,
	})

	require.NoError(t, err, "Create stripe app must not return error")
	require.NotNil(t, app, "Create stripe app must return app")

	// Create test customers
	customer, err := s.Env.Customer().CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: s.namespace,
		Customer: customerentity.Customer{
			Name: "Test Customer",
			Apps: []customerentity.CustomerApp{
				{
					Type: appentitybase.AppTypeStripe,
					Data: appstripeentity.CustomerAppData{
						StripeCustomerID: "cus_123",
					},
				},
			},
		},
	})

	require.NoError(t, err, "Create customer must not return error")
	require.NotNil(t, customer, "Create customer must return customer")

	checkoutSession, err := s.Env.AppStripe().CreateCheckoutSession(ctx, appstripeentity.CreateCheckoutSessionInput{
		AppID:      app.GetID(),
		CustomerID: customer.GetID(),
		Options:    stripeclient.StripeCheckoutSessionOptions{},
	})

	require.NoError(t, err, "Create checkout session must not return error")

	require.Equal(t, stripeclient.StripeCheckoutSession{
		SessionID:     "cs_123",
		SetupIntentID: "seti_123",
		Mode:          stripe.CheckoutSessionModeSetup,
		URL:           "https://checkout.stripe.com/cs_123/test",
	}, checkoutSession, "Create checkout session must match")
}
