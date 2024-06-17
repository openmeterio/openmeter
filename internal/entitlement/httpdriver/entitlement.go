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
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type EntitlementHandler interface {
	CreateEntitlement() CreateEntitlementHandler
	GetEntitlementValue() GetEntitlementValueHandler
	GetEntitlementsOfSubjectHandler() GetEntitlementsOfSubjectHandler
}

type CreateEntitlementParams struct {
	SubjectIdOrKey string
}

type CreateEntitlementHandler httptransport.HandlerWithArgs[entitlement.CreateEntitlementInputs, api.EntitlementMetered, CreateEntitlementParams]

type GetEntitlementValueInputs struct {
	ID models.NamespacedID
	At time.Time
}
type GetEntitlementValueHandler httptransport.HandlerWithArgs[GetEntitlementValueInputs, api.EntitlementValue, EntitlementValueParams]

type GetEntitlementsOfSubjectHandler httptransport.HandlerWithArgs[models.NamespacedID, []api.EntitlementMetered, string]

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
	}
}

func (h *entitlementHandler) CreateEntitlement() CreateEntitlementHandler {
	return httptransport.NewHandlerWithArgs[entitlement.CreateEntitlementInputs, api.EntitlementMetered, CreateEntitlementParams](
		func(ctx context.Context, r *http.Request, arg CreateEntitlementParams) (entitlement.CreateEntitlementInputs, error) {
			entitlement := entitlement.CreateEntitlementInputs{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &entitlement); err != nil {
				return entitlement, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return entitlement, err
			}
			entitlement.Namespace = ns

			return entitlement, nil
		},
		func(ctx context.Context, request entitlement.CreateEntitlementInputs) (api.EntitlementMetered, error) {
			res, err := h.connector.CreateEntitlement(ctx, request)
			return api.EntitlementMetered{
				Id:         &res.ID,
				FeatureId:  res.FeatureID,
				CreatedAt:  &res.CreatedAt,
				UpdatedAt:  &res.UpdatedAt,
				DeletedAt:  res.DeletedAt,
				Subjectkey: &res.SubjectKey,
				Type:       "metered",
			}, err
		},
		// api.Entitlement is a pseuo type due to the openapi magic so it has no fields
		// FIXME: assert that the types are actually comptible...
		commonhttp.JSONResponseEncoder[api.EntitlementMetered],
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
				return false
			}),
		)...,
	)
}

type EntitlementValueParams struct {
	SubjectIdOrKey            string
	EntitlementIdOrFeatureKey string
	Params                    api.GetEntitlementValueParams
}

func (h *entitlementHandler) GetEntitlementValue() GetEntitlementValueHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params EntitlementValueParams) (GetEntitlementValueInputs, error) {
			// TODO: use subjectIdOrKey

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetEntitlementValueInputs{}, err
			}

			return GetEntitlementValueInputs{
				ID: models.NamespacedID{
					Namespace: ns,
					ID:        params.EntitlementIdOrFeatureKey,
				},
			}, nil
		},
		func(ctx context.Context, request GetEntitlementValueInputs) (api.EntitlementValue, error) {
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

func (h *entitlementHandler) GetEntitlementsOfSubjectHandler() GetEntitlementsOfSubjectHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, subjectIdOrKey string) (models.NamespacedID, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return models.NamespacedID{}, err
			}

			return models.NamespacedID{
				Namespace: ns,
				ID:        subjectIdOrKey,
			}, nil
		},
		func(ctx context.Context, id models.NamespacedID) ([]api.EntitlementMetered, error) {
			entitlements, err := h.connector.GetEntitlementsOfSubject(ctx, id.Namespace, models.SubjectKey(id.ID))
			if err != nil {
				return nil, err
			}

			res := make([]api.EntitlementMetered, len(entitlements))
			for i, ent := range entitlements {
				res[i] = api.EntitlementMetered{
					Id:         &ent.ID,
					FeatureId:  ent.FeatureID,
					CreatedAt:  &ent.CreatedAt,
					UpdatedAt:  &ent.UpdatedAt,
					DeletedAt:  ent.DeletedAt,
					Subjectkey: &ent.SubjectKey,
					Type:       "metered",
				}
			}

			return res, nil
		},
		commonhttp.JSONResponseEncoder[[]api.EntitlementMetered],
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
