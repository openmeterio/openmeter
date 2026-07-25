package apps

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

// UninstallAppHandler is a handler to uninstall an app by id
type (
	UninstallAppRequest  = app.UninstallAppInput
	UninstallAppResponse = any
	UninstallAppHandler  httptransport.HandlerWithArgs[UninstallAppRequest, UninstallAppResponse, string]
)

// UninstallApp returns a handler to uninstall an app
func (h *handler) UninstallApp() UninstallAppHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, appId string) (UninstallAppRequest, error) {
			// Resolve namespace
			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return UninstallAppRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return UninstallAppRequest{
				Namespace: namespace,
				ID:        appId,
			}, nil
		},
		func(ctx context.Context, request UninstallAppRequest) (UninstallAppResponse, error) {
			// Check if the app is not used by any billing profile
			if err := h.billingService.IsAppUsed(ctx, request); err != nil {
				return nil, err
			}

			// Uninstall app
			err := h.appService.UninstallApp(ctx, request)
			if err != nil {
				return nil, fmt.Errorf("failed to uninstall app: %w", err)
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[UninstallAppResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("uninstall-app"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
