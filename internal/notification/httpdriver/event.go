package httpdriver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/notification"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	ListEventsRequest  = notification.ListEventsInput
	ListEventsResponse = api.NotificationEventsResponse
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
				OrderBy:    defaultx.WithDefault(params.OrderBy, notification.EventOrderByID),
				Page: pagination.Page{
					PageSize:   defaultx.WithDefault(params.PageSize, notification.DefaultPageSize),
					PageNumber: defaultx.WithDefault(params.Page, notification.DefaultPageNumber),
				},
				Subjects: defaultx.WithDefault(params.Subject, nil),
				Features: defaultx.WithDefault(params.Feature, nil),
				From:     defaultx.WithDefault(params.From, time.Time{}),
				To:       defaultx.WithDefault(params.To, time.Time{}),
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

				item, err = event.AsNotificationEvent()
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
	GetEventHandler  httptransport.HandlerWithArgs[GetEventRequest, GetEventResponse, api.EventId]
)

func (h *handler) GetEvent() GetEventHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, eventID api.EventId) (GetEventRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetEventRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			req := GetEventRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        eventID,
				},
			}

			return req, nil
		},
		func(ctx context.Context, request GetEventRequest) (GetEventResponse, error) {
			event, err := h.service.GetEvent(ctx, request)
			if err != nil {
				return GetEventResponse{}, fmt.Errorf("failed to get event: %w", err)
			}

			return event.AsNotificationEvent()
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
	CreateEventRequest  = notification.CreateEventInput
	CreateEventResponse = api.NotificationEvent
	CreateEventHandler  httptransport.Handler[CreateEventRequest, CreateEventResponse]
)

func (h *handler) CreateEvent() CreateEventHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateEventRequest, error) {
			body := api.NotificationEventCreateRequest{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CreateEventRequest{}, fmt.Errorf("field to decode create channel request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateEventRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			payloadValue, err := body.Payload.ValueByDiscriminator()
			if err != nil {
				return CreateEventRequest{}, notification.ValidationError{
					Err: err,
				}
			}

			payload := notification.EventPayload{}
			switch v := payloadValue.(type) {
			case api.NotificationEventBalanceThresholdPayload:
				payload = payload.FromNotificationEventBalanceThresholdPayload(v)
				if err != nil {
					return CreateEventRequest{}, fmt.Errorf("failed to unmarshal payload as BalanceThresholdPayload: %w", err)
				}

			default:
				return CreateEventRequest{}, fmt.Errorf("unknown event type: %s", body.Type)

			}

			req := CreateEventRequest{
				NamespacedModel: models.NamespacedModel{
					Namespace: ns,
				},
				Type:    notification.EventType(body.Type),
				Payload: payload,
				RuleID:  body.RuleId,
			}

			return req, nil
		},
		func(ctx context.Context, request CreateEventRequest) (CreateEventResponse, error) {
			event, err := h.service.CreateEvent(ctx, request)
			if err != nil {
				return CreateEventResponse{}, fmt.Errorf("failed to get event: %w", err)
			}

			return event.AsNotificationEvent()
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateEventResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createNotificationEvent"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
