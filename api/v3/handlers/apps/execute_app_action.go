package apps

import (
	"context"
	"fmt"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	ExecuteAppActionRequest  = app.ExecuteAppActionInput
	ExecuteAppActionResponse = api.BillingApp
	ExecuteAppActionHandler  httptransport.HandlerWithArgs[ExecuteAppActionRequest, ExecuteAppActionResponse, string]
)

func (h *handler) ExecuteAppAction() ExecuteAppActionHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, appId string) (ExecuteAppActionRequest, error) {
			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return ExecuteAppActionRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			var body api.ExecuteAppActionJSONRequestBody
			if err := request.ParseBody(r, &body); err != nil {
				return ExecuteAppActionRequest{}, err
			}

			actionType, err := body.Discriminator()
			if err == nil && !api.BillingAppActionType(actionType).Valid() {
				err = fmt.Errorf("invalid action type: %s", actionType)
			}
			if err != nil {
				return ExecuteAppActionRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{
						Field:  "action_type",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceBody,
					},
				})
			}

			return ExecuteAppActionRequest{
				AppID: app.AppID{
					Namespace: namespace,
					ID:        appId,
				},
				Type: app.AppActionType(actionType),
			}, nil
		},
		func(ctx context.Context, request ExecuteAppActionRequest) (ExecuteAppActionResponse, error) {
			updatedApp, err := h.appService.ExecuteAppAction(ctx, request)
			if err != nil {
				return ExecuteAppActionResponse{}, fmt.Errorf("failed to execute app action: %w", err)
			}

			return ToAPIBillingApp(updatedApp)
		},
		commonhttp.JSONResponseEncoder[ExecuteAppActionResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("execute-app-action"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
