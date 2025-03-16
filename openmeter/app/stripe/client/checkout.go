package client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/stripe/stripe-go/v80"

	"github.com/openmeterio/openmeter/api"
	app "github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// CreateCheckoutSession creates a checkout session
func (c *stripeAppClient) CreateCheckoutSession(ctx context.Context, input CreateCheckoutSessionInput) (StripeCheckoutSession, error) {
	if err := input.Validate(); err != nil {
		return StripeCheckoutSession{}, models.NewGenericValidationError(err)
	}

	metadata := lo.FromPtr(input.Options.Metadata)
	if metadata == nil {
		metadata = map[string]string{}
	}

	metadata[SetupIntentDataMetadataNamespace] = input.AppID.Namespace
	metadata[SetupIntentDataMetadataAppID] = input.AppID.ID
	metadata[SetupIntentDataMetadataCustomerID] = input.CustomerID.ID

	// Create checkout session
	params := &stripe.CheckoutSessionParams{
		Customer: lo.ToPtr(input.StripeCustomerID),
		Mode:     lo.ToPtr(string(stripe.CheckoutSessionModeSetup)),
		SetupIntentData: &stripe.CheckoutSessionSetupIntentDataParams{
			Metadata: metadata,
		},
	}

	if input.Options.BillingAddressCollection != nil {
		params.BillingAddressCollection = lo.ToPtr(string(*input.Options.BillingAddressCollection))
	}

	if input.Options.CancelURL != nil {
		params.CancelURL = input.Options.CancelURL
	}

	if input.Options.ClientReferenceID != nil {
		params.ClientReferenceID = input.Options.ClientReferenceID
	}

	if input.Options.CustomerUpdate != nil {
		params.CustomerUpdate = &stripe.CheckoutSessionCustomerUpdateParams{}

		if input.Options.CustomerUpdate.Address != nil {
			params.CustomerUpdate.Address = lo.ToPtr(string(*input.Options.CustomerUpdate.Address))
		}

		if input.Options.CustomerUpdate.Name != nil {
			params.CustomerUpdate.Name = lo.ToPtr(string(*input.Options.CustomerUpdate.Name))
		}

		if input.Options.CustomerUpdate.Shipping != nil {
			params.CustomerUpdate.Shipping = lo.ToPtr(string(*input.Options.CustomerUpdate.Shipping))
		}
	}

	if input.Options.Currency != nil {
		params.Currency = input.Options.Currency
	}

	if input.Options.ConsentCollection != nil {
		params.ConsentCollection = &stripe.CheckoutSessionConsentCollectionParams{}

		if input.Options.ConsentCollection.PaymentMethodReuseAgreement != nil {
			params.ConsentCollection.PaymentMethodReuseAgreement = &stripe.CheckoutSessionConsentCollectionPaymentMethodReuseAgreementParams{}

			if input.Options.ConsentCollection.PaymentMethodReuseAgreement.Position != nil {
				params.ConsentCollection.PaymentMethodReuseAgreement.Position = lo.ToPtr(string(*input.Options.ConsentCollection.PaymentMethodReuseAgreement.Position))
			}
		}

		if input.Options.ConsentCollection.Promotions != nil {
			params.ConsentCollection.Promotions = lo.ToPtr(string(*input.Options.ConsentCollection.Promotions))
		}

		if input.Options.ConsentCollection.TermsOfService != nil {
			params.ConsentCollection.TermsOfService = lo.ToPtr(string(*input.Options.ConsentCollection.TermsOfService))
		}
	}

	if input.Options.CustomText != nil {
		params.CustomText = &stripe.CheckoutSessionCustomTextParams{}

		if input.Options.CustomText.AfterSubmit != nil {
			params.CustomText.AfterSubmit = &stripe.CheckoutSessionCustomTextAfterSubmitParams{}

			if input.Options.CustomText.AfterSubmit.Message != nil {
				params.CustomText.AfterSubmit.Message = input.Options.CustomText.AfterSubmit.Message
			}
		}

		if input.Options.CustomText.ShippingAddress != nil {
			params.CustomText.ShippingAddress = &stripe.CheckoutSessionCustomTextShippingAddressParams{}

			if input.Options.CustomText.ShippingAddress.Message != nil {
				params.CustomText.ShippingAddress.Message = input.Options.CustomText.ShippingAddress.Message
			}
		}

		if input.Options.CustomText.Submit != nil {
			params.CustomText.Submit = &stripe.CheckoutSessionCustomTextSubmitParams{}

			if input.Options.CustomText.Submit.Message != nil {
				params.CustomText.Submit.Message = input.Options.CustomText.Submit.Message
			}
		}

		if input.Options.CustomText.TermsOfServiceAcceptance != nil {
			params.CustomText.TermsOfServiceAcceptance = &stripe.CheckoutSessionCustomTextTermsOfServiceAcceptanceParams{}

			if input.Options.CustomText.TermsOfServiceAcceptance.Message != nil {
				params.CustomText.TermsOfServiceAcceptance.Message = input.Options.CustomText.TermsOfServiceAcceptance.Message
			}
		}
	}

	if input.Options.ExpiresAt != nil {
		params.ExpiresAt = input.Options.ExpiresAt
	}

	if input.Options.Locale != nil {
		params.Locale = input.Options.Locale
	}

	if input.Options.ReturnURL != nil {
		params.ReturnURL = input.Options.ReturnURL
	}

	if input.Options.SuccessURL != nil {
		params.SuccessURL = input.Options.SuccessURL
	}

	if input.Options.UiMode != nil {
		params.UIMode = lo.ToPtr(string(*input.Options.UiMode))
	}

	if input.Options.PaymentMethodTypes != nil {
		params.PaymentMethodTypes = lo.Map(
			*input.Options.PaymentMethodTypes,
			func(paymentMethodType string, _ int) *string {
				return &paymentMethodType
			},
		)
	}

	if input.Options.RedirectOnCompletion != nil {
		params.RedirectOnCompletion = lo.ToPtr(string(*input.Options.RedirectOnCompletion))
	}

	if input.Options.TaxIdCollection != nil {
		params.TaxIDCollection = &stripe.CheckoutSessionTaxIDCollectionParams{
			Enabled: &input.Options.TaxIdCollection.Enabled,
		}

		if input.Options.TaxIdCollection.Required != nil {
			params.TaxIDCollection.Required = lo.ToPtr(string(*input.Options.TaxIdCollection.Required))
		}
	}

	// Create checkout session
	session, err := c.client.CheckoutSessions.New(params)
	if err != nil {
		return StripeCheckoutSession{}, c.providerError(err)
	}

	// Create output
	if session.SetupIntent == nil {
		return StripeCheckoutSession{}, errors.New("setup intent is required")
	}

	stripeCheckoutSession := StripeCheckoutSession{
		Mode:          session.Mode,
		SessionID:     session.ID,
		SetupIntentID: session.SetupIntent.ID,
		CreatedAt:     time.Unix(session.Created, 0),
	}

	if session.CancelURL != "" {
		stripeCheckoutSession.CancelURL = &session.CancelURL
	}

	if session.ClientSecret != "" {
		stripeCheckoutSession.ClientSecret = &session.ClientSecret
	}

	if session.ClientReferenceID != "" {
		stripeCheckoutSession.ClientReferenceID = &session.ClientReferenceID
	}

	if session.Currency != "" {
		stripeCheckoutSession.Currency = lo.ToPtr(FromStripeCurrency(session.Currency))
	}

	if session.CustomerEmail != "" {
		stripeCheckoutSession.CustomerEmail = &session.CustomerEmail
	}

	if session.Metadata != nil {
		stripeCheckoutSession.Metadata = &session.Metadata
	}

	if session.ReturnURL != "" {
		stripeCheckoutSession.ReturnURL = &session.ReturnURL
	}

	if session.SuccessURL != "" {
		stripeCheckoutSession.SuccessURL = &session.SuccessURL
	}

	if session.Status != "" {
		stripeCheckoutSession.Status = &session.Status
	}

	if session.URL != "" {
		stripeCheckoutSession.URL = &session.URL
	}

	if session.ExpiresAt != 0 {
		stripeCheckoutSession.ExpiresAt = lo.ToPtr(time.Unix(session.ExpiresAt, 0))
	}

	return stripeCheckoutSession, nil
}

type StripeCheckoutSession struct {
	CancelURL         *string
	ClientSecret      *string
	ClientReferenceID *string
	CustomerEmail     *string
	Currency          *currencyx.Code
	CreatedAt         time.Time
	ExpiresAt         *time.Time
	Mode              stripe.CheckoutSessionMode
	Metadata          *map[string]string
	ReturnURL         *string
	SessionID         string
	SetupIntentID     string
	Status            *stripe.CheckoutSessionStatus
	SuccessURL        *string
	URL               *string

	// We don't add payment intent and status here because we always use setup mode
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
	AppID            app.AppID
	CustomerID       customer.CustomerID
	StripeCustomerID string
	Options          api.CreateStripeCheckoutSessionRequestOptions
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

	// Let's validate metadata for reserved keys
	metadata := lo.FromPtr(i.Options.Metadata)
	if metadata != nil {
		for _, reservedKey := range SetupIntentReservedMetadataKeys {
			if _, ok := metadata[reservedKey]; ok {
				return fmt.Errorf("metadata key %s is reserved", reservedKey)
			}
		}
	}
	return nil
}
