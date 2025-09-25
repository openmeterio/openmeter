package common

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	appadapter "github.com/openmeterio/openmeter/openmeter/app/adapter"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	appcustominvoicingadapter "github.com/openmeterio/openmeter/openmeter/app/custominvoicing/adapter"
	appcustominvoicingservice "github.com/openmeterio/openmeter/openmeter/app/custominvoicing/service"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	appservice "github.com/openmeterio/openmeter/openmeter/app/service"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	appstripeadapter "github.com/openmeterio/openmeter/openmeter/app/stripe/adapter"
	appstripeservice "github.com/openmeterio/openmeter/openmeter/app/stripe/service"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/secret"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

var App = wire.NewSet(
	NewAppRegistry,
	NewAppService,
	NewAppStripeService,
	NewAppSandboxFactory,
	NewAppSandboxProvisioner,
	NewAppCustomInvoicingService,
)

type AppSandboxProvisioner func(ctx context.Context, orgID string) error

func NewAppService(
	logger *slog.Logger,
	db *entdb.Client,
	publisher eventbus.Publisher,
) (app.Service, error) {
	appAdapter, err := appadapter.New(appadapter.Config{
		Client: db,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create app adapter: %w", err)
	}

	return appservice.New(appservice.Config{
		Adapter:   appAdapter,
		Publisher: publisher,
	})
}

func NewAppStripeService(logger *slog.Logger, db *entdb.Client, appsConfig config.AppsConfiguration, appService app.Service, customerService customer.Service, secretService secret.Service, billingService billing.Service, publisher eventbus.Publisher) (appstripe.Service, error) {
	appStripeAdapter, err := appstripeadapter.New(appstripeadapter.Config{
		Client:          db,
		AppService:      appService,
		CustomerService: customerService,
		SecretService:   secretService,
		Logger:          logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create appstripe adapter: %w", err)
	}

	webhookGenerator, err := appstripeservice.NewBaseURLWebhookURLGenerator(appsConfig.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook generator: %w", err)
	}

	if appsConfig.Stripe.WebhookURLPattern != "" {
		webhookGenerator, err = appstripeservice.NewPatternWebhookURLGenerator(appsConfig.Stripe.WebhookURLPattern)
		if err != nil {
			return nil, fmt.Errorf("failed to create webhook generator: %w", err)
		}
	}

	return appstripeservice.New(appstripeservice.Config{
		Adapter:                    appStripeAdapter,
		AppService:                 appService,
		SecretService:              secretService,
		BillingService:             billingService,
		Logger:                     logger,
		DisableWebhookRegistration: appsConfig.Stripe.DisableWebhookRegistration,
		Publisher:                  publisher,
		WebhookURLGenerator:        webhookGenerator,
	})
}

func NewAppSandboxFactory(
	appsConfig config.AppsConfiguration,
	appService app.Service,
	billingService billing.Service,
) (*appsandbox.Factory, error) {
	factory, err := appsandbox.NewFactory(appsandbox.Config{
		AppService:     appService,
		BillingService: billingService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize app sandbox factory: %w", err)
	}

	return factory, nil
}

func NewAppSandboxProvisioner(ctx context.Context, logger *slog.Logger, appsConfig config.AppsConfiguration, appService app.Service, namespaceManager *namespace.Manager, billingService billing.Service, _ *appsandbox.Factory,
) (AppSandboxProvisioner, error) {
	return func(ctx context.Context, orgID string) error {
		app, err := appsandbox.AutoProvision(ctx, appsandbox.AutoProvisionInput{
			Namespace:  orgID,
			AppService: appService,
		})
		if err != nil {
			return fmt.Errorf("failed to auto-provision sandbox app: %w", err)
		}

		logger.Info("sandbox app auto-provisioned", "app_id", app.GetID().ID, "org_id", orgID)

		return nil
	}, nil
}

func NewAppCustomInvoicingService(logger *slog.Logger, db *entdb.Client, appsConfig config.AppsConfiguration, appService app.Service, customerService customer.Service, secretService secret.Service, billingService billing.Service, publisher eventbus.Publisher) (appcustominvoicing.Service, error) {
	appCustomInvoicingAdapter, err := appcustominvoicingadapter.New(appcustominvoicingadapter.Config{
		Client: db,
		Logger: logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create appcustominvoicing adapter: %w", err)
	}

	service, err := appcustominvoicingservice.New(appcustominvoicingservice.Config{
		Adapter:        appCustomInvoicingAdapter,
		Logger:         logger,
		AppService:     appService,
		BillingService: billingService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create appcustominvoicing service: %w", err)
	}

	// This registers the app with the marketplace as a side-effect
	_, err = appcustominvoicing.NewFactory(appcustominvoicing.FactoryConfig{
		AppService:             appService,
		CustomInvoicingService: service,
		BillingService:         billingService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create appcustominvoicing factory: %w", err)
	}

	return service, nil
}

type AppRegistry struct {
	Service            app.Service
	SandboxProvisioner AppSandboxProvisioner
	Stripe             appstripe.Service
	CustomInvoicing    appcustominvoicing.Service
}

func NewAppRegistry(
	Service app.Service,
	SandboxProvisioner AppSandboxProvisioner,
	Stripe appstripe.Service,
	CustomInvoicing appcustominvoicing.Service,
) AppRegistry {
	return AppRegistry{
		Service:            Service,
		SandboxProvisioner: SandboxProvisioner,
		Stripe:             Stripe,
		CustomInvoicing:    CustomInvoicing,
	}
}
