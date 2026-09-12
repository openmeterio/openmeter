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
	"github.com/openmeterio/openmeter/openmeter/app/billingprofile"
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
			// make the billing profile provisioning transactional
			request.CreateDefaultBillingProfileFn = func(ctx context.Context, installedApp app.App) ([]app.CapabilityType, error) {
				return billingprofile.CreateDefault(ctx, h.billingService, h.stripeAppService, installedApp)
			}

			resp, err := h.appService.InstallApp(ctx, request)
			if err != nil {
				return InstallAppResponse{}, fmt.Errorf("failed to install app: %w", err)
			}

			capabilities, err := lo.MapErr(resp.DefaultCapabilies, func(c app.CapabilityType, _ int) (api.BillingAppCapabilityType, error) {
				return ToAPIBillingAppCapabilityTypeFromCapabilityType(c)
			})
			if err != nil {
				return InstallAppResponse{}, fmt.Errorf("error converting default capabilities to api: %w", err)
			}

			response, err := ToAPIBillingInstallAppResponse(resp.App, capabilities)
			if err != nil {
				return InstallAppResponse{}, fmt.Errorf("error converting installed app to api: %w", err)
			}

			return response, nil
		},
		commonhttp.JSONResponseEncoder[InstallAppResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("install-app"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
