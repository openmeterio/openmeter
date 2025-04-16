package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

const (
	DefaultPageSize   = 100
	DefaultPageNumber = 1
)

type (
	ListPlanAddonsRequest  = planaddon.ListPlanAddonsInput
	ListPlanAddonsResponse = api.PlanAddonPaginatedResponse
	ListPlanAddonsParams   struct {
		api.ListPlanAddonsParams

		// AddonID or Key.
		PlanIDOrKey string
	}
	ListPlanAddonsHandler httptransport.HandlerWithArgs[ListPlanAddonsRequest, ListPlanAddonsResponse, ListPlanAddonsParams]
)

func (h *handler) ListPlanAddons() ListPlanAddonsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListPlanAddonsParams) (ListPlanAddonsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListPlanAddonsRequest{}, fmt.Errorf("failed to resolve namespace [namespace=%s]: %w", ns, err)
			}

			req := ListPlanAddonsRequest{
				Page: pagination.Page{
					PageSize:   lo.FromPtrOr(params.PageSize, DefaultPageSize),
					PageNumber: lo.FromPtrOr(params.Page, DefaultPageNumber),
				},
				OrderBy:          planaddon.OrderBy(lo.FromPtrOr(params.OrderBy, api.PlanAddonOrderById)),
				Order:            sortx.Order(lo.FromPtrOr(params.Order, api.SortOrderDESC)),
				Namespaces:       []string{ns},
				PlanIDs:          []string{params.PlanIDOrKey},
				PlanKeys:         []string{params.PlanIDOrKey},
				AddonIDs:         lo.FromPtrOr(params.Id, nil),
				AddonKeys:        lo.FromPtrOr(params.Key, nil),
				AddonKeyVersions: lo.FromPtrOr(params.KeyVersion, nil),
				IncludeDeleted:   lo.FromPtr(params.IncludeDeleted),
			}

			return req, nil
		},
		func(ctx context.Context, request ListPlanAddonsRequest) (ListPlanAddonsResponse, error) {
			resp, err := h.service.ListPlanAddons(ctx, request)
			if err != nil {
				return ListPlanAddonsResponse{}, fmt.Errorf("failed to list plan add-on assignments: %w", err)
			}

			items := make([]api.PlanAddon, 0, len(resp.Items))

			for _, a := range resp.Items {
				var item api.PlanAddon

				item, err = FromPlanAddon(a)
				if err != nil {
					return ListPlanAddonsResponse{}, fmt.Errorf("failed to cast plan add-on assignment [namespace=%s plan.id=%s addon.id=%s]: %w",
						a.Namespace, a.Plan.ID, a.Addon.ID, err)
				}

				items = append(items, item)
			}

			return ListPlanAddonsResponse{
				Items:      items,
				Page:       resp.Page.PageNumber,
				PageSize:   resp.Page.PageSize,
				TotalCount: resp.TotalCount,
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListPlanAddonsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listPlanAddons"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	CreatePlanAddonRequest  = planaddon.CreatePlanAddonInput
	CreatePlanAddonResponse = api.PlanAddon
	CreatePlanAddonHandler  httptransport.HandlerWithArgs[CreatePlanAddonRequest, CreatePlanAddonResponse, string]
)

func (h *handler) CreatePlanAddon() CreatePlanAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, planID string) (CreatePlanAddonRequest, error) {
			body := api.PlanAddonCreate{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CreatePlanAddonRequest{}, fmt.Errorf("failed to decode create plan add-on assignment request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreatePlanAddonRequest{}, fmt.Errorf("failed to resolve namespace [namespace=%s]: %w", ns, err)
			}

			req, err := AsCreatePlanAddonRequest(body, ns, planID)
			if err != nil {
				return CreatePlanAddonRequest{}, fmt.Errorf("failed to parse create plan add-on assignment request [namespace=%s plan.id=%s]: %w",
					ns, planID, err)
			}

			return req, nil
		},
		func(ctx context.Context, request CreatePlanAddonRequest) (CreatePlanAddonResponse, error) {
			a, err := h.service.CreatePlanAddon(ctx, request)
			if err != nil {
				return CreatePlanAddonResponse{}, fmt.Errorf("failed to create plan add-on assignment request [namespace=%s plan.id=%s addon.id=%s]: %w",
					request.Namespace, request.PlanID, request.AddonID, err)
			}

			return FromPlanAddon(*a)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreatePlanAddonResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createPlanAddon"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	UpdatePlanAddonRequest = planaddon.UpdatePlanAddonInput
	UpdatePlanAddonParams  struct {
		PlanID  string
		AddonID string
	}
	UpdatePlanAddonResponse = api.PlanAddon
	UpdatePlanAddonHandler  httptransport.HandlerWithArgs[UpdatePlanAddonRequest, UpdatePlanAddonResponse, UpdatePlanAddonParams]
)

func (h *handler) UpdatePlanAddon() UpdatePlanAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params UpdatePlanAddonParams) (UpdatePlanAddonRequest, error) {
			body := api.PlanAddonReplaceUpdate{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return UpdatePlanAddonRequest{}, fmt.Errorf("failed to decode update plan add-on assignment request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdatePlanAddonRequest{}, fmt.Errorf("failed to resolve namespace [namespace=%s]: %w", ns, err)
			}

			req, err := AsUpdatePlanAddonRequest(body, ns, params.PlanID, params.AddonID)
			if err != nil {
				return UpdatePlanAddonRequest{}, fmt.Errorf("failed to parse update plan add-on assignment request [namespace=%s plan.id=%s addon.id=%s]: %w",
					ns, params.PlanID, params.AddonID, err)
			}

			return req, nil
		},
		func(ctx context.Context, request UpdatePlanAddonRequest) (UpdatePlanAddonResponse, error) {
			a, err := h.service.UpdatePlanAddon(ctx, request)
			if err != nil {
				return UpdatePlanAddonResponse{}, fmt.Errorf("failed to update plan add-on assignment [namespace=%s plan.id=%s addon.id=%s]: %w",
					request.Namespace, request.PlanID, request.AddonID, err)
			}

			return FromPlanAddon(*a)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdatePlanAddonResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("updatePlanAddon"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	DeletePlanAddonRequest = planaddon.DeletePlanAddonInput
	DeletePlanAddonParams  struct {
		PlanID  string
		AddonID string
	}
	DeletePlanAddonResponse = interface{}
	DeletePlanAddonHandler  httptransport.HandlerWithArgs[DeletePlanAddonRequest, DeletePlanAddonResponse, DeletePlanAddonParams]
)

func (h *handler) DeletePlanAddon() DeletePlanAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params DeletePlanAddonParams) (DeletePlanAddonRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeletePlanAddonRequest{}, fmt.Errorf("failed to resolve namespace [namespace=%s]: %w", ns, err)
			}

			return DeletePlanAddonRequest{
				NamespacedModel: models.NamespacedModel{
					Namespace: ns,
				},
				PlanID:  params.PlanID,
				AddonID: params.AddonID,
			}, nil
		},
		func(ctx context.Context, request DeletePlanAddonRequest) (DeletePlanAddonResponse, error) {
			err := h.service.DeletePlanAddon(ctx, request)
			if err != nil {
				return nil, fmt.Errorf("failed to delete plan add-on assignment [namespace=%s plan.id=%s addon.id=%s]: %w",
					request.Namespace, request.PlanID, request.AddonID, err)
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[DeletePlanAddonResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("deletePlanAddon"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	GetPlanAddonRequest = planaddon.GetPlanAddonInput
	GetPlanAddonParams  struct {
		PlanIDOrKey  string
		AddonIDOrKey string
	}
	GetPlanAddonResponse = api.PlanAddon
	GetPlanAddonHandler  httptransport.HandlerWithArgs[GetPlanAddonRequest, GetPlanAddonResponse, GetPlanAddonParams]
)

func (h *handler) GetPlanAddon() GetPlanAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetPlanAddonParams) (GetPlanAddonRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetPlanAddonRequest{}, fmt.Errorf("failed to resolve namespace [namespace=%s]: %w", ns, err)
			}

			return GetPlanAddonRequest{
				NamespacedModel: models.NamespacedModel{
					Namespace: ns,
				},
				PlanIDOrKey:  params.PlanIDOrKey,
				AddonIDOrKey: params.AddonIDOrKey,
			}, nil
		},
		func(ctx context.Context, request GetPlanAddonRequest) (GetPlanAddonResponse, error) {
			a, err := h.service.GetPlanAddon(ctx, request)
			if err != nil {
				return GetPlanAddonResponse{}, fmt.Errorf("failed to get plan add-on assignment [namespace=%s plan.idOrKey=%s addon.idOrKey=%s]: %w",
					request.Namespace, request.PlanIDOrKey, request.AddonIDOrKey, err)
			}

			return FromPlanAddon(*a)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetPlanAddonResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getPlanAddon"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
