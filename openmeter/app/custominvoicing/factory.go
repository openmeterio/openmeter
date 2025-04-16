package appcustominvoicing

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
)

var (
	MarketplaceListing = app.MarketplaceListing{
		Type:        app.AppTypeCustomInvoicing,
		Name:        "Custom Invoicing",
		Description: "Custom Invoicing can be used to interface with third party invoicing and payment systems",
		Capabilities: []app.Capability{
			CollectPaymentCapability,
			CalculateTaxCapability,
			InvoiceCustomerCapability,
		},
		InstallMethods: []app.InstallMethod{
			app.InstallMethodNoCredentials,
		},
	}

	CollectPaymentCapability = app.Capability{
		Type:        app.CapabilityTypeCollectPayments,
		Key:         "custom_invoicing_collect_payment",
		Name:        "Payment",
		Description: "Process payments",
	}

	CalculateTaxCapability = app.Capability{
		Type:        app.CapabilityTypeCalculateTax,
		Key:         "custom_invoicing_calculate_tax",
		Name:        "Calculate Tax",
		Description: "Calculate tax for a payment",
	}

	InvoiceCustomerCapability = app.Capability{
		Type:        app.CapabilityTypeInvoiceCustomers,
		Key:         "custom_invoicing_invoice_customer",
		Name:        "Invoice Customer",
		Description: "Invoice a customer",
	}
)

type Factory struct {
	appService             app.Service
	billingService         billing.Service
	customInvoicingService Service
}

type FactoryConfig struct {
	AppService             app.Service
	CustomInvoicingService Service
}

func (c FactoryConfig) Validate() error {
	if c.AppService == nil {
		return fmt.Errorf("app service is required")
	}

	if c.CustomInvoicingService == nil {
		return fmt.Errorf("custom invoicing service is required")
	}

	return nil
}

func NewFactory(config FactoryConfig) (*Factory, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	fact := &Factory{
		appService:             config.AppService,
		customInvoicingService: config.CustomInvoicingService,
	}

	err := config.AppService.RegisterMarketplaceListing(app.RegistryItem{
		Listing: MarketplaceListing,
		Factory: fact,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register marketplace listing: %w", err)
	}

	return fact, nil
}

// Factory
func (f *Factory) NewApp(ctx context.Context, appBase app.AppBase) (app.App, error) {
	cfg, err := f.customInvoicingService.GetAppConfiguration(ctx, appBase.GetID())
	if err != nil {
		return nil, fmt.Errorf("failed to get app config: %w", err)
	}

	return App{
		AppBase:                appBase,
		Configuration:          cfg,
		billingService:         f.billingService,
		customInvoicingService: f.customInvoicingService,
	}, nil
}

func (f *Factory) InstallApp(ctx context.Context, input app.AppFactoryInstallAppInput) (app.App, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	appBase, err := f.customInvoicingService.CreateApp(ctx, CreateAppInput{
		Namespace: input.Namespace,
		Name:      input.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create app: %w", err)
	}

	return f.NewApp(ctx, appBase)
}

func (f *Factory) UninstallApp(ctx context.Context, input app.UninstallAppInput) error {
	return f.customInvoicingService.DeleteApp(ctx, input)
}

// Service types

type CreateAppInput struct {
	Namespace string
	Name      string

	Config Configuration
}

func (i CreateAppInput) Validate() error {
	if i.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	if i.Name == "" {
		return fmt.Errorf("name is required")
	}

	return nil
}

type UpsertAppConfigurationInput struct {
	AppID         app.AppID
	Configuration Configuration
}
