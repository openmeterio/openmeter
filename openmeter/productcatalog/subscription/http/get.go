package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	GetSubscriptionRequest = struct {
		ID    models.NamespacedID
		Query api.GetSubscriptionParams
	}
	GetSubscriptionResponse = api.SubscriptionExpanded
	GetSubscriptionParams   = struct {
		Query api.GetSubscriptionParams
		ID    string
	}
	GetSubscriptionHandler = httptransport.HandlerWithArgs[GetSubscriptionRequest, GetSubscriptionResponse, GetSubscriptionParams]
)

func (h *handler) GetSubscription() GetSubscriptionHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetSubscriptionParams) (GetSubscriptionRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetSubscriptionRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return GetSubscriptionRequest{
				ID: models.NamespacedID{
					Namespace: ns,
					ID:        params.ID,
				},
				Query: params.Query,
			}, nil
		},
		func(ctx context.Context, req GetSubscriptionRequest) (GetSubscriptionResponse, error) {
			var def GetSubscriptionResponse

			if req.Query.At != nil {
				return def, commonhttp.NewHTTPError(http.StatusNotImplemented, fmt.Errorf("historical queries are not supported"))
			}

			view, err := h.SubscriptionService.GetView(ctx, req.ID)
			if err != nil {
				return def, err
			}

			return MapSubscriptionViewToAPI(view)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetSubscriptionResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.Options,
			httptransport.WithOperationName("getSubscription"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	ListCustomerSubscriptionsParams = struct {
		CustomerID string
		Params     api.ListCustomerSubscriptionsParams
	}
	ListCustomerSubscriptionsRequest = struct {
		CustomerID models.NamespacedID
		Page       pagination.Page
		Expand     bool
	}
	ListCustomerSubscriptionsResponse = commonhttp.Either[pagination.PagedResponse[api.Subscription], pagination.PagedResponse[api.SubscriptionExpanded]]
	ListCustomerSubscriptionsHandler  = httptransport.HandlerWithArgs[ListCustomerSubscriptionsRequest, ListCustomerSubscriptionsResponse, ListCustomerSubscriptionsParams]
)

func (h *handler) ListCustomerSubscriptions() ListCustomerSubscriptionsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListCustomerSubscriptionsParams) (ListCustomerSubscriptionsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCustomerSubscriptionsRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return ListCustomerSubscriptionsRequest{
				CustomerID: models.NamespacedID{
					Namespace: ns,
					ID:        params.CustomerID,
				},
				Page:   pagination.NewPageFromRef(params.Params.Page, params.Params.PageSize),
				Expand: lo.FromPtrOr(params.Params.ExpandToView, false),
			}, nil
		},
		func(ctx context.Context, req ListCustomerSubscriptionsRequest) (ListCustomerSubscriptionsResponse, error) {
			var def ListCustomerSubscriptionsResponse

			subs, err := h.SubscriptionService.List(ctx, subscription.ListSubscriptionsInput{
				Page:         req.Page,
				Namespaces:   []string{req.CustomerID.Namespace},
				Customers:    []string{req.CustomerID.ID},
				ExpandToView: req.Expand,
			})
			if err != nil {
				return def, err
			}

			mapped := mo.Fold[subscription.PagedSubscriptions, subscription.PagedSubscriptionViews, mo.Result[ListCustomerSubscriptionsResponse]](subs, func(subs subscription.PagedSubscriptionViews) mo.Result[ListCustomerSubscriptionsResponse] {
				apiSubs := make([]api.SubscriptionExpanded, len(subs.Items))
				for i, sub := range subs.Items {
					apiSubs[i], err = MapSubscriptionViewToAPI(sub)
					if err != nil {
						return mo.Err[ListCustomerSubscriptionsResponse](err)
					}
				}

				return mo.Ok[ListCustomerSubscriptionsResponse](ListCustomerSubscriptionsResponse{
					Either: mo.Right[pagination.PagedResponse[api.Subscription], pagination.PagedResponse[api.SubscriptionExpanded]](pagination.PagedResponse[api.SubscriptionExpanded]{
						Page:       subs.Page,
						TotalCount: subs.TotalCount,
						Items:      apiSubs,
					}),
				})
			}, func(subs subscription.PagedSubscriptions) mo.Result[ListCustomerSubscriptionsResponse] {
				apiSubs := make([]api.Subscription, len(subs.Items))
				for i, sub := range subs.Items {
					apiSubs[i] = MapSubscriptionToAPI(sub)
				}

				return mo.Ok[ListCustomerSubscriptionsResponse](ListCustomerSubscriptionsResponse{
					Either: mo.Left[pagination.PagedResponse[api.Subscription], pagination.PagedResponse[api.SubscriptionExpanded]](pagination.PagedResponse[api.Subscription]{
						Page:       subs.Page,
						TotalCount: subs.TotalCount,
						Items:      apiSubs,
					}),
				})
			})

			if mapped.IsError() {
				return def, mapped.Error()
			}

			return mapped.MustGet(), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListCustomerSubscriptionsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.Options,
			httptransport.WithOperationName("getSubscription"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
