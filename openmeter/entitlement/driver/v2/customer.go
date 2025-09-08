package entitlementdriverv2

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	CreateCustomerEntitlementHandlerRequest  = entitlement.CreateEntitlementInputs
	CreateCustomerEntitlementHandlerResponse = api.EntitlementV2
	CreateCustomerEntitlementHandlerParams   = string // customerIdOrKey
)

type CreateCustomerEntitlementHandler httptransport.HandlerWithArgs[CreateCustomerEntitlementHandlerRequest, CreateCustomerEntitlementHandlerResponse, CreateCustomerEntitlementHandlerParams]

func (h *customerEntitlementHandler) CreateCustomerEntitlement() CreateCustomerEntitlementHandler {
	return httptransport.NewHandlerWithArgs[
		CreateCustomerEntitlementHandlerRequest,
		CreateCustomerEntitlementHandlerResponse,
		CreateCustomerEntitlementHandlerParams,
	](
		func(ctx context.Context, r *http.Request, customerIdOrKey string) (entitlement.CreateEntitlementInputs, error) {
			inp := &api.EntitlementCreateInputs{}
			request := entitlement.CreateEntitlementInputs{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &inp); err != nil {
				return request, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return request, err
			}

			// Resolve customer
			cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerIDOrKey: &customer.CustomerIDOrKey{
					Namespace: ns,
					IDOrKey:   customerIdOrKey,
				},
			})
			if err != nil {
				return request, err
			}

			if cus != nil && cus.IsDeleted() {
				return request, models.NewGenericPreConditionFailedError(
					fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
				)
			}

			// Reuse v1 parser to build entitlement create inputs using the subject key
			return entitlementdriver.ParseAPICreateInput(inp, ns, cus.GetUsageAttribution())
		},
		func(ctx context.Context, request CreateCustomerEntitlementHandlerRequest) (CreateCustomerEntitlementHandlerResponse, error) {
			ent, err := h.connector.CreateEntitlement(ctx, request)
			if err != nil {
				return api.EntitlementV2{}, err
			}

			if ent.Customer == nil {
				return api.EntitlementV2{}, commonhttp.NewHTTPError(http.StatusNotFound, errors.New("customer not found"))
			}

			v2, err := ParserV2.ToAPIGenericV2(ent, ent.Customer.ID, ent.Customer.Key)
			if err != nil {
				return api.EntitlementV2{}, err
			}
			return *v2, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateCustomerEntitlementHandlerResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createCustomerEntitlementV2"),
			httptransport.WithErrorEncoder(getErrorEncoder()),
		)...,
	)
}

type (
	ListCustomerEntitlementsHandlerRequest  = entitlement.ListEntitlementsParams
	ListCustomerEntitlementsHandlerResponse = api.EntitlementV2PaginatedResponse
	ListCustomerEntitlementsHandlerParams   struct {
		CustomerIdOrKey string
		Params          api.ListCustomerEntitlementsV2Params
	}
)

type ListCustomerEntitlementsHandler httptransport.HandlerWithArgs[ListCustomerEntitlementsHandlerRequest, ListCustomerEntitlementsHandlerResponse, ListCustomerEntitlementsHandlerParams]

func (h *customerEntitlementHandler) ListCustomerEntitlements() ListCustomerEntitlementsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, p ListCustomerEntitlementsHandlerParams) (ListCustomerEntitlementsHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return entitlement.ListEntitlementsParams{}, err
			}

			// Resolve customer to get subject keys
			cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerIDOrKey: &customer.CustomerIDOrKey{Namespace: ns, IDOrKey: p.CustomerIdOrKey},
			})
			if err != nil {
				return entitlement.ListEntitlementsParams{}, err
			}

			if cus != nil && cus.IsDeleted() {
				return entitlement.ListEntitlementsParams{}, models.NewGenericPreConditionFailedError(
					fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
				)
			}

			// Build list params
			req := entitlement.ListEntitlementsParams{
				Namespaces:     []string{ns},
				SubjectKeys:    cus.UsageAttribution.SubjectKeys,
				IncludeDeleted: defaultIncludeDeleted(p.Params.IncludeDeleted),
				Order:          commonhttp.GetSortOrder(api.SortOrderASC, p.Params.Order),
			}

			// Pagination: support both page/pageSize and limit/offset
			if p.Params.Page != nil || p.Params.PageSize != nil {
				req.Page.PageNumber = lo.FromPtrOr(p.Params.Page, 0)
				req.Page.PageSize = lo.FromPtrOr(p.Params.PageSize, commonhttp.DefaultPageSize)
				if req.Page.PageNumber == 0 {
					req.Page.PageNumber = commonhttp.DefaultPage
				}
			}
			req.Limit = lo.FromPtrOr(p.Params.Limit, 0)
			req.Offset = lo.FromPtrOr(p.Params.Offset, 0)

			// OrderBy mapping
			switch lo.FromPtrOr((*string)(p.Params.OrderBy), "createdAt") {
			case "createdAt":
				req.OrderBy = entitlement.ListEntitlementsOrderByCreatedAt
			case "updatedAt":
				req.OrderBy = entitlement.ListEntitlementsOrderByUpdatedAt
			default:
				req.OrderBy = entitlement.ListEntitlementsOrderByCreatedAt
			}

			return req, nil
		},
		func(ctx context.Context, req ListCustomerEntitlementsHandlerRequest) (ListCustomerEntitlementsHandlerResponse, error) {
			paged, err := h.connector.ListEntitlements(ctx, req)
			if err != nil {
				return ListCustomerEntitlementsHandlerResponse{}, err
			}

			// Map paged response -> []api.EntitlementV2 using ParserV2
			mapped, err := pagination.MapPagedResponseError(paged, func(e entitlement.Entitlement) (api.EntitlementV2, error) {
				var customerId string
				var customerKey *string
				if e.Customer != nil {
					customerId = e.Customer.ID
					customerKey = e.Customer.Key
				}
				v2, err := ParserV2.ToAPIGenericV2(&e, customerId, customerKey)
				if err != nil {
					return api.EntitlementV2{}, err
				}
				return *v2, nil
			})
			if err != nil {
				return ListCustomerEntitlementsHandlerResponse{}, err
			}

			return api.EntitlementV2PaginatedResponse{
				Items:      mapped.Items,
				TotalCount: mapped.TotalCount,
				Page:       mapped.Page.PageNumber,
				PageSize:   mapped.Page.PageSize,
			}, nil
		},
		commonhttp.JSONResponseEncoder[ListCustomerEntitlementsHandlerResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listCustomerEntitlementsV2"),
			httptransport.WithErrorEncoder(getErrorEncoder()),
		)...,
	)
}

type (
	GetCustomerEntitlementHandlerParams struct {
		CustomerIDOrKey           string
		EntitlementIdOrFeatureKey string
	}
	GetCustomerEntitlementHandlerRequest struct {
		CustomerIDOrKey           string
		EntitlementIdOrFeatureKey string
		Namespace                 string
	}
	GetCustomerEntitlementHandlerResponse = *api.EntitlementV2
)

type GetCustomerEntitlementHandler httptransport.HandlerWithArgs[GetCustomerEntitlementHandlerRequest, GetCustomerEntitlementHandlerResponse, GetCustomerEntitlementHandlerParams]

func (h *customerEntitlementHandler) GetCustomerEntitlement() GetCustomerEntitlementHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetCustomerEntitlementHandlerParams) (GetCustomerEntitlementHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetCustomerEntitlementHandlerRequest{}, err
			}

			return GetCustomerEntitlementHandlerRequest{
				CustomerIDOrKey:           params.CustomerIDOrKey,
				EntitlementIdOrFeatureKey: params.EntitlementIdOrFeatureKey,
				Namespace:                 ns,
			}, nil
		},
		func(ctx context.Context, request GetCustomerEntitlementHandlerRequest) (GetCustomerEntitlementHandlerResponse, error) {
			// First we resolve the customer
			cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerIDOrKey: &customer.CustomerIDOrKey{
					Namespace: request.Namespace,
					IDOrKey:   request.CustomerIDOrKey,
				},
			})
			if err != nil {
				return nil, err
			}

			if cus != nil && cus.IsDeleted() {
				return nil, models.NewGenericPreConditionFailedError(
					fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
				)
			}

			// Then we resolve the entitlement
			entitlement, err := h.connector.GetEntitlementOfCustomerAt(ctx, request.Namespace, cus.ID, request.EntitlementIdOrFeatureKey, clock.Now())
			if err != nil {
				return nil, err
			}

			return ParserV2.ToAPIGenericV2(entitlement, entitlement.Customer.ID, entitlement.Customer.Key)
		},
		commonhttp.JSONResponseEncoder[GetCustomerEntitlementHandlerResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getCustomerEntitlementV2"),
			httptransport.WithErrorEncoder(getErrorEncoder()),
		)...,
	)
}

type (
	DeleteCustomerEntitlementHandlerParams struct {
		CustomerIDOrKey           string
		EntitlementIdOrFeatureKey string
	}
	DeleteCustomerEntitlementHandlerRequest struct {
		CustomerIDOrKey           string
		EntitlementIdOrFeatureKey string
		Namespace                 string
	}
	DeleteCustomerEntitlementHandlerResponse = interface{}
)

type DeleteCustomerEntitlementHandler httptransport.HandlerWithArgs[DeleteCustomerEntitlementHandlerRequest, DeleteCustomerEntitlementHandlerResponse, DeleteCustomerEntitlementHandlerParams]

func (h *customerEntitlementHandler) DeleteCustomerEntitlement() DeleteCustomerEntitlementHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params DeleteCustomerEntitlementHandlerParams) (DeleteCustomerEntitlementHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteCustomerEntitlementHandlerRequest{}, err
			}

			return DeleteCustomerEntitlementHandlerRequest{
				CustomerIDOrKey:           params.CustomerIDOrKey,
				EntitlementIdOrFeatureKey: params.EntitlementIdOrFeatureKey,
				Namespace:                 ns,
			}, nil
		},
		func(ctx context.Context, request DeleteCustomerEntitlementHandlerRequest) (DeleteCustomerEntitlementHandlerResponse, error) {
			// First we resolve the customer
			cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerIDOrKey: &customer.CustomerIDOrKey{
					Namespace: request.Namespace,
					IDOrKey:   request.CustomerIDOrKey,
				},
			})
			if err != nil {
				return nil, err
			}

			if cus != nil && cus.IsDeleted() {
				return request, models.NewGenericPreConditionFailedError(
					fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
				)
			}

			ent, err := h.connector.GetEntitlementOfCustomerAt(ctx, request.Namespace, cus.ID, request.EntitlementIdOrFeatureKey, clock.Now())
			if err != nil {
				return nil, err
			}

			return nil, h.connector.DeleteEntitlement(ctx, request.Namespace, ent.ID, clock.Now())
		},
		commonhttp.JSONResponseEncoderWithStatus[DeleteCustomerEntitlementHandlerResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("deleteCustomerEntitlementV2"),
			httptransport.WithErrorEncoder(getErrorEncoder()),
		)...,
	)
}

type (
	OverrideCustomerEntitlementHandlerParams struct {
		CustomerIDOrKey           string
		EntitlementIdOrFeatureKey string
	}
	OverrideCustomerEntitlementHandlerRequest struct {
		CustomerID                string
		EntitlementIDOrFeatureKey string
		Namespace                 string
		Inputs                    entitlement.CreateEntitlementInputs
	}
	OverrideCustomerEntitlementHandlerResponse = *api.EntitlementV2
)

type OverrideCustomerEntitlementHandler httptransport.HandlerWithArgs[OverrideCustomerEntitlementHandlerRequest, OverrideCustomerEntitlementHandlerResponse, OverrideCustomerEntitlementHandlerParams]

func (h *customerEntitlementHandler) OverrideCustomerEntitlement() OverrideCustomerEntitlementHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params OverrideCustomerEntitlementHandlerParams) (OverrideCustomerEntitlementHandlerRequest, error) {
			var def OverrideCustomerEntitlementHandlerRequest

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return OverrideCustomerEntitlementHandlerRequest{}, err
			}

			apiInp := &api.EntitlementCreateInputs{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &apiInp); err != nil {
				return OverrideCustomerEntitlementHandlerRequest{}, err
			}

			// Resolve customer
			cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerIDOrKey: &customer.CustomerIDOrKey{
					Namespace: ns,
					IDOrKey:   params.CustomerIDOrKey,
				},
			})
			if err != nil {
				return def, err
			}

			if cus != nil && cus.IsDeleted() {
				return def, models.NewGenericPreConditionFailedError(
					fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
				)
			}

			// Reuse v1 parser to build entitlement create inputs using the subject key
			createInp, err := entitlementdriver.ParseAPICreateInput(apiInp, ns, cus.GetUsageAttribution())
			if err != nil {
				return OverrideCustomerEntitlementHandlerRequest{}, err
			}

			return OverrideCustomerEntitlementHandlerRequest{
				CustomerID:                cus.ID,
				EntitlementIDOrFeatureKey: params.EntitlementIdOrFeatureKey,
				Namespace:                 ns,
				Inputs:                    createInp,
			}, nil
		},
		func(ctx context.Context, request OverrideCustomerEntitlementHandlerRequest) (OverrideCustomerEntitlementHandlerResponse, error) {
			oldEnt, err := h.connector.GetEntitlementOfCustomerAt(ctx, request.Namespace, request.CustomerID, request.EntitlementIDOrFeatureKey, clock.Now())
			if err != nil {
				return nil, err
			}

			ent, err := h.connector.OverrideEntitlement(ctx, request.CustomerID, oldEnt.ID, request.Inputs)
			if err != nil {
				return nil, err
			}

			return ParserV2.ToAPIGenericV2(ent, ent.Customer.ID, ent.Customer.Key)
		},
		commonhttp.JSONResponseEncoder[OverrideCustomerEntitlementHandlerResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("overrideCustomerEntitlementV2"),
			httptransport.WithErrorEncoder(getErrorEncoder()),
		)...,
	)
}

func defaultIncludeDeleted(p *bool) bool { return lo.FromPtrOr(p, false) }

func (h *customerEntitlementHandler) resolveNamespace(ctx context.Context) (string, error) {
	ns, ok := h.namespaceDecoder.GetNamespace(ctx)
	if !ok {
		return "", commonhttp.NewHTTPError(http.StatusInternalServerError, errors.New("internal server error"))
	}

	return ns, nil
}
