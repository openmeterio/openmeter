package httpdriver

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/api/types"
	"github.com/openmeterio/openmeter/internal/credit"
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
	ListEntitlements() ListEntitlementsHandler
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

type APIEntitlementResponse = api.EntitlementMetered

type CreateEntitlementHandlerRequest = entitlement.CreateEntitlementInputs
type CreateEntitlementHandlerParams = string

type CreateEntitlementHandler httptransport.HandlerWithArgs[CreateEntitlementHandlerRequest, APIEntitlementResponse, CreateEntitlementHandlerParams]

func (h *entitlementHandler) CreateEntitlement() CreateEntitlementHandler {
	return httptransport.NewHandlerWithArgs[CreateEntitlementHandlerRequest, APIEntitlementResponse, string](
		func(ctx context.Context, r *http.Request, subjectIdOrKey string) (entitlement.CreateEntitlementInputs, error) {
			req := types.CreateEntitlementJSONBody{}
			// TODO: parse rest of the fields from the request (period, issuing, etc...)
			if err := commonhttp.JSONRequestBodyDecoder(r, &req); err != nil {
				return entitlement.CreateEntitlementInputs{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return entitlement.CreateEntitlementInputs{}, err
			}

			createRequest := entitlement.CreateEntitlementInputs{
				Namespace:  ns,
				FeatureID:  req.FeatureId,
				SubjectKey: subjectIdOrKey,
				UsagePeriod: entitlement.Recurrence{
					Period: credit.RecurrencePeriod(req.UsagePeriod.Interval),
					Anchor: req.UsagePeriod.Anchor,
				},
			}

			if !createRequest.UsagePeriod.Period.IsValid() {
				return createRequest, errors.New("invalid usage period")
			}

			return createRequest, nil
		},
		func(ctx context.Context, request CreateEntitlementHandlerRequest) (APIEntitlementResponse, error) {
			res, err := h.connector.CreateEntitlement(ctx, request)
			return mapEntitlementToAPI(res), err
		},
		commonhttp.JSONResponseEncoderWithStatus[APIEntitlementResponse](http.StatusCreated),
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
				if err, ok := err.(*entitlement.EntitlementAlreadyExistsError); ok {
					commonhttp.NewHTTPError(
						http.StatusConflict,
						err,
						commonhttp.ExtendProblem("conflictingEntityId", err.EntitlementID),
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
				if _, ok := err.(*entitlement.EntitlementNotFoundError); ok {
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
				res[i] = mapEntitlementToAPI(ent)
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

type ListEntitlementsHandlerRequest = entitlement.ListEntitlementsParams
type ListEntitlementsHandlerResponse = []APIEntitlementResponse
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
		func(ctx context.Context, request ListEntitlementsHandlerRequest) ([]APIEntitlementResponse, error) {
			entitlements, err := h.connector.ListEntitlements(ctx, request)
			if err != nil {
				return nil, err
			}

			res := make([]APIEntitlementResponse, len(entitlements))
			for i, ent := range entitlements {
				res[i] = mapEntitlementToAPI(ent)
			}

			return res, nil
		},
		commonhttp.JSONResponseEncoder[[]APIEntitlementResponse],
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

func mapEntitlementToAPI(ent entitlement.Entitlement) APIEntitlementResponse {
	return api.EntitlementMetered{
		Id:         &ent.ID,
		FeatureId:  ent.FeatureID,
		CreatedAt:  &ent.CreatedAt,
		UpdatedAt:  &ent.UpdatedAt,
		DeletedAt:  ent.DeletedAt,
		SubjectKey: ent.SubjectKey,
		Type:       "metered",
		UsagePeriod: types.RecurringPeriodWithNextReset{
			RecurringPeriod: types.RecurringPeriod{
				Interval: types.RecurringPeriodEnum(ent.UsagePeriod.Period),
				Anchor:   ent.UsagePeriod.Anchor,
			},
			NextReset: ent.UsagePeriod.NextReset,
		},
	}
}
