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
	appstripedb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripe"
	appstripecustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripecustomer"
	"github.com/openmeterio/openmeter/openmeter/secret"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
	"github.com/openmeterio/openmeter/pkg/models"
)

const APIKeySecretKey = "stripe_api_key"

// App represents an installed Stripe app
type App struct {
	appentitybase.AppBase
	StripeAccountId string `json:"stripeAccountId"`
	Livemode        bool   `json:"livemode"`

	Client              *entdb.Client
	StripeClientFactory StripeClientFactory
	SecretService       secret.Service
}

func (a App) Validate() error {
	if err := a.AppBase.Validate(); err != nil {
		return fmt.Errorf("error validating app: %w", err)
	}

	if a.Type != appentitybase.AppTypeStripe {
		return errors.New("app type must be stripe")
	}

	if a.Client == nil {
		return errors.New("client is required")
	}

	if a.StripeAccountId == "" {
		return errors.New("stripe account id is required")
	}

	if a.StripeClientFactory == nil {
		return errors.New("stripe client factory is required")
	}

	if a.SecretService == nil {
		return errors.New("secret service is required")
	}

	return nil
}

// ValidateCustomer validates if the app can run for the given customer
func (a App) ValidateCustomer(ctx context.Context, customer *customerentity.Customer, capabilities []appentitybase.CapabilityType) error {
	// Validate if the app supports the given capabilities
	if err := a.ValidateCapabilities(capabilities); err != nil {
		return fmt.Errorf("error validating capabilities: %w", err)
	}

	// Get Stripe Customer
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

	// Get Stripe App
	stripeApp, err := a.Client.AppStripe.
		Query().
		Where(appstripedb.Namespace(a.Namespace)).
		Where(appstripedb.ID(a.ID)).
		First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return app.AppNotFoundError{
				AppID: a.GetID(),
			}
		}

		return fmt.Errorf("failed to get stripe app: %w", err)
	}

	// Get Stripe API Key
	apiKeySecret, err := a.SecretService.GetAppSecret(ctx, secretentity.GetAppSecretInput{
		NamespacedID: models.NamespacedID{
			Namespace: stripeApp.Namespace,
			ID:        stripeApp.ID,
		},
		Key: APIKeySecretKey,
	})
	if err != nil {
		return fmt.Errorf("failed to get stripe api key secret: %w", err)
	}

	// Stripe Client
	stripeClient, err := a.StripeClientFactory(StripeClientConfig{
		Namespace: stripeApp.Namespace,
		APIKey:    apiKeySecret.Value,
	})
	if err != nil {
		return fmt.Errorf("failed to create stripe client: %w", err)
	}

	// Check if the customer exists in Stripe
	_, err = stripeClient.GetCustomer(ctx, *stripeCustomer.StripeCustomerID)
	if err != nil {
		if _, ok := err.(stripeCustomerNotFoundError); ok {
			return app.CustomerPreConditionError{
				AppID:      a.GetID(),
				AppType:    a.GetType(),
				CustomerID: customer.GetID(),
				Condition:  fmt.Sprintf("stripe customer %s not found in stripe account %s", *stripeCustomer.StripeCustomerID, stripeApp.StripeAccountID),
			}
		}

		return err
	}

	// Invoice and payment capabilities need to check if the customer has a country and default payment method via the Stripe API
	if slices.Contains(capabilities, appentitybase.CapabilityTypeCalculateTax) || slices.Contains(capabilities, appentitybase.CapabilityTypeInvoiceCustomers) || slices.Contains(capabilities, appentitybase.CapabilityTypeCollectPayments) {
		// Get payment methods
		paymentMethods, err := stripeClient.GetCustomerPaymentMethods(ctx, *stripeCustomer.StripeCustomerID)
		if err != nil {
			return fmt.Errorf("failed to get stripe customer payment methods: %w", err)
		}

		// Must have at least one payment method
		if len(paymentMethods) == 0 {
			return app.CustomerPreConditionError{
				AppID:      a.GetID(),
				AppType:    a.GetType(),
				CustomerID: customer.GetID(),
				Condition:  "customer must have at least one payment method",
			}
		}

		// Must have at least one payment method with a billing address
		// Billing address is required for tax calculation and invoice creation
		hasAddress := false

		for _, paymentMethod := range paymentMethods {
			if paymentMethod.BillingAddress != nil {
				hasAddress = true
				break
			}
		}

		if !hasAddress {
			return app.CustomerPreConditionError{
				AppID:      a.GetID(),
				AppType:    a.GetType(),
				CustomerID: customer.GetID(),
				Condition:  "customer must have at least one payment method with a billing address",
			}
		}
	}

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

type StripePaymentMethod struct {
	ID             string
	BillingAddress *models.Address
}
