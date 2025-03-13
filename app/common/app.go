package common

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	appadapter "github.com/openmeterio/openmeter/openmeter/app/adapter"
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
	NewAppService,
	NewAppStripeService,
	NewAppSandboxProvisioner,
)

type AppSandboxProvisioner func() error

func NewAppService(logger *slog.Logger, db *entdb.Client, appsConfig config.AppsConfiguration) (app.Service, error) {
	// TODO: remove this check after enabled by default
	if !appsConfig.Enabled || db == nil {
		return nil, nil
	}

	appAdapter, err := appadapter.New(appadapter.Config{
		Client:  db,
		BaseURL: appsConfig.BaseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create app adapter: %w", err)
	}

	return appservice.New(appservice.Config{
		Adapter: appAdapter,
	})
}

func NewAppStripeService(logger *slog.Logger, db *entdb.Client, appsConfig config.AppsConfiguration, appService app.Service, customerService customer.Service, secretService secret.Service, billingService billing.Service, publisher eventbus.Publisher) (appstripe.Service, error) {
	// TODO: remove this check after enabled by default
	if !appsConfig.Enabled || db == nil {
		return nil, nil
	}

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

	return appstripeservice.New(appstripeservice.Config{
		Adapter:                    appStripeAdapter,
		AppService:                 appService,
		SecretService:              secretService,
		BillingService:             billingService,
		Logger:                     logger,
		DisableWebhookRegistration: appsConfig.Stripe.DisableWebhookRegistration,
		Publisher:                  publisher,
	})
}

func NewAppSandboxProvisioner(ctx context.Context, logger *slog.Logger, appsConfig config.AppsConfiguration, appService app.Service, namespaceManager *namespace.Manager, billingService billing.Service) (AppSandboxProvisioner, error) {
	if !appsConfig.Enabled {
		return nil, nil
	}

	_, err := appsandbox.NewFactory(appsandbox.Config{
		AppService:     appService,
		BillingService: billingService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize app sandbox factory: %w", err)
	}

	return func() error {
		app, err := appsandbox.AutoProvision(ctx, appsandbox.AutoProvisionInput{
			Namespace:  namespaceManager.GetDefaultNamespace(),
			AppService: appService,
		})
		if err != nil {
			return fmt.Errorf("failed to auto-provision sandbox app: %w", err)
		}

		logger.Info("sandbox app auto-provisioned", "app_id", app.GetID().ID)

		return nil
	}, nil
}
