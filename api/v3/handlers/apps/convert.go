//go:generate go run github.com/jmattheis/goverter/cmd/goverter gen ./
package apps

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
)

// goverter:variables
// goverter:skipCopySameType
// goverter:output:file ./convert.gen.go
// goverter:useZeroValueOnPointerInconsistency
// goverter:useUnderlyingTypeMethods
// goverter:matchIgnoreCase
// goverter:extend IntToFloat32
// goverter:extend ToAPIBillingApp
// goverter:enum:unknown @error
var (
	ToAPIAppPagePaginatedResponse func(source response.PagePaginationResponse[api.BillingApp]) api.AppPagePaginatedResponse

	ToAPIBillingAppCatalogItem func(source app.MarketplaceListing) (api.BillingAppCatalogItem, error)

	// goverter:enum:map AppTypeStripe BillingAppTypeStripe
	// goverter:enum:map AppTypeSandbox BillingAppTypeSandbox
	// goverter:enum:map AppTypeCustomInvoicing BillingAppTypeExternalInvoicing
	ToAPIBillingAppTypeFromDomain func(source app.AppType) (api.BillingAppType, error)

	ToAPIBillingApps func(source []app.App) ([]api.BillingApp, error)
)

func IntToFloat32(i int) float32 {
	return float32(i)
}

// ToAPIBillingApp maps an app to a v3 API app
func ToAPIBillingApp(item app.App) (api.BillingApp, error) {
	if item == nil {
		return api.BillingApp{}, errors.New("invalid app: nil")
	}

	switch item.GetType() {
	case app.AppTypeStripe:
		stripeApp, ok := item.(appstripe.App)
		if !ok {
			return api.BillingApp{}, fmt.Errorf("expected stripe app, got %T", item)
		}

		billingAppStripe, err := toAPIBillingAppStripe(stripeApp.Meta)
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

		billingAppSandbox, err := toAPIBillingAppSandbox(sandboxApp.Meta)
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

		billingAppExternalInvoicing, err := toAPIBillingAppExternalInvoicing(customInvoicingApp.Meta)
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

func toAPIBillingAppSandbox(sandboxApp appsandbox.Meta) (api.BillingAppSandbox, error) {
	definition, err := ToAPIBillingAppCatalogItem(sandboxApp.GetListing())
	if err != nil {
		return api.BillingAppSandbox{}, err
	}

	return api.BillingAppSandbox{
		Id:          sandboxApp.GetID().ID,
		Type:        api.BillingAppSandboxTypeSandbox,
		Name:        sandboxApp.GetName(),
		Status:      api.BillingAppStatus(sandboxApp.GetStatus()),
		Definition:  definition,
		Labels:      labels.FromMetadata(sandboxApp.GetMetadata()),
		Description: sandboxApp.GetDescription(),
		CreatedAt:   lo.ToPtr(sandboxApp.CreatedAt),
		UpdatedAt:   lo.ToPtr(sandboxApp.UpdatedAt),
		DeletedAt:   sandboxApp.DeletedAt,
	}, nil
}

func toAPIBillingAppStripe(
	stripeApp appstripe.Meta,
) (api.BillingAppStripe, error) {
	definition, err := ToAPIBillingAppCatalogItem(stripeApp.GetListing())
	if err != nil {
		return api.BillingAppStripe{}, err
	}

	apiStripeApp := api.BillingAppStripe{
		Id:          stripeApp.GetID().ID,
		Type:        api.BillingAppStripeType(stripeApp.GetType()),
		Name:        stripeApp.Name,
		Status:      api.BillingAppStatus(stripeApp.GetStatus()),
		Definition:  definition,
		Labels:      labels.FromMetadata(stripeApp.GetMetadata()),
		Description: stripeApp.GetDescription(),
		CreatedAt:   lo.ToPtr(stripeApp.CreatedAt),
		UpdatedAt:   lo.ToPtr(stripeApp.UpdatedAt),
		DeletedAt:   stripeApp.DeletedAt,

		MaskedApiKey: stripeApp.MaskedAPIKey,
		AccountId:    stripeApp.StripeAccountID,
		Livemode:     stripeApp.Livemode,
	}

	return apiStripeApp, nil
}

func toAPIBillingAppExternalInvoicing(customInvoicingApp appcustominvoicing.Meta) (api.BillingAppExternalInvoicing, error) {
	definition, err := ToAPIBillingAppCatalogItem(customInvoicingApp.GetListing())
	if err != nil {
		return api.BillingAppExternalInvoicing{}, err
	}

	return api.BillingAppExternalInvoicing{
		Id:          customInvoicingApp.GetID().ID,
		Type:        api.BillingAppExternalInvoicingTypeExternalInvoicing,
		Name:        customInvoicingApp.GetName(),
		Status:      api.BillingAppStatus(customInvoicingApp.GetStatus()),
		Definition:  definition,
		Labels:      labels.FromMetadata(customInvoicingApp.GetMetadata()),
		Description: customInvoicingApp.GetDescription(),
		CreatedAt:   lo.ToPtr(customInvoicingApp.CreatedAt),
		UpdatedAt:   lo.ToPtr(customInvoicingApp.UpdatedAt),
		DeletedAt:   customInvoicingApp.DeletedAt,

		EnableDraftSyncHook:   customInvoicingApp.Configuration.EnableDraftSyncHook,
		EnableIssuingSyncHook: customInvoicingApp.Configuration.EnableIssuingSyncHook,
	}, nil
}
