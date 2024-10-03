package appstripeentity

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	appstripecustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripecustomer"
)

const APIKeySecretKey = "stripe_api_key"

// App represents an installed Stripe app
type App struct {
	appentitybase.AppBase

	Client *entdb.Client
	// BillingService   billing.Service

	StripeAccountId string `json:"stripeAccountId"`
	Livemode        bool   `json:"livemode"`
}

func (a App) Validate() error {
	if err := a.AppBase.Validate(); err != nil {
		return fmt.Errorf("error validating app: %w", err)
	}

	if a.Type != appentitybase.AppTypeStripe {
		return errors.New("app type must be stripe")
	}

	if a.StripeAccountId == "" {
		return errors.New("stripe account id is required")
	}

	if a.Client == nil {
		return errors.New("client is required")
	}

	return nil
}

// ValidateCustomer validates if the app can run for the given customer
func (a App) ValidateCustomer(ctx context.Context, customer *customerentity.Customer, capabilities []appentitybase.CapabilityType) error {
	// Validate if the app supports the given capabilities
	if err := a.ValidateCapabilities(capabilities); err != nil {
		return fmt.Errorf("error validating capabilities: %w", err)
	}

	stripeCustomer, err := a.Client.AppStripeCustomer.
		Query().
		Where(appstripecustomerdb.Namespace(a.Namespace)).
		Where(appstripecustomerdb.AppID(a.ID)).
		Where(appstripecustomerdb.CustomerID(customer.ID)).
		Only(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return app.CustomerPreConditionError{
				AppID:      a.GetID(),
				AppType:    a.GetType(),
				CustomerID: customer.GetID(),
				Condition:  "customer has no data for stripe app",
			}
		}

		return fmt.Errorf("error getting stripe customer: %w", err)
	}

	// Check if the customer has a Stripe customer ID
	if stripeCustomer.StripeCustomerID == nil || *stripeCustomer.StripeCustomerID == "" {
		return app.CustomerPreConditionError{
			AppID:      a.GetID(),
			AppType:    a.GetType(),
			CustomerID: customer.GetID(),
			Condition:  "customer must have a stripe customer id",
		}
	}

	// TODO: check if the customer exists in Stripe

	// TODO: implement
	// Invoice and payment capabilities need to check if the customer has a country and default payment method via the Stripe API
	// if slices.Contains(capabilities, appentitybase.CapabilityTypeCalculateTax) || slices.Contains(capabilities, appentitybase.CapabilityTypeInvoiceCustomers) || slices.Contains(capabilities, appentitybase.CapabilityTypeCollectPayments) {
	// 	// TODO: go to Stripe and check if customer exists by customer.External.StripeCustomerID
	// 	// Also check if the customer has a country and default payment method

	// 	return errors.New("not implemented")
	// }

	return nil
}

// CustomerAppData represents the Stripe associated data for an app used by a customer
type CustomerAppData struct {
	StripeCustomerID string `json:"stripeCustomerId"`
}

func (d CustomerAppData) Validate() error {
	if d.StripeCustomerID == "" {
		return errors.New("stripe customer id is required")
	}

	return nil
}

type StripeAccount struct {
	StripeAccountID string
}

type StripeCustomer struct {
	StripeCustomerID string
}
