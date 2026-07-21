package apps

import (
	"context"
	"fmt"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	UpdateAppRequest  = app.UpdateAppInput
	UpdateAppResponse = api.BillingApp
	UpdateAppHandler  httptransport.HandlerWithArgs[UpdateAppRequest, UpdateAppResponse, string]
)

func (h *handler) UpdateApp() UpdateAppHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, appId string) (UpdateAppRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateAppRequest{}, err
			}

			var body api.UpdateAppJSONRequestBody
			if err := request.ParseBody(r, &body); err != nil {
				return UpdateAppRequest{}, err
			}

			discType, err := body.Discriminator()
			if err != nil {
				return UpdateAppRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{Field: "type", Reason: err.Error(), Source: apierrors.InvalidParamSourceBody},
				})
			}

			convertedType := api.BillingAppType(discType)

			if !convertedType.Valid() {
				err := fmt.Errorf("invalid app type: %s", discType)
				return UpdateAppRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{Field: "type", Reason: err.Error(), Source: apierrors.InvalidParamSourceBody},
				})
			}

			appID := app.AppID{
				Namespace: ns,
				ID:        appId,
			}

			switch convertedType {
			case api.BillingAppTypeSandbox:
				sandbox, err := body.AsUpdateAppSandboxRequest()
				if err != nil {
					return UpdateAppRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "body", Reason: err.Error(), Source: apierrors.InvalidParamSourceBody},
					})
				}

				metadata, err := metadataFromAPILabels(sandbox.Labels)
				if err != nil {
					return UpdateAppRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "labels", Reason: err.Error(), Source: apierrors.InvalidParamSourceBody},
					})
				}

				return UpdateAppRequest{
					AppID:           appID,
					Name:            sandbox.Name,
					Description:     sandbox.Description,
					Metadata:        metadata,
					AppConfigUpdate: appsandbox.Configuration{},
				}, nil
			case api.BillingAppTypeStripe:
				stripe, err := body.AsUpdateAppStripeRequest()
				if err != nil {
					return UpdateAppRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "body", Reason: err.Error(), Source: apierrors.InvalidParamSourceBody},
					})
				}

				metadata, err := metadataFromAPILabels(stripe.Labels)
				if err != nil {
					return UpdateAppRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "labels", Reason: err.Error(), Source: apierrors.InvalidParamSourceBody},
					})
				}

				return UpdateAppRequest{
					AppID:       appID,
					Name:        stripe.Name,
					Description: stripe.Description,
					Metadata:    metadata,
					AppConfigUpdate: appstripe.Configuration{
						SecretAPIKey: stripe.SecretApiKey,
					},
				}, nil
			case api.BillingAppTypeExternalInvoicing:
				externalInvoicing, err := body.AsUpdateAppExternalInvoicingRequest()
				if err != nil {
					return UpdateAppRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "body", Reason: err.Error(), Source: apierrors.InvalidParamSourceBody},
					})
				}

				metadata, err := metadataFromAPILabels(externalInvoicing.Labels)
				if err != nil {
					return UpdateAppRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "labels", Reason: err.Error(), Source: apierrors.InvalidParamSourceBody},
					})
				}

				return UpdateAppRequest{
					AppID:       appID,
					Name:        externalInvoicing.Name,
					Description: externalInvoicing.Description,
					Metadata:    metadata,
					AppConfigUpdate: appcustominvoicing.Configuration{
						EnableDraftSyncHook:   externalInvoicing.EnableDraftSyncHook,
						EnableIssuingSyncHook: externalInvoicing.EnableIssuingSyncHook,
					},
				}, nil
			default:
				return UpdateAppRequest{}, fmt.Errorf("unsupported app type: %s", discType)
			}
		},
		func(ctx context.Context, request UpdateAppRequest) (UpdateAppResponse, error) {
			updated, err := h.appService.UpdateApp(ctx, request)
			if err != nil {
				return UpdateAppResponse{}, fmt.Errorf("failed to update app: %w", err)
			}

			return ToAPIBillingApp(updated)
		},
		commonhttp.JSONResponseEncoder[UpdateAppResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-app"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

// metadataFromAPILabels converts the API labels of an app update request into the
// domain Metadata pointer expected by app.UpdateAppInput. Update is a full replace
// (mirrors the v1 PUT semantics), so a nil labels field is passed through as a nil
// pointer, which the adapter's SetOrClearMetadata interprets as "clear metadata" -
// omitting "labels" in the request body removes any previously configured labels.
func metadataFromAPILabels(l *api.Labels) (*map[string]string, error) {
	if l == nil {
		return nil, nil
	}

	metadata, err := labels.ToMetadata(l)
	if err != nil {
		return nil, err
	}

	m := map[string]string(metadata)

	return &m, nil
}
