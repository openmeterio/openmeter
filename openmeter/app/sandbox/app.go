package appsandbox

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/sequence"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerapp "github.com/openmeterio/openmeter/openmeter/customer/app"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
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
	_ app.CustomerData                    = (*CustomerData)(nil)

	InvoiceSequenceNumber = sequence.Definition{
		Prefix:         "OM-SANDBOX",
		SuffixTemplate: "{{.CustomerPrefix}}-{{.NextSequenceNumber}}",
		Scope:          "invoices/app/sandbox",
	}
)

type Meta struct {
	app.AppBase
}

var _ app.EventAppParser = (*Meta)(nil)

func (m *Meta) FromEventAppData(event app.EventApp) error {
	m.AppBase = event.AppBase

	return nil
}

type App struct {
	Meta

	sequenceService sequence.Service
}

func (a App) ValidateCustomer(ctx context.Context, customer *customer.Customer, capabilities []app.CapabilityType) error {
	if err := a.ValidateCapabilities(capabilities...); err != nil {
		return fmt.Errorf("error validating capabilities: %w", err)
	}

	return nil
}

func (a App) GetCustomerData(ctx context.Context, input app.GetAppInstanceCustomerDataInput) (app.CustomerData, error) {
	return CustomerData{}, nil
}

func (a App) UpsertCustomerData(ctx context.Context, input app.UpsertAppInstanceCustomerDataInput) error {
	return nil
}

func (a App) DeleteCustomerData(ctx context.Context, input app.DeleteAppInstanceCustomerDataInput) error {
	return nil
}

func (a App) ValidateStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) error {
	return nil
}

func (a App) UpdateAppConfig(ctx context.Context, input app.AppConfigUpdate) error {
	return nil
}

func (a App) UpsertStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) (*billing.UpsertStandardInvoiceResult, error) {
	return billing.NewUpsertStandardInvoiceResult(), nil
}

func (a App) FinalizeStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) (*billing.FinalizeStandardInvoiceResult, error) {
	invoiceNumber, err := a.sequenceService.GenerateInvoiceSequenceNumber(
		ctx,
		sequence.GenerationInput{
			Namespace:    invoice.Namespace,
			CustomerName: invoice.Customer.Name,
			Currency:     invoice.Currency,
		},
		InvoiceSequenceNumber,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate invoice sequence number: %w", err)
	}

	return billing.NewFinalizeStandardInvoiceResult().
		SetInvoiceNumber(invoiceNumber).
		SetSentToCustomerAt(clock.Now()), nil
}

func (a App) DeleteStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) error {
	return nil
}

func (a App) PostAdvanceStandardInvoiceHook(ctx context.Context, invoice billing.StandardInvoice) (*billing.PostAdvanceHookResult, error) {
	if invoice.Status != billing.StandardInvoiceStatusPaymentProcessingPending {
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
			Invoice: invoice.GetInvoiceID(),
			Trigger: billing.TriggerFailed,
			ValidationErrors: &billing.InvoiceTriggerValidationInput{
				Operation: billing.StandardInvoiceOpInitiatePayment,
				Errors:    []error{ErrSimulatedPaymentFailure},
			},
		}), nil
	case TargetPaymentStatusUncollectible:
		return out.InvokeTrigger(billing.InvoiceTriggerInput{
			Invoice: invoice.GetInvoiceID(),
			Trigger: billing.TriggerPaymentUncollectible,
		}), nil
	case TargetPaymentStatusActionRequired:
		return out.InvokeTrigger(billing.InvoiceTriggerInput{
			Invoice: invoice.GetInvoiceID(),
			Trigger: billing.TriggerActionRequired,
		}), nil
	case TargetPaymentStatusPaid:
		fallthrough
	default:
		return out.InvokeTrigger(billing.InvoiceTriggerInput{
			Invoice: invoice.GetInvoiceID(),
			Trigger: billing.TriggerPaid,
		}), nil
	}
}

func (a App) GetEventAppData() (app.EventAppData, error) {
	return app.EventAppData{}, nil
}

type CustomerData struct{}

func (c CustomerData) Validate() error {
	return nil
}

type Factory struct {
	appService      app.Service
	sequenceService sequence.Service
}

type Config struct {
	AppService      app.Service
	SequenceService sequence.Service
}

func (c Config) Validate() error {
	if c.AppService == nil {
		return fmt.Errorf("app service is required")
	}

	if c.SequenceService == nil {
		return fmt.Errorf("sequence service is required")
	}

	return nil
}

func NewFactory(config Config) (*Factory, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	fact := &Factory{
		appService:      config.AppService,
		sequenceService: config.SequenceService,
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
func (a *Factory) NewApp(_ context.Context, appBase app.AppBase) (app.App, error) {
	return App{
		Meta: Meta{
			AppBase: appBase,
		},
		sequenceService: a.sequenceService,
	}, nil
}

func (a *Factory) InstallApp(ctx context.Context, input app.AppFactoryInstallAppInput) (app.App, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Sandbox is a singleton per namespace — only one instance makes sense since all
	// instances are functionally identical (no credentials, no external state).
	existing, err := a.appService.ListApps(ctx, app.ListAppInput{
		Namespace: input.Namespace,
		Type:      lo.ToPtr(app.AppTypeSandbox),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list sandbox apps: %w", err)
	}

	if existing.TotalCount > 0 {
		return nil, models.NewGenericConflictError(fmt.Errorf("sandbox app: %s already exists", existing.Items[0].GetName()))
	}

	appBase, err := a.appService.CreateApp(ctx, app.CreateAppInput{
		Namespace: input.Namespace,
		Name:      input.Name,
		Type:      app.AppTypeSandbox,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create app: %w", err)
	}

	return a.NewApp(ctx, appBase.GetAppBase())
}

func (a *Factory) UninstallApp(ctx context.Context, input app.UninstallAppInput) error {
	return nil
}
