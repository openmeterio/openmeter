package entitlementdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/strcase"
)

type EntitlementHandler interface {
	CreateEntitlement() CreateEntitlementHandler
	OverrideEntitlement() OverrideEntitlementHandler // Maybe rename (both here & API) to Supersede
	GetEntitlement() GetEntitlementHandler
	GetEntitlementById() GetEntitlementByIdHandler
	DeleteEntitlement() DeleteEntitlementHandler
	GetEntitlementValue() GetEntitlementValueHandler
	GetEntitlementsOfSubjectHandler() GetEntitlementsOfSubjectHandler
	ListEntitlements() ListEntitlementsHandler
}

type entitlementHandler struct {
	namespaceDecoder namespacedriver.NamespaceDecoder
	options          []httptransport.HandlerOption
	connector        entitlement.Service
	customerService  customer.Service
	subjectService   subject.Service
}

func NewEntitlementHandler(
	connector entitlement.Service,
	customerService customer.Service,
	subjectService subject.Service,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	options ...httptransport.HandlerOption,
) EntitlementHandler {
	return &entitlementHandler{
		namespaceDecoder: namespaceDecoder,
		options:          options,
		connector:        connector,
		customerService:  customerService,
		subjectService:   subjectService,
	}
}

type (
	CreateEntitlementHandlerRequest = struct {
		Inputs         entitlement.CreateEntitlementInputs
		SubjectIdOrKey string
		Namespace      string
	}
	CreateEntitlementHandlerResponse = *api.Entitlement
	CreateEntitlementHandlerParams   = string
)

type CreateEntitlementHandler httptransport.HandlerWithArgs[CreateEntitlementHandlerRequest, CreateEntitlementHandlerResponse, CreateEntitlementHandlerParams]

func (h *entitlementHandler) CreateEntitlement() CreateEntitlementHandler {
	return httptransport.NewHandlerWithArgs[CreateEntitlementHandlerRequest, CreateEntitlementHandlerResponse, string](
		func(ctx context.Context, r *http.Request, subjectIdOrKey string) (CreateEntitlementHandlerRequest, error) {
			inp := &api.EntitlementCreateInputs{}
			request := CreateEntitlementHandlerRequest{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &inp); err != nil {
				return request, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return request, err
			}

			// We'll populate the usageattribution in the handler down below, this is somewhat hacky
			createInput, err := ParseAPICreateInput(inp, ns, streaming.CustomerUsageAttribution{})
			if err != nil {
				return request, err
			}

			request.Inputs = createInput
			request.SubjectIdOrKey = subjectIdOrKey
			request.Namespace = ns

			return request, nil
		},
		func(ctx context.Context, request CreateEntitlementHandlerRequest) (CreateEntitlementHandlerResponse, error) {
			cust, err := h.resolveCustomerFromSubject(ctx, request.Namespace, request.SubjectIdOrKey)
			if err != nil {
				return nil, err
			}

			request.Inputs.UsageAttribution = cust.GetUsageAttribution()

			res, err := h.connector.CreateEntitlement(ctx, request.Inputs, nil)
			if err != nil {
				return nil, err
			}
			return Parser.ToAPIGeneric(&entitlement.EntitlementWithCustomer{Entitlement: lo.FromPtr(res), Customer: *cust})
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateEntitlementHandlerResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createEntitlement"),
			httptransport.WithErrorEncoder(GetErrorEncoder()),
		)...,
	)
}

type (
	OverrideEntitlementHandlerRequest struct {
		Inputs                    entitlement.CreateEntitlementInputs
		SubjectIdOrKey            string
		EntitlementIdOrFeatureKey string
	}
	OverrideEntitlementHandlerResponse = *api.Entitlement
	OverrideEntitlementHandlerParams   struct {
		SubjectIdOrKey            string
		EntitlementIdOrFeatureKey string
	}
)

type OverrideEntitlementHandler httptransport.HandlerWithArgs[OverrideEntitlementHandlerRequest, OverrideEntitlementHandlerResponse, OverrideEntitlementHandlerParams]

func (h *entitlementHandler) OverrideEntitlement() OverrideEntitlementHandler {
	return httptransport.NewHandlerWithArgs[OverrideEntitlementHandlerRequest, OverrideEntitlementHandlerResponse, OverrideEntitlementHandlerParams](
		func(ctx context.Context, r *http.Request, params OverrideEntitlementHandlerParams) (OverrideEntitlementHandlerRequest, error) {
			inp := &api.EntitlementCreateInputs{}
			request := OverrideEntitlementHandlerRequest{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &inp); err != nil {
				return request, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return request, err
			}

			// We'll populate the usageattribution in the handler down below, this is somewhat hacky
			eInp, err := ParseAPICreateInput(inp, ns, streaming.CustomerUsageAttribution{})
			if err != nil {
				return request, err
			}

			request.Inputs = eInp
			request.SubjectIdOrKey = params.SubjectIdOrKey
			request.EntitlementIdOrFeatureKey = params.EntitlementIdOrFeatureKey

			return request, nil
		},
		func(ctx context.Context, request OverrideEntitlementHandlerRequest) (OverrideEntitlementHandlerResponse, error) {
			cust, err := h.resolveCustomerFromSubject(ctx, request.Inputs.Namespace, request.SubjectIdOrKey)
			if err != nil {
				return nil, err
			}

			request.Inputs.UsageAttribution = cust.GetUsageAttribution()

			res, err := h.connector.OverrideEntitlement(ctx, cust.ID, request.EntitlementIdOrFeatureKey, request.Inputs, nil)
			if err != nil {
				return nil, err
			}
			return Parser.ToAPIGeneric(&entitlement.EntitlementWithCustomer{Entitlement: lo.FromPtr(res), Customer: *cust})
		},
		commonhttp.JSONResponseEncoderWithStatus[OverrideEntitlementHandlerResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("overrideEntitlement"),
			httptransport.WithErrorEncoder(GetErrorEncoder()),
		)...,
	)
}

type GetEntitlementValueHandlerRequest struct {
	EntitlementIdOrFeatureKey string
	SubjectKey                string
	Namespace                 string
	At                        time.Time
}
type (
	GetEntitlementValueHandlerResponse = api.EntitlementValue
	GetEntitlementValueHandlerParams   struct {
		SubjectKey                string
		EntitlementIdOrFeatureKey string
		Params                    api.GetEntitlementValueParams
	}
)
type GetEntitlementValueHandler httptransport.HandlerWithArgs[GetEntitlementValueHandlerRequest, GetEntitlementValueHandlerResponse, GetEntitlementValueHandlerParams]

func (h *entitlementHandler) GetEntitlementValue() GetEntitlementValueHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetEntitlementValueHandlerParams) (GetEntitlementValueHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetEntitlementValueHandlerRequest{}, err
			}

			return GetEntitlementValueHandlerRequest{
				SubjectKey:                params.SubjectKey,
				EntitlementIdOrFeatureKey: params.EntitlementIdOrFeatureKey,
				Namespace:                 ns,
				At:                        defaultx.WithDefault(params.Params.Time, clock.Now()),
			}, nil
		},
		func(ctx context.Context, request GetEntitlementValueHandlerRequest) (api.EntitlementValue, error) {
			cust, err := h.resolveCustomerFromSubject(ctx, request.Namespace, request.SubjectKey)
			if err != nil {
				return api.EntitlementValue{}, err
			}

			entitlementValue, err := h.connector.GetEntitlementValue(ctx, request.Namespace, cust.ID, request.EntitlementIdOrFeatureKey, request.At)
			if err != nil {
				return api.EntitlementValue{}, err
			}
			return MapEntitlementValueToAPI(entitlementValue)
		},
		commonhttp.JSONResponseEncoder[api.EntitlementValue],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getEntitlementValue"),
			httptransport.WithErrorEncoder(GetErrorEncoder()),
		)...,
	)
}

type (
	GetEntitlementsOfSubjectHandlerRequest = struct {
		Namespace      string
		SubjectIdOrKey string
		Params         api.ListSubjectEntitlementsParams
	}
	GetEntitlementsOfSubjectHandlerResponse = []api.Entitlement
	GetEntitlementsOfSubjectHandlerParams   struct {
		SubjectIdOrKey string
		Params         api.ListSubjectEntitlementsParams
	}
)

type GetEntitlementsOfSubjectHandler httptransport.HandlerWithArgs[GetEntitlementsOfSubjectHandlerRequest, GetEntitlementsOfSubjectHandlerResponse, GetEntitlementsOfSubjectHandlerParams]

func (h *entitlementHandler) GetEntitlementsOfSubjectHandler() GetEntitlementsOfSubjectHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetEntitlementsOfSubjectHandlerParams) (GetEntitlementsOfSubjectHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetEntitlementsOfSubjectHandlerRequest{}, err
			}

			return GetEntitlementsOfSubjectHandlerRequest{
				Namespace:      ns,
				SubjectIdOrKey: params.SubjectIdOrKey,
			}, nil
		},
		func(ctx context.Context, id GetEntitlementsOfSubjectHandlerRequest) (GetEntitlementsOfSubjectHandlerResponse, error) {
			cust, err := h.resolveCustomerFromSubject(ctx, id.Namespace, id.SubjectIdOrKey)
			if err != nil {
				return nil, err
			}

			if cust != nil && cust.IsDeleted() {
				return GetEntitlementsOfSubjectHandlerResponse{}, models.NewGenericPreConditionFailedError(
					fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cust.Namespace, cust.ID),
				)
			}

			if cust == nil {
				return GetEntitlementsOfSubjectHandlerResponse{}, models.NewGenericPreConditionFailedError(
					fmt.Errorf("customer not found [namespace=%s subject.id=%s]", id.Namespace, id.SubjectIdOrKey),
				)
			}

			now := clock.Now()

			ents, err := h.connector.ListEntitlements(ctx, entitlement.ListEntitlementsParams{
				CustomerIDs:    []string{cust.ID},
				Namespaces:     []string{id.Namespace},
				ActiveAt:       lo.ToPtr(now),
				IncludeDeleted: lo.FromPtr(id.Params.IncludeDeleted),
			})
			if err != nil {
				return nil, err
			}

			entitlements := ents.Items

			res := make([]api.Entitlement, 0, len(entitlements))
			for _, e := range entitlements {
				ent, err := Parser.ToAPIGeneric(&entitlement.EntitlementWithCustomer{Entitlement: e, Customer: *cust})
				if err != nil {
					return nil, err
				}
				res = append(res, *ent)
			}

			return res, nil
		},
		commonhttp.JSONResponseEncoder[GetEntitlementsOfSubjectHandlerResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getEntitlementsOfSubject"),
			httptransport.WithErrorEncoder(GetErrorEncoder()),
		)...,
	)
}

type (
	ListEntitlementsHandlerRequest  = entitlement.ListEntitlementsParams
	ListEntitlementsHandlerResponse = commonhttp.Union[[]api.Entitlement, pagination.Result[api.Entitlement]]
	ListEntitlementsHandlerParams   = api.ListEntitlementsParams
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
					PageSize:   defaultx.WithDefault(params.PageSize, 0),
					PageNumber: defaultx.WithDefault(params.Page, 0),
				},
				Limit:  defaultx.WithDefault(params.Limit, commonhttp.DefaultPageSize),
				Offset: defaultx.WithDefault(params.Offset, 0),
				OrderBy: entitlement.ListEntitlementsOrderBy(
					strcase.CamelToSnake(defaultx.WithDefault((*string)(params.OrderBy), string(entitlement.ListEntitlementsOrderByCreatedAt))),
				),
				Order:            commonhttp.GetSortOrder(api.SortOrderASC, params.Order),
				SubjectKeys:      convert.DerefHeaderPtr[string](params.Subject),
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
			response := ListEntitlementsHandlerResponse{
				Option1: &[]api.Entitlement{},
				Option2: &pagination.Result[api.Entitlement]{},
			}
			paged, err := h.connector.ListEntitlementsWithCustomer(ctx, request)
			if err != nil {
				return response, err
			}

			entitlements := paged.Entitlements.Items

			mapped := make([]api.Entitlement, 0, len(entitlements))
			for _, e := range entitlements {
				cust, ok := paged.CustomersByID[models.NamespacedID{Namespace: e.Namespace, ID: e.CustomerID}]
				if !ok || cust == nil {
					return response, models.NewGenericPreConditionFailedError(fmt.Errorf("customer not found [namespace=%s customer.id=%s]", e.Namespace, e.CustomerID))
				}

				ent, err := Parser.ToAPIGeneric(&entitlement.EntitlementWithCustomer{Entitlement: e, Customer: *cust})
				if err != nil {
					return response, err
				}
				mapped = append(mapped, *ent)
			}

			if request.Page.IsZero() {
				response.Option1 = &mapped
			} else {
				response.Option1 = nil
				response.Option2 = &pagination.Result[api.Entitlement]{
					Items:      mapped,
					TotalCount: paged.Entitlements.TotalCount,
					Page:       paged.Entitlements.Page,
				}
			}

			return response, nil
		},
		commonhttp.JSONResponseEncoder[ListEntitlementsHandlerResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listEntitlements"),
			httptransport.WithErrorEncoder(GetErrorEncoder()),
		)...,
	)
}

type GetEntitlementHandlerRequest struct {
	EntitlementId string
	Namespace     string
}
type (
	GetEntitlementHandlerResponse = *api.Entitlement
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
			entitlement, err := h.connector.GetEntitlementWithCustomer(ctx, request.Namespace, request.EntitlementId)
			if err != nil {
				return nil, err
			}

			return Parser.ToAPIGeneric(entitlement)
		},
		commonhttp.JSONResponseEncoder[GetEntitlementHandlerResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getEntitlement"),
			httptransport.WithErrorEncoder(GetErrorEncoder()),
		)...,
	)
}

type GetEntitlementByIdHandlerRequest struct {
	EntitlementId string
	Namespace     string
}
type (
	GetEntitlementByIdHandlerResponse = *api.Entitlement
	GetEntitlementByIdHandlerParams   struct {
		EntitlementId string
	}
)
type GetEntitlementByIdHandler httptransport.HandlerWithArgs[GetEntitlementByIdHandlerRequest, GetEntitlementByIdHandlerResponse, GetEntitlementByIdHandlerParams]

func (h *entitlementHandler) GetEntitlementById() GetEntitlementByIdHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetEntitlementByIdHandlerParams) (GetEntitlementByIdHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetEntitlementByIdHandlerRequest{}, err
			}

			return GetEntitlementByIdHandlerRequest{
				EntitlementId: params.EntitlementId,
				Namespace:     ns,
			}, nil
		},
		func(ctx context.Context, request GetEntitlementByIdHandlerRequest) (GetEntitlementByIdHandlerResponse, error) {
			entitlement, err := h.connector.GetEntitlementWithCustomer(ctx, request.Namespace, request.EntitlementId)
			if err != nil {
				return nil, err
			}

			return Parser.ToAPIGeneric(entitlement)
		},
		commonhttp.JSONResponseEncoder[GetEntitlementByIdHandlerResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getEntitlement"),
			httptransport.WithErrorEncoder(GetErrorEncoder()),
		)...,
	)
}

type DeleteEntitlementHandlerRequest struct {
	EntitlementId string
	Namespace     string
}
type (
	DeleteEntitlementHandlerResponse = interface{}
	DeleteEntitlementHandlerParams   struct {
		EntitlementId string
	}
)
type DeleteEntitlementHandler httptransport.HandlerWithArgs[DeleteEntitlementHandlerRequest, DeleteEntitlementHandlerResponse, DeleteEntitlementHandlerParams]

func (h *entitlementHandler) DeleteEntitlement() DeleteEntitlementHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params DeleteEntitlementHandlerParams) (DeleteEntitlementHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteEntitlementHandlerRequest{}, err
			}

			return DeleteEntitlementHandlerRequest{
				EntitlementId: params.EntitlementId,
				Namespace:     ns,
			}, nil
		},
		func(ctx context.Context, request DeleteEntitlementHandlerRequest) (DeleteEntitlementHandlerResponse, error) {
			err := h.connector.DeleteEntitlement(ctx, request.Namespace, request.EntitlementId, clock.Now())
			return nil, err
		},
		commonhttp.EmptyResponseEncoder[DeleteEntitlementHandlerResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("deleteEntitlement"),
			httptransport.WithErrorEncoder(GetErrorEncoder()),
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

func (h *entitlementHandler) resolveCustomerFromSubject(ctx context.Context, namespace string, subjectIdOrKey string) (*customer.Customer, error) {
	subj, err := h.subjectService.GetByIdOrKey(ctx, namespace, subjectIdOrKey)
	if err != nil {
		return nil, err
	}

	if subj.IsDeleted() {
		return nil, models.NewGenericPreConditionFailedError(
			fmt.Errorf("subject is deleted [namespace: %s, subject: %s]", namespace, subjectIdOrKey),
		)
	}

	cust, err := h.customerService.GetCustomerByUsageAttribution(ctx, customer.GetCustomerByUsageAttributionInput{
		Namespace: namespace,
		Key:       subj.Key,
	})
	if err != nil {
		return nil, err
	}

	return cust, nil
}
