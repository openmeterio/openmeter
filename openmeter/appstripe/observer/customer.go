package appstripeobserver

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	appobserver "github.com/openmeterio/openmeter/openmeter/app/observer"
	"github.com/openmeterio/openmeter/openmeter/appstripe"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
)

var _ appobserver.Observer[customerentity.Customer] = (*CustomerObserver)(nil)

type CustomerObserver struct {
	appService       app.Service
	appstripeService appstripe.Service
}

type Config struct {
	AppService       app.Service
	AppstripeService appstripe.Service
}

func (c Config) Validate() error {
	if c.AppService == nil {
		return errors.New("app service cannot be null")
	}

	if c.AppstripeService == nil {
		return errors.New("app stripe service cannot be null")
	}

	return nil
}

func New(config Config) (*CustomerObserver, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &CustomerObserver{
		appService:       config.AppService,
		appstripeService: config.AppstripeService,
	}, nil
}

func (c CustomerObserver) PostCreate(customer *customerentity.Customer) error {
	return c.upsertDefault(customer)
}

func (c CustomerObserver) PostUpdate(customer *customerentity.Customer) error {
	return c.upsertDefault(customer)
}

func (c CustomerObserver) PostDelete(customer *customerentity.Customer) error {
	// Delete stripe customer data for all Stripe apps for the customer in the namespace
	err := c.appstripeService.DeleteStripeCustomerData(context.Background(), appstripeentity.DeleteStripeCustomerDataInput{
		CustomerID: customer.GetID(),
	})
	if err != nil {
		return fmt.Errorf("failed to delete stripe customer data: %w", err)
	}

	return nil
}

// upsertDefault upserts default stripe customer data
func (c CustomerObserver) upsertDefault(customer *customerentity.Customer) error {
	// if customer.External == nil || customer.External.StripeCustomerID == nil {
	// 	return nil
	// }

	// // Get default app
	// // TODO: we need more information to decide to which app this stripe customer belongs to the default app
	// app, err := c.appService.GetDefaultApp(context.Background(), appentity.GetDefaultAppInput{
	// 	Namespace: customer.GetID().Namespace,
	// 	Type:      appentity.AppTypeStripe,
	// })
	// if err != nil {
	// 	return fmt.Errorf("failed to get default app: %w", err)
	// }

	// // Upsert stripe customer data
	// err = c.appstripeService.UpsertStripeCustomerData(context.Background(), appstripeentity.UpsertStripeCustomerDataInput{
	// 	AppID:            app.GetID(),
	// 	CustomerID:       customer.GetID(),
	// 	StripeCustomerID: *customer.External.StripeCustomerID,
	// })
	// if err != nil {
	// 	return fmt.Errorf("failed to upsert stripe customer data: %w", err)
	// }

	return nil
}
