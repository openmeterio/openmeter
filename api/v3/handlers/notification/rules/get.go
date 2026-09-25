package rules

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
	GetNotificationRuleRequest  = notification.GetRuleInput
	GetNotificationRuleResponse = api.NotificationRule
	GetNotificationRuleParams   = string
	GetNotificationRuleHandler  = httptransport.HandlerWithArgs[GetNotificationRuleRequest, GetNotificationRuleResponse, GetNotificationRuleParams]
)

func (h *handler) GetNotificationRule() GetNotificationRuleHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, ruleID GetNotificationRuleParams) (GetNotificationRuleRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetNotificationRuleRequest{}, err
			}

			return GetNotificationRuleRequest{
				Namespace: ns,
				ID:        ruleID,
			}, nil
		},
		func(ctx context.Context, req GetNotificationRuleRequest) (GetNotificationRuleResponse, error) {
			rule, err := h.service.GetRule(ctx, req)
			if err != nil {
				return GetNotificationRuleResponse{}, err
			}

			if rule == nil {
				return GetNotificationRuleResponse{}, fmt.Errorf("failed to get notification rule: nil rule returned")
			}

			return ToAPIRule(*rule)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetNotificationRuleResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-notification-rule"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
