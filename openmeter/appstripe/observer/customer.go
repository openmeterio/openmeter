package appstripeobserver

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appobserver "github.com/openmeterio/openmeter/openmeter/app/observer"
	"github.com/openmeterio/openmeter/openmeter/appstripe"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

var _ appobserver.Observer[customer.Customer] = (*CustomerObserver)(nil)

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

func (c CustomerObserver) PostCreate(customer *customer.Customer) error {
	return c.upsertDefault(customer)
}

func (c CustomerObserver) PostUpdate(customer *customer.Customer) error {
	return c.upsertDefault(customer)
}

func (c CustomerObserver) PostDelete(customer *customer.Customer) error {
	// TODO: we need more information to decide to which app this stripe customer belongs to the default app
	app, err := c.getDefaultApp(context.Background(), customer.GetID().Namespace)
	if err != nil {
		return fmt.Errorf("failed to get default app: %w", err)
	}

	// Delete stripe customer data
	err = c.appstripeService.DeleteStripeCustomerData(context.Background(), appstripeentity.DeleteStripeCustomerDataInput{
		AppID:      app.GetID(),
		CustomerID: customer.GetID(),
	})
	if err != nil {
		return fmt.Errorf("failed to delete stripe customer data: %w", err)
	}

	return nil
}

// upsertDefault upserts default stripe customer data
func (c CustomerObserver) upsertDefault(customer *customer.Customer) error {
	if customer.External == nil || customer.External.StripeCustomerID == nil {
		return nil
	}

	// TODO: we need more information to decide to which app this stripe customer belongs to the default app
	app, err := c.getDefaultApp(context.Background(), customer.GetID().Namespace)
	if err != nil {
		return fmt.Errorf("failed to get default app: %w", err)
	}

	// Upsert stripe customer data
	err = c.appstripeService.UpsertStripeCustomerData(context.Background(), appstripeentity.UpsertStripeCustomerDataInput{
		AppID:            app.GetID(),
		CustomerID:       customer.GetID(),
		StripeCustomerID: *customer.External.StripeCustomerID,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert stripe customer data: %w", err)
	}

	return nil
}

// TODO: use explicit default insted of returning the first one
func (c CustomerObserver) getDefaultApp(ctx context.Context, namespace string) (appentity.App, error) {
	appType := appentity.AppTypeStripe

	apps, err := c.appService.ListApps(ctx, appentity.ListAppInput{
		Namespace: namespace,
		Type:      &appType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get default app for namespace %s: %w", namespace, err)
	}

	if len(apps.Items) == 0 {
		return nil, errors.New("no default app found")
	}

	return apps.Items[0], nil
}
