package rules

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
	CreateNotificationRuleRequest  = notification.CreateRuleInput
	CreateNotificationRuleResponse = api.NotificationRule
	CreateNotificationRuleHandler  = httptransport.Handler[CreateNotificationRuleRequest, CreateNotificationRuleResponse]
)

func (h *handler) CreateNotificationRule() CreateNotificationRuleHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateNotificationRuleRequest, error) {
			body := api.NotificationRuleRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return CreateNotificationRuleRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateNotificationRuleRequest{}, err
			}

			return FromAPICreateRuleRequest(ns, body)
		},
		func(ctx context.Context, req CreateNotificationRuleRequest) (CreateNotificationRuleResponse, error) {
			rule, err := h.service.CreateRule(ctx, req)
			if err != nil {
				return CreateNotificationRuleResponse{}, err
			}

			if rule == nil {
				return CreateNotificationRuleResponse{}, fmt.Errorf("failed to create notification rule: nil rule returned")
			}

			return ToAPIRule(*rule)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateNotificationRuleResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-notification-rule"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
