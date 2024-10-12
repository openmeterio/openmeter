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
	appstripeentityapp "github.com/openmeterio/openmeter/openmeter/app/stripe/entity/app"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// ListAppsHandler is a handler for listing marketplace listings
type (
	ListAppsRequest  = appentity.ListAppInput
	ListAppsResponse = api.AppList
	ListAppsParams   = api.ListAppsParams
	ListAppsHandler  httptransport.HandlerWithArgs[ListAppsRequest, ListAppsResponse, ListAppsParams]
)

// ListApps returns a handler for listing marketplace listings
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

			return ListAppsResponse{
				Page:       result.Page.PageNumber,
				PageSize:   result.Page.PageSize,
				TotalCount: result.TotalCount,
				Items: lo.Map(result.Items, func(item appentity.App, _ int) api.App {
					return mapAppToAPI(item)
				}),
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

func mapAppToAPI(item appentity.App) api.App {
	switch item.GetType() {
	case appentitybase.AppTypeStripe:
		stripeApp := item.(appstripeentityapp.App)
		return mapStripeAppToAPI(stripeApp)
	default:
		return api.App{
			Id:        item.GetAppBase().ID,
			Type:      api.StripeAppType(item.GetType()),
			Name:      item.GetName(),
			Status:    api.OpenMeterAppAppStatus(item.GetAppBase().Status),
			Listing:   mapMarketplaceListing(item.GetAppBase().Listing),
			CreatedAt: item.GetAppBase().CreatedAt,
			UpdatedAt: item.GetAppBase().UpdatedAt,
			DeletedAt: item.GetAppBase().DeletedAt,
			// TODO
			// Description: "TODO",
			// Metadata:  &map[string]string{},
		}
	}
}

func mapStripeAppToAPI(stripeApp appstripeentityapp.App) api.StripeApp {
	return api.StripeApp{
		Id:              stripeApp.ID,
		Type:            api.StripeAppType(stripeApp.Type),
		Name:            stripeApp.Name,
		Status:          api.OpenMeterAppAppStatus(stripeApp.Status),
		Listing:         mapMarketplaceListing(stripeApp.Listing),
		CreatedAt:       stripeApp.CreatedAt,
		UpdatedAt:       stripeApp.UpdatedAt,
		DeletedAt:       stripeApp.DeletedAt,
		StripeAccountId: stripeApp.StripeAccountID,
		Livemode:        stripeApp.Livemode,
		// TODO
		// Description: "TODO",
		// Metadata:  &map[string]string{},
	}
}
