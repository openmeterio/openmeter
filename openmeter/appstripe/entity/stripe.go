package appstripeentity

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	appstripecustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripecustomer"
)

var (
	StripeMarketplaceListing = appentitybase.MarketplaceListing{
		Type:        appentitybase.AppTypeStripe,
		Name:        "Stripe",
		Description: "Stripe is a payment processing platform.",
		IconURL:     "https://stripe.com/favicon.ico",
		Capabilities: []appentitybase.Capability{
			StripeCollectPaymentCapability,
			StripeCalculateTaxCapability,
			StripeInvoiceCustomerCapability,
		},
	}

	StripeCollectPaymentCapability = appentitybase.Capability{
		Type:        appentitybase.CapabilityTypeCollectPayments,
		Key:         "stripe_collect_payment",
		Name:        "Payment",
		Description: "Process payments",
	}

	StripeCalculateTaxCapability = appentitybase.Capability{
		Type:        appentitybase.CapabilityTypeCalculateTax,
		Key:         "stripe_calculate_tax",
		Name:        "Calculate Tax",
		Description: "Calculate tax for a payment",
	}

	StripeInvoiceCustomerCapability = appentitybase.Capability{
		Type:        appentitybase.CapabilityTypeInvoiceCustomers,
		Key:         "stripe_invoice_customer",
		Name:        "Invoice Customer",
		Description: "Invoice a customer",
	}
)

// if err != nil {
// 	return nil, fmt.Errorf("failed to register marketplace listing: %w", err)
// }

// App represents an installed Stripe app
type App struct {
	appentitybase.AppBase

	// TODO: cycle dependencies
	Client *entdb.Client
	// AppService app.Service
	// AppStripeService appstripe.Service
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

	// Invoice and payment capabilities need to check if the customer has a country and default payment method via the Stripe API
	if slices.Contains(capabilities, appentitybase.CapabilityTypeCalculateTax) || slices.Contains(capabilities, appentitybase.CapabilityTypeInvoiceCustomers) || slices.Contains(capabilities, appentitybase.CapabilityTypeCollectPayments) {
		// TODO: go to Stripe and check if customer exists by customer.External.StripeCustomerID
		// Also check if the customer has a country and default payment method

		// return errors.New("not implemented")
	}

	return nil
}

type CustomerAppData struct {
	StripeCustomerID string `json:"stripeCustomerId"`
}

func (d CustomerAppData) Validate() error {
	if d.StripeCustomerID == "" {
		return errors.New("stripe customer id is required")
	}

	return nil
}

type CreateAppStripeInput struct {
	Namespace       string
	Name            string
	Description     string
	StripeAccountID string
	Livemode        bool
}

func (i CreateAppStripeInput) Validate() error {
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

	return nil
}

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
