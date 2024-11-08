package httpdriver

import (
	"context"
	"fmt"
	"net/http"

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

type PhaseKeyPlanParams struct {
	// Key the user provided unique identifier (in the scope of the plan) of the plan phase.
	Key string

	// PlanID is the plan unique identifier in ULID format.
	PlanID string
}

type (
	ListPhasesRequest  = plan.ListPhasesInput
	ListPhasesResponse = api.PlanPhasePaginatedResponse
	ListPhasesParams   = api.ListPlanPhasesParams
	ListPhasesHandler  httptransport.HandlerWithArgs[ListPhasesRequest, ListPhasesResponse, ListPhasesParams]
)

func (h *handler) ListPhases() ListPhasesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListPhasesParams) (ListPhasesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListPhasesRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			req := ListPhasesRequest{
				Namespaces: []string{ns},
				OrderBy:    plan.OrderBy(lo.FromPtrOr(params.OrderBy, api.PhasesOrderByKey)),
				Order:      sortx.Order(defaultx.WithDefault(params.Order, api.SortOrderDESC)),
				Page: pagination.Page{
					PageSize:   defaultx.WithDefault(params.PageSize, notification.DefaultPageSize),
					PageNumber: defaultx.WithDefault(params.Page, notification.DefaultPageNumber),
				},
			}

			return req, nil
		},
		func(ctx context.Context, request ListPhasesRequest) (ListPhasesResponse, error) {
			resp, err := h.service.ListPhases(ctx, request)
			if err != nil {
				return ListPhasesResponse{}, fmt.Errorf("failed to list plan phases: %w", err)
			}

			items := make([]api.PlanPhase, 0, len(resp.Items))

			for _, phase := range resp.Items {
				var item api.PlanPhase

				item, err = fromPlanPhase(phase)
				if err != nil {
					return ListPhasesResponse{}, fmt.Errorf("failed to cast plan phase: %w", err)
				}

				items = append(items, item)
			}

			return ListPhasesResponse{
				Items:      items,
				Page:       resp.Page.PageNumber,
				PageSize:   resp.Page.PageSize,
				TotalCount: resp.TotalCount,
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListPhasesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listPlanPhases"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	CreatePhaseRequest  = plan.CreatePhaseInput
	CreatePhaseResponse = api.PlanPhase
	CreatePhaseHandler  httptransport.HandlerWithArgs[CreatePhaseRequest, CreatePhaseResponse, string]
)

func (h *handler) CreatePhase() CreatePhaseHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, planID string) (CreatePhaseRequest, error) {
			body := api.PlanPhaseCreate{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CreatePhaseRequest{}, fmt.Errorf("failed to decode create plan phase request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreatePhaseRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			req, err := asCreatePhaseRequest(body, ns, planID)
			if err != nil {
				return CreatePhaseRequest{}, fmt.Errorf("failed to create phase request: %w", err)
			}

			req.NamespacedModel = models.NamespacedModel{
				Namespace: ns,
			}

			return req, nil
		},
		func(ctx context.Context, request CreatePhaseRequest) (CreatePhaseResponse, error) {
			phase, err := h.service.CreatePhase(ctx, request)
			if err != nil {
				return CreatePhaseResponse{}, fmt.Errorf("failed to create phase: %w", err)
			}

			return fromPlanPhase(*phase)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreatePhaseResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createPlanPhase"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	UpdatePhaseRequest       = plan.UpdatePhaseInput
	UpdatePhaseRequestParams = PhaseKeyPlanParams
	UpdatePhaseResponse      = api.PlanPhase
	UpdatePhaseHandler       httptransport.HandlerWithArgs[UpdatePhaseRequest, UpdatePhaseResponse, UpdatePhaseRequestParams]
)

func (h *handler) UpdatePhase() UpdatePhaseHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params UpdatePhaseRequestParams) (UpdatePhaseRequest, error) {
			body := api.PlanPhaseUpdate{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return UpdatePhaseRequest{}, fmt.Errorf("failed to decode update plan phase request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdatePhaseRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			req, err := asUpdatePhaseRequest(body, ns, params.PlanID, params.Key)
			if err != nil {
				return UpdatePhaseRequest{}, fmt.Errorf("failed to update plan phase request: %w", err)
			}

			req.NamespacedID = models.NamespacedID{
				Namespace: ns,
			}
			req.Key = params.Key
			req.PlanID = params.PlanID

			return req, nil
		},
		func(ctx context.Context, request UpdatePhaseRequest) (UpdatePhaseResponse, error) {
			phase, err := h.service.UpdatePhase(ctx, request)
			if err != nil {
				return UpdatePhaseResponse{}, fmt.Errorf("failed to update plan phase: %w", err)
			}

			return fromPlanPhase(*phase)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdatePhaseResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("updatePlanPhase"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	DeletePhaseRequest       = plan.DeletePhaseInput
	DeletePhaseRequestParams = PhaseKeyPlanParams
	DeletePhaseResponse      = interface{}
	DeletePhaseHandler       httptransport.HandlerWithArgs[DeletePhaseRequest, DeletePhaseResponse, DeletePhaseRequestParams]
)

func (h *handler) DeletePhase() DeletePhaseHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params DeletePhaseRequestParams) (DeletePhaseRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeletePhaseRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return DeletePhaseRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
				},
				Key:    params.Key,
				PlanID: params.PlanID,
			}, nil
		},
		func(ctx context.Context, request DeletePhaseRequest) (DeletePhaseResponse, error) {
			err := h.service.DeletePhase(ctx, request)
			if err != nil {
				return nil, fmt.Errorf("failed to delete plan phase: %w", err)
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[DeletePhaseResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("deletePlanPhase"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	GetPhaseRequest       = plan.GetPhaseInput
	GetPhaseRequestParams = PhaseKeyPlanParams
	GetPhaseResponse      = api.PlanPhase
	GetPhaseHandler       httptransport.HandlerWithArgs[GetPhaseRequest, GetPhaseResponse, GetPhaseRequestParams]
)

func (h *handler) GetPhase() GetPhaseHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetPhaseRequestParams) (GetPhaseRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetPhaseRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return GetPhaseRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
				},
				Key:    params.Key,
				PlanID: params.PlanID,
			}, nil
		},
		func(ctx context.Context, request GetPhaseRequest) (GetPhaseResponse, error) {
			phase, err := h.service.GetPhase(ctx, request)
			if err != nil {
				return GetPhaseResponse{}, fmt.Errorf("failed to get plan phase: %w", err)
			}

			return fromPlanPhase(*phase)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetPhaseResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getPlanPhase"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
