package rules

import (
	"context"
	"fmt"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/handlers/notification/events"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/testevent"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	TestNotificationRuleRequest  = notification.GetRuleInput
	TestNotificationRuleResponse = api.NotificationEvent
	TestNotificationRuleParams   = string
	TestNotificationRuleHandler  = httptransport.HandlerWithArgs[TestNotificationRuleRequest, TestNotificationRuleResponse, TestNotificationRuleParams]
)

// TestNotificationRule generates an event with sample data for the rule and persists
// it like any other event, annotated as a test event. Delivery to the rule's channels
// then happens through the regular delivery path.
func (h *handler) TestNotificationRule() TestNotificationRuleHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, ruleID TestNotificationRuleParams) (TestNotificationRuleRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return TestNotificationRuleRequest{}, err
			}

			return TestNotificationRuleRequest{
				Namespace: ns,
				ID:        ruleID,
			}, nil
		},
		func(ctx context.Context, req TestNotificationRuleRequest) (TestNotificationRuleResponse, error) {
			rule, err := h.service.GetRule(ctx, req)
			if err != nil {
				return TestNotificationRuleResponse{}, err
			}

			if rule == nil {
				return TestNotificationRuleResponse{}, fmt.Errorf("failed to get notification rule: nil rule returned")
			}

			payload, err := h.testEventGenerator.Generate(ctx, testevent.GeneratorInput{
				Namespace: req.Namespace,
				EventType: rule.Type,
			})
			if err != nil {
				return TestNotificationRuleResponse{}, fmt.Errorf("failed to generate test event: %w", err)
			}

			event, err := h.service.CreateEvent(ctx, notification.CreateEventInput{
				NamespacedModel: models.NamespacedModel{Namespace: req.Namespace},
				Type:            rule.Type,
				Payload:         payload,
				RuleID:          rule.ID,
				Annotations: models.Annotations{
					notification.AnnotationRuleTestEvent: true,
				},
			})
			if err != nil {
				return TestNotificationRuleResponse{}, fmt.Errorf("failed to create test event: %w", err)
			}

			if event == nil {
				return TestNotificationRuleResponse{}, fmt.Errorf("failed to create test event: nil event returned")
			}

			return events.ToAPIEvent(*event)
		},
		commonhttp.JSONResponseEncoderWithStatus[TestNotificationRuleResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("test-notification-rule"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
