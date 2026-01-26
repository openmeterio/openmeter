package httpdriver

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	appstripeentityapp "github.com/openmeterio/openmeter/openmeter/app/stripe/entity/app"
)

// MapAppToAPI maps an app to an API app
func MapAppToAPI(item app.App) (api.App, error) {
	if item == nil {
		return api.App{}, errors.New("invalid app: nil")
	}

	switch item.GetType() {
	case app.AppTypeStripe:
		stripeApp := item.(appstripeentityapp.App)

		app := api.App{}
		if err := app.FromStripeApp(mapStripeAppToAPI(stripeApp.Meta)); err != nil {
			return app, err
		}

		return app, nil
	case app.AppTypeSandbox:
		sandboxApp := item.(appsandbox.App)

		app := api.App{}
		if err := app.FromSandboxApp(mapSandboxAppToAPI(sandboxApp.Meta)); err != nil {
			return app, err
		}

		return app, nil
	case app.AppTypeCustomInvoicing:
		customInvoicingApp := item.(appcustominvoicing.App)

		app := api.App{}
		if err := app.FromCustomInvoicingApp(mapCustomInvoicingAppToAPI(customInvoicingApp.Meta)); err != nil {
			return app, err
		}

		return app, nil
	default:
		return api.App{}, fmt.Errorf("unsupported app type: %s", item.GetType())
	}
}

func mapSandboxAppToAPI(app appsandbox.Meta) api.SandboxApp {
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

func mapStripeAppToAPI(
	stripeApp appstripeentityapp.Meta,
) api.StripeApp {
	apiStripeApp := api.StripeApp{
		Id:              stripeApp.GetID().ID,
		Type:            api.StripeAppType(stripeApp.GetType()),
		Name:            stripeApp.Name,
		Status:          api.AppStatus(stripeApp.GetStatus()),
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
		apiStripeApp.Metadata = lo.ToPtr(api.Metadata(stripeApp.GetMetadata()))
	}

	return apiStripeApp
}

func mapCustomInvoicingAppToAPI(app appcustominvoicing.Meta) api.CustomInvoicingApp {
	return api.CustomInvoicingApp{
		Id:          app.GetID().ID,
		Type:        api.CustomInvoicingAppTypeCustomInvoicing,
		Name:        app.GetName(),
		Status:      api.AppStatus(app.GetStatus()),
		Listing:     mapMarketplaceListing(app.GetListing()),
		Metadata:    lo.EmptyableToPtr(api.Metadata(app.GetMetadata())),
		Description: app.GetDescription(),
		CreatedAt:   app.CreatedAt,
		UpdatedAt:   app.UpdatedAt,
		DeletedAt:   app.DeletedAt,

		EnableDraftSyncHook:   app.Configuration.EnableDraftSyncHook,
		EnableIssuingSyncHook: app.Configuration.EnableIssuingSyncHook,
	}
}

func MapEventAppToAPI(event app.EventApp) (api.App, error) {
	switch event.GetType() {
	case app.AppTypeStripe:
		target := appstripeentityapp.App{}
		if err := target.FromEventAppData(event); err != nil {
			return api.App{}, err
		}

		app := api.App{}
		if err := app.FromStripeApp(mapStripeAppToAPI(target.Meta)); err != nil {
			return api.App{}, err
		}

		return app, nil
	case app.AppTypeSandbox:
		target := appsandbox.Meta{}
		if err := target.FromEventAppData(event); err != nil {
			return api.App{}, err
		}

		app := api.App{}
		if err := app.FromSandboxApp(mapSandboxAppToAPI(target)); err != nil {
			return api.App{}, err
		}

		return app, nil
	case app.AppTypeCustomInvoicing:
		target := appcustominvoicing.App{}
		if err := target.FromEventAppData(event); err != nil {
			return api.App{}, err
		}

		app := api.App{}
		if err := app.FromCustomInvoicingApp(mapCustomInvoicingAppToAPI(target.Meta)); err != nil {
			return api.App{}, err
		}

		return app, nil
	default:
		return api.App{}, fmt.Errorf("unsupported app type: %s", event.GetType())
	}
}

// fromAPIAppStripeCustomerData maps an API stripe customer data to an app stripe customer data
func fromAPIAppStripeCustomerData(apiStripeCustomerData api.StripeCustomerAppData) appstripeentity.CustomerData {
	return appstripeentity.CustomerData{
		StripeCustomerID:             apiStripeCustomerData.StripeCustomerId,
		StripeDefaultPaymentMethodID: apiStripeCustomerData.StripeDefaultPaymentMethodId,
	}
}

// customerAppToAPI converts a CustomerApp to an API CustomerAppData
func ToAPIStripeCustomerAppData(
	customerAppData appstripeentity.CustomerData,
	stripeApp appstripeentityapp.App,
) api.StripeCustomerAppData {
	apiStripeCustomerAppData := api.StripeCustomerAppData{
		Id:                           lo.ToPtr(stripeApp.GetID().ID),
		Type:                         api.StripeCustomerAppDataTypeStripe,
		App:                          lo.ToPtr(mapStripeAppToAPI(stripeApp.Meta)),
		StripeCustomerId:             customerAppData.StripeCustomerID,
		StripeDefaultPaymentMethodId: customerAppData.StripeDefaultPaymentMethodID,
	}

	return apiStripeCustomerAppData
}
