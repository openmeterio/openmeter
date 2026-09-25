package channels

import (
	"context"
	"fmt"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	CreateNotificationChannelRequest  = notification.CreateChannelInput
	CreateNotificationChannelResponse = api.NotificationChannel
	CreateNotificationChannelHandler  = httptransport.Handler[CreateNotificationChannelRequest, CreateNotificationChannelResponse]
)

// CreateNotificationChannel returns a handler for creating a notification channel.
func (h *handler) CreateNotificationChannel() CreateNotificationChannelHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateNotificationChannelRequest, error) {
			body := api.CreateNotificationChannelRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return CreateNotificationChannelRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateNotificationChannelRequest{}, err
			}

			return FromAPICreateChannelRequest(ns, body)
		},
		func(ctx context.Context, req CreateNotificationChannelRequest) (CreateNotificationChannelResponse, error) {
			channel, err := h.service.CreateChannel(ctx, req)
			if err != nil {
				return CreateNotificationChannelResponse{}, fmt.Errorf("failed to create notification channel: %w", err)
			}

			if channel == nil {
				return CreateNotificationChannelResponse{}, fmt.Errorf("failed to create notification channel: nil channel returned")
			}

			return ToAPIChannel(*channel)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateNotificationChannelResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-notification-channel"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
