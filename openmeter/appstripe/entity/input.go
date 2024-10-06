package appstripeentity

import (
	"errors"
	"fmt"
	"strings"

	"github.com/stripe/stripe-go/v80"

	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
)

type CreateAppStripeInput struct {
	ID              *string
	Namespace       string
	Name            string
	Description     string
	StripeAccountID string
	Livemode        bool
	APIKey          secretentity.SecretID
	WebhookSecret   secretentity.SecretID
}

func (i CreateAppStripeInput) Validate() error {
	if i.ID != nil && *i.ID == "" {
		return errors.New("id cannot be empty if provided")
	}

	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.Name == "" {
		return errors.New("name is required")
	}

	if i.Description == "" {
		return errors.New("description is required")
	}

	if i.StripeAccountID == "" {
		return errors.New("stripe account id is required")
	}

	if err := i.APIKey.Validate(); err != nil {
		return fmt.Errorf("error validating api key: %w", err)
	}

	if i.APIKey.Namespace != i.Namespace {
		return errors.New("api key must be in the same namespace as the app")
	}

	if err := i.WebhookSecret.Validate(); err != nil {
		return fmt.Errorf("error validating webhook secret: %w", err)
	}

	if i.WebhookSecret.Namespace != i.Namespace {
		return errors.New("webhook secret must be in the same namespace as the app")
	}

	return nil
}

type CreateStripeCustomerInput struct {
	AppID      appentitybase.AppID
	CustomerID customerentity.CustomerID
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

	return nil
}

type CreateStripeCustomerOutput struct {
	StripeCustomerID string
}

func (o CreateStripeCustomerOutput) Validate() error {
	if o.StripeCustomerID == "" {
		return errors.New("stripe customer id is required")
	}

	return nil
}

type StripeClientSetupWebhookInput struct {
	AppID   appentitybase.AppID
	BaseURL string
}

func (i StripeClientSetupWebhookInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return fmt.Errorf("error validating app id: %w", err)
	}

	if i.BaseURL == "" {
		return errors.New("base url is required")
	}

	return nil
}

type (
	StripeClientCreateStripeCustomerInput = CreateStripeCustomerInput
)

type UpsertStripeCustomerDataInput struct {
	AppID            appentitybase.AppID
	CustomerID       customerentity.CustomerID
	StripeCustomerID string
}

func (i UpsertStripeCustomerDataInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return fmt.Errorf("error validating app id: %w", err)
	}

	if err := i.CustomerID.Validate(); err != nil {
		return fmt.Errorf("error validating customer id: %w", err)
	}

	if i.AppID.Namespace != i.CustomerID.Namespace {
		return errors.New("app and customer must be in the same namespace")
	}

	if i.StripeCustomerID == "" {
		return errors.New("stripe customer id is required")
	}

	return nil
}

type DeleteStripeCustomerDataInput struct {
	AppID      *appentitybase.AppID
	CustomerID customerentity.CustomerID
}

func (i DeleteStripeCustomerDataInput) Validate() error {
	if i.CustomerID.ID == "" {
		return errors.New("customer id is required")
	}

	if i.CustomerID.Namespace == "" {
		return errors.New("customer namespace is required")
	}

	if i.AppID != nil {
		if i.AppID.ID == "" {
			return errors.New("app id is required")
		}

		if i.AppID.Namespace == "" {
			return errors.New("app namespace is required")
		}

		if i.AppID.Namespace != i.CustomerID.Namespace {
			return errors.New("app and customer must be in the same namespace")
		}
	}

	return nil
}

type StripeCheckoutSessionOptions struct {
	CancelURL          *string
	ClientReferenceID  *string
	CustomText         *stripe.CheckoutSessionCustomTextParams
	Metadata           map[string]string
	ReturnURL          *string
	SuccessURL         *string
	UIMode             *stripe.CheckoutSessionUIMode
	PaymentMethodTypes *[]*string
}

type GetAppInput = appentitybase.AppID

type CreateCheckoutSessionInput struct {
	AppID            appentitybase.AppID
	CustomerID       customerentity.CustomerID
	StripeCustomerID *string
	Options          StripeCheckoutSessionOptions
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

	if i.StripeCustomerID != nil && !strings.HasPrefix(*i.StripeCustomerID, "cus_") {
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

	return nil
}

type StripeClientCreateCheckoutSessionInput struct {
	AppID            appentitybase.AppID
	CustomerID       customerentity.CustomerID
	StripeCustomerID string
	Options          StripeCheckoutSessionOptions
}

func (i StripeClientCreateCheckoutSessionInput) Validate() error {
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

	return nil
}
