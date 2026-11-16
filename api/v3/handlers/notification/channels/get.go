package channels

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	GetNotificationChannelRequest  = notification.GetChannelInput
	GetNotificationChannelResponse = apiv3models.BillingNotificationChannel
	GetNotificationChannelParams   = string
	GetNotificationChannelHandler  = httptransport.HandlerWithArgs[GetNotificationChannelRequest, GetNotificationChannelResponse, GetNotificationChannelParams]
)

// GetNotificationChannel returns a handler for getting a notification channel.
func (h *handler) GetNotificationChannel() GetNotificationChannelHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, channelID GetNotificationChannelParams) (GetNotificationChannelRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetNotificationChannelRequest{}, err
			}

			return GetNotificationChannelRequest{
				Namespace: ns,
				ID:        channelID,
			}, nil
		},
		func(ctx context.Context, req GetNotificationChannelRequest) (GetNotificationChannelResponse, error) {
			channel, err := h.service.GetChannel(ctx, req)
			if err != nil {
				return GetNotificationChannelResponse{}, err
			}

			if channel == nil {
				return GetNotificationChannelResponse{}, fmt.Errorf("failed to get notification channel: nil channel returned")
			}

			return ToAPIChannel(*channel)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetNotificationChannelResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-billing-notification-channel"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
