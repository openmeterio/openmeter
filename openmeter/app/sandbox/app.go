package appsandbox

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/openmeter/billing"
	customerapp "github.com/openmeterio/openmeter/openmeter/customer/app"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/clock"
)

const (
	TargetPaymentStatusMetadataKey = "openmeter.io/sandbox/target-payment-status"

	TargetPaymentStatusPaid           = "paid"
	TargetPaymentStatusFailed         = "failed"
	TargetPaymentStatusUncollectible  = "uncollectible"
	TargetPaymentStatusActionRequired = "action_required"
)

var (
	_ customerapp.App                     = (*App)(nil)
	_ billing.InvoicingApp                = (*App)(nil)
	_ billing.InvoicingAppPostAdvanceHook = (*App)(nil)
	_ appentity.CustomerData              = (*CustomerData)(nil)

	InvoiceSequenceNumber = billing.SequenceDefinition{
		Template: "OM-SANDBOX-{{.CustomerPrefix}}-{{.NextSequenceNumber}}",
		Scope:    "invoices/app/sandbox",
	}
)

type App struct {
	appentitybase.AppBase

	billingService billing.Service
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

func (a App) ValidateInvoice(ctx context.Context, invoice billing.Invoice) error {
	return nil
}

func (a App) UpsertInvoice(ctx context.Context, invoice billing.Invoice) (*billing.UpsertInvoiceResult, error) {
	return billing.NewUpsertInvoiceResult(), nil
}

func (a App) FinalizeInvoice(ctx context.Context, invoice billing.Invoice) (*billing.FinalizeInvoiceResult, error) {
	invoiceNumber, err := a.billingService.GenerateInvoiceSequenceNumber(
		ctx,
		billing.SequenceGenerationInput{
			Namespace:    invoice.Namespace,
			CustomerName: invoice.Customer.Name,
			Currency:     invoice.Currency,
		},
		InvoiceSequenceNumber,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate invoice sequence number: %w", err)
	}

	return billing.NewFinalizeInvoiceResult().
		SetInvoiceNumber(invoiceNumber).
		SetSentToCustomerAt(clock.Now()), nil
}

func (a App) DeleteInvoice(ctx context.Context, invoice billing.Invoice) error {
	return nil
}

func (a App) PostAdvanceInvoiceHook(ctx context.Context, invoice billing.Invoice) (*billing.PostAdvanceHookResult, error) {
	if invoice.Status != billing.InvoiceStatusPaymentProcessingPending {
		return nil, nil
	}

	targetStatus := TargetPaymentStatusPaid

	// Allow overriding via metadata for testing (unit, customer) purposes
	override, ok := invoice.Metadata[TargetPaymentStatusMetadataKey]
	if ok && override != "" {
		targetStatus = override
	}

	out := billing.NewPostAdvanceHookResult()
	// Let's simulate the payment status by invoking the right trigger
	switch targetStatus {
	case TargetPaymentStatusFailed:
		return out.InvokeTrigger(billing.InvoiceTriggerInput{
			Invoice: invoice.InvoiceID(),
			Trigger: billing.TriggerFailed,
			ValidationErrors: &billing.InvoiceTriggerValidationInput{
				Operation: billing.InvoiceOpInitiatePayment,
				Errors:    []error{ErrSimulatedPaymentFailure},
			},
		}), nil
	case TargetPaymentStatusUncollectible:
		return out.InvokeTrigger(billing.InvoiceTriggerInput{
			Invoice: invoice.InvoiceID(),
			Trigger: billing.TriggerPaymentUncollectible,
		}), nil
	case TargetPaymentStatusActionRequired:
		return out.InvokeTrigger(billing.InvoiceTriggerInput{
			Invoice: invoice.InvoiceID(),
			Trigger: billing.TriggerActionRequired,
		}), nil
	case TargetPaymentStatusPaid:
		fallthrough
	default:
		return out.InvokeTrigger(billing.InvoiceTriggerInput{
			Invoice: invoice.InvoiceID(),
			Trigger: billing.TriggerPaid,
		}), nil
	}
}

type CustomerData struct{}

func (c CustomerData) Validate() error {
	return nil
}

type Factory struct {
	appService     app.Service
	billingService billing.Service
}

type Config struct {
	AppService     app.Service
	BillingService billing.Service
}

func (c Config) Validate() error {
	if c.AppService == nil {
		return fmt.Errorf("app service is required")
	}

	if c.BillingService == nil {
		return fmt.Errorf("billing service is required")
	}

	return nil
}

func NewFactory(config Config) (*Factory, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	fact := &Factory{
		appService:     config.AppService,
		billingService: config.BillingService,
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
		AppBase:        appBase,
		billingService: a.billingService,
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
