package httpdriver

import (
	"fmt"
	"log/slog"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
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
func (a *AppMapper) MapAppToAPI(item appentity.App) (api.App, error) {
	switch item.GetType() {
	case appentitybase.AppTypeStripe:
		stripeApp := item.(appstripeentityapp.App)

		stripeAPIApp := a.mapStripeAppToAPI(stripeApp)

		app := api.App{}
		if err := app.FromStripeApp(stripeAPIApp); err != nil {
			return app, err
		}

		return app, nil
	case appentitybase.AppTypeSandbox:
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
		Listing:   mapMarketplaceListing(app.GetListing()),
		CreatedAt: app.CreatedAt,
		UpdatedAt: app.UpdatedAt,
		DeletedAt: app.DeletedAt,
	}
}

func (a *AppMapper) mapStripeAppToAPI(
	stripeApp appstripeentityapp.App,
) api.StripeApp {
	// Get masked API key
	maskedAPIKey, err := a.stripeAppService.GetMaskedSecretAPIKey(stripeApp.APIKey)
	if err != nil {
		a.logger.Error("failed to get stripe app masked api key", "id", stripeApp.GetID())

		// Fallback to empty string
		maskedAPIKey = ""
	}

	apiStripeApp := api.StripeApp{
		Id:              stripeApp.GetID().ID,
		Type:            api.StripeAppType(stripeApp.GetType()),
		Name:            stripeApp.Name,
		Status:          api.AppStatus(stripeApp.GetStatus()),
		Listing:         mapMarketplaceListing(stripeApp.GetListing()),
		MaskedAPIKey:    maskedAPIKey,
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
