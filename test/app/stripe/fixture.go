package appstripe

import (
	"context"
	"fmt"
	"time"

	"github.com/stretchr/testify/mock"
	"golang.org/x/exp/rand"

	"github.com/openmeterio/openmeter/openmeter/app"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

const defaultStripeCustomerID = "cus_123"

func NewFixture(
	app app.Service,
	customer customer.Service,
	stripeClient *StripeClientMock,
	stripeAppClient *StripeAppClientMock,
) *Fixture {
	return &Fixture{
		app:             app,
		customer:        customer,
		stripeClient:    stripeClient,
		stripeAppClient: stripeAppClient,
	}
}

type Fixture struct {
	app             app.Service
	customer        customer.Service
	stripeClient    *StripeClientMock
	stripeAppClient *StripeAppClientMock
}

// setupAppWithCustomer creates a stripe app and a customer with customer data
func (s *Fixture) setupAppWithCustomer(ctx context.Context, namespace string) (app.App, *customer.Customer, appstripeentity.CustomerData, error) {
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
func (s *Fixture) setupApp(ctx context.Context, namespace string) (app.App, error) {
	s.stripeClient.
		On("GetAccount").
		Return(stripeclient.StripeAccount{
			StripeAccountID: getStripeAccountId(),
		}, nil)

	s.stripeClient.
		On("SetupWebhook", mock.Anything).
		Return(stripeclient.StripeWebhookEndpoint{
			EndpointID: "we_123",
			Secret:     "whsec_123",
		}, nil)

	// TODO: do not share env between tests
	defer s.stripeClient.Restore()

	app, err := s.app.InstallMarketplaceListingWithAPIKey(ctx, app.InstallAppWithAPIKeyInput{
		InstallAppInput: app.InstallAppInput{
			MarketplaceListingID: app.MarketplaceListingID{
				Type: app.AppTypeStripe,
			},

			Namespace: namespace,
		},
		APIKey: TestStripeAPIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("install stripe app failed: %w", err)
	}

	return app, nil
}

// Create test customers
func (s *Fixture) setupCustomer(ctx context.Context, namespace string) (*customer.Customer, error) {
	customerKey := fmt.Sprintf("test-customer-%d", rand.Intn(1000000))
	customer, err := s.customer.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: namespace,
		CustomerMutate: customer.CustomerMutate{
			Name: "Test Customer",
			Key:  &customerKey,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create customer failed: %w", err)
	}

	return customer, nil
}

// Add customer data to the app
func (s *Fixture) setupAppCustomerData(ctx context.Context, customerApp app.App, customer *customer.Customer) (appstripeentity.CustomerData, error) {
	data := appstripeentity.CustomerData{
		StripeCustomerID: defaultStripeCustomerID,
	}

	s.stripeAppClient.
		On("GetCustomer", data.StripeCustomerID).
		Return(stripeclient.StripeCustomer{
			StripeCustomerID: data.StripeCustomerID,
		}, nil)

	defer s.stripeAppClient.Restore()

	err := customerApp.UpsertCustomerData(ctx, app.UpsertAppInstanceCustomerDataInput{
		CustomerID: customer.GetID(),
		Data:       data,
	})
	if err != nil {
		return data, fmt.Errorf("upsert customer data failed: %w", err)
	}

	return data, nil
}

// Get a random stripe account id
func getStripeAccountId() string {
	length := 6
	rand.Seed(uint64(time.Now().UnixNano()))
	b := make([]byte, length+2)
	_, _ = rand.Read(b)
	s := fmt.Sprintf("%x", b)[2 : length+2]

	return "acct_" + s
}
