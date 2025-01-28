package client

import (
	"errors"
	"fmt"
	"strings"

	"github.com/stripe/stripe-go/v80"

	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
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
	// ID of a payment method that’s attached to the customer,
	// to be used as the customer’s default payment method for invoices.
	DefaultPaymentMethod *StripePaymentMethod
}

type StripePaymentMethod struct {
	ID               string
	StripeCustomerID *string
	Name             string
	Email            string
	BillingAddress   *models.Address
}

type StripeCheckoutSession struct {
	SessionID     string
	SetupIntentID string
	Mode          stripe.CheckoutSessionMode

	URL        *string
	CancelURL  *string
	SuccessURL *string
	ReturnURL  *string
}

func (o StripeCheckoutSession) Validate() error {
	if o.SessionID == "" {
		return errors.New("session id is required")
	}

	if o.SetupIntentID == "" {
		return errors.New("setup intent id is required")
	}

	if o.Mode != stripe.CheckoutSessionModeSetup {
		return errors.New("mode must be setup")
	}

	return nil
}

type CreateCheckoutSessionInput struct {
	AppID            appentitybase.AppID
	CustomerID       customerentity.CustomerID
	StripeCustomerID string
	Options          StripeCheckoutSessionInputOptions
}

func (i CreateCheckoutSessionInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return fmt.Errorf("error validating app id: %w", err)
	}

	if err := i.CustomerID.Validate(); err != nil {
		return fmt.Errorf("error validating customer id: %w", err)
	}

	if i.AppID.Namespace != i.CustomerID.Namespace {
		return errors.New("app and customer must be in the same namespace")
	}

	if i.StripeCustomerID != "" && !strings.HasPrefix(i.StripeCustomerID, "cus_") {
		return errors.New("stripe customer id must start with cus_")
	}

	if i.Options.UIMode != nil {
		switch *i.Options.UIMode {
		case stripe.CheckoutSessionUIModeEmbedded:
			if i.Options.ReturnURL == nil {
				return errors.New("return url is required for embedded ui mode")
			}

			if i.Options.CancelURL != nil {
				return errors.New("cancel url is not allowed for embedded ui mode")
			}
		case stripe.CheckoutSessionUIModeHosted:
			if i.Options.SuccessURL == nil {
				return errors.New("success url is required for hosted ui mode")
			}
		}
	}

	if err := i.Options.Validate(); err != nil {
		return fmt.Errorf("error validating options: %w", err)
	}

	return nil
}

type StripeCheckoutSessionInputOptions struct {
	CancelURL          *string
	Currency           *string
	ClientReferenceID  *string
	CustomText         *stripe.CheckoutSessionCustomTextParams
	Metadata           map[string]string
	ReturnURL          *string
	SuccessURL         *string
	UIMode             *stripe.CheckoutSessionUIMode
	PaymentMethodTypes *[]*string
}

func (i StripeCheckoutSessionInputOptions) Validate() error {
	return nil
}

type SetupWebhookInput struct {
	AppID   appentitybase.AppID
	BaseURL string
}

func (i SetupWebhookInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return fmt.Errorf("error validating app id: %w", err)
	}

	if i.BaseURL == "" {
		return errors.New("base url is required")
	}

	return nil
}

type DeleteWebhookInput struct {
	AppID           appentitybase.AppID
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
	AppID      appentitybase.AppID
	CustomerID customerentity.CustomerID

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
