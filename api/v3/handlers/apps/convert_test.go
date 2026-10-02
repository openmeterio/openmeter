package apps

import (
	"testing"

	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/app"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestToAPIBillingAppActions(t *testing.T) {
	appBase := func(appType app.AppType) app.AppBase {
		return app.AppBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				ID:        "app-id",
				Namespace: "namespace",
				Name:      "app",
			}),
			Type:    appType,
			Listing: app.MarketplaceListing{Type: appType},
		}
	}

	t.Run("stripe app with outdated webhook schema exposes the reconcile action", func(t *testing.T) {
		apiApp, err := toAPIBillingAppStripe(appstripe.Meta{
			AppBase: appBase(app.AppTypeStripe),
			AppData: appstripe.AppData{WebhookSchemaVersion: appstripe.LatestWebhookSchemaVersion - 1},
		})
		require.NoError(t, err)

		require.NotNil(t, apiApp.Actions)
		require.Len(t, *apiApp.Actions, 1)
		require.Equal(t, api.BillingAppActionTypeReconcileWebhookEvents, (*apiApp.Actions)[0].Type)
	})

	t.Run("apps without actions omit the field", func(t *testing.T) {
		stripeApp, err := toAPIBillingAppStripe(appstripe.Meta{
			AppBase: appBase(app.AppTypeStripe),
			AppData: appstripe.AppData{WebhookSchemaVersion: appstripe.LatestWebhookSchemaVersion},
		})
		require.NoError(t, err)
		require.Nil(t, stripeApp.Actions)

		sandboxApp, err := toAPIBillingAppSandbox(appsandbox.Meta{AppBase: appBase(app.AppTypeSandbox)})
		require.NoError(t, err)
		require.Nil(t, sandboxApp.Actions)
	})
}
