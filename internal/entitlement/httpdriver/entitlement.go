package httpdriver

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/entitlement"
	booleanentitlement "github.com/openmeterio/openmeter/internal/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/internal/entitlement/metered"
	staticentitlement "github.com/openmeterio/openmeter/internal/entitlement/static"
	"github.com/openmeterio/openmeter/internal/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type EntitlementHandler interface {
	CreateEntitlement() CreateEntitlementHandler
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

type CreateEntitlementHandlerRequest = entitlement.CreateEntitlementInputs
type CreateEntitlementHandlerResponse = *api.Entitlement
type CreateEntitlementHandlerParams = string

type CreateEntitlementHandler httptransport.HandlerWithArgs[CreateEntitlementHandlerRequest, CreateEntitlementHandlerResponse, CreateEntitlementHandlerParams]

func (h *entitlementHandler) CreateEntitlement() CreateEntitlementHandler {
	return httptransport.NewHandlerWithArgs[CreateEntitlementHandlerRequest, CreateEntitlementHandlerResponse, string](
		func(ctx context.Context, r *http.Request, subjectIdOrKey string) (entitlement.CreateEntitlementInputs, error) {
			// TODO: we could use the API generated type here
			entitlement := entitlement.CreateEntitlementInputs{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &entitlement); err != nil {
				return entitlement, err
			}
			entitlement.SubjectKey = subjectIdOrKey

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return entitlement, err
			}
			entitlement.Namespace = ns

			return entitlement, nil
		},
		func(ctx context.Context, request CreateEntitlementHandlerRequest) (CreateEntitlementHandlerResponse, error) {
			res, err := h.connector.CreateEntitlement(ctx, entitlement.CreateEntitlementInputs(request))
			if err != nil {
				return nil, err
			}
			return Parser.ToAPIGeneric(res)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateEntitlementHandlerResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createEntitlement"),
			httptransport.WithErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter) bool {
				if _, ok := err.(*productcatalog.FeatureNotFoundError); ok {
					commonhttp.NewHTTPError(
						http.StatusNotFound,
						err,
					).EncodeError(ctx, w)
					return true
				}
				if _, ok := err.(*entitlement.NotFoundError); ok {
					commonhttp.NewHTTPError(
						http.StatusNotFound,
						err,
					).EncodeError(ctx, w)
					return true
				}
				if err, ok := err.(*entitlement.AlreadyExistsError); ok {
					commonhttp.NewHTTPError(
						http.StatusConflict,
						err,
						commonhttp.ExtendProblem("conflictingEntityId", err.EntitlementID),
					).EncodeError(ctx, w)
					return true
				}
				if err, ok := err.(*entitlement.InvalidValueError); ok {
					commonhttp.NewHTTPError(
						http.StatusBadRequest,
						err,
					).EncodeError(ctx, w)
					return true
				}
				return false
			}),
		)...,
	)
}

type GetEntitlementValueHandlerRequest struct {
	EntitlementIdOrFeatureKey string
	SubjectKey                string
	Namespace                 string
	At                        time.Time
}
type GetEntitlementValueHandlerResponse = api.EntitlementValue
type GetEntitlementValueHandlerParams struct {
	SubjectKey                string
	EntitlementIdOrFeatureKey string
	Params                    api.GetEntitlementValueParams
}
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
				At:                        defaultx.WithDefault(params.Params.Time, time.Now()),
			}, nil
		},
		func(ctx context.Context, request GetEntitlementValueHandlerRequest) (api.EntitlementValue, error) {
			entitlement, err := h.connector.GetEntitlementValue(ctx, request.Namespace, request.SubjectKey, request.EntitlementIdOrFeatureKey, request.At)
			if err != nil {
				return api.EntitlementValue{}, err
			}

			switch ent := entitlement.(type) {
			case *meteredentitlement.MeteredEntitlementValue:
				return api.EntitlementValue{
					HasAccess: convert.ToPointer(ent.HasAccess()),
					Balance:   &ent.Balance,
					Usage:     &ent.UsageInPeriod,
					Overage:   &ent.Overage,
				}, nil
			case *staticentitlement.StaticEntitlementValue:
				return api.EntitlementValue{
					HasAccess: convert.ToPointer(ent.HasAccess()),
					Config:    &ent.Config,
				}, nil
			case *booleanentitlement.BooleanEntitlementValue:
				return api.EntitlementValue{
					HasAccess: convert.ToPointer(ent.HasAccess()),
				}, nil
			default:
				return api.EntitlementValue{}, errors.New("unknown entitlement type")
			}

		},
		commonhttp.JSONResponseEncoder[api.EntitlementValue],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getEntitlementValue"),
			httptransport.WithErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter) bool {
				if _, ok := err.(*productcatalog.FeatureNotFoundError); ok {
					commonhttp.NewHTTPError(
						http.StatusNotFound,
						err,
					).EncodeError(ctx, w)
					return true
				}
				if _, ok := err.(*entitlement.NotFoundError); ok {
					commonhttp.NewHTTPError(
						http.StatusNotFound,
						err,
					).EncodeError(ctx, w)
					return true
				}
				return false
			}),
		)...,
	)
}

type GetEntitlementsOfSubjectHandlerRequest = models.NamespacedID
type GetEntitlementsOfSubjectHandlerResponse = []api.Entitlement
type GetEntitlementsOfSubjectHandlerParams struct {
	SubjectIdOrKey string
	Params         api.ListSubjectEntitlementsParams
}

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
				ID:        params.SubjectIdOrKey, // TODO: should work with ID as well & should use params.Params values
			}, nil
		},
		func(ctx context.Context, id GetEntitlementsOfSubjectHandlerRequest) (GetEntitlementsOfSubjectHandlerResponse, error) {
			entitlements, err := h.connector.GetEntitlementsOfSubject(ctx, id.Namespace, models.SubjectKey(id.ID))
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
		)...,
	)
}

type ListEntitlementsHandlerRequest = entitlement.ListEntitlementsParams
type ListEntitlementsHandlerResponse = []api.Entitlement
type ListEntitlementsHandlerParams = api.ListEntitlementsParams

type ListEntitlementsHandler httptransport.HandlerWithArgs[ListEntitlementsHandlerRequest, ListEntitlementsHandlerResponse, ListEntitlementsHandlerParams]

func (h *entitlementHandler) ListEntitlements() ListEntitlementsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListEntitlementsHandlerParams) (entitlement.ListEntitlementsParams, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return entitlement.ListEntitlementsParams{}, err
			}

			p := entitlement.ListEntitlementsParams{
				Namespace: ns,
				Limit:     defaultx.WithDefault(params.Limit, 1000),
				Offset:    defaultx.WithDefault(params.Offset, 0),
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
			entitlements, err := h.connector.ListEntitlements(ctx, request)
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
		commonhttp.JSONResponseEncoder[ListEntitlementsHandlerResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listEntitlements"),
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
