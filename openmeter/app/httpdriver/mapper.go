package httpdriver

import (
	"fmt"
	"log/slog"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	appstripeentityapp "github.com/openmeterio/openmeter/openmeter/app/stripe/entity/app"
)

// NewAppMapper creates a new app mapper
func NewAppMapper(
	logger *slog.Logger,
	stripeAppService appstripe.Service,
) *AppMapper {
	return &AppMapper{
		logger:           logger,
		stripeAppService: stripeAppService,
	}
}

// AppMapper maps app models to API models
type AppMapper struct {
	logger           *slog.Logger
	stripeAppService appstripe.Service
}

// MapAppToAPI maps an app to an API app
func (a *AppMapper) MapAppToAPI(item app.App) (api.App, error) {
	switch item.GetType() {
	case app.AppTypeStripe:
		stripeApp := item.(appstripeentityapp.App)

		app := api.App{}
		if err := app.FromStripeApp(a.mapStripeAppToAPI(stripeApp)); err != nil {
			return app, err
		}

		return app, nil
	case app.AppTypeSandbox:
		sandboxApp := item.(appsandbox.App)

		app := api.App{}
		if err := app.FromSandboxApp(a.mapSandboxAppToAPI(sandboxApp)); err != nil {
			return app, err
		}

		return app, nil
	default:
		return api.App{}, fmt.Errorf("unsupported app type: %s", item.GetType())
	}
}

func (a *AppMapper) mapSandboxAppToAPI(app appsandbox.App) api.SandboxApp {
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

func (a *AppMapper) mapStripeAppToAPI(
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
