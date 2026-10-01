package rules

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	DeleteNotificationRuleRequest  = notification.DeleteRuleInput
	DeleteNotificationRuleResponse = any
	DeleteNotificationRuleParams   = string
	DeleteNotificationRuleHandler  = httptransport.HandlerWithArgs[DeleteNotificationRuleRequest, DeleteNotificationRuleResponse, DeleteNotificationRuleParams]
)

func (h *handler) DeleteNotificationRule() DeleteNotificationRuleHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, ruleID DeleteNotificationRuleParams) (DeleteNotificationRuleRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteNotificationRuleRequest{}, err
			}

			return DeleteNotificationRuleRequest{
				Namespace: ns,
				ID:        ruleID,
			}, nil
		},
		func(ctx context.Context, req DeleteNotificationRuleRequest) (DeleteNotificationRuleResponse, error) {
			return nil, h.service.DeleteRule(ctx, req)
		},
		commonhttp.EmptyResponseEncoder[DeleteNotificationRuleResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("delete-notification-rule"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
