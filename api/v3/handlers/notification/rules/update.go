package rules

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	UpdateNotificationRuleRequest  = notification.UpdateRuleInput
	UpdateNotificationRuleResponse = apiv3models.BillingNotificationRule
	UpdateNotificationRuleParams   = string
	UpdateNotificationRuleHandler  = httptransport.HandlerWithArgs[UpdateNotificationRuleRequest, UpdateNotificationRuleResponse, UpdateNotificationRuleParams]
)

func (h *handler) UpdateNotificationRule() UpdateNotificationRuleHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, ruleID UpdateNotificationRuleParams) (UpdateNotificationRuleRequest, error) {
			body := apiv3models.BillingNotificationRuleRequest{}
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
			rule, err := h.service.UpdateRuleView(ctx, req)
			if err != nil {
				return UpdateNotificationRuleResponse{}, err
			}

			return ToAPIBillingNotificationRule(rule)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateNotificationRuleResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-billing-notification-rule"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
