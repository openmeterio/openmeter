package appstripeentity

import (
	"errors"
	"fmt"
	"strings"

	"github.com/stripe/stripe-go/v80"

	"github.com/openmeterio/openmeter/api"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
)

const (
	APIKeySecretKey  = "stripe_api_key"
	WebhookSecretKey = "stripe_webhook_secret"
)

type CreateAppStripeInput struct {
	appentity.CreateAppInput

	StripeAccountID string
	Livemode        bool
	APIKey          secretentity.SecretID
	StripeWebhookID string
	WebhookSecret   secretentity.SecretID
}

func (i CreateAppStripeInput) Validate() error {
	if i.CreateAppInput.Type != appentitybase.AppTypeStripe {
		return errors.New("app type must be stripe")
	}

	if err := i.ID.Validate(); err != nil {
		return errors.New("id cannot be empty if provided")
	}

	if err := i.CreateAppInput.Validate(); err != nil {
		return fmt.Errorf("error validating create app input: %w", err)
	}

	if i.StripeAccountID == "" {
		return errors.New("stripe account id is required")
	}

	if err := i.APIKey.Validate(); err != nil {
		return fmt.Errorf("error validating api key: %w", err)
	}

	if i.ID != nil && i.APIKey.Namespace != i.ID.Namespace {
		return errors.New("api key must be in the same namespace as the app")
	}

	if err := i.WebhookSecret.Validate(); err != nil {
		return fmt.Errorf("error validating webhook secret: %w", err)
	}

	if i.StripeWebhookID == "" {
		return errors.New("stripe webhook id is required")
	}

	if i.ID != nil && i.WebhookSecret.Namespace != i.ID.Namespace {
		return errors.New("webhook secret must be in the same namespace as the app")
	}

	return nil
}

type GetStripeAppDataInput struct {
	AppID appentitybase.AppID
}

func (i GetStripeAppDataInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return fmt.Errorf("error validating app id: %w", err)
	}

	return nil
}

type DeleteStripeAppDataInput struct {
	AppID appentitybase.AppID
}

func (i DeleteStripeAppDataInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return fmt.Errorf("error validating app id: %w", err)
	}

	return nil
}

type GetStripeCustomerDataInput struct {
	AppID      appentitybase.AppID
	CustomerID customerentity.CustomerID
}

func (i GetStripeCustomerDataInput) Validate() error {
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

type CreateStripeCustomerOutput struct {
	StripeCustomerID string
}

func (o CreateStripeCustomerOutput) Validate() error {
	if o.StripeCustomerID == "" {
		return errors.New("stripe customer id is required")
	}

	return nil
}

type UpsertStripeCustomerDataInput struct {
	AppID                        appentitybase.AppID
	CustomerID                   customerentity.CustomerID
	StripeCustomerID             string
	StripeDefaultPaymentMethodID *string
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

	if i.StripeDefaultPaymentMethodID != nil && !strings.HasPrefix(*i.StripeDefaultPaymentMethodID, "pm_") {
		return errors.New("stripe default payment method must start with pm_")
	}

	return nil
}

type DeleteStripeCustomerDataInput struct {
	AppID      *appentitybase.AppID
	CustomerID *customerentity.CustomerID
}

func (i DeleteStripeCustomerDataInput) Validate() error {
	if i.AppID == nil && i.CustomerID == nil {
		return errors.New("app id or customer id is required")
	}

	if i.CustomerID != nil {
		if i.CustomerID.ID == "" {
			return errors.New("customer id is required")
		}

		if i.CustomerID.Namespace == "" {
			return errors.New("customer namespace is required")
		}
	}

	if i.AppID != nil {
		if i.AppID.ID == "" {
			return errors.New("app id is required")
		}

		if i.AppID.Namespace == "" {
			return errors.New("app namespace is required")
		}
	}

	if i.AppID != nil && i.CustomerID != nil && i.AppID.Namespace != i.CustomerID.Namespace {
		return errors.New("app and customer must be in the same namespace")
	}

	return nil
}

type GetAppInput = appentitybase.AppID

type GetWebhookSecretInput struct {
	AppID string
}

func (i GetWebhookSecretInput) Validate() error {
	if i.AppID == "" {
		return errors.New("app id is required")
	}

	return nil
}

type GetWebhookSecretOutput = secretentity.Secret

type UpdateAPIKeyInput struct {
	AppID  appentitybase.AppID
	APIKey string
}

func (i UpdateAPIKeyInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return fmt.Errorf("error validating app id: %w", err)
	}

	if i.APIKey == "" {
		return errors.New("api key is required")
	}

	return nil
}

type CreateCheckoutSessionInput struct {
	Namespace           string
	AppID               *appentitybase.AppID
	CreateCustomerInput *customerentity.CreateCustomerInput
	CustomerID          *customerentity.CustomerID
	CustomerKey         *string
	StripeCustomerID    *string
	Options             api.CreateStripeCheckoutSessionRequestOptions
}

func (i CreateCheckoutSessionInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.AppID != nil {
		if err := i.AppID.Validate(); err != nil {
			return fmt.Errorf("error validating app id: %w", err)
		}
	}

	// Least one of customer or customer id is required
	if i.CreateCustomerInput == nil && i.CustomerID == nil && i.CustomerKey == nil {
		return errors.New("create customer input or customer id or customer key is required")
	}

	// Mutually exclusive
	if i.CreateCustomerInput != nil && i.CustomerID != nil {
		return errors.New("create customer input and customer id cannot be provided at the same time")
	}

	if i.CreateCustomerInput != nil && i.CustomerKey != nil {
		return errors.New("create customer input and customer key cannot be provided at the same time")
	}

	if i.CustomerID != nil && i.CustomerKey != nil {
		return errors.New("create customer id and customer key cannot be provided at the same time")
	}

	if i.CreateCustomerInput != nil {
		if err := i.CreateCustomerInput.Validate(); err != nil {
			return fmt.Errorf("error validating create customer input: %w", err)
		}
	}

	if i.CustomerID != nil {
		if err := i.CustomerID.Validate(); err != nil {
			return fmt.Errorf("error validating customer id: %w", err)
		}
	}

	if i.CustomerID != nil && i.Namespace != i.CustomerID.Namespace {
		return errors.New("app and customer must be in the same namespace")
	}

	if i.CustomerKey != nil && *i.CustomerKey == "" {
		return errors.New("customer key cannot be empty if provided")
	}

	if i.StripeCustomerID != nil && !strings.HasPrefix(*i.StripeCustomerID, "cus_") {
		return errors.New("stripe customer id must start with cus_")
	}

	if i.Options.UiMode != nil {
		switch *i.Options.UiMode {
		case api.CheckoutSessionUIModeEmbedded:
			if i.Options.ReturnURL == nil {
				return errors.New("return url is required for embedded ui mode")
			}

			if i.Options.CancelURL != nil {
				return errors.New("cancel url is not allowed for embedded ui mode")
			}
		case api.CheckoutSessionUIModeHosted:
			if i.Options.SuccessURL == nil {
				return errors.New("success url is required for hosted ui mode")
			}
		}
	}

	return nil
}

type CreateCheckoutSessionOutput struct {
	CustomerID       customerentity.CustomerID
	StripeCustomerID string

	SessionID     string
	SetupIntentID string
	URL           *string
	Mode          stripe.CheckoutSessionMode

	CancelURL  *string
	SuccessURL *string
	ReturnURL  *string
}

func (o CreateCheckoutSessionOutput) Validate() error {
	if err := o.CustomerID.Validate(); err != nil {
		return fmt.Errorf("error validating customer id: %w", err)
	}

	if o.StripeCustomerID == "" {
		return errors.New("stripe customer id is required")
	}

	if o.SessionID == "" {
		return errors.New("session id is required")
	}

	if o.SetupIntentID == "" {
		return errors.New("setup intent id is required")
	}

	return nil
}

type AppBase struct {
	appentitybase.AppBase
	AppData
}

// AppData represents the Stripe associated data for the app
type AppData struct {
	StripeAccountID string
	Livemode        bool
	APIKey          secretentity.SecretID
	StripeWebhookID string
	WebhookSecret   secretentity.SecretID
}

func (d AppData) Validate() error {
	if d.StripeAccountID == "" {
		return errors.New("stripe account id is required")
	}

	if err := d.APIKey.Validate(); err != nil {
		return fmt.Errorf("error validating api key: %w", err)
	}

	if d.StripeWebhookID == "" {
		return errors.New("stripe webhook id is required")
	}

	if err := d.WebhookSecret.Validate(); err != nil {
		return fmt.Errorf("error validating webhook secret: %w", err)
	}

	return nil
}

type SetCustomerDefaultPaymentMethodInput struct {
	AppID            appentitybase.AppID
	StripeCustomerID string
	PaymentMethodID  string
}

type SetCustomerDefaultPaymentMethodOutput struct {
	CustomerID customerentity.CustomerID
}

func (i SetCustomerDefaultPaymentMethodInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return fmt.Errorf("app id: %w", err)
	}

	if i.StripeCustomerID == "" {
		return errors.New("stripe customer id is required")
	}

	if i.PaymentMethodID == "" {
		return errors.New("payment method id is required")
	}

	return nil
}

// GetSupplierContactInput to get the default supplier
type GetSupplierContactInput struct {
	AppID appentitybase.AppID
}

func (i GetSupplierContactInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return fmt.Errorf("error validating app id: %w", err)
	}

	return nil
}
