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
	UpdateNotificationChannelRequest  = notification.UpdateChannelInput
	UpdateNotificationChannelResponse = api.NotificationChannel
	UpdateNotificationChannelParams   = string
	UpdateNotificationChannelHandler  = httptransport.HandlerWithArgs[UpdateNotificationChannelRequest, UpdateNotificationChannelResponse, UpdateNotificationChannelParams]
)

// UpdateNotificationChannel returns a handler for updating a notification channel.
func (h *handler) UpdateNotificationChannel() UpdateNotificationChannelHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, channelID UpdateNotificationChannelParams) (UpdateNotificationChannelRequest, error) {
			body := api.UpdateNotificationChannelRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return UpdateNotificationChannelRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateNotificationChannelRequest{}, err
			}

			return FromAPIUpdateChannelRequest(ns, channelID, body)
		},
		func(ctx context.Context, req UpdateNotificationChannelRequest) (UpdateNotificationChannelResponse, error) {
			channel, err := h.service.UpdateChannel(ctx, req)
			if err != nil {
				return UpdateNotificationChannelResponse{}, err
			}

			if channel == nil {
				return UpdateNotificationChannelResponse{}, fmt.Errorf("failed to update notification channel: nil channel returned")
			}

			return ToAPIChannel(*channel)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateNotificationChannelResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-notification-channel"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
