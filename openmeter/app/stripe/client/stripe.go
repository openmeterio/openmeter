package client

import (
	"errors"
	"fmt"
	"strings"

	"github.com/stripe/stripe-go/v80"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
)

type StripeWebhookEndpoint struct {
	EndpointID string
	Secret     string
}

type StripeAccount struct {
	StripeAccountID string
	BusinessProfile *stripe.AccountBusinessProfile
	Country         models.CountryCode
}

type StripeCustomer struct {
	StripeCustomerID string
	Name             *string
	Currency         *string
	Email            *string
	// ID of a payment method that’s attached to the customer,
	// to be used as the customer’s default payment method for invoices.
	DefaultPaymentMethod *StripePaymentMethod
	Tax                  *StripeCustomerTax
}

type StripeCustomerTax struct {
	AutomaticTax StripeCustomerAutomaticTax
}

// https://docs.stripe.com/api/customers/object#customer_object-tax-automatic_tax
type StripeCustomerAutomaticTax string

const (
	// There was an error determining the customer’s location. This is usually caused by a temporary issue. Retrieve the customer to try again.
	StripeCustomerAutomaticTaxFailed StripeCustomerAutomaticTax = "failed"
	// The customer is located in a country or state where you’re not registered to collect tax. Also returned when automatic tax calculation is not supported in the customer’s location.
	StripeCustomerAutomaticTaxNotCollecting StripeCustomerAutomaticTax = "not_collecting"
	// The customer is located in a country or state where you’re collecting tax
	StripeCustomerAutomaticTaxSupported StripeCustomerAutomaticTax = "supported"
	// The customer’s location couldn’t be determined. Make sure the provided address information is valid and supported in the customer’s country.
	StripeCustomerAutomaticTaxUnrecognizedLocation StripeCustomerAutomaticTax = "unrecognized_location"
)

type StripePaymentMethod struct {
	ID               string
	StripeCustomerID *string
	Name             string
	Email            string
	BillingAddress   *models.Address
}

type SetupWebhookInput struct {
	AppID      app.AppID
	WebhookURL string
}

func (i SetupWebhookInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return fmt.Errorf("error validating app id: %w", err)
	}

	if i.WebhookURL == "" {
		return errors.New("webhook url is required")
	}

	return nil
}

type DeleteWebhookInput struct {
	AppID           app.AppID
	StripeWebhookID string
}

func (i DeleteWebhookInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return fmt.Errorf("error validating app id: %w", err)
	}

	if i.StripeWebhookID == "" {
		return errors.New("stripe webhook id is required")
	}

	return nil
}

type CreateStripeCustomerInput struct {
	AppID      app.AppID
	CustomerID customer.CustomerID

	Name  *string
	Email *string
}

func (i CreateStripeCustomerInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return fmt.Errorf("error validating app id: %w", err)
	}

	if err := i.CustomerID.Validate(); err != nil {
		return fmt.Errorf("error validating customer id: %w", err)
	}

	if i.AppID.Namespace != i.CustomerID.Namespace {
		return errors.New("app and customer must be in the same namespace")
	}

	if i.Name != nil && *i.Name == "" {
		return errors.New("name cannot be empty if provided")
	}

	if i.Email != nil && *i.Email == "" {
		return errors.New("email cannot be empty if provided")
	}

	return nil
}

// IsAPIKeyLiveMode checks if the API key is a live mode key
func IsAPIKeyLiveMode(apiKey string) bool {
	// Root keys start with "sk_"
	if strings.HasPrefix(apiKey, "sk_test") {
		return false
	}

	// Restricted keys start with "rk_"
	if strings.HasPrefix(apiKey, "rk_test") {
		return false
	}

	return true
}
