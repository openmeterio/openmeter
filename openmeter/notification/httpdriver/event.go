package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type (
	ListEventsRequest  = notification.ListEventsInput
	ListEventsResponse = api.NotificationEventPaginatedResponse
	ListEventsParams   = api.ListNotificationEventsParams
	ListEventsHandler  httptransport.HandlerWithArgs[ListEventsRequest, ListEventsResponse, ListEventsParams]
)

// inFilter converts a legacy repeated query parameter into a set-membership filter.
// An absent or empty parameter yields nil so that no predicate is applied at all —
// an empty In list would otherwise match nothing. The values are cloned so the filter
// does not alias the request's parsed query parameters.
func inFilter(values []string) *filter.FilterString {
	if len(values) == 0 {
		return nil
	}

	return &filter.FilterString{In: lo.ToPtr(slices.Clone(values))}
}

func (h *handler) ListEvents() ListEventsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListEventsParams) (ListEventsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListEventsRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			// ListEventsInput no longer range-checks the period, so keep the v1 error
			// here rather than degrading an inverted period to an empty result set.
			if params.From != nil && params.To != nil && params.From.After(*params.To) {
				return ListEventsRequest{}, models.NewGenericValidationError(
					fmt.Errorf("invalid time period: period start (%s) is after the period end (%s)", *params.From, *params.To),
				)
			}

			req := ListEventsRequest{
				Namespaces: []string{ns},
				Order:      sortx.Order(lo.FromPtrOr(params.Order, api.SortOrderDESC)),
				OrderBy:    notification.OrderBy(lo.FromPtrOr(params.OrderBy, api.NotificationEventOrderByCreatedAt)),
				Page: pagination.Page{
					PageSize:   lo.FromPtrOr(params.PageSize, notification.DefaultPageSize),
					PageNumber: lo.FromPtrOr(params.Page, notification.DefaultPageNumber),
				},
				CreatedAt: filter.NewFilterTime(params.From, params.To),
				Subject:   inFilter(lo.FromPtr(params.Subject)),
				Feature:   inFilter(lo.FromPtr(params.Feature)),
				ChannelID: inFilter(lo.FromPtr(params.Channel)),
			}

			if rules := lo.FromPtr(params.Rule); len(rules) > 0 {
				req.RuleID = &filter.FilterULID{FilterString: *inFilter(rules)}
			}

			return req, nil
		},
		func(ctx context.Context, request ListEventsRequest) (ListEventsResponse, error) {
			resp, err := h.service.ListEvents(ctx, request)
			if err != nil {
				return ListEventsResponse{}, fmt.Errorf("failed to list events: %w", err)
			}

			items := make([]api.NotificationEvent, 0, len(resp.Items))

			for _, event := range resp.Items {
				var item api.NotificationEvent

				item, err = FromEvent(event)
				if err != nil {
					return ListEventsResponse{}, fmt.Errorf("failed to cast event: %w", err)
				}

				items = append(items, item)
			}

			return ListEventsResponse{
				Items:      items,
				Page:       resp.Page.PageNumber,
				PageSize:   resp.Page.PageSize,
				TotalCount: resp.TotalCount,
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListEventsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listNotificationEvents"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	GetEventRequest  = notification.GetEventInput
	GetEventResponse = api.NotificationEvent
	GetEventHandler  httptransport.HandlerWithArgs[GetEventRequest, GetEventResponse, string]
)

func (h *handler) GetEvent() GetEventHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, eventID string) (GetEventRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetEventRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			req := GetEventRequest{
				Namespace: ns,
				ID:        eventID,
			}

			return req, nil
		},
		func(ctx context.Context, request GetEventRequest) (GetEventResponse, error) {
			event, err := h.service.GetEvent(ctx, request)
			if err != nil {
				return GetEventResponse{}, fmt.Errorf("failed to get event: %w", err)
			}

			if event == nil {
				return GetEventResponse{}, errors.New("failed to create test event: nil event returned")
			}

			return FromEvent(*event)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetEventResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getNotificationEvent"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	ResendEventRequest  = notification.ResendEventInput
	ResendEventResponse = interface{}
	ResendEventHandler  httptransport.HandlerWithArgs[ResendEventRequest, ResendEventResponse, string]
)

func (h *handler) ResendEvent() ResendEventHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, eventID string) (ResendEventRequest, error) {
			body := api.NotificationEventResendRequest{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return ResendEventRequest{}, fmt.Errorf("field to decode resend event request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ResendEventRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			req := ResendEventRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        eventID,
				},
				Channels: lo.FromPtr(body.Channels),
			}

			return req, nil
		},
		func(ctx context.Context, request ResendEventRequest) (ResendEventResponse, error) {
			err := h.service.ResendEvent(ctx, request)
			if err != nil {
				return nil, fmt.Errorf("failed to resend event: %w", err)
			}
			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[ResendEventResponse](http.StatusAccepted),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("resendNotificationEvent"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
