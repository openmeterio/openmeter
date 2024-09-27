package app

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
)

var (
	StripeMarketplaceListing = MarketplaceListing{
		Type:        AppTypeStripe,
		Key:         "stripe",
		Name:        "Stripe",
		Description: "Stripe is a payment processing platform.",
		IconURL:     "https://stripe.com/favicon.ico",
		Capabilities: []Capability{
			StripeCollectPaymentCapability,
			StripeCalculateTaxCapability,
			StripeInvoiceCustomerCapability,
		},
	}

	StripeCollectPaymentCapability = Capability{
		Key:         "stripe_collect_payment",
		Name:        "Payment",
		Description: "Process payments",
		Requirements: []Requirement{
			RequirementCustomerExternalStripeCustomerId,
		},
	}

	StripeCalculateTaxCapability = Capability{
		Key:         "stripe_calculate_tax",
		Name:        "Calculate Tax",
		Description: "Calculate tax for a payment",
		Requirements: []Requirement{
			RequirementCustomerExternalStripeCustomerId,
		},
	}

	StripeInvoiceCustomerCapability = Capability{
		Key:         "stripe_invoice_customer",
		Name:        "Invoice Customer",
		Description: "Invoice a customer",
		Requirements: []Requirement{
			RequirementCustomerExternalStripeCustomerId,
		},
	}
)

// StripeApp represents an installed Stripe app
type StripeApp struct {
	AppBase

	StripeAccountId string `json:"stripeAccountId"`
	Livemode        bool   `json:"livemode"`
}

func (a StripeApp) Validate() error {
	if err := a.AppBase.Validate(); err != nil {
		return fmt.Errorf("error validating app: %w", err)
	}

	if a.Type != AppTypeStripe {
		return errors.New("app type must be stripe")
	}

	if a.StripeAccountId == "" {
		return errors.New("stripe account id is required")
	}

	return nil
}

// ValidateCustomer validates if the app can run for the given customer
func (a StripeApp) ValidateCustomer(customer customer.Customer) error {
	if customer.External == nil && *customer.External.StripeCustomerID == "" {
		return CustomerPreConditionError{
			AppID:          a.GetID(),
			AppType:        a.GetType(),
			AppRequirement: RequirementCustomerExternalStripeCustomerId,
			CustomerID:     customer.GetID(),
		}
	}

	// TODO: go to Stripe and check if customer exists by customer.External.StripeCustomerID
	// Also check if the customer has a country and default payment method

	return nil
}
