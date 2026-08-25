package events

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
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type (
	ListNotificationEventsRequest  = notification.ListEventsInput
	ListNotificationEventsResponse = response.PagePaginationResponse[api.NotificationEvent]
	ListNotificationEventsParams   = api.ListNotificationEventsParams
	ListNotificationEventsHandler  = httptransport.HandlerWithArgs[ListNotificationEventsRequest, ListNotificationEventsResponse, ListNotificationEventsParams]
)

func (h *handler) ListNotificationEvents() ListNotificationEventsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListNotificationEventsParams) (ListNotificationEventsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListNotificationEventsRequest{}, err
			}

			page := pagination.NewPage(1, 20)
			if params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Page.Number, 1),
					lo.FromPtrOr(params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListNotificationEventsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Field:  "page",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}

			req := ListNotificationEventsRequest{
				Namespaces: []string{ns},
				Page:       page,
				// The adapter falls back to ordering by id, but v1 lists events newest
				// first. Set the default explicitly so the two APIs agree.
				OrderBy: notification.OrderByCreatedAt,
				Order:   sortx.OrderDesc,
			}

			if params.Sort != nil {
				sort, err := request.ParseSortBy(*params.Sort)
				if err != nil {
					return ListNotificationEventsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						apierrors.InvalidParameter{
							Field:  "sort",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceQuery,
						},
					})
				}

				orderBy, err := FromAPIEventSortField(ctx, sort.Field)
				if err != nil {
					return ListNotificationEventsRequest{}, err
				}
				req.OrderBy = orderBy
				req.Order = sort.Order.ToSortxOrder()
			}

			if params.Filter != nil {
				if err := applyAPIEventFilters(ctx, &req, *params.Filter); err != nil {
					return ListNotificationEventsRequest{}, err
				}
			}

			return req, nil
		},
		func(ctx context.Context, req ListNotificationEventsRequest) (ListNotificationEventsResponse, error) {
			result, err := h.service.ListEvents(ctx, req)
			if err != nil {
				return ListNotificationEventsResponse{}, fmt.Errorf("failed to list notification events: %w", err)
			}

			items := make([]api.NotificationEvent, 0, len(result.Items))
			for _, item := range result.Items {
				apiEvent, err := ToAPIEvent(item)
				if err != nil {
					return ListNotificationEventsResponse{}, err
				}
				items = append(items, apiEvent)
			}

			return response.NewPagePaginationResponse(items, response.PageMetaPage{
				Size:   req.Page.PageSize,
				Number: req.Page.PageNumber,
				Total:  lo.ToPtr(result.TotalCount),
			}), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListNotificationEventsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-notification-events"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

// applyAPIEventFilters converts the deepObject filter parameters into the domain
// predicates the service expects. Every failure is reported against the query parameter
// that caused it so the client can tell which filter was rejected.
func applyAPIEventFilters(ctx context.Context, req *ListNotificationEventsRequest, params api.ListNotificationEventsParamsFilter) error {
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

	typeFilter, err := filters.FromAPIFilterStringExact(params.Type)
	if err != nil {
		return badRequest("filter[type]", err)
	}
	// The column stores the dotted domain value ("invoice.created"), not the snake_case
	// wire value, so translate before this reaches the adapter.
	typeFilter, err = mapAPIEnumFilter(typeFilter, func(v string) (string, error) {
		domain, err := ToDomainEventType(api.NotificationEventType(v))
		return string(domain), err
	})
	if err != nil {
		return badRequest("filter[type]", err)
	}
	req.Type = typeFilter

	createdAt, err := filters.FromAPIFilterDateTime(params.CreatedAt)
	if err != nil {
		return badRequest("filter[created_at]", err)
	}
	req.CreatedAt = createdAt

	ruleID, err := filters.FromAPIFilterULID(params.RuleId)
	if err != nil {
		return badRequest("filter[rule_id]", err)
	}
	req.RuleID = ruleID

	channelID, err := filters.FromAPIFilterULID(params.ChannelId)
	if err != nil {
		return badRequest("filter[channel_id]", err)
	}
	if channelID != nil {
		if err := requireExactFilter("channel_id", &channelID.FilterString); err != nil {
			return badRequest("filter[channel_id]", err)
		}
		req.ChannelID = &channelID.FilterString
	}

	deliveryStatus, err := filters.FromAPIFilterStringExact(params.DeliveryStatus)
	if err != nil {
		return badRequest("filter[delivery_status]", err)
	}
	if err := requireExactFilter("delivery_status", deliveryStatus); err != nil {
		return badRequest("filter[delivery_status]", err)
	}
	// The column stores the uppercase domain state ("FAILED"), not the wire value.
	deliveryStatus, err = mapAPIEnumFilter(deliveryStatus, func(v string) (string, error) {
		domain, err := ToDomainDeliveryState(api.NotificationEventDeliveryState(v))
		return string(domain), err
	})
	if err != nil {
		return badRequest("filter[delivery_status]", err)
	}
	req.DeliveryStatus = deliveryStatus

	subjectKey, err := filters.FromAPIFilterStringExact(params.SubjectKey)
	if err != nil {
		return badRequest("filter[subject_key]", err)
	}
	req.SubjectKey = subjectKey

	subjectID, err := filters.FromAPIFilterULID(params.SubjectId)
	if err != nil {
		return badRequest("filter[subject_id]", err)
	}
	req.SubjectID = subjectID

	featureKey, err := filters.FromAPIFilterStringExact(params.FeatureKey)
	if err != nil {
		return badRequest("filter[feature_key]", err)
	}
	req.FeatureKey = featureKey

	featureID, err := filters.FromAPIFilterULID(params.FeatureId)
	if err != nil {
		return badRequest("filter[feature_id]", err)
	}
	req.FeatureID = featureID

	return nil
}
