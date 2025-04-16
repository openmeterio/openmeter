package appcustominvoicing

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerapp "github.com/openmeterio/openmeter/openmeter/customer/app"
)

var _ customerapp.App = (*App)(nil)

// TODO[later]: Implement invoicing app
// _ billing.InvoicingApp                = (*App)(nil)
// _ billing.InvoicingAppPostAdvanceHook = (*App)(nil)

type Configuration struct {
	SkipDraftSyncHook   bool `json:"skip_draft_sync_hook"`
	SkipIssuingSyncHook bool `json:"skip_issuing_sync_hook"`
}

func (c Configuration) Validate() error {
	return nil
}

type App struct {
	app.AppBase
	Configuration

	billingService         billing.Service
	customInvoicingService Service
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
