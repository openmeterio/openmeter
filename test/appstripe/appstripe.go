package appstripe

import (
	"context"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appcustomerentity "github.com/openmeterio/openmeter/openmeter/appcustomer/entity"
	appstripecustomer "github.com/openmeterio/openmeter/openmeter/appstripe/customer"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
)

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

	app, err := s.Env.AppStripe().CreateStripeApp(ctx, appstripeentity.CreateAppStripeInput{
		Namespace:       s.namespace,
		Name:            "Test App",
		Description:     "Test App Description",
		StripeAccountID: "acct_123",
		Livemode:        true,
	})

	require.NoError(t, err, "Create stripe app must not return error")
	require.NotNil(t, app, "Create stripe app must return app")
}

// TestCustomerCreate tests stripe app behavior when creating a new customer
func (s *AppHandlerTestSuite) TestCustomerCreate(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	// Create a stripe app first
	app, err := s.Env.AppStripe().CreateStripeApp(ctx, appstripeentity.CreateAppStripeInput{
		Namespace:       s.namespace,
		Name:            "Test App",
		Description:     "Test App Description",
		StripeAccountID: "acct_123",
		Livemode:        true,
	})

	require.NoError(t, err, "Create stripe app must not return error")
	require.NotNil(t, app, "Create stripe app must return app")

	// Create a customer
	customer, err := s.Env.Customer().CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: s.namespace,
		Customer: customerentity.Customer{
			Name: "Test Customer",
			Apps: []appcustomerentity.CustomerApp{
				{
					Type: appentity.AppTypeStripe,
					Data: appstripecustomer.CustomerData{
						StripeCustomerID: "cus_123",
					},
				},
			},
		},
	})

	require.NoError(t, err, "Create customer must not return error")
	require.NotNil(t, customer, "Create customer must return customer")
}
