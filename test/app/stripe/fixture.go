package appstripe

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/stretchr/testify/mock"
)

func NewFixture(
	app app.Service,
	customer customer.Service,
	stripeClient *StripeClientMock,
) *Fixture {
	return &Fixture{
		app:          app,
		customer:     customer,
		stripeClient: stripeClient,
	}
}

type Fixture struct {
	app          app.Service
	customer     customer.Service
	stripeClient *StripeClientMock
}

// setupAppWithCustomer creates a stripe app and a customer with customer data
func (s *Fixture) setupAppWithCustomer(ctx context.Context, namespace string) (appentity.App, *customerentity.Customer, appstripeentity.CustomerData, error) {
	app, err := s.setupApp(ctx, namespace)
	if err != nil {
		return nil, nil, appstripeentity.CustomerData{}, fmt.Errorf("setup app failed: %w", err)
	}

	customer, err := s.setupCustomer(ctx, namespace)
	if err != nil {
		return nil, nil, appstripeentity.CustomerData{}, fmt.Errorf("setup customer failed: %w", err)
	}

	data, err := s.setupAppCustomerData(ctx, app, customer)
	if err != nil {
		return nil, nil, appstripeentity.CustomerData{}, fmt.Errorf("setup app customer data failed: %w", err)
	}

	return app, customer, data, nil
}

// Create a stripe app first
func (s *Fixture) setupApp(ctx context.Context, namespace string) (appentity.App, error) {
	s.stripeClient.
		On("GetAccount").
		Return(stripeclient.StripeAccount{
			StripeAccountID: "stripe-account-id",
		}, nil)

	s.stripeClient.
		On("SetupWebhook", mock.Anything).
		Return(stripeclient.StripeWebhookEndpoint{
			EndpointID: "we_123",
			Secret:     "whsec_123",
		}, nil)

	// TODO: do not share env between tests
	defer s.stripeClient.Restore()

	app, err := s.app.InstallMarketplaceListingWithAPIKey(ctx, appentity.InstallAppWithAPIKeyInput{
		MarketplaceListingID: appentity.MarketplaceListingID{
			Type: appentitybase.AppTypeStripe,
		},

		Namespace: namespace,
		APIKey:    TestStripeAPIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("install stripe app failed: %w", err)
	}

	return app, nil
}

// Create test customers
func (s *Fixture) setupCustomer(ctx context.Context, namespace string) (*customerentity.Customer, error) {
	customer, err := s.customer.CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: namespace,
		CustomerMutate: customerentity.CustomerMutate{
			Name: "Test Customer",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create customer failed: %w", err)
	}

	return customer, nil
}

// Add customer data to the app
func (s *Fixture) setupAppCustomerData(ctx context.Context, app appentity.App, customer *customerentity.Customer) (appstripeentity.CustomerData, error) {
	data := appstripeentity.CustomerData{
		StripeCustomerID: "cus_123",
	}

	err := app.UpsertCustomerData(ctx, appentity.UpsertAppInstanceCustomerDataInput{
		CustomerID: customer.GetID(),
		Data:       data,
	})
	if err != nil {
		return data, fmt.Errorf("Upsert customer data failed: %w", err)
	}

	return data, nil
}
