package appsandbox

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
)

var _ customerentity.App = (*App)(nil)

type App struct {
	appentitybase.AppBase
}

func (a App) ValidateCustomer(ctx context.Context, customer *customerentity.Customer, capabilities []appentitybase.CapabilityType) error {
	if err := a.ValidateCapabilities(capabilities...); err != nil {
		return fmt.Errorf("error validating capabilities: %w", err)
	}

	return nil
}

type Factory struct {
	appService app.Service
}

type Config struct {
	AppService app.Service
}

func (c Config) Validate() error {
	if c.AppService == nil {
		return fmt.Errorf("app service is required")
	}

	return nil
}

func NewFactory(config Config) (*Factory, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	fact := &Factory{
		appService: config.AppService,
	}

	err := config.AppService.RegisterMarketplaceListing(appentity.RegistryItem{
		Listing: MarketplaceListing,
		Factory: fact,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register marketplace listing: %w", err)
	}

	return fact, nil
}

// Factory
func (a *Factory) NewApp(ctx context.Context, appBase appentitybase.AppBase) (appentity.App, error) {
	return App{
		AppBase: appBase,
	}, nil
}

func (a *Factory) InstallAppWithAPIKey(ctx context.Context, input appentity.AppFactoryInstallAppWithAPIKeyInput) (appentity.App, error) {
	// Validate input
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	appBase, err := a.appService.CreateApp(ctx, appentity.CreateAppInput{
		Namespace: input.Namespace,
		Name:      input.Name,
		Type:      appentitybase.AppTypeSandbox,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create app: %w", err)
	}

	return appBase, nil
}

func (a *Factory) UninstallApp(ctx context.Context, input appentity.UninstallAppInput) error {
	return nil
}
