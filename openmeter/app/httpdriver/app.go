package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	appstripeentityapp "github.com/openmeterio/openmeter/openmeter/app/stripe/entity/app"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// ListAppsHandler is a handler for listing apps
type (
	ListAppsRequest  = app.ListAppInput
	ListAppsResponse = api.AppPaginatedResponse
	ListAppsParams   = api.ListAppsParams
	ListAppsHandler  httptransport.HandlerWithArgs[ListAppsRequest, ListAppsResponse, ListAppsParams]
)

// ListApps returns a handler for listing apps
func (h *handler) ListApps() ListAppsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListAppsParams) (ListAppsRequest, error) {
			// Resolve namespace
			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListAppsRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return ListAppsRequest{
				Namespace: namespace,
				Page: pagination.Page{
					PageSize:   lo.FromPtrOr(params.PageSize, app.DefaultPageSize),
					PageNumber: lo.FromPtrOr(params.Page, app.DefaultPageNumber),
				},
			}, nil
		},
		func(ctx context.Context, request ListAppsRequest) (ListAppsResponse, error) {
			result, err := h.service.ListApps(ctx, request)
			if err != nil {
				return ListAppsResponse{}, fmt.Errorf("failed to list apps: %w", err)
			}

			items := make([]api.App, 0, len(result.Items))
			for _, item := range result.Items {
				app, err := MapAppToAPI(item)
				if err != nil {
					return ListAppsResponse{}, fmt.Errorf("failed to map app to api: %w", err)
				}

				items = append(items, app)
			}

			return ListAppsResponse{
				Page:       result.Page.PageNumber,
				PageSize:   result.Page.PageSize,
				TotalCount: result.TotalCount,
				Items:      items,
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListAppsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listApps"),
		)...,
	)
}

// GetAppHandler is a handler to get an app by id
type (
	GetAppRequest  = app.GetAppInput
	GetAppResponse = api.App
	GetAppHandler  httptransport.HandlerWithArgs[GetAppRequest, GetAppResponse, string]
)

// GetApp returns an app handler
func (h *handler) GetApp() GetAppHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, appId string) (GetAppRequest, error) {
			// Resolve namespace
			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetAppRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return GetAppRequest{
				Namespace: namespace,
				ID:        appId,
			}, nil
		},
		func(ctx context.Context, request GetAppRequest) (GetAppResponse, error) {
			app, err := h.service.GetApp(ctx, request)
			if err != nil {
				return GetAppResponse{}, fmt.Errorf("failed to get app: %w", err)
			}

			return MapAppToAPI(app)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetAppResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getApp"),
		)...,
	)
}

// UpdateAppHandler is a handler to update an app
type (
	UpdateAppRequest  = app.UpdateAppInput
	UpdateAppResponse = api.App
	UpdateAppHandler  httptransport.HandlerWithArgs[UpdateAppRequest, UpdateAppResponse, string]
)

// UpdateApp returns an app handler
func (h *handler) UpdateApp() UpdateAppHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, appId string) (UpdateAppRequest, error) {
			// Resolve namespace
			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateAppRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			var body api.UpdateAppJSONRequestBody
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return UpdateAppRequest{}, fmt.Errorf("field to decode upsert customer data request: %w", err)
			}

			updateType, err := body.Discriminator()
			if err != nil {
				return UpdateAppRequest{}, models.NewGenericValidationError(fmt.Errorf("failed to get update type: %w", err))
			}

			switch updateType {
			case string(app.AppTypeStripe):
				payload, err := body.AsStripeAppReplaceUpdate()
				if err != nil {
					return UpdateAppRequest{}, fmt.Errorf("failed to get stripe app replace update: %w", err)
				}

				return UpdateAppRequest{
					AppID: app.AppID{
						ID:        appId,
						Namespace: namespace,
					},
					Name:        payload.Name,
					Description: payload.Description,
					Metadata:    payload.Metadata,
					AppConfigUpdate: appstripeentityapp.Configuration{
						SecretAPIKey: payload.SecretAPIKey,
					},
				}, nil

			case string(app.AppTypeSandbox):
				payload, err := body.AsSandboxAppReplaceUpdate()
				if err != nil {
					return UpdateAppRequest{}, fmt.Errorf("failed to get sandbox app replace update: %w", err)
				}

				return UpdateAppRequest{
					AppID: app.AppID{
						ID:        appId,
						Namespace: namespace,
					},
					Name:            payload.Name,
					Description:     payload.Description,
					Metadata:        payload.Metadata,
					AppConfigUpdate: appsandbox.Configuration{},
				}, nil
			case string(app.AppTypeCustomInvoicing):
				payload, err := body.AsCustomInvoicingAppReplaceUpdate()
				if err != nil {
					return UpdateAppRequest{}, fmt.Errorf("failed to get custom invoicing app replace update: %w", err)
				}

				return UpdateAppRequest{
					AppID: app.AppID{
						ID:        appId,
						Namespace: namespace,
					},
					Name:        payload.Name,
					Description: payload.Description,
					Metadata:    payload.Metadata,
					AppConfigUpdate: appcustominvoicing.Configuration{
						EnableDraftSyncHook:   payload.EnableDraftSyncHook,
						EnableIssuingSyncHook: payload.EnableIssuingSyncHook,
					},
				}, nil
			default:
				return UpdateAppRequest{}, models.NewGenericValidationError(fmt.Errorf("invalid app type: %s", updateType))
			}
		},
		func(ctx context.Context, request UpdateAppRequest) (UpdateAppResponse, error) {
			app, err := h.service.UpdateApp(ctx, request)
			if err != nil {
				return UpdateAppResponse{}, fmt.Errorf("failed to update app: %w", err)
			}

			return MapAppToAPI(app)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateAppResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("updateApp"),
		)...,
	)
}

// UninstallAppHandler is a handler to uninstalls an app by id
type (
	UninstallAppRequest  = app.UninstallAppInput
	UninstallAppResponse = interface{}
	UninstallAppHandler  httptransport.HandlerWithArgs[UninstallAppRequest, UninstallAppResponse, string]
)

// UninstallApp uninstalls an app
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
			err := h.service.UninstallApp(ctx, request)
			if err != nil {
				return nil, fmt.Errorf("failed to uninstall app: %w", err)
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[UninstallAppResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("uninstallApp"),
		)...,
	)
}
