package entitlementdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
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
	connector        entitlement.Connector
}

func NewEntitlementHandler(
	connector entitlement.Connector,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	options ...httptransport.HandlerOption,
) EntitlementHandler {
	return &entitlementHandler{
		namespaceDecoder: namespaceDecoder,
		options:          options,
		connector:        connector,
	}
}

type (
	CreateEntitlementHandlerRequest  = entitlement.CreateEntitlementInputs
	CreateEntitlementHandlerResponse = *api.Entitlement
	CreateEntitlementHandlerParams   = string
)

type CreateEntitlementHandler httptransport.HandlerWithArgs[CreateEntitlementHandlerRequest, CreateEntitlementHandlerResponse, CreateEntitlementHandlerParams]

func (h *entitlementHandler) CreateEntitlement() CreateEntitlementHandler {
	return httptransport.NewHandlerWithArgs[CreateEntitlementHandlerRequest, CreateEntitlementHandlerResponse, string](
		func(ctx context.Context, r *http.Request, subjectIdOrKey string) (entitlement.CreateEntitlementInputs, error) {
			inp := &api.EntitlementCreateInputs{}
			request := entitlement.CreateEntitlementInputs{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &inp); err != nil {
				return request, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return request, err
			}

			return ParseAPICreateInput(inp, ns, subjectIdOrKey)
		},
		func(ctx context.Context, request CreateEntitlementHandlerRequest) (CreateEntitlementHandlerResponse, error) {
			res, err := h.connector.CreateEntitlement(ctx, request)
			if err != nil {
				return nil, err
			}
			return Parser.ToAPIGeneric(res)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateEntitlementHandlerResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createEntitlement"),
			httptransport.WithErrorEncoder(getErrorEncoder()),
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

			eInp, err := ParseAPICreateInput(inp, ns, params.SubjectIdOrKey)
			if err != nil {
				return request, err
			}

			request.Inputs = eInp
			request.SubjectIdOrKey = params.SubjectIdOrKey
			request.EntitlementIdOrFeatureKey = params.EntitlementIdOrFeatureKey

			return request, nil
		},
		func(ctx context.Context, request OverrideEntitlementHandlerRequest) (OverrideEntitlementHandlerResponse, error) {
			ent, err := h.connector.GetEntitlementOfSubjectAt(ctx, request.Inputs.Namespace, request.SubjectIdOrKey, request.EntitlementIdOrFeatureKey, clock.Now())
			if err != nil {
				return nil, err
			}

			if ent == nil {
				return nil, fmt.Errorf("unexpected nil entitlement")
			}

			if ent.SubscriptionManaged {
				return nil, &models.GenericForbiddenError{Inner: fmt.Errorf("entitlement is managed by subscription")}
			}

			res, err := h.connector.OverrideEntitlement(ctx, request.SubjectIdOrKey, request.EntitlementIdOrFeatureKey, request.Inputs)
			if err != nil {
				return nil, err
			}
			return Parser.ToAPIGeneric(res)
		},
		commonhttp.JSONResponseEncoderWithStatus[OverrideEntitlementHandlerResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("overrideEntitlement"),
			httptransport.WithErrorEncoder(getErrorEncoder()),
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
			entitlementValue, err := h.connector.GetEntitlementValue(ctx, request.Namespace, request.SubjectKey, request.EntitlementIdOrFeatureKey, request.At)
			if err != nil {
				return api.EntitlementValue{}, err
			}
			return MapEntitlementValueToAPI(entitlementValue)
		},
		commonhttp.JSONResponseEncoder[api.EntitlementValue],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getEntitlementValue"),
			httptransport.WithErrorEncoder(getErrorEncoder()),
		)...,
	)
}

type (
	GetEntitlementsOfSubjectHandlerRequest  = models.NamespacedID
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
				return models.NamespacedID{}, err
			}

			return models.NamespacedID{
				Namespace: ns,
				ID:        params.SubjectIdOrKey,
			}, nil
		},
		func(ctx context.Context, id GetEntitlementsOfSubjectHandlerRequest) (GetEntitlementsOfSubjectHandlerResponse, error) {
			entitlements, err := h.connector.GetEntitlementsOfSubject(ctx, id.Namespace, id.ID, clock.Now())
			if err != nil {
				return nil, err
			}

			res := make([]api.Entitlement, 0, len(entitlements))
			for _, e := range entitlements {
				ent, err := Parser.ToAPIGeneric(&e)
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
			httptransport.WithErrorEncoder(getErrorEncoder()),
		)...,
	)
}

type (
	ListEntitlementsHandlerRequest  = entitlement.ListEntitlementsParams
	ListEntitlementsHandlerResponse = commonhttp.Union[[]api.Entitlement, pagination.PagedResponse[api.Entitlement]]
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
				Option2: &pagination.PagedResponse[api.Entitlement]{},
			}
			paged, err := h.connector.ListEntitlements(ctx, request)
			if err != nil {
				return response, err
			}

			entitlements := paged.Items

			mapped := make([]api.Entitlement, 0, len(entitlements))
			for _, e := range entitlements {
				ent, err := Parser.ToAPIGeneric(&e)
				if err != nil {
					return response, err
				}
				mapped = append(mapped, *ent)
			}

			if request.Page.IsZero() {
				response.Option1 = &mapped
			} else {
				response.Option1 = nil
				response.Option2 = &pagination.PagedResponse[api.Entitlement]{
					Items:      mapped,
					TotalCount: paged.TotalCount,
					Page:       paged.Page,
				}
			}

			return response, nil
		},
		commonhttp.JSONResponseEncoder[ListEntitlementsHandlerResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listEntitlements"),
			httptransport.WithErrorEncoder(getErrorEncoder()),
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
			entitlement, err := h.connector.GetEntitlement(ctx, request.Namespace, request.EntitlementId)
			if err != nil {
				return nil, err
			}

			return Parser.ToAPIGeneric(entitlement)
		},
		commonhttp.JSONResponseEncoder[GetEntitlementHandlerResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getEntitlement"),
			httptransport.WithErrorEncoder(getErrorEncoder()),
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
			entitlement, err := h.connector.GetEntitlement(ctx, request.Namespace, request.EntitlementId)
			if err != nil {
				return nil, err
			}

			return Parser.ToAPIGeneric(entitlement)
		},
		commonhttp.JSONResponseEncoder[GetEntitlementByIdHandlerResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getEntitlement"),
			httptransport.WithErrorEncoder(getErrorEncoder()),
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
			ent, err := h.connector.GetEntitlement(ctx, request.Namespace, request.EntitlementId)
			if err != nil {
				return nil, err
			}

			if ent == nil {
				return nil, fmt.Errorf("unexpected nil entitlement")
			}

			if ent.SubscriptionManaged {
				return nil, &models.GenericForbiddenError{Inner: fmt.Errorf("entitlement is managed by subscription")}
			}

			err = h.connector.DeleteEntitlement(ctx, request.Namespace, request.EntitlementId, clock.Now())
			return nil, err
		},
		commonhttp.EmptyResponseEncoder[DeleteEntitlementHandlerResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("deleteEntitlement"),
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
