package appsandbox

import (
	"context"
	"fmt"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	customerapp "github.com/openmeterio/openmeter/openmeter/customer/app"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
)

const (
	InvoiceTSFormat = "20060102-150405"
)

var (
	_ customerapp.App            = (*App)(nil)
	_ billingentity.InvoicingApp = (*App)(nil)
	_ appentity.CustomerData     = (*CustomerData)(nil)
)

type App struct {
	appentitybase.AppBase
}

func (a App) ValidateCustomer(ctx context.Context, customer *customerentity.Customer, capabilities []appentitybase.CapabilityType) error {
	if err := a.ValidateCapabilities(capabilities...); err != nil {
		return fmt.Errorf("error validating capabilities: %w", err)
	}

	return nil
}

func (a App) GetCustomerData(ctx context.Context, input appentity.GetAppInstanceCustomerDataInput) (appentity.CustomerData, error) {
	return CustomerData{}, nil
}

func (a App) UpsertCustomerData(ctx context.Context, input appentity.UpsertAppInstanceCustomerDataInput) error {
	return nil
}

func (a App) DeleteCustomerData(ctx context.Context, input appentity.DeleteAppInstanceCustomerDataInput) error {
	return nil
}

func (a App) ValidateInvoice(ctx context.Context, invoice billingentity.Invoice) error {
	return nil
}

func (a App) UpsertInvoice(ctx context.Context, invoice billingentity.Invoice) (*billingentity.UpsertInvoiceResult, error) {
	id, err := ulid.Parse(invoice.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse invoice ID: %w", err)
	}

	idTime := ulid.Time(id.Time())

	out := billingentity.NewUpsertInvoiceResult()
	out.SetInvoiceNumber(fmt.Sprintf("SANDBOX-%s", idTime.Format(InvoiceTSFormat)))

	return billingentity.NewUpsertInvoiceResult(), nil
}

func (a App) FinalizeInvoice(ctx context.Context, invoice billingentity.Invoice) (*billingentity.FinalizeInvoiceResult, error) {
	return nil, nil
}

func (a App) DeleteInvoice(ctx context.Context, invoice billingentity.Invoice) error {
	return nil
}

type CustomerData struct{}

func (c CustomerData) Validate() error {
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
func (a *Factory) NewApp(_ context.Context, appBase appentitybase.AppBase) (appentity.App, error) {
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

	return a.NewApp(ctx, appBase)
}

func (a *Factory) UninstallApp(ctx context.Context, input appentity.UninstallAppInput) error {
	return nil
}
