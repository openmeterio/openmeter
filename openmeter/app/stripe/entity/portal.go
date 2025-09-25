package appstripeentity

import (
	"errors"
	"fmt"
	"time"

	"github.com/stripe/stripe-go/v82"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
)

// CreateStripePortalSessionInput is the input for creating a stripe customer portal session.
type CreateStripePortalSessionInput struct {
	AppID           app.AppID
	CustomerID      customer.CustomerID
	Locale          *string
	ConfigurationID *string
	ReturnURL       *string
}

// Validate validates the input for creating a stripe customer portal session.
func (i CreateStripePortalSessionInput) Validate() error {
	var errs []error

	if err := i.AppID.Validate(); err != nil {
		errs = append(errs, models.NewGenericValidationError(fmt.Errorf("app id is required: %w", err)))
	}

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, models.NewGenericValidationError(errors.New("customer id is required")))
	}

	if i.ReturnURL != nil && *i.ReturnURL == "" {
		errs = append(errs, models.NewGenericValidationError(errors.New("return url cannot be empty if provided")))
	}

	if i.Locale != nil && *i.Locale == "" {
		errs = append(errs, models.NewGenericValidationError(errors.New("locale cannot be empty if provided")))
	}

	return errors.Join(errs...)
}

// StripePortalSession is the response from the Stripe API for a customer portal session.
type StripePortalSession struct {
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
