package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/notification"
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

func (h *handler) ListEvents() ListEventsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListEventsParams) (ListEventsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListEventsRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			req := ListEventsRequest{
				Namespaces: []string{ns},
				Order:      sortx.Order(lo.FromPtrOr(params.Order, api.SortOrderDESC)),
				OrderBy:    notification.OrderBy(lo.FromPtrOr(params.OrderBy, api.NotificationEventOrderByCreatedAt)),
				Page: pagination.Page{
					PageSize:   lo.FromPtrOr(params.PageSize, notification.DefaultPageSize),
					PageNumber: lo.FromPtrOr(params.Page, notification.DefaultPageNumber),
				},
				Subjects: lo.FromPtr(params.Subject),
				Features: lo.FromPtr(params.Feature),
				Rules:    lo.FromPtr(params.Rule),
				Channels: lo.FromPtr(params.Channel),
				From:     lo.FromPtr(params.From),
				To:       lo.FromPtr(params.To),
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
