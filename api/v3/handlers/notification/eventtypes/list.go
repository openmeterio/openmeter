package eventtypes

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/filters"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type ListNotificationEventTypesRequest struct {
	Page pagination.Page
	Type *filter.FilterString
}

type (
	ListNotificationEventTypesResponse = response.PagePaginationResponse[apiv3models.BillingNotificationEventTypeSummary]
	ListNotificationEventTypesParams   = apiv3models.ListBillingNotificationEventTypesParams
	ListNotificationEventTypesHandler  = httptransport.HandlerWithArgs[ListNotificationEventTypesRequest, ListNotificationEventTypesResponse, ListNotificationEventTypesParams]
)

func (h *handler) ListNotificationEventTypes() ListNotificationEventTypesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListNotificationEventTypesParams) (ListNotificationEventTypesRequest, error) {
			page := pagination.NewPage(1, 20)
			if params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Page.Number, 1),
					lo.FromPtrOr(params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListNotificationEventTypesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{Field: "page", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
				})
			}

			req := ListNotificationEventTypesRequest{Page: page}

			if params.Filter != nil {
				typeFilter, err := filters.FromAPIFilterString(params.Filter.Type)
				if err != nil {
					return ListNotificationEventTypesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[type]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}

				req.Type = typeFilter
			}

			return req, nil
		},
		func(ctx context.Context, req ListNotificationEventTypesRequest) (ListNotificationEventTypesResponse, error) {
			// The registry is small and in memory, so filtering and paging happen
			// here, in its version-then-name order.
			matching := make([]apiv3models.BillingNotificationEventTypeSummary, 0)

			for _, d := range h.registry.EventTypes(nil) {
				if req.Type != nil {
					ok, err := req.Type.Match(d.EventType.String())
					if err != nil {
						return ListNotificationEventTypesResponse{}, err
					}

					if !ok {
						continue
					}
				}

				matching = append(matching, ToAPIBillingNotificationEventTypeSummary(d))
			}

			total := len(matching)
			start := min(req.Page.Offset(), total)
			end := min(start+req.Page.Limit(), total)

			return response.NewPagePaginationResponse(matching[start:end], response.PageMetaPage{
				Size:   req.Page.PageSize,
				Number: req.Page.PageNumber,
				Total:  &total,
			}), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListNotificationEventTypesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-billing-notification-event-types"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
