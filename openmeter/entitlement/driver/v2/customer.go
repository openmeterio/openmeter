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
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/strcase"
)

type (
	CreateCustomerEntitlementHandlerRequest = struct {
		Namespace       string
		CustomerIDOrKey string
		APIInput        *api.EntitlementV2CreateInputs
	}
	CreateCustomerEntitlementHandlerResponse = api.EntitlementV2
	CreateCustomerEntitlementHandlerParams   = string // customerIdOrKey
)

type CreateCustomerEntitlementHandler httptransport.HandlerWithArgs[CreateCustomerEntitlementHandlerRequest, CreateCustomerEntitlementHandlerResponse, CreateCustomerEntitlementHandlerParams]

func (h *entitlementHandler) CreateCustomerEntitlement() CreateCustomerEntitlementHandler {
	return httptransport.NewHandlerWithArgs[
		CreateCustomerEntitlementHandlerRequest,
		CreateCustomerEntitlementHandlerResponse,
		CreateCustomerEntitlementHandlerParams,
	](
		func(ctx context.Context, r *http.Request, customerIdOrKey string) (CreateCustomerEntitlementHandlerRequest, error) {
			inp := &api.EntitlementV2CreateInputs{}
			def := CreateCustomerEntitlementHandlerRequest{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &inp); err != nil {
				return def, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return def, err
			}

			// Reuse v1 parser to build entitlement create inputs using the subject key
			return CreateCustomerEntitlementHandlerRequest{
				Namespace:       ns,
				APIInput:        inp,
				CustomerIDOrKey: customerIdOrKey,
			}, nil
		},
		func(ctx context.Context, request CreateCustomerEntitlementHandlerRequest) (CreateCustomerEntitlementHandlerResponse, error) {
			// Let's resolve the customer
			// Resolve customer
			cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerIDOrKey: &customer.CustomerIDOrKey{
					Namespace: request.Namespace,
					IDOrKey:   request.CustomerIDOrKey,
				},
			})
			if err != nil {
				return CreateCustomerEntitlementHandlerResponse{}, err
			}

			if cus != nil && cus.IsDeleted() {
				return CreateCustomerEntitlementHandlerResponse{}, models.NewGenericPreConditionFailedError(
					fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
				)
			}

			createInp, grantsInp, err := ParseAPICreateInputV2(request.APIInput, request.Namespace, cus.GetUsageAttribution())
			if err != nil {
				return CreateCustomerEntitlementHandlerResponse{}, err
			}

			ent, err := h.connector.CreateEntitlement(ctx, createInp, grantsInp)
			if err != nil {
				return api.EntitlementV2{}, err
			}

			v2, err := ParserV2.ToAPIGenericV2(ent, cus.ID, cus.Key)
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
	ListCustomerEntitlementsHandlerRequest = struct {
		Namespace       string
		CustomerIdOrKey string
		ListParams      entitlement.ListEntitlementsParams
	}
	ListCustomerEntitlementsHandlerResponse = api.EntitlementV2PaginatedResponse
	ListCustomerEntitlementsHandlerParams   struct {
		CustomerIdOrKey string
		Params          api.ListCustomerEntitlementsV2Params
	}
)

type ListCustomerEntitlementsHandler httptransport.HandlerWithArgs[ListCustomerEntitlementsHandlerRequest, ListCustomerEntitlementsHandlerResponse, ListCustomerEntitlementsHandlerParams]

func (h *entitlementHandler) ListCustomerEntitlements() ListCustomerEntitlementsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, p ListCustomerEntitlementsHandlerParams) (ListCustomerEntitlementsHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCustomerEntitlementsHandlerRequest{}, err
			}

			cust, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerIDOrKey: &customer.CustomerIDOrKey{
					Namespace: ns,
					IDOrKey:   p.CustomerIdOrKey,
				},
			})
			if err != nil {
				return ListCustomerEntitlementsHandlerRequest{}, err
			}
			if cust != nil && cust.IsDeleted() {
				return ListCustomerEntitlementsHandlerRequest{}, models.NewGenericPreConditionFailedError(
					fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cust.Namespace, cust.ID),
				)
			}

			now := clock.Now()

			return ListCustomerEntitlementsHandlerRequest{
				Namespace:       ns,
				CustomerIdOrKey: p.CustomerIdOrKey,
				ListParams: entitlement.ListEntitlementsParams{
					CustomerIDs: []string{cust.ID},
					Namespaces:  []string{ns},
					ActiveAt:    lo.ToPtr(now),
					Page: pagination.NewPage(
						defaultx.WithDefault(p.Params.Page, 1),
						defaultx.WithDefault(p.Params.PageSize, 100),
					),
					OrderBy: entitlement.ListEntitlementsOrderBy(
						strcase.CamelToSnake(defaultx.WithDefault((*string)(p.Params.OrderBy), string(entitlement.ListEntitlementsOrderByCreatedAt))),
					),
					Order:               commonhttp.GetSortOrder(api.SortOrderASC, p.Params.Order),
					IncludeDeletedAfter: now,
				},
			}, nil
		},
		func(ctx context.Context, req ListCustomerEntitlementsHandlerRequest) (ListCustomerEntitlementsHandlerResponse, error) {
			ents, err := h.connector.ListEntitlementsWithCustomer(ctx, req.ListParams)
			if err != nil {
				return ListCustomerEntitlementsHandlerResponse{}, err
			}

			mapped, err := pagination.MapResultErr(ents.Entitlements, func(e entitlement.Entitlement) (api.EntitlementV2, error) {
				cust, ok := ents.CustomersByID[models.NamespacedID{Namespace: e.Namespace, ID: e.CustomerID}]
				if !ok {
					return api.EntitlementV2{}, models.NewGenericPreConditionFailedError(fmt.Errorf("customer not found [namespace=%s customer.id=%s]", e.Namespace, e.CustomerID))
				}
				v2, err := ParserV2.ToAPIGenericV2(&e, cust.ID, cust.Key)
				if err != nil {
					return api.EntitlementV2{}, err
				}
				return lo.FromPtr(v2), nil
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

func (h *entitlementHandler) GetCustomerEntitlement() GetCustomerEntitlementHandler {
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

			return ParserV2.ToAPIGenericV2(entitlement, entitlement.CustomerID, cus.Key)
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

func (h *entitlementHandler) DeleteCustomerEntitlement() DeleteCustomerEntitlementHandler {
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

			err = h.connector.DeleteEntitlement(ctx, request.Namespace, ent.ID, clock.Now())
			if err != nil {
				return nil, err
			}

			// 204, no content
			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[DeleteCustomerEntitlementHandlerResponse](http.StatusNoContent),
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
		EntitlementIDOrFeatureKey string
		CustomerIDOrKey           string
		Namespace                 string
		APIInput                  *api.EntitlementV2CreateInputs
	}
	OverrideCustomerEntitlementHandlerResponse = *api.EntitlementV2
)

type OverrideCustomerEntitlementHandler httptransport.HandlerWithArgs[OverrideCustomerEntitlementHandlerRequest, OverrideCustomerEntitlementHandlerResponse, OverrideCustomerEntitlementHandlerParams]

func (h *entitlementHandler) OverrideCustomerEntitlement() OverrideCustomerEntitlementHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params OverrideCustomerEntitlementHandlerParams) (OverrideCustomerEntitlementHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return OverrideCustomerEntitlementHandlerRequest{}, err
			}

			apiInp := &api.EntitlementV2CreateInputs{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &apiInp); err != nil {
				return OverrideCustomerEntitlementHandlerRequest{}, err
			}

			return OverrideCustomerEntitlementHandlerRequest{
				Namespace:                 ns,
				CustomerIDOrKey:           params.CustomerIDOrKey,
				EntitlementIDOrFeatureKey: params.EntitlementIdOrFeatureKey,
				APIInput:                  apiInp,
			}, nil
		},
		func(ctx context.Context, request OverrideCustomerEntitlementHandlerRequest) (OverrideCustomerEntitlementHandlerResponse, error) {
			// Resolve customer
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

			oldEnt, err := h.connector.GetEntitlementOfCustomerAt(ctx, request.Namespace, cus.ID, request.EntitlementIDOrFeatureKey, clock.Now())
			if err != nil {
				return nil, err
			}

			createInp, grantsInp, err := ParseAPICreateInputV2(request.APIInput, request.Namespace, cus.GetUsageAttribution())
			if err != nil {
				return nil, err
			}

			ent, err := h.connector.OverrideEntitlement(ctx, cus.ID, oldEnt.ID, createInp, grantsInp)
			if err != nil {
				return nil, err
			}

			return ParserV2.ToAPIGenericV2(ent, ent.CustomerID, cus.Key)
		},
		commonhttp.JSONResponseEncoder[OverrideCustomerEntitlementHandlerResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("overrideCustomerEntitlementV2"),
			httptransport.WithErrorEncoder(getErrorEncoder()),
		)...,
	)
}

func (h *entitlementHandler) resolveNamespace(ctx context.Context) (string, error) {
	ns, ok := h.namespaceDecoder.GetNamespace(ctx)
	if !ok {
		return "", commonhttp.NewHTTPError(http.StatusInternalServerError, errors.New("internal server error"))
	}

	return ns, nil
}
