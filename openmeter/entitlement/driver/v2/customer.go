package entitlementdriverv2

import (
	"context"
	"errors"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// CustomerEntitlementHandler exposes V2 customer entitlement endpoints
type CustomerEntitlementHandler interface {
	CreateCustomerEntitlement() CreateCustomerEntitlementHandler
	ListCustomerEntitlements() ListCustomerEntitlementsHandler
}

type customerEntitlementHandler struct {
	namespaceDecoder namespacedriver.NamespaceDecoder
	options          []httptransport.HandlerOption
	connector        entitlement.Connector
	customerService  customer.Service
}

func NewCustomerEntitlementHandler(
	connector entitlement.Connector,
	customerService customer.Service,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	options ...httptransport.HandlerOption,
) CustomerEntitlementHandler {
	return &customerEntitlementHandler{
		namespaceDecoder: namespaceDecoder,
		options:          options,
		connector:        connector,
		customerService:  customerService,
	}
}

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

			subjectKey, err := cus.UsageAttribution.GetSubjectKey()
			if err != nil {
				return request, commonhttp.NewHTTPError(http.StatusConflict, err)
			}

			// Reuse v1 parser to build entitlement create inputs using the subject key
			return entitlementdriver.ParseAPICreateInput(inp, ns, subjectKey)
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

func defaultIncludeDeleted(p *bool) bool { return lo.FromPtrOr(p, false) }

func (h *customerEntitlementHandler) resolveNamespace(ctx context.Context) (string, error) {
	ns, ok := h.namespaceDecoder.GetNamespace(ctx)
	if !ok {
		return "", commonhttp.NewHTTPError(http.StatusInternalServerError, errors.New("internal server error"))
	}

	return ns, nil
}
