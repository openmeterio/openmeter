package httpdriver

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/internal/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type EntitlementHandler interface {
	CreateEntitlement() CreateEntitlementHandler
	GetEntitlementValue() GetEntitlementValueHandler
	GetEntitlementsOfSubjectHandler() GetEntitlementsOfSubjectHandler
}

type entitlementHandler struct {
	namespaceDecoder namespacedriver.NamespaceDecoder
	options          []httptransport.HandlerOption
	connector        entitlement.EntitlementConnector
}

func NewEntitlementHandler(
	connector entitlement.EntitlementConnector,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	options ...httptransport.HandlerOption,
) EntitlementHandler {
	return &entitlementHandler{
		namespaceDecoder: namespaceDecoder,
		options:          options,
		connector:        connector,
	}
}

// The generated api.EntitlementMetered type doesn't really follow our openapi spec
// so we have to manually override some fields...
// FIXME: APIs can drift due to this
type APIEntitlementResponse struct {
	api.EntitlementMetered

	UsagePeriod *api.RecurringPeriod `json:"usagePeriod,omitempty"`
}

type CreateEntitlementHandlerRequest = entitlement.CreateEntitlementInputs
type CreateEntitlementHandlerResponse = APIEntitlementResponse
type CreateEntitlementHandlerParams = string

type CreateEntitlementHandler httptransport.HandlerWithArgs[CreateEntitlementHandlerRequest, CreateEntitlementHandlerResponse, CreateEntitlementHandlerParams]

func (h *entitlementHandler) CreateEntitlement() CreateEntitlementHandler {
	return httptransport.NewHandlerWithArgs[CreateEntitlementHandlerRequest, CreateEntitlementHandlerResponse, string](
		func(ctx context.Context, r *http.Request, subjectIdOrKey string) (entitlement.CreateEntitlementInputs, error) {
			entitlement := entitlement.CreateEntitlementInputs{}
			// TODO: parse rest of the fields from the request (period, issuing, etc...)
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
		func(ctx context.Context, request CreateEntitlementHandlerRequest) (APIEntitlementResponse, error) {
			res, err := h.connector.CreateEntitlement(ctx, request)
			return APIEntitlementResponse{
				EntitlementMetered: api.EntitlementMetered{
					Id:         &res.ID,
					FeatureId:  res.FeatureID,
					CreatedAt:  &res.CreatedAt,
					UpdatedAt:  &res.UpdatedAt,
					DeletedAt:  res.DeletedAt,
					Subjectkey: &res.SubjectKey,
					Type:       "metered",
				},
				UsagePeriod: nil,
			}, err
		},
		// api.Entitlement is a pseuo type due to the openapi magic so it has no fields
		// FIXME: assert that the types are actually comptible...
		commonhttp.JSONResponseEncoder[APIEntitlementResponse],
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
				if _, ok := err.(*entitlement.EntitlementNotFoundError); ok {
					commonhttp.NewHTTPError(
						http.StatusNotFound,
						err,
					).EncodeError(ctx, w)
					return true
				}
				if _, ok := err.(*entitlement.EntitlementAlreadyExistsError); ok {
					commonhttp.NewHTTPError(
						http.StatusConflict,
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
	ID models.NamespacedID
	At time.Time
}
type GetEntitlementValueHandlerResponse = api.EntitlementValue
type GetEntitlementValueHandlerParams struct {
	SubjectIdOrKey            string
	EntitlementIdOrFeatureKey string
	Params                    api.GetEntitlementValueParams
}
type GetEntitlementValueHandler httptransport.HandlerWithArgs[GetEntitlementValueHandlerRequest, GetEntitlementValueHandlerResponse, GetEntitlementValueHandlerParams]

func (h *entitlementHandler) GetEntitlementValue() GetEntitlementValueHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetEntitlementValueHandlerParams) (GetEntitlementValueHandlerRequest, error) {
			// TODO: use subjectIdOrKey

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetEntitlementValueHandlerRequest{}, err
			}

			return GetEntitlementValueHandlerRequest{
				ID: models.NamespacedID{
					Namespace: ns,
					ID:        params.EntitlementIdOrFeatureKey,
				},
				At: defaultx.WithDefault(params.Params.Time, time.Now()),
			}, nil
		},
		func(ctx context.Context, request GetEntitlementValueHandlerRequest) (api.EntitlementValue, error) {
			entitlement, err := h.connector.GetEntitlementValue(ctx, models.NamespacedID{
				Namespace: request.ID.Namespace,
				ID:        request.ID.ID,
			}, request.At)
			if err != nil {
				return api.EntitlementValue{}, err
			}

			return api.EntitlementValue{
				HasAccess: &entitlement.HasAccess,
				Balance:   &entitlement.Balance,
				Usage:     &entitlement.Usage,
				Overage:   &entitlement.Overage,
			}, nil
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
				return false
			}),
		)...,
	)
}

type GetEntitlementsOfSubjectHandlerRequest = models.NamespacedID
type GetEntitlementsOfSubjectHandlerResponse = []APIEntitlementResponse
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
		func(ctx context.Context, id GetEntitlementsOfSubjectHandlerRequest) ([]APIEntitlementResponse, error) {
			entitlements, err := h.connector.GetEntitlementsOfSubject(ctx, id.Namespace, models.SubjectKey(id.ID))
			if err != nil {
				return nil, err
			}

			res := make([]APIEntitlementResponse, len(entitlements))
			for i, ent := range entitlements {
				res[i] = APIEntitlementResponse{
					EntitlementMetered: api.EntitlementMetered{
						Id:         &ent.ID,
						FeatureId:  ent.FeatureID,
						CreatedAt:  &ent.CreatedAt,
						UpdatedAt:  &ent.UpdatedAt,
						DeletedAt:  ent.DeletedAt,
						Subjectkey: &ent.SubjectKey,
						Type:       "metered",
					},
					UsagePeriod: nil,
				}
			}

			return res, nil
		},
		commonhttp.JSONResponseEncoder[[]APIEntitlementResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getEntitlementsOfSubject"),
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
