package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	appstripeentityapp "github.com/openmeterio/openmeter/openmeter/app/stripe/entity/app"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// ListAppsHandler is a handler for listing apps
type (
	ListAppsRequest  = appentity.ListAppInput
	ListAppsResponse = api.AppList
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
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

// GetAppHandler is a handler to get an app by id
type (
	GetAppRequest  = appentity.GetAppInput
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
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

// UninstallAppHandler is a handler to uninstalls an app by id
type (
	UninstallAppRequest  = appentity.UninstallAppInput
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
			err := h.service.UninstallApp(ctx, request)
			if err != nil {
				return nil, fmt.Errorf("failed to uninstall app: %w", err)
			}

			return nil, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[UninstallAppResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("uninstallApp"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

func MapAppToAPI(item appentity.App) (api.App, error) {
	switch item.GetType() {
	case appentitybase.AppTypeStripe:
		stripeApp := item.(appstripeentityapp.App)

		app := api.App{}
		if err := app.FromStripeApp(mapStripeAppToAPI(stripeApp)); err != nil {
			return app, err
		}

		return app, nil
	case appentitybase.AppTypeSandbox:
		sandboxApp := item.(appsandbox.App)

		app := api.App{}
		if err := app.FromSandboxApp(mapSandboxAppToAPI(sandboxApp)); err != nil {
			return app, err
		}

		return app, nil
	default:
		return api.App{}, fmt.Errorf("unsupported app type: %s", item.GetType())
	}
}

func mapSandboxAppToAPI(app appsandbox.App) api.SandboxApp {
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

func mapStripeAppToAPI(stripeApp appstripeentityapp.App) api.StripeApp {
	apiStripeApp := api.StripeApp{
		Id:              stripeApp.GetID().ID,
		Type:            api.StripeAppType(stripeApp.GetType()),
		Name:            stripeApp.Name,
		Status:          api.AppStatus(stripeApp.GetStatus()),
		Listing:         mapMarketplaceListing(stripeApp.GetListing()),
		CreatedAt:       stripeApp.CreatedAt,
		UpdatedAt:       stripeApp.UpdatedAt,
		DeletedAt:       stripeApp.DeletedAt,
		StripeAccountId: stripeApp.StripeAccountID,
		Livemode:        stripeApp.Livemode,
	}

	apiStripeApp.Description = stripeApp.GetDescription()

	if stripeApp.GetMetadata() != nil {
		apiStripeApp.Metadata = lo.ToPtr(stripeApp.GetMetadata())
	}

	return apiStripeApp
}
