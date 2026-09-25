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
	UpdateNotificationRuleRequest  = notification.UpdateRuleInput
	UpdateNotificationRuleResponse = api.NotificationRule
	UpdateNotificationRuleParams   = string
	UpdateNotificationRuleHandler  = httptransport.HandlerWithArgs[UpdateNotificationRuleRequest, UpdateNotificationRuleResponse, UpdateNotificationRuleParams]
)

func (h *handler) UpdateNotificationRule() UpdateNotificationRuleHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, ruleID UpdateNotificationRuleParams) (UpdateNotificationRuleRequest, error) {
			body := api.NotificationRuleRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return UpdateNotificationRuleRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateNotificationRuleRequest{}, err
			}

			return FromAPIUpdateRuleRequest(ns, ruleID, body)
		},
		func(ctx context.Context, req UpdateNotificationRuleRequest) (UpdateNotificationRuleResponse, error) {
			rule, err := h.service.UpdateRule(ctx, req)
			if err != nil {
				return UpdateNotificationRuleResponse{}, err
			}

			if rule == nil {
				return UpdateNotificationRuleResponse{}, fmt.Errorf("failed to update notification rule: nil rule returned")
			}

			return ToAPIRule(*rule)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateNotificationRuleResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-notification-rule"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
