package httpdriver

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	appstripeentityapp "github.com/openmeterio/openmeter/openmeter/app/stripe/entity/app"
)

// MapAppToAPI maps an app to an API app
func MapAppToAPI(item app.App) (api.App, error) {
	switch item.GetType() {
	case app.AppTypeStripe:
		stripeApp := item.(appstripeentityapp.App)

		app := api.App{}
		if err := app.FromStripeApp(mapStripeAppToAPI(stripeApp)); err != nil {
			return app, err
		}

		return app, nil
	case app.AppTypeSandbox:
		sandboxApp := item.(appsandbox.App)

		app := api.App{}
		if err := app.FromSandboxApp(mapSandboxAppToAPI(sandboxApp)); err != nil {
			return app, err
		}

		return app, nil
	case app.AppTypeCustomInvoicing:
		customInvoicingApp := item.(appcustominvoicing.App)

		app := api.App{}
		if err := app.FromCustomInvoicingApp(mapCustomInvoicingAppToAPI(customInvoicingApp)); err != nil {
			return app, err
		}

		return app, nil
	default:
		return api.App{}, fmt.Errorf("unsupported app type: %s", item.GetType())
	}
}

func mapSandboxAppToAPI(app appsandbox.App) api.SandboxApp {
	return api.SandboxApp{
		Id:        app.GetID().ID,
		Type:      api.SandboxAppTypeSandbox,
		Name:      app.GetName(),
		Status:    api.AppStatus(app.GetStatus()),
		Default:   app.Default,
		Listing:   mapMarketplaceListing(app.GetListing()),
		CreatedAt: app.CreatedAt,
		UpdatedAt: app.UpdatedAt,
		DeletedAt: app.DeletedAt,
	}
}

func mapStripeAppToAPI(
	stripeApp appstripeentityapp.App,
) api.StripeApp {
	apiStripeApp := api.StripeApp{
		Id:              stripeApp.GetID().ID,
		Type:            api.StripeAppType(stripeApp.GetType()),
		Name:            stripeApp.Name,
		Status:          api.AppStatus(stripeApp.GetStatus()),
		Default:         stripeApp.Default,
		Listing:         mapMarketplaceListing(stripeApp.GetListing()),
		MaskedAPIKey:    stripeApp.MaskedAPIKey,
		CreatedAt:       stripeApp.CreatedAt,
		UpdatedAt:       stripeApp.UpdatedAt,
		DeletedAt:       stripeApp.DeletedAt,
		StripeAccountId: stripeApp.StripeAccountID,
		Livemode:        stripeApp.Livemode,
	}

	apiStripeApp.Description = stripeApp.GetDescription()

	if stripeApp.GetMetadata() != nil {
		apiStripeApp.Metadata = lo.ToPtr(stripeApp.GetMetadata())
	}

	return apiStripeApp
}

func mapCustomInvoicingAppToAPI(app appcustominvoicing.App) api.CustomInvoicingApp {
	return api.CustomInvoicingApp{
		Id:          app.GetID().ID,
		Type:        api.CustomInvoicingAppTypeCustomInvoicing,
		Name:        app.GetName(),
		Status:      api.AppStatus(app.GetStatus()),
		Default:     app.Default,
		Listing:     mapMarketplaceListing(app.GetListing()),
		Metadata:    lo.EmptyableToPtr(app.GetMetadata()),
		Description: app.GetDescription(),
		CreatedAt:   app.CreatedAt,
		UpdatedAt:   app.UpdatedAt,
		DeletedAt:   app.DeletedAt,

		EnableDraftSyncHook:   app.Configuration.EnableDraftSyncHook,
		EnableIssuingSyncHook: app.Configuration.EnableIssuingSyncHook,
	}
}
