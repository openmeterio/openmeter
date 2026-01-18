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
		stripeApp, ok := item.(appstripeentityapp.App)
		if !ok {
			return api.BillingApp{}, fmt.Errorf("expected stripe app, got %T", item)
		}

		billingAppStripe, err := mapStripeAppToAPI(stripeApp.Meta)
		if err != nil {
			return api.BillingApp{}, fmt.Errorf("failed to map stripe app to API: %w", err)
		}

		billingApp := api.BillingApp{}
		if err := billingApp.FromBillingAppStripe(billingAppStripe); err != nil {
			return billingApp, err
		}

		return billingApp, nil
	case app.AppTypeSandbox:
		sandboxApp, ok := item.(appsandbox.App)
		if !ok {
			return api.BillingApp{}, fmt.Errorf("expected sandbox app, got %T", item)
		}

		billingAppSandbox, err := mapSandboxAppToAPI(sandboxApp.Meta)
		if err != nil {
			return api.BillingApp{}, fmt.Errorf("failed to map sandbox app to API: %w", err)
		}

		billingApp := api.BillingApp{}
		if err := billingApp.FromBillingAppSandbox(billingAppSandbox); err != nil {
			return billingApp, err
		}

		return billingApp, nil
	case app.AppTypeCustomInvoicing:
		customInvoicingApp, ok := item.(appcustominvoicing.App)
		if !ok {
			return api.BillingApp{}, fmt.Errorf("expected custom invoicing app, got %T", item)
		}

		billingAppExternalInvoicing, err := mapCustomInvoicingAppToAPI(customInvoicingApp.Meta)
		if err != nil {
			return api.BillingApp{}, fmt.Errorf("failed to map custom invoicing app to API: %w", err)
		}

		billingApp := api.BillingApp{}
		if err := billingApp.FromBillingAppExternalInvoicing(billingAppExternalInvoicing); err != nil {
			return billingApp, err
		}

		return billingApp, nil
	default:
		return api.BillingApp{}, fmt.Errorf("unsupported app type: %s", item.GetType())
	}
}

func mapSandboxAppToAPI(sandboxApp appsandbox.Meta) (api.BillingAppSandbox, error) {
	definition, err := ConvertMarketplaceListingToV3Api(sandboxApp.GetListing())
	if err != nil {
		return api.BillingAppSandbox{}, err
	}

	return api.BillingAppSandbox{
		Id:         sandboxApp.GetID().ID,
		Type:       api.BillingAppSandboxTypeSandbox,
		Name:       sandboxApp.GetName(),
		Status:     api.BillingAppStatus(sandboxApp.GetStatus()),
		Definition: definition,
		CreatedAt:  lo.ToPtr(sandboxApp.CreatedAt),
		UpdatedAt:  lo.ToPtr(sandboxApp.UpdatedAt),
		DeletedAt:  sandboxApp.DeletedAt,
	}, nil
}

func mapStripeAppToAPI(
	stripeApp appstripeentityapp.Meta,
) (api.BillingAppStripe, error) {
	definition, err := ConvertMarketplaceListingToV3Api(stripeApp.GetListing())
	if err != nil {
		return api.BillingAppStripe{}, err
	}

	apiStripeApp := api.BillingAppStripe{
		Id:           stripeApp.GetID().ID,
		Type:         api.BillingAppStripeType(stripeApp.GetType()),
		Name:         stripeApp.Name,
		Status:       api.BillingAppStatus(stripeApp.GetStatus()),
		Definition:   definition,
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

	return apiStripeApp, nil
}

func mapCustomInvoicingAppToAPI(customInvoicingApp appcustominvoicing.Meta) (api.BillingAppExternalInvoicing, error) {
	definition, err := ConvertMarketplaceListingToV3Api(customInvoicingApp.GetListing())
	if err != nil {
		return api.BillingAppExternalInvoicing{}, err
	}

	return api.BillingAppExternalInvoicing{
		Id:          customInvoicingApp.GetID().ID,
		Type:        api.BillingAppExternalInvoicingTypeExternalInvoicing,
		Name:        customInvoicingApp.GetName(),
		Status:      api.BillingAppStatus(customInvoicingApp.GetStatus()),
		Definition:  definition,
		Labels:      ConvertMetadataToLabels(customInvoicingApp.GetMetadata()),
		Description: customInvoicingApp.GetDescription(),
		CreatedAt:   lo.ToPtr(customInvoicingApp.CreatedAt),
		UpdatedAt:   lo.ToPtr(customInvoicingApp.UpdatedAt),
		DeletedAt:   customInvoicingApp.DeletedAt,

		EnableDraftSyncHook:   customInvoicingApp.Configuration.EnableDraftSyncHook,
		EnableIssuingSyncHook: customInvoicingApp.Configuration.EnableIssuingSyncHook,
	}, nil
}
