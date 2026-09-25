package rules

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
	ListNotificationRulesRequest  = notification.ListRulesInput
	ListNotificationRulesResponse = response.PagePaginationResponse[api.NotificationRule]
	ListNotificationRulesParams   = api.ListNotificationRulesParams
	ListNotificationRulesHandler  = httptransport.HandlerWithArgs[ListNotificationRulesRequest, ListNotificationRulesResponse, ListNotificationRulesParams]
)

func (h *handler) ListNotificationRules() ListNotificationRulesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListNotificationRulesParams) (ListNotificationRulesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListNotificationRulesRequest{}, err
			}

			page := pagination.NewPage(1, 20)
			if params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Page.Number, 1),
					lo.FromPtrOr(params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListNotificationRulesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Field:  "page",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}

			req := ListNotificationRulesRequest{
				Namespaces: []string{ns},
				Page:       page,
			}

			if params.Sort != nil {
				sort, err := request.ParseSortBy(*params.Sort)
				if err != nil {
					return ListNotificationRulesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						apierrors.InvalidParameter{
							Field:  "sort",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceQuery,
						},
					})
				}

				orderBy, err := FromAPIRuleSortField(ctx, sort.Field)
				if err != nil {
					return ListNotificationRulesRequest{}, err
				}

				req.OrderBy = orderBy
				req.Order = sort.Order.ToSortxOrder()
			}

			if params.Filter != nil {
				if err := applyAPIRuleFilters(ctx, &req, *params.Filter); err != nil {
					return ListNotificationRulesRequest{}, err
				}
			}

			return req, nil
		},
		func(ctx context.Context, req ListNotificationRulesRequest) (ListNotificationRulesResponse, error) {
			result, err := h.service.ListRules(ctx, req)
			if err != nil {
				return ListNotificationRulesResponse{}, fmt.Errorf("failed to list notification rules: %w", err)
			}

			items := make([]api.NotificationRule, 0, len(result.Items))
			for _, item := range result.Items {
				apiRule, err := ToAPIRule(item)
				if err != nil {
					return ListNotificationRulesResponse{}, err
				}

				items = append(items, apiRule)
			}

			return response.NewPagePaginationResponse(items, response.PageMetaPage{
				Size:   req.Page.PageSize,
				Number: req.Page.PageNumber,
				Total:  lo.ToPtr(result.TotalCount),
			}), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListNotificationRulesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-notification-rules"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

// applyAPIRuleFilters converts the deepObject filter parameters into the domain
// predicates the service expects. Every failure is reported against the query
// parameter that caused it so the client can tell which filter was rejected.
func applyAPIRuleFilters(ctx context.Context, req *ListNotificationRulesRequest, params api.ListNotificationRulesParamsFilter) error {
	badRequest := func(field string, err error) error {
		return apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
			{Field: field, Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
		})
	}

	id, err := filters.FromAPIFilterULID(params.Id)
	if err != nil {
		return badRequest("filter[id]", err)
	}
	req.ID = id

	name, err := filters.FromAPIFilterString(params.Name)
	if err != nil {
		return badRequest("filter[name]", err)
	}
	req.Name = name

	typeFilter, err := filters.FromAPIFilterStringExact(params.Type)
	if err != nil {
		return badRequest("filter[type]", err)
	}

	typeFilter, err = filters.MapValues(typeFilter, func(v string) (string, error) {
		return v, notification.EventType(v).Validate()
	})
	if err != nil {
		return badRequest("filter[type]", err)
	}
	req.Type = typeFilter

	disabled, err := filters.FromAPIFilterBoolean(params.Disabled)
	if err != nil {
		return badRequest("filter[disabled]", err)
	}
	req.Disabled = disabled

	createdAt, err := filters.FromAPIFilterDateTime(params.CreatedAt)
	if err != nil {
		return badRequest("filter[created_at]", err)
	}
	req.CreatedAt = createdAt

	updatedAt, err := filters.FromAPIFilterDateTime(params.UpdatedAt)
	if err != nil {
		return badRequest("filter[updated_at]", err)
	}
	req.UpdatedAt = updatedAt

	channelID, err := filters.FromAPIFilterULID(params.ChannelId)
	if err != nil {
		return badRequest("filter[channel_id]", err)
	}

	if channelID != nil {
		if err := filters.RequireExact("channel_id", &channelID.FilterString); err != nil {
			return badRequest("filter[channel_id]", err)
		}

		req.ChannelID = &channelID.FilterString
	}

	return nil
}
