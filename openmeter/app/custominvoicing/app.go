package appcustominvoicing

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerapp "github.com/openmeterio/openmeter/openmeter/customer/app"
)

var (
	_ customerapp.App                 = (*App)(nil)
	_ billing.InvoicingApp            = (*App)(nil)
	_ billing.InvoicingAppAsyncSyncer = (*App)(nil)
)

var DefaultInvoiceSequenceNumber = billing.SequenceDefinition{
	Prefix:         "INV",
	SuffixTemplate: "{{.CustomerPrefix}}-{{.NextSequenceNumber}}",
	Scope:          "invoices/custom-invoicing",
}

type Configuration struct {
	EnableDraftSyncHook   bool `json:"enable_draft_sync_hook"`
	EnableIssuingSyncHook bool `json:"enable_issuing_sync_hook"`
}

const (
	MetadataKeyDraftSyncedAt = "openmeter.io/custominvoicing/draft-synced-at"
	MetadataKeyFinalizedAt   = "openmeter.io/custominvoicing/finalized-at"
)

func (c Configuration) Validate() error {
	return nil
}

type Meta struct {
	app.AppBase
	Configuration
}

var _ app.EventAppParser = (*Meta)(nil)

func (m *Meta) FromEventAppData(event app.EventApp) error {
	m.AppBase = event.AppBase

	if err := event.AppData.ParseInto(&m.Configuration); err != nil {
		return fmt.Errorf("error parsing app data: %w", err)
	}

	return nil
}

type App struct {
	Meta

	customInvoicingService Service
	billingService         billing.Service
}

func (a App) ValidateCustomer(ctx context.Context, customer *customer.Customer, capabilities []app.CapabilityType) error {
	return nil
}

func (a App) UpdateAppConfig(ctx context.Context, input app.AppConfigUpdate) error {
	cfg, ok := input.(Configuration)
	if !ok {
		return fmt.Errorf("invalid configuration")
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	return a.customInvoicingService.UpsertAppConfiguration(ctx, UpsertAppConfigurationInput{
		AppID:         a.GetID(),
		Configuration: cfg,
	})
}

func (a App) GetEventAppData() (app.EventAppData, error) {
	return app.NewEventAppData(a.Configuration)
}

// InvoicingApp
// These are no-ops as whatever is meaningful, is handled via the http driver of the custominvoicing app.

// ValidateStandardInvoice is a no-op as any validation issues are published via the draft.syncing and finalizations syncing
// flow.
func (a App) ValidateStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) error {
	return nil
}

func (a App) UpsertStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) (*billing.UpsertStandardInvoiceResult, error) {
	return nil, nil
}

func (a App) FinalizeStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) (*billing.FinalizeStandardInvoiceResult, error) {
	canAdvance, err := a.CanIssuingSyncAdvance(invoice)
	if err != nil {
		return nil, err
	}

	res := billing.NewFinalizeStandardInvoiceResult()

	// If we are done with the hook work, let's make sure that the invoice has a non-draft invoice number
	if canAdvance {
		// If the invoice still has a draft invoice number, let's generate a non-draft one
		if billing.DraftInvoiceSequenceNumber.PrefixMatches(invoice.Number) {
			invoiceNumber, err := a.billingService.GenerateInvoiceSequenceNumber(ctx,
				billing.SequenceGenerationInput{
					Namespace:    invoice.Namespace,
					CustomerName: invoice.Customer.Name,
					Currency:     invoice.Currency,
				},
				DefaultInvoiceSequenceNumber,
			)
			if err != nil {
				return nil, fmt.Errorf("generating invoice number: %w", err)
			}

			res.SetInvoiceNumber(invoiceNumber)
		}
	}

	return res, nil
}

// DeleteStandardInvoice is a no-op as this should happen via the notifications webhook
func (a App) DeleteStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) error {
	return nil
}

// InvoicingAppAsyncSyncer

func (a App) CanDraftSyncAdvance(invoice billing.StandardInvoice) (bool, error) {
	if !a.Configuration.EnableDraftSyncHook {
		return true, nil
	}

	if invoice.Metadata == nil {
		return false, nil
	}

	if _, ok := invoice.Metadata[MetadataKeyDraftSyncedAt]; ok {
		return true, nil
	}

	return false, nil
}

func (a App) CanIssuingSyncAdvance(invoice billing.StandardInvoice) (bool, error) {
	if !a.Configuration.EnableIssuingSyncHook {
		return true, nil
	}

	if invoice.Metadata == nil {
		return false, nil
	}

	if _, ok := invoice.Metadata[MetadataKeyFinalizedAt]; ok {
		return true, nil
	}

	return false, nil
}
