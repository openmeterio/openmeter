package subscriptions

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	ListSubscriptionsRequest  = subscription.ListSubscriptionsInput
	ListSubscriptionsResponse = response.PagePaginationResponse[api.BillingSubscription]
	ListSubscriptionsParams   = api.ListSubscriptionsParams
	ListSubscriptionsHandler  httptransport.HandlerWithArgs[ListSubscriptionsRequest, ListSubscriptionsResponse, ListSubscriptionsParams]
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
				// Filter by customer ID
				if params.Filter.CustomerId != nil {
					// Get the customer to validate it exists
					_, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
						CustomerID: &customer.CustomerID{
							Namespace: ns,
							ID:        *params.Filter.CustomerId,
						},
					})
					if err != nil {
						return ListSubscriptionsRequest{}, err
					}

					// Add the customer ID filter to the request
					req.CustomerIDs = []string{*params.Filter.CustomerId}
				}
			}

			return req, nil
		},
		func(ctx context.Context, request ListSubscriptionsRequest) (ListSubscriptionsResponse, error) {
			resp, err := h.subscriptionService.List(ctx, request)
			if err != nil {
				return ListSubscriptionsResponse{}, err
			}

			subscriptions := lo.Map(resp.Items, func(item subscription.Subscription, _ int) api.BillingSubscription {
				return ConvertSubscriptionToAPISubscription(item)
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
