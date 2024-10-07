package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type (
	ListChannelsRequest  = notification.ListChannelsInput
	ListChannelsResponse = api.NotificationChannelPaginatedResponse
	ListChannelsParams   = api.ListNotificationChannelsParams
	ListChannelsHandler  httptransport.HandlerWithArgs[ListChannelsRequest, ListChannelsResponse, ListChannelsParams]
)

func (h *handler) ListChannels() ListChannelsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListChannelsParams) (ListChannelsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListChannelsRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			req := ListChannelsRequest{
				Namespaces:      []string{ns},
				IncludeDisabled: defaultx.WithDefault(params.IncludeDisabled, notification.DefaultDisabled),
				OrderBy:         defaultx.WithDefault(params.OrderBy, api.NotificationChannelOrderById),
				Order:           sortx.Order(defaultx.WithDefault(params.Order, api.SortOrderDESC)),
				Page: pagination.Page{
					PageSize:   defaultx.WithDefault(params.PageSize, notification.DefaultPageSize),
					PageNumber: defaultx.WithDefault(params.Page, notification.DefaultPageNumber),
				},
			}

			return req, nil
		},
		func(ctx context.Context, request ListChannelsRequest) (ListChannelsResponse, error) {
			resp, err := h.service.ListChannels(ctx, request)
			if err != nil {
				return ListChannelsResponse{}, fmt.Errorf("failed to list channels: %w", err)
			}

			items := make([]api.NotificationChannel, 0, len(resp.Items))

			for _, channel := range resp.Items {
				var item api.NotificationChannel

				item, err = channel.AsNotificationChannel()
				if err != nil {
					return ListChannelsResponse{}, fmt.Errorf("failed to cast notification channel: %w", err)
				}

				items = append(items, item)
			}

			return ListChannelsResponse{
				Items:      items,
				Page:       resp.Page.PageNumber,
				PageSize:   resp.Page.PageSize,
				TotalCount: resp.TotalCount,
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListChannelsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listNotificationChannels"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	CreateChannelRequest  = notification.CreateChannelInput
	CreateChannelResponse = api.NotificationChannel
	CreateChannelHandler  httptransport.Handler[CreateChannelRequest, CreateChannelResponse]
)

func (h *handler) CreateChannel() CreateChannelHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateChannelRequest, error) {
			body := api.NotificationChannelCreateRequest{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CreateChannelRequest{}, fmt.Errorf("field to decode create channel request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateChannelRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			req := CreateChannelRequest{
				NamespacedModel: models.NamespacedModel{
					Namespace: ns,
				},
			}

			req = req.FromNotificationChannelWebhookCreateRequest(body)

			return req, nil
		},
		func(ctx context.Context, request CreateChannelRequest) (CreateChannelResponse, error) {
			channel, err := h.service.CreateChannel(ctx, request)
			if err != nil {
				return CreateChannelResponse{}, fmt.Errorf("failed to create channel: %w", err)
			}

			return channel.AsNotificationChannel()
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateChannelResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createNotificationChannel"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	UpdateChannelRequest  = notification.UpdateChannelInput
	UpdateChannelResponse = api.NotificationChannel
	UpdateChannelHandler  httptransport.HandlerWithArgs[UpdateChannelRequest, UpdateChannelResponse, string]
)

func (h *handler) UpdateChannel() UpdateChannelHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, channelID string) (UpdateChannelRequest, error) {
			body := api.NotificationChannelCreateRequest{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return UpdateChannelRequest{}, fmt.Errorf("field to decode update channel request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateChannelRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			req := UpdateChannelRequest{
				NamespacedModel: models.NamespacedModel{
					Namespace: ns,
				},
				ID: channelID,
			}

			req = req.FromNotificationChannelWebhookCreateRequest(body)

			return req, nil
		},
		func(ctx context.Context, request UpdateChannelRequest) (UpdateChannelResponse, error) {
			channel, err := h.service.UpdateChannel(ctx, request)
			if err != nil {
				return UpdateChannelResponse{}, fmt.Errorf("failed to update channel: %w", err)
			}

			return channel.AsNotificationChannel()
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateChannelResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("updateNotificationChannel"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	DeleteChannelRequest  = notification.DeleteChannelInput
	DeleteChannelResponse = interface{}
	DeleteChannelHandler  httptransport.HandlerWithArgs[DeleteChannelRequest, DeleteChannelResponse, string]
)

func (h *handler) DeleteChannel() DeleteChannelHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, channelID string) (DeleteChannelRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteChannelRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return DeleteChannelRequest{
				Namespace: ns,
				ID:        channelID,
			}, nil
		},
		func(ctx context.Context, request DeleteChannelRequest) (DeleteChannelResponse, error) {
			err := h.service.DeleteChannel(ctx, request)
			if err != nil {
				return nil, fmt.Errorf("failed to delete channel: %w", err)
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[DeleteChannelResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("deleteNotificationChannel"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	GetChannelRequest  = notification.GetChannelInput
	GetChannelResponse = api.NotificationChannel
	GetChannelHandler  httptransport.HandlerWithArgs[GetChannelRequest, GetChannelResponse, string]
)

func (h *handler) GetChannel() GetChannelHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, channelID string) (GetChannelRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetChannelRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return GetChannelRequest{
				Namespace: ns,
				ID:        channelID,
			}, nil
		},
		func(ctx context.Context, request GetChannelRequest) (GetChannelResponse, error) {
			channel, err := h.service.GetChannel(ctx, request)
			if err != nil {
				return GetChannelResponse{}, fmt.Errorf("failed to get channel: %w", err)
			}

			return channel.AsNotificationChannel()
		},
		commonhttp.JSONResponseEncoderWithStatus[GetChannelResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getNotificationChannel"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
