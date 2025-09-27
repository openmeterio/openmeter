package client

import (
	"context"
	"errors"
	"time"

	"github.com/samber/lo"
	"github.com/stripe/stripe-go/v80"

	"github.com/openmeterio/openmeter/pkg/models"
)

// CreatePortalSessionInput is the input for creating a customer portal session.
type CreatePortalSessionInput struct {
	StripeCustomerID string
	ConfigurationID  *string
	ReturnURL        *string
	Locale           *string
}

// Validate validates the input for creating a customer portal session.
func (i CreatePortalSessionInput) Validate() error {
	var errs []error

	if i.StripeCustomerID == "" {
		errs = append(errs, models.NewGenericValidationError(errors.New("stripe customer id is required")))
	}

	if i.ReturnURL != nil && *i.ReturnURL == "" {
		errs = append(errs, models.NewGenericValidationError(errors.New("return url cannot be empty if provided")))
	}

	if i.Locale != nil && *i.Locale == "" {
		errs = append(errs, models.NewGenericValidationError(errors.New("locale cannot be empty if provided")))
	}

	return errors.Join(errs...)
}

// PortalSession is the response from the Stripe API for a customer portal session.
type PortalSession struct {
	// The ID of the customer portal session.
	// See: https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-id
	ID string

	// Configuration Configuration used to customize the customer portal.
	// See: https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-configuration
	Configuration    *stripe.BillingPortalConfiguration
	CreatedAt        time.Time
	StripeCustomerID string

	// Livemode Livemode.
	Livemode bool

	// Locale Status.
	// The IETF language tag of the locale customer portal is displayed in.
	// See: https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-locale
	Locale string

	// ReturnUrl Return URL.
	// See: https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-return_url
	ReturnURL string

	// The URL to redirect the customer to after they have completed
	// their requested actions.
	URL string
}

// CreatePortalSession creates a customer portal session.
func (c *stripeAppClient) CreatePortalSession(ctx context.Context, input CreatePortalSessionInput) (PortalSession, error) {
	if err := input.Validate(); err != nil {
		return PortalSession{}, err
	}

	portalSession, err := c.client.BillingPortalSessions.New(&stripe.BillingPortalSessionParams{
		Customer:      lo.ToPtr(input.StripeCustomerID),
		Configuration: input.ConfigurationID,
		ReturnURL:     input.ReturnURL,
		Locale:        input.Locale,
	})
	if err != nil {
		// Stripe customer not found error
		if stripeErr, ok := err.(*stripe.Error); ok && stripeErr.Code == stripe.ErrorCodeResourceMissing {
			return PortalSession{}, NewStripeCustomerNotFoundError(input.StripeCustomerID)
		}

		return PortalSession{}, c.providerError(err)
	}

	stripePortalSession := PortalSession{
		ID:               portalSession.ID,
		Configuration:    portalSession.Configuration,
		StripeCustomerID: portalSession.Customer,
		Livemode:         portalSession.Livemode,
		Locale:           portalSession.Locale,
		ReturnURL:        portalSession.ReturnURL,
		URL:              portalSession.URL,
		CreatedAt:        time.Unix(portalSession.Created, 0),
	}

	return stripePortalSession, nil
}
