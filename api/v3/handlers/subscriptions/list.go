package subscriptions

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/filters"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	ListSubscriptionsRequest  = subscription.ListSubscriptionsInput
	ListSubscriptionsResponse = response.PagePaginationResponse[api.BillingSubscription]
	ListSubscriptionsParams   = api.ListSubscriptionsParams
	ListSubscriptionsHandler  = httptransport.HandlerWithArgs[ListSubscriptionsRequest, ListSubscriptionsResponse, ListSubscriptionsParams]
)

func (h *handler) ListSubscriptions() ListSubscriptionsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListSubscriptionsParams) (ListSubscriptionsRequest, error) {
			// Resolve namespace
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListSubscriptionsRequest{}, err
			}

			// Pagination
			page := pagination.NewPage(1, 20)
			if params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Page.Number, 1),
					lo.FromPtrOr(params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListSubscriptionsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Field:  "page",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}

			// Build request
			req := ListSubscriptionsRequest{
				Namespaces: []string{ns},
				Page:       page,
			}

			// Filters
			if params.Filter != nil {
				customerID, err := filters.FromAPIFilterULID(params.Filter.CustomerId)
				if err != nil {
					return ListSubscriptionsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[customer_id]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.CustomerID = customerID

				id, err := filters.FromAPIFilterULID(params.Filter.Id)
				if err != nil {
					return ListSubscriptionsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[id]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.ID = id

				status, err := filters.FromAPIStatusFilter[subscription.SubscriptionStatus](ctx, params.Filter.Status)
				if err != nil {
					return ListSubscriptionsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[status]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.Status = status

				planID, err := filters.FromAPIFilterULID(params.Filter.PlanId)
				if err != nil {
					return ListSubscriptionsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[plan_id]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.PlanID = planID

				planKey, err := filters.FromAPIFilterStringExact(params.Filter.PlanKey)
				if err != nil {
					return ListSubscriptionsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[plan_key]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.PlanKey = planKey
			}

			// Sort
			if params.Sort != nil {
				sort, err := request.ParseSortBy(*params.Sort)
				if err != nil {
					return ListSubscriptionsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "sort", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.OrderBy = subscription.OrderBy(sort.Field)
				req.Order = sort.Order.ToSortxOrder()
			}

			return req, nil
		},
		func(ctx context.Context, request ListSubscriptionsRequest) (ListSubscriptionsResponse, error) {
			resp, err := h.subscriptionService.List(ctx, request)
			if err != nil {
				return ListSubscriptionsResponse{}, err
			}

			subscriptions := lo.Map(resp.Items, func(item subscription.Subscription, _ int) api.BillingSubscription {
				return ToAPIBillingSubscription(item)
			})

			r := response.NewPagePaginationResponse(subscriptions, response.PageMetaPage{
				Size:   request.Page.PageSize,
				Number: request.Page.PageNumber,
				Total:  lo.ToPtr(resp.TotalCount),
			})

			return r, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListSubscriptionsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-subscriptions"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
