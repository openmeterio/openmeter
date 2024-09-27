package stripe

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
)

const (
	AppTypeStripe appentity.AppType = "stripe"
)

var StripeMarketplaceListing = appentity.MarketplaceListing{
	Type:        AppTypeStripe,
	Key:         "stripe",
	Name:        "Stripe",
	Description: "Stripe is a payment processing platform.",
	IconURL:     "https://stripe.com/favicon.ico",
}

func Register(reg *app.Registry, appService app.Service, db entdb.Client) error {
	stripeAppFactory := NewAppFactory(appService, db)
	reg.RegisterListing(
		AppTypeStripe,
		appentity.Integration{
			Listing: StripeMarketplaceListing,
			Factory: stripeAppFactory,
		},
	)

	return nil
}

type AppFactory struct {
	AppService     app.Service
	BillingService billing.Service
	Client         entdb.Client
}

// TODO: add validation single input etc.
func NewAppFactory(appService app.Service, client entdb.Client) AppFactory {
	return AppFactory{
		AppService: appService,
		Client:     client,
	}
}

func (f AppFactory) NewIntegration(ctx context.Context, app appentity.AppBase) (appentity.App, error) {
	return &App{
		App:        app,
		AppService: f.AppService,
		// TODO: add other dependencies
	}, nil
}

func (f AppFactory) Capabilities() []appentity.CapabilityType {
	return []appentity.CapabilityType{
		customer.CapabilityCustomerManagement,
		// TODO: billing capabilities
	}
}

// App represents an installed Stripe app
type App struct {
	appentity.App

	AppService     app.Service
	BillingService billing.Service

	StripeAccountId string `json:"stripeAccountId"`
	Livemode        bool   `json:"livemode"`
}

// ValidateCustomer validates if the app can run for the given customer
func (a App) ValidateCustomer(ctx context.Context, customer customer.Customer) error {
	// Validate the customer's mandatory fields as per stripe

	// Let's validate for additional fields required by billing
	customerProfile, err := a.BillingService.GetProfileWithCustomerOverride(ctx, billing.GetProfileWithCustomerOverrideInput{
		Namespace:  customer.Namespace,
		CustomerID: customer.ID,
	})
	if err != nil {
		// The customer profile is not available => the customer is not ready for billing
		return nil
	}

	// If this plugin is responsible for invoicing, validate the invoicing configuration
	// TODOs:
	// - The provider configuration should be part of this app
	// if the customer profile has this provider set as app then let's go into stricer mode
	// if customerProfile.Profile.InvoicingConfiguration.Type == provider.InvoicingProviderStripeInvoicing {
	// 		Validate whatever is needed
	// }

	return nil
}

func (a App) ImportCustomers(ctx context.Context) ([]customer.CustomerImportInput, error) {
	return nil, errors.New("not implemented")
}

func (a App) UpsertCustomer(ctx context.Context, customer customer.Customer) error {
	return errors.New("not implemented")
}
