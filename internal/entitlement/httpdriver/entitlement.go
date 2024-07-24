package httpdriver

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/internal/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

type EntitlementHandler interface {
	CreateEntitlement() CreateEntitlementHandler
	GetEntitlement() GetEntitlementHandler
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

			value, err := inp.ValueByDiscriminator()
			if err != nil {
				return request, err
			}

			switch v := value.(type) {
			case api.EntitlementMeteredCreateInputs:
				request = entitlement.CreateEntitlementInputs{
					Namespace:       ns,
					FeatureID:       v.FeatureId,
					FeatureKey:      v.FeatureKey,
					SubjectKey:      subjectIdOrKey,
					EntitlementType: entitlement.EntitlementTypeMetered,
					IsSoftLimit:     v.IsSoftLimit,
					IssueAfterReset: v.IssueAfterReset,
					IssueAfterResetPriority: convert.SafeDeRef(v.IssueAfterResetPriority, func(i int) *uint8 {
						return convert.ToPointer(uint8(i))
					}),
					UsagePeriod: &entitlement.UsagePeriod{
						Anchor:   defaultx.WithDefault(v.UsagePeriod.Anchor, clock.Now()), // TODO: shouldn't we truncate this?
						Interval: recurrence.RecurrenceInterval(v.UsagePeriod.Interval),
					},
				}
				if v.Metadata != nil {
					request.Metadata = *v.Metadata
				}
			case api.EntitlementStaticCreateInputs:
				request = entitlement.CreateEntitlementInputs{
					Namespace:       ns,
					FeatureID:       v.FeatureId,
					FeatureKey:      v.FeatureKey,
					SubjectKey:      subjectIdOrKey,
					EntitlementType: entitlement.EntitlementTypeStatic,
					Config:          []byte(v.Config),
				}
				if v.UsagePeriod != nil {
					request.UsagePeriod = &entitlement.UsagePeriod{
						Anchor:   defaultx.WithDefault(v.UsagePeriod.Anchor, clock.Now()), // TODO: shouldn't we truncate this?
						Interval: recurrence.RecurrenceInterval(v.UsagePeriod.Interval),
					}
				}
				if v.Metadata != nil {
					request.Metadata = *v.Metadata
				}
			case api.EntitlementBooleanCreateInputs:
				request = entitlement.CreateEntitlementInputs{
					Namespace:       ns,
					FeatureID:       v.FeatureId,
					FeatureKey:      v.FeatureKey,
					SubjectKey:      subjectIdOrKey,
					EntitlementType: entitlement.EntitlementTypeBoolean,
				}
				if v.UsagePeriod != nil {
					request.UsagePeriod = &entitlement.UsagePeriod{
						Anchor:   defaultx.WithDefault(v.UsagePeriod.Anchor, clock.Now()), // TODO: shouldn't we truncate this?
						Interval: recurrence.RecurrenceInterval(v.UsagePeriod.Interval),
					}
				}
				if v.Metadata != nil {
					request.Metadata = *v.Metadata
				}
			default:
				return request, errors.New("unknown entitlement type")
			}

			return request, nil
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
			httptransport.WithErrorEncoder(getErrorEncoder()),
		)...,
	)
}

type (
	ListEntitlementsHandlerRequest  = entitlement.ListEntitlementsParams
	ListEntitlementsHandlerResponse = []api.Entitlement
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

			p := entitlement.ListEntitlementsParams{
				Namespaces: []string{ns},
				Page: pagination.Page{
					PageSize:   defaultx.WithDefault(params.PageSize, commonhttp.DefaultPageSize),
					PageNumber: defaultx.WithDefault(params.Page, commonhttp.DefaultPage),
				},
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
			paged, err := h.connector.ListEntitlements(ctx, request)
			if err != nil {
				return nil, err
			}

			entitlements := paged.Items

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
			err := h.connector.DeleteEntitlement(ctx, request.Namespace, request.EntitlementId)
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
