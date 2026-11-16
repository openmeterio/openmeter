//go:generate go run github.com/jmattheis/goverter/cmd/goverter gen ./
package apps

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api/v3/labels"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
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
	ToAPIAppPagePaginatedResponse func(source response.PagePaginationResponse[apiv3models.BillingApp]) apiv3models.AppPagePaginatedResponse

	ToAPIBillingAppCatalogItem func(source app.MarketplaceListing) (apiv3models.BillingAppCatalogItem, error)

	// goverter:enum:map AppTypeStripe BillingAppTypeStripe
	// goverter:enum:map AppTypeSandbox BillingAppTypeSandbox
	// goverter:enum:map AppTypeCustomInvoicing BillingAppTypeExternalInvoicing
	ToAPIBillingAppTypeFromDomain func(source app.AppType) (apiv3models.BillingAppType, error)

	// goverter:enum:map BillingAppTypeStripe AppTypeStripe
	// goverter:enum:map BillingAppTypeSandbox AppTypeSandbox
	// goverter:enum:map BillingAppTypeExternalInvoicing AppTypeCustomInvoicing
	ToDomainAppTypeFromAPIBillingAppType func(source apiv3models.BillingAppType) (app.AppType, error)

	// goverter:enum:map CapabilityTypeReportUsage BillingAppCapabilityTypeReportUsage
	// goverter:enum:map CapabilityTypeReportEvents BillingAppCapabilityTypeReportEvents
	// goverter:enum:map CapabilityTypeCalculateTax BillingAppCapabilityTypeCalculateTax
	// goverter:enum:map CapabilityTypeInvoiceCustomers BillingAppCapabilityTypeInvoiceCustomers
	// goverter:enum:map CapabilityTypeCollectPayments BillingAppCapabilityTypeCollectPayments
	ToAPIBillingAppCapabilityTypeFromCapabilityType func(source app.CapabilityType) (apiv3models.BillingAppCapabilityType, error)

	// goverter:enum:map InstallMethodOAuth2 BillingAppInstallMethodsWithOauth2
	// goverter:enum:map InstallMethodAPIKey BillingAppInstallMethodsWithApiKey
	// goverter:enum:map InstallMethodNoCredentials BillingAppInstallMethodsNoCredentialsRequired
	ToAPIBillingAppInstallMethodsFromInstallMethod func(source app.InstallMethod) (apiv3models.BillingAppInstallMethods, error)

	ToAPIBillingApps func(source []app.App) ([]apiv3models.BillingApp, error)
)

func IntToFloat32(i int) float32 {
	return float32(i)
}

// ToAPIBillingApp maps an app to a v3 API app
func ToAPIBillingApp(item app.App) (apiv3models.BillingApp, error) {
	if item == nil {
		return apiv3models.BillingApp{}, errors.New("invalid app: nil")
	}

	switch item.GetType() {
	case app.AppTypeStripe:
		stripeApp, ok := item.(appstripe.App)
		if !ok {
			return apiv3models.BillingApp{}, fmt.Errorf("expected stripe app, got %T", item)
		}

		billingAppStripe, err := toAPIBillingAppStripe(stripeApp.Meta)
		if err != nil {
			return apiv3models.BillingApp{}, fmt.Errorf("failed to map stripe app to API: %w", err)
		}

		billingApp := apiv3models.BillingApp{}
		if err := billingApp.FromBillingAppStripe(billingAppStripe); err != nil {
			return billingApp, err
		}

		return billingApp, nil
	case app.AppTypeSandbox:
		sandboxApp, ok := item.(appsandbox.App)
		if !ok {
			return apiv3models.BillingApp{}, fmt.Errorf("expected sandbox app, got %T", item)
		}

		billingAppSandbox, err := toAPIBillingAppSandbox(sandboxApp.Meta)
		if err != nil {
			return apiv3models.BillingApp{}, fmt.Errorf("failed to map sandbox app to API: %w", err)
		}

		billingApp := apiv3models.BillingApp{}
		if err := billingApp.FromBillingAppSandbox(billingAppSandbox); err != nil {
			return billingApp, err
		}

		return billingApp, nil
	case app.AppTypeCustomInvoicing:
		customInvoicingApp, ok := item.(appcustominvoicing.App)
		if !ok {
			return apiv3models.BillingApp{}, fmt.Errorf("expected custom invoicing app, got %T", item)
		}

		billingAppExternalInvoicing, err := toAPIBillingAppExternalInvoicing(customInvoicingApp.Meta)
		if err != nil {
			return apiv3models.BillingApp{}, fmt.Errorf("failed to map custom invoicing app to API: %w", err)
		}

		billingApp := apiv3models.BillingApp{}
		if err := billingApp.FromBillingAppExternalInvoicing(billingAppExternalInvoicing); err != nil {
			return billingApp, err
		}

		return billingApp, nil
	default:
		return apiv3models.BillingApp{}, fmt.Errorf("unsupported app type: %s", item.GetType())
	}
}

func ToAPIBillingInstallAppResponse(item app.App, defaultForCapabilityTypes []apiv3models.BillingAppCapabilityType) (apiv3models.BillingInstallAppResponse, error) {
	if item == nil {
		return apiv3models.BillingInstallAppResponse{}, errors.New("invalid app: nil")
	}

	response := apiv3models.BillingInstallAppResponse{}

	switch item.GetType() {
	case app.AppTypeStripe:
		stripeApp, ok := item.(appstripe.App)
		if !ok {
			return response, fmt.Errorf("expected stripe app, got %T", item)
		}

		base, err := toAPIBillingAppStripe(stripeApp.Meta)
		if err != nil {
			return response, fmt.Errorf("failed to map stripe app to API: %w", err)
		}

		if err := response.FromBillingInstalledAppStripe(apiv3models.BillingInstalledAppStripe{
			Id:          base.Id,
			Type:        apiv3models.BillingInstalledAppStripeTypeStripe,
			Name:        base.Name,
			Status:      base.Status,
			Definition:  base.Definition,
			Labels:      base.Labels,
			Description: base.Description,
			CreatedAt:   base.CreatedAt,
			UpdatedAt:   base.UpdatedAt,
			DeletedAt:   base.DeletedAt,

			MaskedApiKey: base.MaskedApiKey,
			AccountId:    base.AccountId,
			Livemode:     base.Livemode,

			DefaultForCapabilityTypes: defaultForCapabilityTypes,
		}); err != nil {
			return response, err
		}

		return response, nil
	case app.AppTypeSandbox:
		sandboxApp, ok := item.(appsandbox.App)
		if !ok {
			return response, fmt.Errorf("expected sandbox app, got %T", item)
		}

		base, err := toAPIBillingAppSandbox(sandboxApp.Meta)
		if err != nil {
			return response, fmt.Errorf("failed to map sandbox app to API: %w", err)
		}

		if err := response.FromBillingInstalledAppSandbox(apiv3models.BillingInstalledAppSandbox{
			Id:          base.Id,
			Type:        apiv3models.BillingInstalledAppSandboxTypeSandbox,
			Name:        base.Name,
			Status:      base.Status,
			Definition:  base.Definition,
			Labels:      base.Labels,
			Description: base.Description,
			CreatedAt:   base.CreatedAt,
			UpdatedAt:   base.UpdatedAt,
			DeletedAt:   base.DeletedAt,

			DefaultForCapabilityTypes: defaultForCapabilityTypes,
		}); err != nil {
			return response, err
		}

		return response, nil
	case app.AppTypeCustomInvoicing:
		customInvoicingApp, ok := item.(appcustominvoicing.App)
		if !ok {
			return response, fmt.Errorf("expected custom invoicing app, got %T", item)
		}

		base, err := toAPIBillingAppExternalInvoicing(customInvoicingApp.Meta)
		if err != nil {
			return response, fmt.Errorf("failed to map custom invoicing app to API: %w", err)
		}

		if err := response.FromBillingInstalledAppExternalInvoicing(apiv3models.BillingInstalledAppExternalInvoicing{
			Id:          base.Id,
			Type:        apiv3models.BillingInstalledAppExternalInvoicingTypeExternalInvoicing,
			Name:        base.Name,
			Status:      base.Status,
			Definition:  base.Definition,
			Labels:      base.Labels,
			Description: base.Description,
			CreatedAt:   base.CreatedAt,
			UpdatedAt:   base.UpdatedAt,
			DeletedAt:   base.DeletedAt,

			EnableDraftSyncHook:   base.EnableDraftSyncHook,
			EnableIssuingSyncHook: base.EnableIssuingSyncHook,

			DefaultForCapabilityTypes: defaultForCapabilityTypes,
		}); err != nil {
			return response, err
		}

		return response, nil
	default:
		return response, fmt.Errorf("unsupported app type: %s", item.GetType())
	}
}

func toAPIBillingAppSandbox(sandboxApp appsandbox.Meta) (apiv3models.BillingAppSandbox, error) {
	definition, err := ToAPIBillingAppCatalogItem(sandboxApp.GetListing())
	if err != nil {
		return apiv3models.BillingAppSandbox{}, err
	}

	return apiv3models.BillingAppSandbox{
		Id:          sandboxApp.GetID().ID,
		Type:        apiv3models.BillingAppSandboxTypeSandbox,
		Name:        sandboxApp.GetName(),
		Status:      apiv3models.BillingAppStatus(sandboxApp.GetStatus()),
		Definition:  definition,
		Labels:      labels.FromMetadata(sandboxApp.GetMetadata()),
		Description: sandboxApp.GetDescription(),
		CreatedAt:   sandboxApp.CreatedAt,
		UpdatedAt:   sandboxApp.UpdatedAt,
		DeletedAt:   sandboxApp.DeletedAt,
		Actions:     toAPIBillingAppActions(sandboxApp),
	}, nil
}

func toAPIBillingAppActions(item interface{ Actions() []app.AppAction }) *[]apiv3models.BillingAppAction {
	actions := item.Actions()
	if len(actions) == 0 {
		return nil
	}

	return lo.ToPtr(lo.Map(actions, func(action app.AppAction, _ int) apiv3models.BillingAppAction {
		return apiv3models.BillingAppAction{
			Type:        apiv3models.BillingAppActionType(action.Type),
			Description: action.Description,
		}
	}))
}

func toAPIBillingAppStripe(
	stripeApp appstripe.Meta,
) (apiv3models.BillingAppStripe, error) {
	definition, err := ToAPIBillingAppCatalogItem(stripeApp.GetListing())
	if err != nil {
		return apiv3models.BillingAppStripe{}, err
	}

	apiStripeApp := apiv3models.BillingAppStripe{
		Id:          stripeApp.GetID().ID,
		Type:        apiv3models.BillingAppStripeType(stripeApp.GetType()),
		Name:        stripeApp.Name,
		Status:      apiv3models.BillingAppStatus(stripeApp.GetStatus()),
		Definition:  definition,
		Labels:      labels.FromMetadata(stripeApp.GetMetadata()),
		Description: stripeApp.GetDescription(),
		CreatedAt:   stripeApp.CreatedAt,
		UpdatedAt:   stripeApp.UpdatedAt,
		DeletedAt:   stripeApp.DeletedAt,
		Actions:     toAPIBillingAppActions(stripeApp),

		MaskedApiKey: stripeApp.MaskedAPIKey,
		AccountId:    stripeApp.StripeAccountID,
		Livemode:     stripeApp.Livemode,
	}

	return apiStripeApp, nil
}

func toAPIBillingAppExternalInvoicing(customInvoicingApp appcustominvoicing.Meta) (apiv3models.BillingAppExternalInvoicing, error) {
	definition, err := ToAPIBillingAppCatalogItem(customInvoicingApp.GetListing())
	if err != nil {
		return apiv3models.BillingAppExternalInvoicing{}, err
	}

	return apiv3models.BillingAppExternalInvoicing{
		Id:          customInvoicingApp.GetID().ID,
		Type:        apiv3models.BillingAppExternalInvoicingTypeExternalInvoicing,
		Name:        customInvoicingApp.GetName(),
		Status:      apiv3models.BillingAppStatus(customInvoicingApp.GetStatus()),
		Definition:  definition,
		Labels:      labels.FromMetadata(customInvoicingApp.GetMetadata()),
		Description: customInvoicingApp.GetDescription(),
		CreatedAt:   customInvoicingApp.CreatedAt,
		UpdatedAt:   customInvoicingApp.UpdatedAt,
		DeletedAt:   customInvoicingApp.DeletedAt,
		Actions:     toAPIBillingAppActions(customInvoicingApp),

		EnableDraftSyncHook:   customInvoicingApp.Configuration.EnableDraftSyncHook,
		EnableIssuingSyncHook: customInvoicingApp.Configuration.EnableIssuingSyncHook,
	}, nil
}
