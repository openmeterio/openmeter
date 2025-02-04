package appstripe

import (
	"context"
	"fmt"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v80"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerapp "github.com/openmeterio/openmeter/openmeter/customer/app"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
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
	stripeAccountID := getStripeAccountId()

	s.setupNamespace(t)

	s.Env.StripeClient().
		On("GetAccount").
		Return(stripeclient.StripeAccount{
			StripeAccountID: stripeAccountID,
		}, nil)

	s.Env.StripeClient().
		On("SetupWebhook", mock.Anything).
		Return(stripeclient.StripeWebhookEndpoint{
			EndpointID: "we_123",
			Secret:     "whsec_123",
		}, nil)

	// TODO: do not share env between tests
	defer s.Env.StripeClient().Restore()

	// Create a stripe app
	createApp, err := s.Env.App().InstallMarketplaceListingWithAPIKey(ctx, app.InstallAppWithAPIKeyInput{
		MarketplaceListingID: app.MarketplaceListingID{
			Type: app.AppTypeStripe,
		},

		Namespace: s.namespace,
		APIKey:    TestStripeAPIKey,
	})

	require.NoError(t, err, "Create stripe app must not return error")
	require.NotNil(t, createApp, "Create stripe app must return app")

	// Create with same Stripe account ID should return conflict
	_, err = s.Env.App().InstallMarketplaceListingWithAPIKey(ctx, app.InstallAppWithAPIKeyInput{
		MarketplaceListingID: app.MarketplaceListingID{
			Type: app.AppTypeStripe,
		},

		Namespace: s.namespace,
		APIKey:    TestStripeAPIKey,
	})

	require.ErrorIs(t, err, app.AppConflictError{
		Namespace: s.namespace,
		Conflict:  fmt.Sprintf("stripe app already exists with stripe account id: %s", stripeAccountID),
	}, "Create stripe app must return conflict error")
}

// TestGet tests getting a stripe app
func (s *AppHandlerTestSuite) TestGet(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	// Create a stripe app first
	createApp, err := s.Env.Fixture().setupApp(ctx, s.namespace)
	require.NoError(t, err, "setup fixture must not return error")

	require.NoError(t, err, "Create stripe app must not return error")
	require.NotNil(t, createApp, "Create stripe app must return app")

	// Get the app
	getApp, err := s.Env.App().GetApp(ctx, createApp.GetID())

	require.NoError(t, err, "Get stripe app must not return error")
	require.Equal(t, createApp.GetID(), getApp.GetID(), "apps must be equal")

	// Get should return 404
	appIdNotFound := app.AppID{
		Namespace: s.namespace,
		ID:        "not_found",
	}

	_, err = s.Env.App().GetApp(ctx, appIdNotFound)
	require.ErrorIs(t, err, app.AppNotFoundError{AppID: appIdNotFound}, "must return app not found error")
}

// TestGetDefault tests getting the default stripe app
func (s *AppHandlerTestSuite) TestGetDefault(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	s.Env.StripeClient().
		On("GetAccount").
		Return(stripeclient.StripeAccount{
			StripeAccountID: getStripeAccountId(),
		}, nil)

	s.Env.StripeClient().
		On("SetupWebhook", mock.Anything).
		Return(stripeclient.StripeWebhookEndpoint{
			EndpointID: "we_123",
			Secret:     "whsec_123",
		}, nil)

	// TODO: do not share env between tests
	defer s.Env.StripeClient().Restore()

	// Create a stripe app first
	createApp1, err := s.Env.App().InstallMarketplaceListingWithAPIKey(ctx, app.InstallAppWithAPIKeyInput{
		MarketplaceListingID: app.MarketplaceListingID{
			Type: app.AppTypeStripe,
		},

		Namespace: s.namespace,
		APIKey:    TestStripeAPIKey,
	})

	require.NoError(t, err, "Create stripe app must not return error")
	require.NotNil(t, createApp1, "Create stripe app must return app")

	// Install with different Stripe account ID
	s.Env.StripeClient().Restore()

	s.Env.StripeClient().
		On("GetAccount").
		Return(stripeclient.StripeAccount{
			StripeAccountID: getStripeAccountId(),
		}, nil)

	s.Env.StripeClient().
		On("SetupWebhook", mock.Anything).
		Return(stripeclient.StripeWebhookEndpoint{
			EndpointID: "we_123",
			Secret:     "whsec_123",
		}, nil)

	createApp2, err := s.Env.App().InstallMarketplaceListingWithAPIKey(ctx, app.InstallAppWithAPIKeyInput{
		MarketplaceListingID: app.MarketplaceListingID{
			Type: app.AppTypeStripe,
		},

		Namespace: s.namespace,
		APIKey:    TestStripeAPIKey,
	})

	require.NoError(t, err, "Create stripe app must not return error")
	require.NotNil(t, createApp2, "Create stripe app must return app")

	// Get the app
	getApp, err := s.Env.App().GetDefaultApp(ctx, app.GetDefaultAppInput{
		Namespace: s.namespace,
		Type:      app.AppTypeStripe,
	})

	require.NoError(t, err, "Get default stripe app must not return error")
	require.Equal(t, createApp1.GetID(), getApp.GetID(), "apps must be equal with first")
}

// TestGetDefaultAfterDelete tests getting the default stripe app after delete
func (s *AppHandlerTestSuite) TestGetDefaultAfterDelete(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	s.Env.StripeClient().
		On("GetAccount").
		Return(stripeclient.StripeAccount{
			StripeAccountID: getStripeAccountId(),
		}, nil)

	s.Env.StripeClient().
		On("SetupWebhook", mock.Anything).
		Return(stripeclient.StripeWebhookEndpoint{
			EndpointID: "we_123",
			Secret:     "whsec_123",
		}, nil)

	// TODO: do not share env between tests
	defer s.Env.StripeClient().Restore()

	// Create a stripe app first
	createApp, err := s.Env.App().InstallMarketplaceListingWithAPIKey(ctx, app.InstallAppWithAPIKeyInput{
		MarketplaceListingID: app.MarketplaceListingID{
			Type: app.AppTypeStripe,
		},

		Namespace: s.namespace,
		APIKey:    TestStripeAPIKey,
	})

	require.NoError(t, err, "Create stripe app must not return error")

	// Delete the app to test the default app can be deleted
	s.Env.StripeAppClient().
		On("DeleteWebhook", stripeclient.DeleteWebhookInput{
			AppID:           createApp.GetID(),
			StripeWebhookID: "we_123",
		}).
		Return(nil)

	err = s.Env.App().UninstallApp(ctx, createApp.GetID())
	require.NoError(t, err, "Uninstall stripe app must not return error")

	// Getting the deleted default app should return error
	_, err = s.Env.App().GetDefaultApp(ctx, app.GetDefaultAppInput{
		Namespace: s.namespace,
		Type:      app.AppTypeStripe,
	})

	require.ErrorAs(t, err, &app.AppDefaultNotFoundError{}, "Get default stripe app must return app not found error")

	// Create a new stripe app that should become the new default
	createApp2, err := s.Env.App().InstallMarketplaceListingWithAPIKey(ctx, app.InstallAppWithAPIKeyInput{
		MarketplaceListingID: app.MarketplaceListingID{
			Type: app.AppTypeStripe,
		},

		Namespace: s.namespace,
		APIKey:    TestStripeAPIKey,
	})

	require.NoError(t, err, "Create stripe app must not return error")
	require.NotNil(t, createApp2, "Create stripe app must return app")
	require.NotEqual(t, createApp.GetID(), createApp2.GetID(), "apps must not be equal")

	// Get the default app
	getApp, err := s.Env.App().GetDefaultApp(ctx, app.GetDefaultAppInput{
		Namespace: s.namespace,
		Type:      app.AppTypeStripe,
	})

	require.NoError(t, err, "Get default stripe app must not return error")
	require.Equal(t, createApp2.GetID(), getApp.GetID(), "apps must be equal with second")
}

// TestUpdate tests updating an app
func (s *AppHandlerTestSuite) TestUpdate(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	// Create an app first
	testApp, err := s.Env.Fixture().setupApp(ctx, s.namespace)
	require.NoError(t, err, "setup fixture must not return error")

	// Update the app
	updateApp, err := s.Env.App().UpdateApp(ctx, app.UpdateAppInput{
		AppID:       testApp.GetID(),
		Name:        "Updated Stripe App 1",
		Description: lo.ToPtr("Updated description 1"),
		Default:     true,
		Metadata:    &map[string]string{"key": "value"},
	})

	require.NoError(t, err, "Update app must not return error")
	require.NotNil(t, updateApp, "Update app must return app")

	// Partial update (only required fields)
	updateApp, err = s.Env.App().UpdateApp(ctx, app.UpdateAppInput{
		AppID:   testApp.GetID(),
		Name:    "Updated Stripe App 2",
		Default: false,
	})

	require.NoError(t, err, "Update app must not return error")
	require.NotNil(t, updateApp, "Update app must return app")

	// Updated fields
	require.Equal(t, "Updated Stripe App 2", updateApp.GetAppBase().Name, "Name must be updated")
	require.Equal(t, false, updateApp.GetAppBase().Default, "Default must remain the same")

	// Remains the same
	require.Equal(t, "Updated description 1", *updateApp.GetAppBase().Description, "Description must be updated")
	require.Equal(t, map[string]string{"key": "value"}, updateApp.GetAppBase().Metadata, "Metadata must be updated")
}

// TestUninstall tests uninstalling a stripe app
func (s *AppHandlerTestSuite) TestUninstall(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	// Create a stripe app first
	createApp, err := s.Env.Fixture().setupApp(ctx, s.namespace)
	require.NoError(t, err, "setup fixture must not return error")

	require.NoError(t, err, "Create stripe app must not return error")
	require.NotNil(t, createApp, "Create stripe app must return app")

	// Mocks
	s.Env.StripeAppClient().
		On("DeleteWebhook", stripeclient.DeleteWebhookInput{
			AppID:           createApp.GetID(),
			StripeWebhookID: "we_123",
		}).
		Return(nil)

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
	getCustomerData, err := testApp.GetCustomerData(ctx, app.GetAppInstanceCustomerDataInput{
		CustomerID: customer.GetID(),
	})

	require.NoError(t, err, "Get customer data must not return error")
	require.Equal(t, appstripeentity.CustomerData{
		StripeCustomerID: customerData.StripeCustomerID,
	}, getCustomerData, "Customer data must match")

	// List customer data should return the customer data
	listCustomerData, err := s.Env.App().ListCustomerData(ctx, app.ListCustomerInput{
		Page: pagination.Page{
			PageSize:   10,
			PageNumber: 1,
		},
		CustomerID: customer.GetID(),
	})

	require.NoError(t, err, "List customer data must not return error")
	require.Equal(t, 1, len(listCustomerData.Items), "List customer data must return one item")
	require.Equal(t, testApp.GetID(), listCustomerData.Items[0].App.GetID(), "App ID must match")

	// Update customer data
	newStripeCustomerID := "cus_456"
	newStripePaymentMethodID := "pm_456"

	s.Env.StripeAppClient().
		On("GetCustomer", newStripeCustomerID).
		Return(stripeclient.StripeCustomer{
			StripeCustomerID: newStripeCustomerID,
		}, nil)

	s.Env.StripeAppClient().
		On("GetPaymentMethod", newStripePaymentMethodID).
		Return(stripeclient.StripePaymentMethod{
			ID:               newStripePaymentMethodID,
			StripeCustomerID: &newStripeCustomerID,
			Name:             "ACME Inc.",
			Email:            "acme@example.com",
		}, nil)

	defer s.Env.StripeAppClient().Restore()

	err = testApp.UpsertCustomerData(ctx, app.UpsertAppInstanceCustomerDataInput{
		CustomerID: customer.GetID(),
		Data: appstripeentity.CustomerData{
			StripeCustomerID:             newStripeCustomerID,
			StripeDefaultPaymentMethodID: &newStripePaymentMethodID,
		},
	})

	require.NoError(t, err, "Update customer data must not return error")

	// Update customer data with non existing stripe customer should return error
	stripeAppData, err := s.Env.AppStripe().GetStripeAppData(ctx, appstripeentity.GetStripeAppDataInput{
		AppID: testApp.GetID(),
	})
	require.NoError(t, err, "Get stripe app data must not return error")

	nonExistingStripeCustomerID := "cus_789"

	s.Env.StripeAppClient().
		On("GetCustomer", nonExistingStripeCustomerID).
		Return(stripeclient.StripeCustomer{}, stripeclient.StripeCustomerNotFoundError{
			StripeCustomerID: nonExistingStripeCustomerID,
		})

	defer s.Env.StripeAppClient().Restore()

	err = testApp.UpsertCustomerData(ctx, app.UpsertAppInstanceCustomerDataInput{
		CustomerID: customer.GetID(),
		Data: appstripeentity.CustomerData{
			StripeCustomerID: nonExistingStripeCustomerID,
		},
	})

	require.ErrorIs(t, err, app.AppCustomerPreConditionError{
		AppID:      testApp.GetID(),
		AppType:    app.AppTypeStripe,
		CustomerID: customer.GetID(),
		Condition:  fmt.Sprintf("stripe customer %s not found in stripe account: %s", nonExistingStripeCustomerID, stripeAppData.StripeAccountID),
	})

	// Updated customer data with non existing payment method should return error
	nonExistingStripePaymentMethodID := "pm_789"

	s.Env.StripeAppClient().Restore()

	s.Env.StripeAppClient().
		On("GetCustomer", newStripeCustomerID).
		Return(stripeclient.StripeCustomer{
			StripeCustomerID: newStripeCustomerID,
		}, nil)

	s.Env.StripeAppClient().
		On("GetPaymentMethod", nonExistingStripePaymentMethodID).
		Return(stripeclient.StripePaymentMethod{}, stripeclient.StripePaymentMethodNotFoundError{
			StripePaymentMethodID: nonExistingStripePaymentMethodID,
		})

	defer s.Env.StripeAppClient().Restore()

	err = testApp.UpsertCustomerData(ctx, app.UpsertAppInstanceCustomerDataInput{
		CustomerID: customer.GetID(),
		Data: appstripeentity.CustomerData{
			StripeCustomerID:             newStripeCustomerID,
			StripeDefaultPaymentMethodID: &nonExistingStripePaymentMethodID,
		},
	})

	require.ErrorIs(t, err, app.AppProviderPreConditionError{
		AppID:     testApp.GetID(),
		Condition: fmt.Sprintf("stripe payment method %s not found in stripe account: %s", nonExistingStripePaymentMethodID, stripeAppData.StripeAccountID),
	})

	// Updated customer data with payment method that does not belong to the customer should return errors
	s.Env.StripeAppClient().Restore()

	s.Env.StripeAppClient().
		On("GetCustomer", newStripeCustomerID).
		Return(stripeclient.StripeCustomer{
			StripeCustomerID: newStripeCustomerID,
		}, nil)

	s.Env.StripeAppClient().
		On("GetPaymentMethod", newStripePaymentMethodID).
		Return(stripeclient.StripePaymentMethod{
			ID:               newStripePaymentMethodID,
			StripeCustomerID: lo.ToPtr("cus_different"),
			Name:             "ACME Inc.",
		}, nil)

	defer s.Env.StripeAppClient().Restore()

	err = testApp.UpsertCustomerData(ctx, app.UpsertAppInstanceCustomerDataInput{
		CustomerID: customer.GetID(),
		Data: appstripeentity.CustomerData{
			StripeCustomerID:             newStripeCustomerID,
			StripeDefaultPaymentMethodID: &newStripePaymentMethodID,
		},
	})

	require.ErrorIs(t, err, app.AppProviderPreConditionError{
		AppID: testApp.GetID(),
		Condition: fmt.Sprintf(
			"stripe payment method %s does not belong to stripe customer %s in stripe account: %s",
			newStripePaymentMethodID,
			newStripeCustomerID,
			stripeAppData.StripeAccountID,
		),
	})

	// Updated customer data must match
	getCustomerData, err = testApp.GetCustomerData(ctx, app.GetAppInstanceCustomerDataInput{
		CustomerID: customer.GetID(),
	})

	require.NoError(t, err, "Get customer data must not return error")
	require.Equal(t, appstripeentity.CustomerData{
		StripeCustomerID: "cus_456",
	}, getCustomerData, "Customer data must match")

	// Delete customer data
	err = testApp.DeleteCustomerData(ctx, app.DeleteAppInstanceCustomerDataInput{
		CustomerID: customer.GetID(),
	})

	require.NoError(t, err, "Delete customer data must not return error")

	// List customer data should return no customer data
	listCustomerData, err = s.Env.App().ListCustomerData(ctx, app.ListCustomerInput{
		Page: pagination.Page{
			PageSize:   10,
			PageNumber: 1,
		},
		CustomerID: customer.GetID(),
	})

	require.NoError(t, err, "List customer data must not return error")
	require.Equal(t, 0, len(listCustomerData.Items), "List customer data must return no item")

	// Get customer data should return 404
	_, err = testApp.GetCustomerData(ctx, app.GetAppInstanceCustomerDataInput{
		CustomerID: customer.GetID(),
	})

	require.ErrorIs(t, err, app.AppCustomerPreConditionError{
		AppID:      testApp.GetID(),
		AppType:    app.AppTypeStripe,
		CustomerID: customer.GetID(),
		Condition:  "customer has no data for stripe app",
	})

	// Restore customer data
	s.Env.StripeAppClient().
		On("GetCustomer", customerData.StripeCustomerID).
		Return(stripeclient.StripeCustomer{
			StripeCustomerID: customerData.StripeCustomerID,
		}, nil)

	defer s.Env.StripeAppClient().Restore()

	err = testApp.UpsertCustomerData(ctx, app.UpsertAppInstanceCustomerDataInput{
		CustomerID: customer.GetID(),
		Data: appstripeentity.CustomerData{
			StripeCustomerID: customerData.StripeCustomerID,
		},
	})

	require.NoError(t, err, "Restore customer data must not return error")

	// List customer data should return the restores customer data
	listCustomerData, err = s.Env.App().ListCustomerData(ctx, app.ListCustomerInput{
		Page: pagination.Page{
			PageSize:   10,
			PageNumber: 1,
		},
		CustomerID: customer.GetID(),
	})

	require.NoError(t, err, "List customer data must not return error")
	require.Equal(t, 1, len(listCustomerData.Items), "List customer data must return one item")
	require.Equal(t, testApp.GetID(), listCustomerData.Items[0].App.GetID(), "App ID must match")
}

// TestCustomerValidate tests stripe app behavior when validating a customer
func (s *AppHandlerTestSuite) TestCustomerValidate(ctx context.Context, t *testing.T) {
	testApp, testCustomer, _, err := s.Env.Fixture().setupAppWithCustomer(ctx, s.namespace)
	require.NoError(t, err, "setup fixture must not return error")

	// Create customer without stripe data
	customerWithoutStripeData, err := s.Env.Customer().CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: s.namespace,
		CustomerMutate: customer.CustomerMutate{
			Name: "Test Customer Without Stripe",
		},
	})

	require.NoError(t, err, "Create customer must not return error")
	require.NotNil(t, customerWithoutStripeData, "Create customer must return customer")

	// Get App
	getApp, err := s.Env.App().GetApp(ctx, testApp.GetID())

	require.NoError(t, err, "Get app must not return error")

	// App should implement Customer App
	customerApp, err := customerapp.AsCustomerApp(getApp)

	require.NoError(t, err, "Get app must not return error")

	// Mocks
	s.Env.StripeAppClient().
		On("GetCustomer", "cus_123").
		Return(stripeclient.StripeCustomer{
			StripeCustomerID: "cus_123",
			DefaultPaymentMethod: &stripeclient.StripePaymentMethod{
				ID:    "pm_123",
				Name:  "ACME Inc.",
				Email: "acme@test.com",
				BillingAddress: &models.Address{
					City:       lo.ToPtr("San Francisco"),
					PostalCode: lo.ToPtr("94103"),
					State:      lo.ToPtr("CA"),
					Country:    lo.ToPtr(models.CountryCode("US")),
					Line1:      lo.ToPtr("123 Market St"),
				},
			},
		}, nil)

	// TODO: do not share env between tests
	defer s.Env.StripeAppClient().Restore()

	// App should validate the customer
	err = customerApp.ValidateCustomer(ctx, testCustomer, []app.CapabilityType{
		app.CapabilityTypeCalculateTax,
		app.CapabilityTypeInvoiceCustomers,
		app.CapabilityTypeCollectPayments,
	})
	require.NoError(t, err, "Validate customer must not return error")

	// Validate the customer with an invalid capability
	err = customerApp.ValidateCustomer(ctx, testCustomer, []app.CapabilityType{app.CapabilityTypeReportEvents})
	require.ErrorContains(t, err, "capability reportEvents is not supported", "Validate customer must return error")

	// Validate the customer without stripe data
	err = customerApp.ValidateCustomer(ctx, customerWithoutStripeData, []app.CapabilityType{app.CapabilityTypeCalculateTax})
	require.ErrorContains(t, err, "customer has no data", "Validate customer must return error")
}

// TestCreateCheckoutSession tests stripe app behavior when creating a new checkout session
func (s *AppHandlerTestSuite) TestCreateCheckoutSession(ctx context.Context, t *testing.T) {
	testApp, testCustomer, _, err := s.Env.Fixture().setupAppWithCustomer(ctx, s.namespace)
	require.NoError(t, err, "setup fixture must not return error")

	// Create checkout session
	appID := testApp.GetID()
	customerID := testCustomer.GetID()

	// Mocks
	s.Env.StripeAppClient().
		On("CreateCheckoutSession", stripeclient.CreateCheckoutSessionInput{
			AppID:            appID,
			CustomerID:       customerID,
			StripeCustomerID: "cus_123",
			Options:          api.CreateStripeCheckoutSessionRequestOptions{},
		}).
		Return(stripeclient.StripeCheckoutSession{
			SessionID:     "cs_123",
			SetupIntentID: "seti_123",
			Mode:          stripe.CheckoutSessionModeSetup,
			URL:           lo.ToPtr("https://checkout.stripe.com/cs_123/test"),
		}, nil)

	// TODO: do not share env between tests
	defer s.Env.StripeAppClient().Restore()

	checkoutSession, err := s.Env.AppStripe().CreateCheckoutSession(ctx, appstripeentity.CreateCheckoutSessionInput{
		Namespace:  s.namespace,
		AppID:      &appID,
		CustomerID: &customerID,
		Options:    api.CreateStripeCheckoutSessionRequestOptions{},
	})

	require.NoError(t, err, "Create checkout session must not return error")

	require.Equal(t, appstripeentity.CreateCheckoutSessionOutput{
		CustomerID:       testCustomer.GetID(),
		StripeCustomerID: "cus_123",
		SessionID:        "cs_123",
		SetupIntentID:    "seti_123",
		Mode:             stripe.CheckoutSessionModeSetup,
		URL:              lo.ToPtr("https://checkout.stripe.com/cs_123/test"),
	}, checkoutSession, "Create checkout session must match")

	// Test app 404 error
	appIdNotFound := app.AppID{
		Namespace: s.namespace,
		ID:        "not_found",
	}

	_, err = s.Env.AppStripe().CreateCheckoutSession(ctx, appstripeentity.CreateCheckoutSessionInput{
		Namespace:  s.namespace,
		AppID:      &appIdNotFound,
		CustomerID: &customerID,
		Options:    api.CreateStripeCheckoutSessionRequestOptions{},
	})

	require.ErrorIs(t, err, app.AppNotFoundError{AppID: appIdNotFound}, "Create checkout session must return app not found error")

	// Test customer 404 error
	customerIdNotFound := customer.CustomerID{
		Namespace: s.namespace,
		ID:        "not_found",
	}

	_, err = s.Env.AppStripe().CreateCheckoutSession(ctx, appstripeentity.CreateCheckoutSessionInput{
		Namespace:  s.namespace,
		AppID:      &appID,
		CustomerID: &customerIdNotFound,
		Options:    api.CreateStripeCheckoutSessionRequestOptions{},
	})

	require.ErrorIs(t, err, customer.NotFoundError{CustomerID: customerIdNotFound}, "Create checkout session must return customer not found error")

	// Test if we pass down customer currency if set
	s.Env.StripeAppClient().Restore()

	s.Env.StripeAppClient().
		On("CreateCheckoutSession", stripeclient.CreateCheckoutSessionInput{
			AppID:            appID,
			CustomerID:       customerID,
			StripeCustomerID: "cus_123",
			Options: api.CreateStripeCheckoutSessionRequestOptions{
				Currency: lo.ToPtr("usd"),
			},
		}).
		Return(stripeclient.StripeCheckoutSession{
			SessionID:     "cs_123",
			SetupIntentID: "seti_123",
			Mode:          stripe.CheckoutSessionModeSetup,
			URL:           lo.ToPtr("https://checkout.stripe.com/cs_123/test"),
		}, nil)

	// TODO: do not share env between tests
	defer s.Env.StripeAppClient().Restore()

	_, err = s.Env.Customer().UpdateCustomer(ctx, customer.UpdateCustomerInput{
		CustomerID: testCustomer.GetID(),
		CustomerMutate: customer.CustomerMutate{
			Name:             testCustomer.Name,
			UsageAttribution: testCustomer.UsageAttribution,
			Currency:         lo.ToPtr(currencyx.Code("USD")),
		},
	})
	require.NoError(t, err, "Update customer must not return error")

	_, err = s.Env.AppStripe().CreateCheckoutSession(ctx, appstripeentity.CreateCheckoutSessionInput{
		Namespace:  s.namespace,
		AppID:      &appID,
		CustomerID: &customerID,
	})
	require.NoError(t, err, "Create checkout session must not return error")
}

// TestUpdateAPIKey tests stripe app behavior when updating the API key
func (s *AppHandlerTestSuite) TestUpdateAPIKey(ctx context.Context, t *testing.T) {
	testApp, err := s.Env.Fixture().setupApp(ctx, s.namespace)
	require.NoError(t, err, "setup fixture must not return error")

	// Get stripe app
	stripeApp, err := s.Env.AppStripe().GetStripeAppData(ctx, appstripeentity.GetStripeAppDataInput{AppID: testApp.GetID()})
	require.NoError(t, err, "Get stripe app data must not return error")

	newAPIKey := "sk_test_abcde"

	// Should not allow to update test mode app with livemode key
	err = s.Env.AppStripe().UpdateAPIKey(ctx, appstripeentity.UpdateAPIKeyInput{
		AppID:  testApp.GetID(),
		APIKey: newAPIKey,
	})

	require.Error(t, err, "Update API key must return error")
	require.Equal(t, err.Error(), "new stripe api key is in test mode but the app is in live mode")

	// Mock the secret service
	s.Env.Secret().EnableMock()
	defer s.Env.Secret().DisableMock()

	newAPIKey = "sk_live_abcde"

	s.Env.StripeAppClient().
		On("GetAccount").
		Return(stripeclient.StripeAccount{
			StripeAccountID: stripeApp.StripeAccountID,
		}, nil)

	s.Env.Secret().
		On("GetAppSecret", secretentity.GetAppSecretInput{
			AppID: testApp.GetID(),
			Key:   appstripeentity.APIKeySecretKey,
		}).
		Return(stripeApp.APIKey, nil)

	s.Env.Secret().
		On("UpdateAppSecret", secretentity.UpdateAppSecretInput{
			AppID:    testApp.GetID(),
			SecretID: stripeApp.APIKey,
			Key:      appstripeentity.APIKeySecretKey,
			Value:    newAPIKey,
		}).
		Return(nil)

	// Update app status to unauthorized so we can check
	// if it is updated to ready after updating the API key.
	err = s.Env.App().UpdateAppStatus(ctx, app.UpdateAppStatusInput{
		ID:     testApp.GetID(),
		Status: app.AppStatusUnauthorized,
	})
	require.NoError(t, err, "Update app status must not return error")

	// Should allow to update test mode app with test mode key
	err = s.Env.AppStripe().UpdateAPIKey(ctx, appstripeentity.UpdateAPIKeyInput{
		AppID:  testApp.GetID(),
		APIKey: newAPIKey,
	})
	require.NoError(t, err, "Update API key must not return error")

	// Get stripe app
	testApp, err = s.Env.App().GetApp(ctx, testApp.GetID())

	require.NoError(t, err, "Get app must not return error")
	require.Equal(t, testApp.GetStatus(), app.AppStatusReady, "App status must be ready")
}
