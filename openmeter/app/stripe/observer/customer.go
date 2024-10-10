package observer

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	appobserver "github.com/openmeterio/openmeter/openmeter/app/observer"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
)

var _ appobserver.Observer[customerentity.Customer] = (*CustomerObserver)(nil)

type CustomerObserver struct {
	appService       app.Service
	appStripeAdapter appstripe.Adapter
}

type CustomerObserverConfig struct {
	AppService       app.Service
	AppStripeAdapter appstripe.Adapter
}

func (c CustomerObserverConfig) Validate() error {
	if c.AppService == nil {
		return errors.New("app service cannot be null")
	}

	if c.AppStripeAdapter == nil {
		return errors.New("app stripe adapter cannot be null")
	}

	return nil
}

func NewCustomerObserver(config CustomerObserverConfig) (*CustomerObserver, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &CustomerObserver{
		appService:       config.AppService,
		appStripeAdapter: config.AppStripeAdapter,
	}, nil
}

func (c CustomerObserver) PostCreate(ctx context.Context, customer *customerentity.Customer) error {
	return c.upsert(ctx, customer)
}

func (c CustomerObserver) PostUpdate(ctx context.Context, customer *customerentity.Customer) error {
	return c.upsert(ctx, customer)
}

func (c CustomerObserver) PostDelete(ctx context.Context, customer *customerentity.Customer) error {
	// Delete stripe customer data for all Stripe apps for the customer in the namespace
	err := c.appStripeAdapter.DeleteStripeCustomerData(ctx, appstripeentity.DeleteStripeCustomerDataInput{
		CustomerID: customer.GetID(),
	})
	if err != nil {
		return fmt.Errorf("failed to delete stripe customer data: %w", err)
	}

	return nil
}

// upsert upserts default stripe customer data
func (c CustomerObserver) upsert(ctx context.Context, customer *customerentity.Customer) error {
	var defaultAppID *appentitybase.AppID

	for _, customerApp := range customer.Apps {
		// Skip non stripe apps
		if customerApp.Type != appentitybase.AppTypeStripe {
			continue
		}

		// Cast app data to stripe customer data
		appStripeCustomer, ok := customerApp.Data.(appstripeentity.CustomerAppData)
		if !ok {
			return errors.New("failed to cast app data to stripe customer data")
		}

		var appID appentitybase.AppID

		// If there is no app id, it's the default app
		if customerApp.AppID != nil {
			appID = *customerApp.AppID
		} else {
			if defaultAppID != nil {
				return fmt.Errorf("multiple default stripe apps found: %s, %s in namespace %s", defaultAppID.ID, customer.GetID(), defaultAppID.Namespace)
			}

			// Get default app
			app, err := c.appService.GetDefaultApp(ctx, appentity.GetDefaultAppInput{
				Namespace: customer.GetID().Namespace,
				Type:      appentitybase.AppTypeStripe,
			})
			if err != nil {
				return fmt.Errorf("failed to get default app: %w", err)
			}

			id := app.GetID()

			appID = id
			defaultAppID = &id
		}

		// Upsert stripe customer data
		upsertStripeCustomerInput := appstripeentity.UpsertStripeCustomerDataInput{
			AppID:            appID,
			CustomerID:       customer.GetID(),
			StripeCustomerID: appStripeCustomer.StripeCustomerID,
		}

		if err := upsertStripeCustomerInput.Validate(); err != nil {
			return fmt.Errorf("failed to validate upsert stripe customer data input: %w", err)
		}

		err := c.appStripeAdapter.UpsertStripeCustomerData(ctx, upsertStripeCustomerInput)
		if err != nil {
			return fmt.Errorf("failed to upsert stripe customer data: %w", err)
		}
	}

	return nil
}
