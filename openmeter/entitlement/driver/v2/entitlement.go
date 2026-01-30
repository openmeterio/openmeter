package entitlementdriverv2

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/strcase"
)

type (
	ListEntitlementsHandlerRequest  = entitlement.ListEntitlementsParams
	ListEntitlementsHandlerResponse = pagination.Result[api.EntitlementV2]
	ListEntitlementsHandlerParams   = api.ListEntitlementsV2Params
)

type ListEntitlementsHandler httptransport.HandlerWithArgs[ListEntitlementsHandlerRequest, ListEntitlementsHandlerResponse, ListEntitlementsHandlerParams]

func (h *entitlementHandler) ListEntitlements() ListEntitlementsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListEntitlementsHandlerParams) (entitlement.ListEntitlementsParams, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return entitlement.ListEntitlementsParams{}, err
			}

			// validate OrderBy
			if params.OrderBy != nil {
				if !slices.Contains(entitlement.ListEntitlementsOrderBy("").StrValues(), strcase.CamelToSnake(string(*params.OrderBy))) {
					return entitlement.ListEntitlementsParams{}, commonhttp.NewHTTPError(http.StatusBadRequest, errors.New("invalid order by"))
				}
			}

			// validate EntitlementType
			if params.EntitlementType != nil {
				ets := convert.DerefHeaderPtr[string](params.EntitlementType)
				for _, et := range ets {
					if !slices.Contains(entitlement.EntitlementType("").StrValues(), et) {
						return entitlement.ListEntitlementsParams{}, commonhttp.NewHTTPError(http.StatusBadRequest, errors.New("invalid entitlement type"))
					}
				}
			}

			p := entitlement.ListEntitlementsParams{
				ExcludeInactive: defaultx.WithDefault(params.ExcludeInactive, false),
				Namespaces:      []string{ns},
				Page: pagination.Page{
					PageSize:   defaultx.WithDefault(params.PageSize, commonhttp.DefaultPageSize),
					PageNumber: defaultx.WithDefault(params.Page, 1),
				},
				Limit:  defaultx.WithDefault(params.Limit, commonhttp.DefaultPageSize),
				Offset: defaultx.WithDefault(params.Offset, 0),
				OrderBy: func() entitlement.ListEntitlementsOrderBy {
					orderBy := entitlement.ListEntitlementsOrderByCreatedAt

					if params.OrderBy != nil {
						orderBy = entitlement.ListEntitlementsOrderBy(strcase.CamelToSnake(string(lo.FromPtr(params.OrderBy))))
					}

					return orderBy
				}(),
				Order:            commonhttp.GetSortOrder(api.SortOrderASC, params.Order),
				CustomerIDs:      convert.DerefHeaderPtr[string](params.CustomerIds),
				CustomerKeys:     convert.DerefHeaderPtr[string](params.CustomerKeys),
				FeatureIDsOrKeys: convert.DerefHeaderPtr[string](params.Feature),
				EntitlementTypes: slicesx.Map[string, entitlement.EntitlementType](convert.DerefHeaderPtr[string](params.EntitlementType), func(s string) entitlement.EntitlementType {
					return entitlement.EntitlementType(s)
				}),
			}
			if !p.Page.IsZero() {
				p.Page.PageNumber = defaultx.IfZero(p.Page.PageNumber, commonhttp.DefaultPage)
				p.Page.PageSize = defaultx.IfZero(p.Page.PageSize, commonhttp.DefaultPageSize)
			}

			switch defaultx.WithDefault(params.OrderBy, "") {
			case "createdAt":
				p.OrderBy = entitlement.ListEntitlementsOrderByCreatedAt
			case "updatedAt":
				p.OrderBy = entitlement.ListEntitlementsOrderByUpdatedAt
			default:
				p.OrderBy = entitlement.ListEntitlementsOrderByCreatedAt
			}

			return p, nil
		},
		func(ctx context.Context, request ListEntitlementsHandlerRequest) (ListEntitlementsHandlerResponse, error) {
			// due to backward compatibility, if pagination is not provided we return a simple array
			paged, err := h.connector.ListEntitlementsWithCustomer(ctx, request)
			if err != nil {
				return ListEntitlementsHandlerResponse{}, err
			}

			return pagination.MapResultErr(paged.Entitlements, func(e entitlement.Entitlement) (api.EntitlementV2, error) {
				cust, ok := paged.CustomersByID[models.NamespacedID{Namespace: e.Namespace, ID: e.CustomerID}]
				if !ok {
					return api.EntitlementV2{}, models.NewGenericPreConditionFailedError(fmt.Errorf("customer not found [namespace=%s customer.id=%s]", e.Namespace, e.CustomerID))
				}

				r, err := ParserV2.ToAPIGenericV2(&e, cust.ID, cust.Key)
				return lo.FromPtr(r), err
			})
		},
		commonhttp.JSONResponseEncoder[ListEntitlementsHandlerResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listEntitlementsV2"),
			httptransport.WithErrorEncoder(getErrorEncoder()),
		)...,
	)
}

type GetEntitlementHandlerRequest struct {
	EntitlementId string
	Namespace     string
}
type (
	GetEntitlementHandlerResponse = *api.EntitlementV2
	GetEntitlementHandlerParams   struct {
		EntitlementId string
	}
)
type GetEntitlementHandler httptransport.HandlerWithArgs[GetEntitlementHandlerRequest, GetEntitlementHandlerResponse, GetEntitlementHandlerParams]

func (h *entitlementHandler) GetEntitlement() GetEntitlementHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetEntitlementHandlerParams) (GetEntitlementHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetEntitlementHandlerRequest{}, err
			}

			return GetEntitlementHandlerRequest{
				EntitlementId: params.EntitlementId,
				Namespace:     ns,
			}, nil
		},
		func(ctx context.Context, request GetEntitlementHandlerRequest) (GetEntitlementHandlerResponse, error) {
			ent, err := h.connector.GetEntitlementWithCustomer(ctx, request.Namespace, request.EntitlementId)
			if err != nil {
				return nil, err
			}

			return ParserV2.ToAPIGenericV2(&ent.Entitlement, ent.Customer.ID, ent.Customer.Key)
		},
		commonhttp.JSONResponseEncoder[GetEntitlementHandlerResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getEntitlementByIdV2"),
			httptransport.WithErrorEncoder(getErrorEncoder()),
		)...,
	)
}
