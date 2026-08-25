package events

import (
	"context"
	"fmt"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	GetNotificationEventRequest  = notification.GetEventInput
	GetNotificationEventResponse = api.NotificationEvent
	GetNotificationEventParams   = string
	GetNotificationEventHandler  = httptransport.HandlerWithArgs[GetNotificationEventRequest, GetNotificationEventResponse, GetNotificationEventParams]
)

// GetNotificationEvent returns a handler for getting a notification event.
func (h *handler) GetNotificationEvent() GetNotificationEventHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, eventID GetNotificationEventParams) (GetNotificationEventRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetNotificationEventRequest{}, err
			}

			return GetNotificationEventRequest{
				Namespace: ns,
				ID:        eventID,
			}, nil
		},
		func(ctx context.Context, req GetNotificationEventRequest) (GetNotificationEventResponse, error) {
			event, err := h.service.GetEvent(ctx, req)
			if err != nil {
				return GetNotificationEventResponse{}, err
			}

			if event == nil {
				return GetNotificationEventResponse{}, fmt.Errorf("failed to get notification event: nil event returned")
			}

			return ToAPIEvent(*event)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetNotificationEventResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-notification-event"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
