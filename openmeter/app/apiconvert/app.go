package apiconvert

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	apimodels "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
)

// ToAPIApp maps an app to an API app
func ToAPIApp(item app.App) (apimodels.App, error) {
	if item == nil {
		return apimodels.App{}, errors.New("invalid app: nil")
	}

	switch item.GetType() {
	case app.AppTypeStripe:
		stripeApp := item.(appstripe.App)

		app := apimodels.App{}
		if err := app.FromStripeApp(ToAPIStripeApp(stripeApp.Meta)); err != nil {
			return app, err
		}

		return app, nil
	case app.AppTypeSandbox:
		sandboxApp := item.(appsandbox.App)

		app := apimodels.App{}
		if err := app.FromSandboxApp(ToAPISandboxApp(sandboxApp.Meta)); err != nil {
			return app, err
		}

		return app, nil
	case app.AppTypeCustomInvoicing:
		customInvoicingApp := item.(appcustominvoicing.App)

		app := apimodels.App{}
		if err := app.FromCustomInvoicingApp(ToAPICustomInvoicingApp(customInvoicingApp.Meta)); err != nil {
			return app, err
		}

		return app, nil
	default:
		return apimodels.App{}, fmt.Errorf("unsupported app type: %s", item.GetType())
	}
}

func ToAPIEventApp(event app.EventApp) (apimodels.App, error) {
	switch event.GetType() {
	case app.AppTypeStripe:
		target := appstripe.App{}
		if err := target.FromEventAppData(event); err != nil {
			return apimodels.App{}, err
		}

		app := apimodels.App{}
		if err := app.FromStripeApp(ToAPIStripeApp(target.Meta)); err != nil {
			return apimodels.App{}, err
		}

		return app, nil
	case app.AppTypeSandbox:
		target := appsandbox.Meta{}
		if err := target.FromEventAppData(event); err != nil {
			return apimodels.App{}, err
		}

		app := apimodels.App{}
		if err := app.FromSandboxApp(ToAPISandboxApp(target)); err != nil {
			return apimodels.App{}, err
		}

		return app, nil
	case app.AppTypeCustomInvoicing:
		target := appcustominvoicing.App{}
		if err := target.FromEventAppData(event); err != nil {
			return apimodels.App{}, err
		}

		app := apimodels.App{}
		if err := app.FromCustomInvoicingApp(ToAPICustomInvoicingApp(target.Meta)); err != nil {
			return apimodels.App{}, err
		}

		return app, nil
	default:
		return apimodels.App{}, fmt.Errorf("unsupported app type: %s", event.GetType())
	}
}

func ToAPISandboxApp(app appsandbox.Meta) apimodels.SandboxApp {
	return apimodels.SandboxApp{
		Id:        app.GetID().ID,
		Type:      apimodels.SandboxAppTypeSandbox,
		Name:      app.GetName(),
		Status:    apimodels.AppStatus(app.GetStatus()),
		Listing:   ToAPIMarketplaceListing(app.GetListing()),
		CreatedAt: app.CreatedAt,
		UpdatedAt: app.UpdatedAt,
		DeletedAt: app.DeletedAt,
	}
}

func ToAPIStripeApp(
	stripeApp appstripe.Meta,
) apimodels.StripeApp {
	apiStripeApp := apimodels.StripeApp{
		Id:              stripeApp.GetID().ID,
		Type:            apimodels.StripeAppType(stripeApp.GetType()),
		Name:            stripeApp.Name,
		Status:          apimodels.AppStatus(stripeApp.GetStatus()),
		Listing:         ToAPIMarketplaceListing(stripeApp.GetListing()),
		MaskedAPIKey:    stripeApp.MaskedAPIKey,
		CreatedAt:       stripeApp.CreatedAt,
		UpdatedAt:       stripeApp.UpdatedAt,
		DeletedAt:       stripeApp.DeletedAt,
		StripeAccountId: stripeApp.StripeAccountID,
		Livemode:        stripeApp.Livemode,
	}

	apiStripeApp.Description = stripeApp.GetDescription()

	if stripeApp.GetMetadata() != nil {
		apiStripeApp.Metadata = lo.ToPtr(apimodels.Metadata(stripeApp.GetMetadata()))
	}

	return apiStripeApp
}

func ToAPICustomInvoicingApp(app appcustominvoicing.Meta) apimodels.CustomInvoicingApp {
	return apimodels.CustomInvoicingApp{
		Id:          app.GetID().ID,
		Type:        apimodels.CustomInvoicingAppTypeCustomInvoicing,
		Name:        app.GetName(),
		Status:      apimodels.AppStatus(app.GetStatus()),
		Listing:     ToAPIMarketplaceListing(app.GetListing()),
		Metadata:    lo.EmptyableToPtr(apimodels.Metadata(app.GetMetadata())),
		Description: app.GetDescription(),
		CreatedAt:   app.CreatedAt,
		UpdatedAt:   app.UpdatedAt,
		DeletedAt:   app.DeletedAt,

		EnableDraftSyncHook:   app.Configuration.EnableDraftSyncHook,
		EnableIssuingSyncHook: app.Configuration.EnableIssuingSyncHook,
	}
}

// Map marketplace listing to API
func ToAPIMarketplaceListing(listing app.MarketplaceListing) apimodels.MarketplaceListing {
	return apimodels.MarketplaceListing{
		Type:        apimodels.AppType(listing.Type),
		Name:        listing.Name,
		Description: listing.Description,
		Capabilities: lo.Map(listing.Capabilities, func(v app.Capability, _ int) apimodels.AppCapability {
			return apimodels.AppCapability{
				Type:        apimodels.AppCapabilityType(v.Type),
				Key:         v.Key,
				Name:        v.Name,
				Description: v.Description,
			}
		}),
		InstallMethods: lo.Map(listing.InstallMethods, func(v app.InstallMethod, _ int) apimodels.InstallMethod {
			return apimodels.InstallMethod(v)
		}),
	}
}
