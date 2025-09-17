package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/customer"
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
		CustomerIDOrKey string
		Params          api.ListCustomerSubscriptionsParams
	}
	ListCustomerSubscriptionsRequest = struct {
		CustomerID customer.CustomerID
		Page       pagination.Page
	}
	ListCustomerSubscriptionsResponse = pagination.Result[api.Subscription]
	ListCustomerSubscriptionsHandler  = httptransport.HandlerWithArgs[ListCustomerSubscriptionsRequest, ListCustomerSubscriptionsResponse, ListCustomerSubscriptionsParams]
)

func (h *handler) ListCustomerSubscriptions() ListCustomerSubscriptionsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListCustomerSubscriptionsParams) (ListCustomerSubscriptionsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCustomerSubscriptionsRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			// Get the customer
			cus, err := h.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerIDOrKey: &customer.CustomerIDOrKey{
					IDOrKey:   params.CustomerIDOrKey,
					Namespace: ns,
				},
			})
			if err != nil {
				return ListCustomerSubscriptionsRequest{}, err
			}

			if cus != nil && cus.IsDeleted() {
				return ListCustomerSubscriptionsRequest{}, models.NewGenericPreConditionFailedError(
					fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
				)
			}

			return ListCustomerSubscriptionsRequest{
				CustomerID: cus.GetID(),
				Page:       pagination.NewPageFromRef(params.Params.Page, params.Params.PageSize),
			}, nil
		},
		func(ctx context.Context, req ListCustomerSubscriptionsRequest) (ListCustomerSubscriptionsResponse, error) {
			var def ListCustomerSubscriptionsResponse

			subs, err := h.SubscriptionService.List(ctx, subscription.ListSubscriptionsInput{
				Page:       req.Page,
				Namespaces: []string{req.CustomerID.Namespace},
				Customers:  []string{req.CustomerID.ID},
			})
			if err != nil {
				return def, err
			}

			apiSubs := make([]api.Subscription, len(subs.Items))

			for i, sub := range subs.Items {
				apiSubs[i] = MapSubscriptionToAPI(sub)
			}

			return ListCustomerSubscriptionsResponse{
				Page:       req.Page,
				TotalCount: subs.TotalCount,
				Items:      apiSubs,
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListCustomerSubscriptionsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.Options,
			httptransport.WithOperationName("getSubscription"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
