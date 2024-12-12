package common

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	appadapter "github.com/openmeterio/openmeter/openmeter/app/adapter"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	appservice "github.com/openmeterio/openmeter/openmeter/app/service"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	appstripeadapter "github.com/openmeterio/openmeter/openmeter/app/stripe/adapter"
	appstripeservice "github.com/openmeterio/openmeter/openmeter/app/stripe/service"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/secret"
)

func NewAppService(logger *slog.Logger, db *entdb.Client, appsConfig config.AppsConfiguration) (app.Service, error) {
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

func NewAppStripeService(logger *slog.Logger, db *entdb.Client, appService app.Service, customerService customer.Service, secretService secret.Service) (appstripe.Service, error) {
	appStripeAdapter, err := appstripeadapter.New(appstripeadapter.Config{
		Client:          db,
		AppService:      appService,
		CustomerService: customerService,
		SecretService:   secretService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create appstripe adapter: %w", err)
	}

	_, err = appstripeservice.New(appstripeservice.Config{
		Adapter:       appStripeAdapter,
		AppService:    appService,
		SecretService: secretService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create appstripe service: %w", err)
	}

	return appstripeservice.New(appstripeservice.Config{
		Adapter:       appStripeAdapter,
		AppService:    appService,
		SecretService: secretService,
	})
}

func NewAppSandbox(ctx context.Context, logger *slog.Logger, db *entdb.Client, appService app.Service, namespaceManager *namespace.Manager) (appsandbox.App, error) {
	_, err := appsandbox.NewFactory(appsandbox.Config{
		AppService: appService,
	})
	if err != nil {
		return appsandbox.App{}, fmt.Errorf("failed to initialize app sandbox factory: %w", err)
	}

	app, err := appsandbox.AutoProvision(ctx, appsandbox.AutoProvisionInput{
		Namespace:  namespaceManager.GetDefaultNamespace(),
		AppService: appService,
	})
	if err != nil {
		return appsandbox.App{}, fmt.Errorf("failed to auto-provision sandbox app: %w", err)
	}

	logger.Info("sandbox app auto-provisioned", "app_id", app.GetID().ID)

	appSandbox, ok := app.(appsandbox.App)
	if !ok {
		return appsandbox.App{}, fmt.Errorf("failed to cast app to sandbox app")
	}

	return appSandbox, nil
}
