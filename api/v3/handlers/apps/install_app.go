package apps

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/app"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	InstallAppRequest  = app.InstallAppV3Input
	InstallAppResponse = api.BillingInstallAppResponse
	InstallAppHandler  httptransport.Handler[InstallAppRequest, InstallAppResponse]
)

func (h *handler) InstallApp() InstallAppHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (InstallAppRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return InstallAppRequest{}, err
			}

			var body api.InstallAppJSONRequestBody
			if err := request.ParseBody(r, &body); err != nil {
				return InstallAppRequest{}, err
			}

			discType, err := body.Discriminator()
			if err != nil {
				return InstallAppRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{Field: "type", Reason: err.Error(), Source: apierrors.InvalidParamSourceBody},
				})
			}

			convertedType := api.BillingAppType(discType)

			if !convertedType.Valid() {
				err := fmt.Errorf("invalid app type: %s", discType)
				return InstallAppRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{Field: "type", Reason: err.Error(), Source: apierrors.InvalidParamSourceBody},
				})
			}

			domainType, err := ToDomainAppTypeFromAPIBillingAppType(api.BillingAppType(discType))
			if err != nil {
				return InstallAppRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{Field: "type", Reason: err.Error(), Source: apierrors.InvalidParamSourceBody},
				})
			}

			listingID := app.MarketplaceListingID{
				Type: domainType,
			}

			switch convertedType {
			case api.BillingAppTypeSandbox:
				sandbox, err := body.AsBillingInstallAppSandbox()
				if err != nil {
					return InstallAppRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "body", Reason: err.Error(), Source: apierrors.InvalidParamSourceBody},
					})
				}

				return InstallAppRequest{
					MarketplaceListingID:        listingID,
					Namespace:                   ns,
					Name:                        sandbox.Name,
					CreateDefaultBillingProfile: sandbox.CreateBillingProfile,
				}, nil
			case api.BillingAppTypeStripe:
				stripe, err := body.AsBillingInstallAppStripeWithApiKey()
				if err != nil {
					return InstallAppRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "body", Reason: err.Error(), Source: apierrors.InvalidParamSourceBody},
					})
				}

				return InstallAppRequest{
					MarketplaceListingID:        listingID,
					Namespace:                   ns,
					Name:                        stripe.Name,
					APIKey:                      lo.ToPtr(stripe.ApiKey),
					CreateDefaultBillingProfile: stripe.CreateBillingProfile,
				}, nil
			case api.BillingAppTypeExternalInvoicing:
				externalInvoicing, err := body.AsBillingInstallAppExternalInvoicing()
				if err != nil {
					return InstallAppRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "body", Reason: err.Error(), Source: apierrors.InvalidParamSourceBody},
					})
				}

				return InstallAppRequest{
					MarketplaceListingID:        listingID,
					Namespace:                   ns,
					Name:                        externalInvoicing.Name,
					CreateDefaultBillingProfile: externalInvoicing.CreateBillingProfile,
				}, nil
			default:
				return InstallAppRequest{}, fmt.Errorf("unsupported app type: %s", discType)
			}
		},
		func(ctx context.Context, request InstallAppRequest) (InstallAppResponse, error) {
			// make h.createBillingProfile transactional
			request.CreateDefaultBillingProfileFn = h.createBillingProfile

			resp, err := h.appService.InstallApp(ctx, request)
			if err != nil {
				return InstallAppResponse{}, fmt.Errorf("failed to install app: %w", err)
			}

			billingApp, err := ToAPIBillingApp(resp.App)
			if err != nil {
				return InstallAppResponse{}, fmt.Errorf("error converting installed app to api: %w", err)
			}

			capabilities, err := lo.MapErr(resp.DefaultCapabilies, func(c app.CapabilityType, _ int) (api.BillingAppCapabilityType, error) {
				return ToAPIBillingAppCapabilityTypeFromCapabilityType(c)
			})
			if err != nil {
				return InstallAppResponse{}, fmt.Errorf("error converting default capabilities to api: %w", err)
			}

			return InstallAppResponse{
				App:                       billingApp,
				DefaultForCapabilityTypes: capabilities,
			}, nil
		},
		commonhttp.JSONResponseEncoder[InstallAppResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("install-app"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

// createBillingProfile creates a default billing profile for the installed app based on its type
func (h *handler) createBillingProfile(ctx context.Context, installedApp app.App) ([]app.CapabilityType, error) {
	switch installedApp.GetType() {
	case app.AppTypeStripe:
		return h.makeStripeDefaultBillingApp(ctx, installedApp)
	case app.AppTypeSandbox:
		namespace := installedApp.GetID().Namespace
		if err := h.billingService.ProvisionDefaultBillingProfile(ctx, namespace); err != nil {
			return nil, fmt.Errorf("provision default billing profile: %w", err)
		}
		return []app.CapabilityType{
			app.CapabilityTypeCalculateTax,
			app.CapabilityTypeInvoiceCustomers,
			app.CapabilityTypeCollectPayments,
		}, nil
	case app.AppTypeCustomInvoicing:
		// TODO: Implement custom invoicing billing profile creation
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown app type: %s", installedApp.GetType())
	}
}

// Make Stripe app the default billing app if current one is Sandbox app
func (h *handler) makeStripeDefaultBillingApp(ctx context.Context, stripeApp app.App) ([]app.CapabilityType, error) {
	defaultForCapabilityTypes := []app.CapabilityType{}

	appID := stripeApp.GetID()

	// Check if it's a Stripe app
	if stripeApp.GetType() != app.AppTypeStripe {
		return defaultForCapabilityTypes, fmt.Errorf("app is not a stripe app: %s", appID.ID)
	}

	// Check if the default billing profile is a sandbox app type
	defaultBillingProfile, err := h.billingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
		Namespace: appID.Namespace,
	})
	if err != nil {
		return defaultForCapabilityTypes, fmt.Errorf("failed to get default billing profile: %w", err)
	}

	// Set default billing profile if the current default is the sandbox
	setDefaultBillingProfile := defaultBillingProfile != nil && defaultBillingProfile.Apps != nil && defaultBillingProfile.Apps.Invoicing.GetType() == app.AppTypeSandbox

	// Get supplier contract from stripe app
	supplierContract, err := h.stripeAppService.GetSupplierContact(ctx, appstripe.GetSupplierContactInput{
		AppID: appID,
	})
	if err != nil {
		return defaultForCapabilityTypes, fmt.Errorf("failed to get supplier contract for stripe app %s: %w", appID.ID, err)
	}

	// Create new default billing profile
	_, err = h.billingService.CreateProfile(ctx, billing.CreateProfileInput{
		Namespace:      appID.Namespace,
		Name:           "Stripe Billing Profile",
		Description:    lo.ToPtr("Stripe Billing Profile, created automatically"),
		Default:        setDefaultBillingProfile,
		Supplier:       supplierContract,
		WorkflowConfig: billing.DefaultWorkflowConfig,
		Apps: billing.ProfileAppReferences{
			Tax:       appID,
			Invoicing: appID,
			Payment:   appID,
		},
	})
	if err != nil {
		return defaultForCapabilityTypes, fmt.Errorf("failed to create billing profile for stripe app %s: %w", appID.ID, err)
	}

	defaultForCapabilityTypes = []app.CapabilityType{
		app.CapabilityTypeCalculateTax,
		app.CapabilityTypeInvoiceCustomers,
		app.CapabilityTypeCollectPayments,
	}

	return defaultForCapabilityTypes, nil
}
