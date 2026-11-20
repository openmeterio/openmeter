package eventtypes

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	v4 "github.com/openmeterio/openmeter/openmeter/notification/v4"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	GetNotificationEventTypeRequest  = v4.EventType
	GetNotificationEventTypeResponse = apiv3models.BillingNotificationEventTypeDefinition
	GetNotificationEventTypeParams   = string
	GetNotificationEventTypeHandler  = httptransport.HandlerWithArgs[GetNotificationEventTypeRequest, GetNotificationEventTypeResponse, GetNotificationEventTypeParams]
)

func (h *handler) GetNotificationEventType() GetNotificationEventTypeHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, eventType GetNotificationEventTypeParams) (GetNotificationEventTypeRequest, error) {
			t, err := v4.ParseEventType(eventType)
			if err != nil {
				return GetNotificationEventTypeRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{Field: "notificationEventType", Reason: err.Error(), Source: apierrors.InvalidParamSourcePath},
				})
			}

			return t, nil
		},
		func(ctx context.Context, t GetNotificationEventTypeRequest) (GetNotificationEventTypeResponse, error) {
			definition, err := h.registry.EventType(t)
			if err != nil {
				if models.IsGenericNotFoundError(err) {
					return GetNotificationEventTypeResponse{}, apierrors.NewNotFoundError(ctx, err, "notification event type")
				}

				return GetNotificationEventTypeResponse{}, err
			}

			return ToAPIBillingNotificationEventTypeDefinition(definition)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetNotificationEventTypeResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-billing-notification-event-type"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
