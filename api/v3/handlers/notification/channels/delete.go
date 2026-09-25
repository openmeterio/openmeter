package channels

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	DeleteNotificationChannelRequest  = notification.DeleteChannelInput
	DeleteNotificationChannelResponse = any
	DeleteNotificationChannelParams   = string
	DeleteNotificationChannelHandler  = httptransport.HandlerWithArgs[DeleteNotificationChannelRequest, DeleteNotificationChannelResponse, DeleteNotificationChannelParams]
)

// DeleteNotificationChannel returns a handler for deleting a notification channel.
func (h *handler) DeleteNotificationChannel() DeleteNotificationChannelHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, channelID DeleteNotificationChannelParams) (DeleteNotificationChannelRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteNotificationChannelRequest{}, err
			}

			return DeleteNotificationChannelRequest{
				Namespace: ns,
				ID:        channelID,
			}, nil
		},
		func(ctx context.Context, req DeleteNotificationChannelRequest) (DeleteNotificationChannelResponse, error) {
			if err := h.service.DeleteChannel(ctx, req); err != nil {
				return nil, err
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[DeleteNotificationChannelResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("delete-notification-channel"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
