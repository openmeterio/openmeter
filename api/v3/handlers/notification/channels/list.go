package channels

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/filters"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	ListNotificationChannelsRequest  = notification.ListChannelsInput
	ListNotificationChannelsResponse = response.PagePaginationResponse[api.NotificationChannel]
	ListNotificationChannelsParams   = api.ListNotificationChannelsParams
	ListNotificationChannelsHandler  = httptransport.HandlerWithArgs[ListNotificationChannelsRequest, ListNotificationChannelsResponse, ListNotificationChannelsParams]
)

func (h *handler) ListNotificationChannels() ListNotificationChannelsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListNotificationChannelsParams) (ListNotificationChannelsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListNotificationChannelsRequest{}, err
			}

			page := pagination.NewPage(1, 20)
			if params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Page.Number, 1),
					lo.FromPtrOr(params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListNotificationChannelsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Field:  "page",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}

			req := ListNotificationChannelsRequest{
				Namespaces: []string{ns},
				Page:       page,
			}

			if params.Sort != nil {
				sort, err := request.ParseSortBy(*params.Sort)
				if err != nil {
					return ListNotificationChannelsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						apierrors.InvalidParameter{
							Field:  "sort",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceQuery,
						},
					})
				}

				orderBy, err := FromAPIChannelSortField(ctx, sort.Field)
				if err != nil {
					return ListNotificationChannelsRequest{}, err
				}
				req.OrderBy = orderBy
				req.Order = sort.Order.ToSortxOrder()
			}

			if params.Filter != nil {
				id, err := filters.FromAPIFilterULID(params.Filter.Id)
				if err != nil {
					return ListNotificationChannelsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[id]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.ID = id

				name, err := filters.FromAPIFilterString(params.Filter.Name)
				if err != nil {
					return ListNotificationChannelsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[name]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.Name = name

				typeFilter, err := filters.FromAPIFilterStringExact(params.Filter.Type)
				if err != nil {
					return ListNotificationChannelsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[type]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				// Translate the wire-format type value(s) ("webhook") into the
				// domain/DB value ("WEBHOOK") before this reaches the adapter.
				typeFilter, err = mapAPIChannelTypeFilter(typeFilter)
				if err != nil {
					return ListNotificationChannelsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[type]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.Type = typeFilter

				disabled, err := filters.FromAPIFilterBoolean(params.Filter.Disabled)
				if err != nil {
					return ListNotificationChannelsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[disabled]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.Disabled = disabled

				createdAt, err := filters.FromAPIFilterDateTime(params.Filter.CreatedAt)
				if err != nil {
					return ListNotificationChannelsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[created_at]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.CreatedAt = createdAt

				updatedAt, err := filters.FromAPIFilterDateTime(params.Filter.UpdatedAt)
				if err != nil {
					return ListNotificationChannelsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[updated_at]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.UpdatedAt = updatedAt
			}

			return req, nil
		},
		func(ctx context.Context, req ListNotificationChannelsRequest) (ListNotificationChannelsResponse, error) {
			result, err := h.service.ListChannels(ctx, req)
			if err != nil {
				return ListNotificationChannelsResponse{}, fmt.Errorf("failed to list notification channels: %w", err)
			}

			items := make([]api.NotificationChannel, 0, len(result.Items))
			for _, item := range result.Items {
				apiChannel, err := ToAPIChannel(item)
				if err != nil {
					return ListNotificationChannelsResponse{}, err
				}
				items = append(items, apiChannel)
			}

			return response.NewPagePaginationResponse(items, response.PageMetaPage{
				Size:   req.Page.PageSize,
				Number: req.Page.PageNumber,
				Total:  lo.ToPtr(result.TotalCount),
			}), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListNotificationChannelsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-notification-channels"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
