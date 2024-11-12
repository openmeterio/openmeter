package httpdriver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type (
	ListPlansRequest  = plan.ListPlansInput
	ListPlansResponse = api.PlanPaginatedResponse
	ListPlansParams   = api.ListPlansParams
	ListPlansHandler  httptransport.HandlerWithArgs[ListPlansRequest, ListPlansResponse, ListPlansParams]
)

func (h *handler) ListPlans() ListPlansHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListPlansParams) (ListPlansRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListPlansRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			req := ListPlansRequest{
				OrderBy: plan.OrderBy(lo.FromPtrOr(params.OrderBy, api.PlanOrderById)),
				Order:   sortx.Order(defaultx.WithDefault(params.Order, api.SortOrderDESC)),
				Page: pagination.Page{
					PageSize:   defaultx.WithDefault(params.PageSize, notification.DefaultPageSize),
					PageNumber: defaultx.WithDefault(params.Page, notification.DefaultPageNumber),
				},
				Namespaces:  []string{ns},
				IDs:         lo.FromPtrOr(params.Id, nil),
				Keys:        lo.FromPtrOr(params.Key, nil),
				KeyVersions: lo.FromPtrOr(params.KeyVersion, nil),
			}

			return req, nil
		},
		func(ctx context.Context, request ListPlansRequest) (ListPlansResponse, error) {
			resp, err := h.service.ListPlans(ctx, request)
			if err != nil {
				return ListPlansResponse{}, fmt.Errorf("failed to list plans: %w", err)
			}

			items := make([]api.Plan, 0, len(resp.Items))

			for _, p := range resp.Items {
				var item api.Plan

				item, err = FromPlan(p)
				if err != nil {
					return ListPlansResponse{}, fmt.Errorf("failed to cast plan pahse: %w", err)
				}

				items = append(items, item)
			}

			return ListPlansResponse{
				Items:      items,
				Page:       resp.Page.PageNumber,
				PageSize:   resp.Page.PageSize,
				TotalCount: resp.TotalCount,
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListPlansResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listPlans"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	CreatePlanRequest  = plan.CreatePlanInput
	CreatePlanResponse = api.Plan
	CreatePlanHandler  httptransport.Handler[CreatePlanRequest, CreatePlanResponse]
)

func (h *handler) CreatePlan() CreatePlanHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreatePlanRequest, error) {
			body := api.PlanCreate{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CreatePlanRequest{}, fmt.Errorf("failed to decode create plan request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreatePlanRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			req, err := AsCreatePlanRequest(body, ns)
			if err != nil {
				return CreatePlanRequest{}, fmt.Errorf("failed to create plan request: %w", err)
			}

			req.NamespacedModel = models.NamespacedModel{
				Namespace: ns,
			}

			return req, nil
		},
		func(ctx context.Context, request CreatePlanRequest) (CreatePlanResponse, error) {
			p, err := h.service.CreatePlan(ctx, request)
			if err != nil {
				return CreatePlanResponse{}, fmt.Errorf("failed to create plan: %w", err)
			}

			return FromPlan(*p)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreatePlanResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createPlan"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	UpdatePlanRequest  = plan.UpdatePlanInput
	UpdatePlanResponse = api.Plan
	UpdatePlanHandler  httptransport.HandlerWithArgs[UpdatePlanRequest, UpdatePlanResponse, string]
)

func (h *handler) UpdatePlan() UpdatePlanHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, planID string) (UpdatePlanRequest, error) {
			body := api.PlanUpdate{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return UpdatePlanRequest{}, fmt.Errorf("failed to decode update plan request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdatePlanRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			req, err := AsUpdatePlanRequest(body, ns, planID)
			if err != nil {
				return UpdatePlanRequest{}, fmt.Errorf("failed to update plan request: %w", err)
			}

			req.NamespacedID = models.NamespacedID{
				Namespace: ns,
				ID:        planID,
			}

			return req, nil
		},
		func(ctx context.Context, request UpdatePlanRequest) (UpdatePlanResponse, error) {
			p, err := h.service.UpdatePlan(ctx, request)
			if err != nil {
				return UpdatePlanResponse{}, fmt.Errorf("failed to update plan: %w", err)
			}

			return FromPlan(*p)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdatePlanResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("updatePlan"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	DeletePlanRequest  = plan.DeletePlanInput
	DeletePlanResponse = interface{}
	DeletePlanHandler  httptransport.HandlerWithArgs[DeletePlanRequest, DeletePlanResponse, string]
)

func (h *handler) DeletePlan() DeletePlanHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, planID string) (DeletePlanRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeletePlanRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return DeletePlanRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        planID,
				},
			}, nil
		},
		func(ctx context.Context, request DeletePlanRequest) (DeletePlanResponse, error) {
			err := h.service.DeletePlan(ctx, request)
			if err != nil {
				return nil, fmt.Errorf("failed to delete plan: %w", err)
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[DeletePlanResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("deletePlan"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	GetPlanRequest       = plan.GetPlanInput
	GetPlanRequestParams struct {
		// PlanID is the plan unique identifier in ULID format.
		ID string

		// Key is the unique key for Plan.
		Key string

		// Version is the version of the Plan.
		// If not set the latest version is assumed.
		Version int

		// AllowLatest defines whether return the latest version regardless of its PlanStatus or with ActiveStatus only if
		// Version is not set.
		IncludeLatest bool
	}
	GetPlanResponse = api.Plan
	GetPlanHandler  httptransport.HandlerWithArgs[GetPlanRequest, GetPlanResponse, GetPlanRequestParams]
)

func (h *handler) GetPlan() GetPlanHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetPlanRequestParams) (GetPlanRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetPlanRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return GetPlanRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        params.ID,
				},
				Key:           params.Key,
				Version:       params.Version,
				IncludeLatest: params.IncludeLatest,
			}, nil
		},
		func(ctx context.Context, request GetPlanRequest) (GetPlanResponse, error) {
			p, err := h.service.GetPlan(ctx, request)
			if err != nil {
				return GetPlanResponse{}, fmt.Errorf("failed to get plan: %w", err)
			}

			return FromPlan(*p)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetPlanResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getPlan"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	PublishPlanRequest  = plan.PublishPlanInput
	PublishPlanResponse = api.Plan
	PublishPlanHandler  httptransport.HandlerWithArgs[PublishPlanRequest, PublishPlanResponse, string]
)

func (h *handler) PublishPlan() PublishPlanHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, planID string) (PublishPlanRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return PublishPlanRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			// TODO(chrisgacsal): update api.Request in TypeSpec definition to allow setting EffectivePeriod

			req := PublishPlanRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        planID,
				},
				EffectivePeriod: plan.EffectivePeriod{
					EffectiveFrom: lo.ToPtr(time.Now()),
				},
			}

			return req, nil
		},
		func(ctx context.Context, request PublishPlanRequest) (PublishPlanResponse, error) {
			p, err := h.service.PublishPlan(ctx, request)
			if err != nil {
				return PublishPlanResponse{}, fmt.Errorf("failed to Publish plan: %w", err)
			}

			return FromPlan(*p)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetPlanResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("publishPlan"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	ArchivePlanRequest  = plan.ArchivePlanInput
	ArchivePlanResponse = api.Plan
	ArchivePlanHandler  httptransport.HandlerWithArgs[ArchivePlanRequest, ArchivePlanResponse, string]
)

func (h *handler) ArchivePlan() ArchivePlanHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, planID string) (ArchivePlanRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ArchivePlanRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			// TODO(chrisgacsal): update api.Request in TypeSpec definition to allow setting EffectivePeriod.To

			req := ArchivePlanRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        planID,
				},
				EffectiveTo: time.Now(),
			}

			return req, nil
		},
		func(ctx context.Context, request ArchivePlanRequest) (ArchivePlanResponse, error) {
			p, err := h.service.ArchivePlan(ctx, request)
			if err != nil {
				return ArchivePlanResponse{}, fmt.Errorf("failed to archive plan: %w", err)
			}

			return FromPlan(*p)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetPlanResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("archivePlan"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	NextPlanRequest  = plan.NextPlanInput
	NextPlanResponse = api.Plan
	NextPlanHandler  httptransport.HandlerWithArgs[NextPlanRequest, NextPlanResponse, string]
)

func (h *handler) NextPlan() NextPlanHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, planID string) (NextPlanRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return NextPlanRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			// TODO(chrisgacsal): update api.Request in TypeSpec definition to allow setting EffectivePeriod.To

			req := NextPlanRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        planID,
				},
				Key:     "",
				Version: 0,
			}

			return req, nil
		},
		func(ctx context.Context, request NextPlanRequest) (NextPlanResponse, error) {
			p, err := h.service.NextPlan(ctx, request)
			if err != nil {
				return NextPlanResponse{}, fmt.Errorf("failed to create next version of Plan: %w", err)
			}

			// TODO(chrisgacsal): update api.Response in TypeSpec definition to allow returning Plan

			return FromPlan(*p)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetPlanResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("nextPlan"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
