package events

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	ResendNotificationEventRequest  = notification.ResendEventInput
	ResendNotificationEventResponse = any
	ResendNotificationEventParams   = string
	ResendNotificationEventHandler  = httptransport.HandlerWithArgs[ResendNotificationEventRequest, ResendNotificationEventResponse, ResendNotificationEventParams]
)

// ResendNotificationEvent returns a handler for re-sending a notification event. The
// service only marks the matching delivery statuses for re-delivery; the delivery worker
// picks them up afterwards, so the response is a bodyless 202.
func (h *handler) ResendNotificationEvent() ResendNotificationEventHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, eventID ResendNotificationEventParams) (ResendNotificationEventRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ResendNotificationEventRequest{}, err
			}

			body := api.ResendNotificationEventRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return ResendNotificationEventRequest{}, err
			}

			return ResendNotificationEventRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        eventID,
				},
				Channels: lo.FromPtr(body.Channels),
			}, nil
		},
		func(ctx context.Context, req ResendNotificationEventRequest) (ResendNotificationEventResponse, error) {
			return nil, h.service.ResendEvent(ctx, req)
		},
		commonhttp.EmptyResponseEncoder[ResendNotificationEventResponse](http.StatusAccepted),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("resend-notification-event"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
