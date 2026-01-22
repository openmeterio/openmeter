package appstripeentity

import (
	"errors"
	"fmt"
	"strings"

	"github.com/stripe/stripe-go/v80"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
)

const (
	APIKeySecretKey  = "stripe_api_key"
	WebhookSecretKey = "stripe_webhook_secret"
)

type CreateAppStripeInput struct {
	app.CreateAppInput

	StripeAccountID string
	Livemode        bool
	APIKey          secretentity.SecretID
	MaskedAPIKey    string
	StripeWebhookID string
	WebhookSecret   secretentity.SecretID
}

func (i CreateAppStripeInput) Validate() error {
	if i.CreateAppInput.Type != app.AppTypeStripe {
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

	if i.MaskedAPIKey == "" {
		return errors.New("masked api key is required")
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
	AppID app.AppID
}

func (i GetStripeAppDataInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return fmt.Errorf("error validating app id: %w", err)
	}

	return nil
}

type DeleteStripeAppDataInput struct {
	AppID app.AppID
}

func (i DeleteStripeAppDataInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return fmt.Errorf("error validating app id: %w", err)
	}

	return nil
}

type GetStripeCustomerDataInput struct {
	AppID      app.AppID
	CustomerID customer.CustomerID
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
	AppID                        app.AppID
	CustomerID                   customer.CustomerID
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
	AppID      *app.AppID
	CustomerID *customer.CustomerID
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

type GetAppInput = app.AppID

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
	AppID  app.AppID
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

type UpdateAPIKeyAdapterInput struct {
	UpdateAPIKeyInput

	MaskedAPIKey string
}

func (i UpdateAPIKeyAdapterInput) Validate() error {
	if err := i.UpdateAPIKeyInput.Validate(); err != nil {
		return fmt.Errorf("error validating update api key input: %w", err)
	}

	if i.MaskedAPIKey == "" {
		return errors.New("masked api key is required")
	}

	return nil
}

type CreateCheckoutSessionInput struct {
	Namespace           string
	AppID               app.AppID
	CreateCustomerInput *customer.CreateCustomerInput
	CustomerID          *customer.CustomerID
	StripeCustomerID    *string
	Options             api.CreateStripeCheckoutSessionRequestOptions
}

func (i CreateCheckoutSessionInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if err := i.AppID.Validate(); err != nil {
		return fmt.Errorf("error validating app id: %w", err)
	}

	if i.AppID.Namespace != i.Namespace {
		return errors.New("app id and namespace must be in the same namespace")
	}

	// Least one of customer, customer id or customer key is required
	if i.CreateCustomerInput == nil && i.CustomerID == nil {
		return errors.New("create customer input or customer id or customer key is required")
	}

	// Mutually exclusive
	if i.CreateCustomerInput != nil {
		if err := i.CreateCustomerInput.Validate(); err != nil {
			return fmt.Errorf("error validating create customer input: %w", err)
		}

		if i.CustomerID != nil {
			return errors.New("create customer input and customer id cannot be provided at the same time")
		}
	}

	if i.CustomerID != nil {
		if err := i.CustomerID.Validate(); err != nil {
			return fmt.Errorf("error validating customer id: %w", err)
		}

		if i.Namespace != i.CustomerID.Namespace {
			return errors.New("app and customer must be in the same namespace")
		}

		if i.CreateCustomerInput != nil {
			return errors.New("customer id and create customer input cannot be provided at the same time")
		}
	}

	if i.StripeCustomerID != nil && !strings.HasPrefix(*i.StripeCustomerID, "cus_") {
		return errors.New("stripe customer id must start with cus_")
	}

	if i.Options.UiMode != nil {
		switch *i.Options.UiMode {
		case api.CheckoutSessionUIModeEmbedded:
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
	AppID            app.AppID
	CustomerID       customer.CustomerID
	StripeCustomerID string

	client.StripeCheckoutSession
}

func (o CreateCheckoutSessionOutput) Validate() error {
	var errs []error

	if err := o.AppID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("error validating app id: %w", err))
	}

	if err := o.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("error validating customer id: %w", err))
	}

	if o.StripeCustomerID == "" {
		errs = append(errs, errors.New("stripe customer id is required"))
	}

	if err := o.StripeCheckoutSession.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("error validating stripe checkout session: %w", err))
	}

	return errors.Join(errs...)
}

type AppBase struct {
	app.AppBase
	AppData
}

// AppData represents the Stripe associated data for the app
type AppData struct {
	StripeAccountID string                `json:"stripeAccountId"`
	Livemode        bool                  `json:"livemode"`
	APIKey          secretentity.SecretID `json:"-"`
	MaskedAPIKey    string                `json:"maskedApiKey"`
	StripeWebhookID string                `json:"stripeWebhookId"`
	WebhookSecret   secretentity.SecretID `json:"-"`
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
	AppID            app.AppID
	StripeCustomerID string
	PaymentMethodID  string
}

type SetCustomerDefaultPaymentMethodOutput struct {
	CustomerID customer.CustomerID
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

type HandleSetupIntentSucceededInput struct {
	SetCustomerDefaultPaymentMethodInput

	PaymentIntentMetadata map[string]string
}

func (i HandleSetupIntentSucceededInput) Validate() error {
	if err := i.SetCustomerDefaultPaymentMethodInput.Validate(); err != nil {
		return fmt.Errorf("error validating set customer default payment method adapter input: %w", err)
	}

	return nil
}

type HandleSetupIntentSucceededOutput struct {
	CustomerID customer.CustomerID
}

// GetSupplierContactInput to get the default supplier
type GetSupplierContactInput struct {
	AppID app.AppID
}

func (i GetSupplierContactInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return fmt.Errorf("error validating app id: %w", err)
	}

	return nil
}

type ValidationErrorsInput struct {
	Op     billing.StandardInvoiceOperation
	Errors []*stripe.Error
}

type HandleInvoiceStateTransitionInput struct {
	AppID   app.AppID
	Invoice stripe.Invoice

	// Trigger setup

	// Trigger is the state machine trigger that will be used to transition the invoice
	Trigger billing.InvoiceTrigger
	// TargetStatus specifies the expected status of the invoice after the transition, needed to filter
	// for duplicate events as the state machine doesn't allow transition into the same state
	TargetStatuses []billing.StandardInvoiceStatus

	// Event filtering

	// IgnoreInvoiceInStatus is a list of invoice statuses. If the invoice is in this status we ignore the event
	// this allows to filter for out of order events.
	IgnoreInvoiceInStatus []billing.StandardInvoiceStatusMatcher
	// ShouldTriggerOnEvent gets the *current* stripe invoice and returns true if the state machine should be triggered
	// useful for filtering late events based on the current state (optional)
	ShouldTriggerOnEvent func(*stripe.Invoice) (bool, error)

	// Validation errors
	// GetValidationErrors is invoked with the current stripe invoice and returns the validation errors if any
	GetValidationErrors func(*stripe.Invoice) (*ValidationErrorsInput, error)
}

func (i HandleInvoiceStateTransitionInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return fmt.Errorf("error validating app id: %w", err)
	}

	if i.Invoice.ID == "" {
		return errors.New("invoice id is required")
	}

	if i.Trigger == nil {
		return errors.New("trigger is required")
	}

	if len(i.TargetStatuses) == 0 {
		return errors.New("target statuses are required")
	}

	return nil
}

type HandleInvoiceSentEventInput struct {
	AppID   app.AppID
	Invoice stripe.Invoice
	SentAt  int64
}

func (i HandleInvoiceSentEventInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return fmt.Errorf("error validating app id: %w", err)
	}

	if i.Invoice.ID == "" {
		return errors.New("invoice id is required")
	}

	if i.SentAt == 0 {
		return errors.New("sent at is required")
	}

	return nil
}

type GetStripeInvoiceInput struct {
	AppID           app.AppID
	StripeInvoiceID string
}

func (i GetStripeInvoiceInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return fmt.Errorf("error validating app id: %w", err)
	}

	if i.StripeInvoiceID == "" {
		return errors.New("stripe invoice id is required")
	}

	return nil
}
