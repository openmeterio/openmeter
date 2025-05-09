package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/internal"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type (
	ListRulesRequest  = notification.ListRulesInput
	ListRulesResponse = api.NotificationRulePaginatedResponse
	ListRulesParams   = api.ListNotificationRulesParams
	ListRulesHandler  httptransport.HandlerWithArgs[ListRulesRequest, ListRulesResponse, ListRulesParams]
)

func (h *handler) ListRules() ListRulesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListRulesParams) (ListRulesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListRulesRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			req := ListRulesRequest{
				Namespaces:      []string{ns},
				IncludeDisabled: lo.FromPtrOr(params.IncludeDisabled, notification.DefaultDisabled),
				OrderBy:         notification.OrderBy(lo.FromPtrOr(params.OrderBy, api.NotificationRuleOrderById)),
				Order:           sortx.Order(lo.FromPtrOr(params.Order, api.SortOrderASC)),
				Page: pagination.Page{
					PageSize:   lo.FromPtrOr(params.PageSize, notification.DefaultPageSize),
					PageNumber: lo.FromPtrOr(params.Page, notification.DefaultPageNumber),
				},
			}

			return req, nil
		},
		func(ctx context.Context, request ListRulesRequest) (ListRulesResponse, error) {
			resp, err := h.service.ListRules(ctx, request)
			if err != nil {
				return ListRulesResponse{}, fmt.Errorf("failed to list rules: %w", err)
			}

			items := make([]api.NotificationRule, 0, len(resp.Items))

			for _, rule := range resp.Items {
				var item CreateRuleResponse

				item, err = FromRule(rule)
				if err != nil {
					return ListRulesResponse{}, fmt.Errorf("failed to cast rule to notification rule: %w", err)
				}

				items = append(items, item)
			}

			return ListRulesResponse{
				Items:      items,
				Page:       resp.Page.PageNumber,
				PageSize:   resp.Page.PageSize,
				TotalCount: resp.TotalCount,
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListRulesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listNotificationRules"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	CreateRuleRequest  = notification.CreateRuleInput
	CreateRuleResponse = api.NotificationRule
	CreateRuleHandler  httptransport.Handler[CreateRuleRequest, CreateRuleResponse]
)

func (h *handler) CreateRule() CreateRuleHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateRuleRequest, error) {
			body := api.NotificationRuleCreateRequest{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CreateRuleRequest{}, fmt.Errorf("field to decode create rule request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateRuleRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			value, err := body.ValueByDiscriminator()
			if err != nil {
				return CreateRuleRequest{}, fmt.Errorf("failed to get request type: %w", err)
			}

			var req CreateRuleRequest

			switch v := value.(type) {
			case api.NotificationRuleBalanceThresholdCreateRequest:
				req = AsRuleBalanceThresholdCreateRequest(v, ns)
			case api.NotificationRuleEntitlementResetCreateRequest:
				req = AsRuleEntitlementResetCreateRequest(v, ns)
			case api.NotificationRuleInvoiceCreatedCreateRequest:
				req = AsRuleInvoiceCreatedCreateRequest(v, ns)
			case api.NotificationRuleInvoiceUpdatedCreateRequest:
				req = AsRuleInvoiceUpdatedCreateRequest(v, ns)
			}

			return req, nil
		},
		func(ctx context.Context, request CreateRuleRequest) (CreateRuleResponse, error) {
			rule, err := h.service.CreateRule(ctx, request)
			if err != nil {
				return CreateRuleResponse{}, fmt.Errorf("failed to create rule: %w", err)
			}

			if rule == nil {
				return CreateRuleResponse{}, errors.New("failed to create rule: nil rule returned")
			}

			return FromRule(*rule)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateRuleResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createNotificationRule"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	UpdateRuleRequest  = notification.UpdateRuleInput
	UpdateRuleResponse = api.NotificationRule
	UpdateRuleHandler  httptransport.HandlerWithArgs[UpdateRuleRequest, UpdateRuleResponse, string]
)

func (h *handler) UpdateRule() UpdateRuleHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, ruleID string) (UpdateRuleRequest, error) {
			body := api.NotificationRuleCreateRequest{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return UpdateRuleRequest{}, fmt.Errorf("field to decode update rule request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateRuleRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			value, err := body.ValueByDiscriminator()
			if err != nil {
				return UpdateRuleRequest{}, fmt.Errorf("failed to get request type: %w", err)
			}

			var req UpdateRuleRequest

			switch v := value.(type) {
			case api.NotificationRuleBalanceThresholdCreateRequest:
				req = AsRuleBalanceThresholdUpdateRequest(v, ns, ruleID)
			case api.NotificationRuleEntitlementResetCreateRequest:
				req = AsRuleEntitlementResetUpdateRequest(v, ns, ruleID)
			case api.NotificationRuleInvoiceCreatedCreateRequest:
				req = AsRuleInvoiceCreatedUpdateRequest(v, ns, ruleID)
			case api.NotificationRuleInvoiceUpdatedCreateRequest:
				req = AsRuleInvoiceUpdatedUpdateRequest(v, ns, ruleID)
			}

			return req, nil
		},
		func(ctx context.Context, request UpdateRuleRequest) (UpdateRuleResponse, error) {
			rule, err := h.service.UpdateRule(ctx, request)
			if err != nil {
				return UpdateRuleResponse{}, fmt.Errorf("failed to update rule: %w", err)
			}

			if rule == nil {
				return CreateRuleResponse{}, errors.New("failed to create rule: nil rule returned")
			}

			return FromRule(*rule)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateRuleResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("updateNotificationRule"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	DeleteRuleRequest  = notification.DeleteRuleInput
	DeleteRuleResponse = interface{}
	DeleteRuleHandler  httptransport.HandlerWithArgs[DeleteRuleRequest, DeleteRuleResponse, string]
)

func (h *handler) DeleteRule() DeleteRuleHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, ruleID string) (DeleteRuleRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteRuleRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return DeleteRuleRequest{
				Namespace: ns,
				ID:        ruleID,
			}, nil
		},
		func(ctx context.Context, request DeleteRuleRequest) (DeleteRuleResponse, error) {
			err := h.service.DeleteRule(ctx, request)
			if err != nil {
				return nil, fmt.Errorf("failed to delete rule: %w", err)
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[DeleteChannelResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("deleteNotificationRule"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	GetRuleRequest  = notification.GetRuleInput
	GetRuleResponse = api.NotificationRule
	GetRuleHandler  httptransport.HandlerWithArgs[GetRuleRequest, GetRuleResponse, string]
)

func (h *handler) GetRule() GetRuleHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, ruleID string) (GetRuleRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetRuleRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return GetRuleRequest{
				Namespace: ns,
				ID:        ruleID,
			}, nil
		},
		func(ctx context.Context, request GetRuleRequest) (GetRuleResponse, error) {
			rule, err := h.service.GetRule(ctx, request)
			if err != nil {
				return GetRuleResponse{}, fmt.Errorf("failed to get rule: %w", err)
			}

			if rule == nil {
				return CreateRuleResponse{}, errors.New("failed to create rule: nil rule returned")
			}

			return FromRule(*rule)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetRuleResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getNotificationRule"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type TestRuleRequest struct {
	models.NamespacedModel

	// RuleID defines the notification Rule that generated this Event.
	RuleID string `json:"ruleId"`
}

type (
	TestRuleResponse = api.NotificationEvent
	TestRuleHandler  httptransport.HandlerWithArgs[TestRuleRequest, TestRuleResponse, string]
)

func (h *handler) TestRule() TestRuleHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, ruleID string) (TestRuleRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return TestRuleRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			req := TestRuleRequest{
				NamespacedModel: models.NamespacedModel{
					Namespace: ns,
				},
				RuleID: ruleID,
			}

			return req, nil
		},
		func(ctx context.Context, request TestRuleRequest) (TestRuleResponse, error) {
			rule, err := h.service.GetRule(ctx, notification.GetRuleInput{
				Namespace: request.NamespacedModel.Namespace,
				ID:        request.RuleID,
			})
			if err != nil {
				return TestRuleResponse{}, fmt.Errorf("failed to get rule: %w", err)
			}

			testEvent, err := h.testEventGenerator.Generate(ctx, internal.EventGeneratorInput{
				Namespace: request.Namespace,
				EventType: rule.Type,
			})
			if err != nil {
				return TestRuleResponse{}, fmt.Errorf("failed to generate test event: %w", err)
			}

			event, err := h.service.CreateEvent(ctx, notification.CreateEventInput{
				NamespacedModel: request.NamespacedModel,
				Type:            rule.Type,
				Payload:         testEvent,
				RuleID:          rule.ID,
				Annotations: models.Annotations{
					notification.AnnotationRuleTestEvent: true,
				},
			})
			if err != nil {
				return TestRuleResponse{}, fmt.Errorf("failed to create test event: %w", err)
			}

			if event == nil {
				return TestRuleResponse{}, errors.New("failed to create test event: nil event returned")
			}

			return FromEvent(*event)
		},
		commonhttp.JSONResponseEncoderWithStatus[TestRuleResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createNotificationEvent"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
