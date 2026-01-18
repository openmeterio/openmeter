package apps

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	appstripeentityapp "github.com/openmeterio/openmeter/openmeter/app/stripe/entity/app"
)

// MapAppToAPI maps an app to an v3 API app
func MapAppToAPI(item app.App) (api.BillingApp, error) {
	if item == nil {
		return api.BillingApp{}, errors.New("invalid app: nil")
	}

	switch item.GetType() {
	case app.AppTypeStripe:
		stripeApp := item.(appstripeentityapp.App)

		app := api.BillingApp{}
		if err := app.FromBillingAppStripe(mapStripeAppToAPI(stripeApp.Meta)); err != nil {
			return app, err
		}

		return app, nil
	case app.AppTypeSandbox:
		sandboxApp := item.(appsandbox.App)

		app := api.BillingApp{}
		if err := app.FromBillingAppSandbox(mapSandboxAppToAPI(sandboxApp.Meta)); err != nil {
			return app, err
		}

		return app, nil
	case app.AppTypeCustomInvoicing:
		customInvoicingApp := item.(appcustominvoicing.App)

		app := api.BillingApp{}
		if err := app.FromBillingAppExternalInvoicing(mapCustomInvoicingAppToAPI(customInvoicingApp.Meta)); err != nil {
			return app, err
		}

		return app, nil
	default:
		return api.BillingApp{}, fmt.Errorf("unsupported app type: %s", item.GetType())
	}
}

func mapSandboxAppToAPI(app appsandbox.Meta) api.BillingAppSandbox {
	return api.BillingAppSandbox{
		Id:         app.GetID().ID,
		Type:       api.BillingAppSandboxTypeSandbox,
		Name:       app.GetName(),
		Status:     api.BillingAppStatus(app.GetStatus()),
		Definition: ConvertMarketplaceListingToV3Api(app.GetListing()),
		CreatedAt:  lo.ToPtr(app.CreatedAt),
		UpdatedAt:  lo.ToPtr(app.UpdatedAt),
		DeletedAt:  app.DeletedAt,
	}
}

func mapStripeAppToAPI(
	stripeApp appstripeentityapp.Meta,
) api.BillingAppStripe {
	apiStripeApp := api.BillingAppStripe{
		Id:           stripeApp.GetID().ID,
		Type:         api.BillingAppStripeType(stripeApp.GetType()),
		Name:         stripeApp.Name,
		Status:       api.BillingAppStatus(stripeApp.GetStatus()),
		Definition:   ConvertMarketplaceListingToV3Api(stripeApp.GetListing()),
		MaskedApiKey: stripeApp.MaskedAPIKey,
		CreatedAt:    lo.ToPtr(stripeApp.CreatedAt),
		UpdatedAt:    lo.ToPtr(stripeApp.UpdatedAt),
		DeletedAt:    stripeApp.DeletedAt,
		AccountId:    stripeApp.StripeAccountID,
		Livemode:     stripeApp.Livemode,
	}

	apiStripeApp.Description = stripeApp.GetDescription()

	if stripeApp.GetMetadata() != nil {
		apiStripeApp.Labels = ConvertMetadataToLabels(stripeApp.GetMetadata())
	}

	return apiStripeApp
}

func mapCustomInvoicingAppToAPI(app appcustominvoicing.Meta) api.BillingAppExternalInvoicing {
	return api.BillingAppExternalInvoicing{
		Id:          app.GetID().ID,
		Type:        api.BillingAppExternalInvoicingTypeExternalInvoicing,
		Name:        app.GetName(),
		Status:      api.BillingAppStatus(app.GetStatus()),
		Definition:  ConvertMarketplaceListingToV3Api(app.GetListing()),
		Labels:      ConvertMetadataToLabels(app.GetMetadata()),
		Description: app.GetDescription(),
		CreatedAt:   lo.ToPtr(app.CreatedAt),
		UpdatedAt:   lo.ToPtr(app.UpdatedAt),
		DeletedAt:   app.DeletedAt,

		EnableDraftSyncHook:   app.Configuration.EnableDraftSyncHook,
		EnableIssuingSyncHook: app.Configuration.EnableIssuingSyncHook,
	}
}
