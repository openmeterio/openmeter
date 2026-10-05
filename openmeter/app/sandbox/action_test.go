package appsandbox

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestExecuteActionIsUnsupported(t *testing.T) {
	appBase := app.AppBase{
		ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
			ID:        "app-id",
			Namespace: "namespace",
			Name:      "Sandbox",
		}),
		Type: app.AppTypeSandbox,
	}

	err := appOperations{appBase: appBase}.ExecuteAction(t.Context(), app.ExecuteAppActionInput{
		AppID: appBase.GetID(),
		Type:  "reconcile_webhook_events",
	})
	require.ErrorAs(t, err, new(*app.AppActionUnsupportedError))
}
