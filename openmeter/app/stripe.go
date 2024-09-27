package app

import (
	"errors"
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/customer"
)

var (
	StripeMarketplaceListing = MarketplaceListing{
		Type:        AppTypeStripe,
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
		Type:        CapabilityTypeCollectPayments,
		Key:         "stripe_collect_payment",
		Name:        "Payment",
		Description: "Process payments",
		Requirements: []Requirement{
			RequirementCustomerExternalStripeCustomerId,
		},
	}

	StripeCalculateTaxCapability = Capability{
		Type:        CapabilityTypeCalculateTax,
		Key:         "stripe_calculate_tax",
		Name:        "Calculate Tax",
		Description: "Calculate tax for a payment",
		Requirements: []Requirement{
			RequirementCustomerExternalStripeCustomerId,
		},
	}

	StripeInvoiceCustomerCapability = Capability{
		Type:        CapabilityTypeInvoiceCustomers,
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
func (a StripeApp) ValidateCustomer(customer customer.Customer, capabilities []CapabilityType) error {
	// Validate if the app supports the given capabilities
	if err := a.ValidateCapabilities(capabilities); err != nil {
		return fmt.Errorf("error validating capabilities: %w", err)
	}

	// All Stripe capabilities require the customer to have a Stripe customer ID associated
	if customer.External == nil && *customer.External.StripeCustomerID == "" {
		return CustomerPreConditionError{
			AppID:          a.GetID(),
			AppType:        a.GetType(),
			AppRequirement: RequirementCustomerExternalStripeCustomerId,
			CustomerID:     customer.GetID(),
		}
	}

	// Invoice and payment capabilities need to check if the customer has a country and default payment method via the Stripe API
	if slices.Contains(capabilities, CapabilityTypeCalculateTax) || slices.Contains(capabilities, CapabilityTypeInvoiceCustomers) || slices.Contains(capabilities, CapabilityTypeCollectPayments) {
		// TODO: go to Stripe and check if customer exists by customer.External.StripeCustomerID
		// Also check if the customer has a country and default payment method

		return errors.New("not implemented")
	}

	return nil
}
