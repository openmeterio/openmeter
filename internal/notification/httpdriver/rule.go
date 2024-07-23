package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/notification"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type (
	ListRulesRequest  = notification.ListRulesInput
	ListRulesResponse = api.NotificationRulesResponse
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
				IncludeDisabled: defaultx.WithDefault(params.IncludeDisabled, notification.DefaultDisabled),
				OrderBy:         notification.RuleOrderBy(defaultx.WithDefault(params.OrderBy, api.ListNotificationRulesParamsOrderById)),
				Order:           sortx.Order(defaultx.WithDefault(params.Order, api.SortOrderASC)),
				Page: pagination.Page{
					PageSize:   defaultx.WithDefault(params.PageSize, notification.DefaultPageSize),
					PageNumber: defaultx.WithDefault(params.Page, notification.DefaultPageNumber),
				},
			}

			return req, nil
		},
		func(ctx context.Context, request ListRulesRequest) (ListRulesResponse, error) {
			resp, err := h.connector.ListRules(ctx, request)
			if err != nil {
				return ListRulesResponse{}, fmt.Errorf("failed to list rules: %w", err)
			}

			items := make([]api.NotificationRule, 0, len(resp.Items))

			for _, rule := range resp.Items {
				var item CreateRuleResponse

				item, err = rule.AsNotificationRule()
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
				return CreateRuleRequest{}, notification.ValidationError{
					Err: err,
				}
			}

			req := CreateRuleRequest{
				NamespacedModel: models.NamespacedModel{
					Namespace: ns,
				},
			}

			switch v := value.(type) {
			case api.NotificationRuleBalanceThresholdCreateRequest:
				req = req.FromNotificationRuleBalanceThresholdCreateRequest(v)
			default:
				return CreateRuleRequest{}, notification.ValidationError{
					Err: fmt.Errorf("invalid channel type: %T", v),
				}
			}

			return req, nil
		},
		func(ctx context.Context, request CreateRuleRequest) (CreateRuleResponse, error) {
			rule, err := h.connector.CreateRule(ctx, request)
			if err != nil {
				return CreateRuleResponse{}, fmt.Errorf("failed to create rule: %w", err)
			}

			return rule.AsNotificationRule()
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
	UpdateRuleHandler  httptransport.HandlerWithArgs[UpdateRuleRequest, UpdateRuleResponse, api.RuleId]
)

func (h *handler) UpdateRule() UpdateRuleHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, ruleID api.RuleId) (UpdateRuleRequest, error) {
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
				return UpdateRuleRequest{}, notification.ValidationError{
					Err: err,
				}
			}

			req := UpdateRuleRequest{
				NamespacedModel: models.NamespacedModel{
					Namespace: ns,
				},
				ID: ruleID,
			}

			switch v := value.(type) {
			case api.NotificationRuleBalanceThresholdCreateRequest:
				req = req.FromNotificationRuleBalanceThresholdCreateRequest(v)
			default:
				return UpdateRuleRequest{}, notification.ValidationError{
					Err: fmt.Errorf("invalid rule type: %T", v),
				}
			}

			return req, nil
		},
		func(ctx context.Context, request UpdateRuleRequest) (UpdateRuleResponse, error) {
			rule, err := h.connector.UpdateRule(ctx, request)
			if err != nil {
				return UpdateRuleResponse{}, fmt.Errorf("failed to update rule: %w", err)
			}

			return rule.AsNotificationRule()
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
	DeleteRuleHandler  httptransport.HandlerWithArgs[DeleteRuleRequest, DeleteRuleResponse, api.RuleId]
)

func (h *handler) DeleteRule() DeleteRuleHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, ruleID api.RuleId) (DeleteRuleRequest, error) {
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
			err := h.connector.DeleteRule(ctx, request)
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
	GetRuleHandler  httptransport.HandlerWithArgs[GetRuleRequest, GetRuleResponse, api.RuleId]
)

func (h *handler) GetRule() GetRuleHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, ruleID api.RuleId) (GetRuleRequest, error) {
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
			rule, err := h.connector.GetRule(ctx, request)
			if err != nil {
				return GetRuleResponse{}, fmt.Errorf("failed to get rule: %w", err)
			}

			return rule.AsNotificationRule()
		},
		commonhttp.JSONResponseEncoderWithStatus[GetRuleResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getNotificationRule"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
