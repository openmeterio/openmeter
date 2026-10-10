package appstripe

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestDeletedAppPreservesIdentityAndRejectsOperations(t *testing.T) {
	appBase := app.AppBase{
		ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
			ID:        "app-id",
			Namespace: "namespace",
			Name:      "Stripe",
		}),
		Type: app.AppTypeStripe,
	}

	deleted := NewDeleted(Meta{AppBase: appBase})
	require.Equal(t, appBase.GetID(), deleted.GetID())
	require.Equal(t, app.AppTypeStripe, deleted.GetType())
	require.ErrorIs(t, deleted.ValidateCapabilities(app.CapabilityTypeInvoiceCustomers), app.ErrAppDeleted)
	require.ErrorIs(t, deleted.ExecuteAction(t.Context(), app.ExecuteAppActionInput{
		AppID: appBase.GetID(),
		Type:  AppActionTypeReconcileWebhookEvents,
	}), app.ErrAppDeleted)
	require.ErrorIs(t, deleted.DeleteStandardInvoice(t.Context(), billing.StandardInvoice{}), billing.WarnInvoiceWorkflowAppDeleteSkipped)
}

func TestMetaActionsReportsOutdatedWebhookSchema(t *testing.T) {
	t.Run("outdated version requests reconcile", func(t *testing.T) {
		actions := Meta{AppData: AppData{WebhookSchemaVersion: LatestWebhookSchemaVersion - 1}}.Actions()

		require.Len(t, actions, 1)
		require.Equal(t, AppActionTypeReconcileWebhookEvents, actions[0].Type)
		require.NotEmpty(t, actions[0].Description)
	})

	t.Run("latest version has no actions", func(t *testing.T) {
		require.Nil(t, Meta{AppData: AppData{WebhookSchemaVersion: LatestWebhookSchemaVersion}}.Actions())
	})

	t.Run("deleted app has no actions", func(t *testing.T) {
		deleted := Meta{AppData: AppData{WebhookSchemaVersion: LatestWebhookSchemaVersion - 1}}
		deleted.DeletedAt = lo.ToPtr(time.Now())

		require.Nil(t, deleted.Actions())
	})
}
