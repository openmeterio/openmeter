package appstripe

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
)

func (a appOperations) ExecuteAction(ctx context.Context, input app.ExecuteAppActionInput) error {
	switch input.Type {
	case AppActionTypeReconcileWebhookEvents:
		return a.reconcileWebhookEvents(ctx)
	default:
		return app.NewAppActionUnsupportedError(a.GetID(), input.Type)
	}
}

// reconcileWebhookEvents re-registers the Stripe endpoint with the current event set and
// records LatestWebhookSchemaVersion so the action stops being reported.
func (a appOperations) reconcileWebhookEvents(ctx context.Context) error {
	stripeAppData, stripeClient, err := a.getStripeClient(ctx, "reconcileWebhookEvents")
	if err != nil {
		return err
	}

	err = stripeClient.UpdateWebhook(ctx, stripeclient.UpdateWebhookInput{
		AppID:           a.GetID(),
		StripeWebhookID: stripeAppData.StripeWebhookID,
		EnabledEvents:   stripeclient.WebhookEnabledEvents,
	})
	if err != nil {
		return fmt.Errorf("failed to update stripe webhook: %w", err)
	}

	return a.StripeAppService.UpdateWebhookSchemaVersion(ctx, UpdateWebhookSchemaVersionInput{
		AppID:                a.GetID(),
		WebhookSchemaVersion: LatestWebhookSchemaVersion,
	})
}
