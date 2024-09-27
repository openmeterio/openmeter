package appstripeentity

import (
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/openmeterio/openmeter/openmeter/app"
	appadapter "github.com/openmeterio/openmeter/openmeter/app/adapter"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

var marketplace = appadapter.DefaultMarketplace()

var createDefaultMarketplaceOnce sync.Once

func RegisterApp() error {
	var err error

	createDefaultMarketplaceOnce.Do(func() {
		err = marketplace.RegisterListing(StripeMarketplaceListing)
	})

	return err
}

var (
	StripeMarketplaceListing = appentity.MarketplaceListing{
		Type:        appentity.AppTypeStripe,
		Name:        "Stripe",
		Description: "Stripe is a payment processing platform.",
		IconURL:     "https://stripe.com/favicon.ico",
		Capabilities: []appentity.Capability{
			StripeCollectPaymentCapability,
			StripeCalculateTaxCapability,
			StripeInvoiceCustomerCapability,
		},
	}

	StripeCollectPaymentCapability = appentity.Capability{
		Type:        appentity.CapabilityTypeCollectPayments,
		Key:         "stripe_collect_payment",
		Name:        "Payment",
		Description: "Process payments",
	}

	StripeCalculateTaxCapability = appentity.Capability{
		Type:        appentity.CapabilityTypeCalculateTax,
		Key:         "stripe_calculate_tax",
		Name:        "Calculate Tax",
		Description: "Calculate tax for a payment",
	}

	StripeInvoiceCustomerCapability = appentity.Capability{
		Type:        appentity.CapabilityTypeInvoiceCustomers,
		Key:         "stripe_invoice_customer",
		Name:        "Invoice Customer",
		Description: "Invoice a customer",
	}
)

// if err != nil {
// 	return nil, fmt.Errorf("failed to register marketplace listing: %w", err)
// }

// App represents an installed Stripe app
type App struct {
	appentity.AppBase

	// TODO: cycle dependencies
	// AppService       app.Service
	// AppStripeService appstripe.Service
	// BillingService   billing.Service

	StripeAccountId string `json:"stripeAccountId"`
	Livemode        bool   `json:"livemode"`
}

func (a App) Validate() error {
	if err := a.AppBase.Validate(); err != nil {
		return fmt.Errorf("error validating app: %w", err)
	}

	if a.Type != appentity.AppTypeStripe {
		return errors.New("app type must be stripe")
	}

	if a.StripeAccountId == "" {
		return errors.New("stripe account id is required")
	}

	return nil
}

// ValidateCustomer validates if the app can run for the given customer
func (a App) ValidateCustomer(customer customer.Customer, capabilities []appentity.CapabilityType) error {
	// Validate if the app supports the given capabilities
	if err := a.ValidateCapabilities(capabilities); err != nil {
		return fmt.Errorf("error validating capabilities: %w", err)
	}

	// All Stripe capabilities require the customer to have a Stripe customer ID associated
	if customer.External == nil && *customer.External.StripeCustomerID == "" {
		return app.CustomerPreConditionError{
			AppID:      a.GetID(),
			AppType:    a.GetType(),
			CustomerID: customer.GetID(),
			Condition:  "customer must have a Stripe customer ID",
		}
	}

	// Invoice and payment capabilities need to check if the customer has a country and default payment method via the Stripe API
	if slices.Contains(capabilities, appentity.CapabilityTypeCalculateTax) || slices.Contains(capabilities, appentity.CapabilityTypeInvoiceCustomers) || slices.Contains(capabilities, appentity.CapabilityTypeCollectPayments) {
		// TODO: go to Stripe and check if customer exists by customer.External.StripeCustomerID
		// Also check if the customer has a country and default payment method

		return errors.New("not implemented")
	}

	return nil
}

type CreateAppStripeInput = App