package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/operation"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type FeatureHandler interface {
	GetFeature() GetFeatureHandler
	CreateFeature() CreateFeatureHandler
	ListFeatures() ListFeaturesHandler
	DeleteFeature() DeleteFeatureHandler
}

type GetFeatureHandler httptransport.HandlerWithArgs[models.NamespacedID, productcatalog.Feature, api.FeatureID]

type CreateFeatureHandler httptransport.Handler[productcatalog.CreateFeatureInputs, productcatalog.Feature]

type ListFeaturesHandler httptransport.HandlerWithArgs[productcatalog.ListFeaturesParams, []productcatalog.Feature, api.ListFeaturesParams]

type DeleteFeatureHandler httptransport.HandlerWithArgs[models.NamespacedID, any, api.FeatureID]

type featureHandlers struct {
	namespaceDecoder namespacedriver.NamespaceDecoder
	options          []httptransport.HandlerOption
	connector        productcatalog.FeatureConnector
}

func NewFeatureHandler(
	connector productcatalog.FeatureConnector,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	options ...httptransport.HandlerOption,
) FeatureHandler {
	return &featureHandlers{
		namespaceDecoder: namespaceDecoder,
		options:          options,
		connector:        connector,
	}
}

func (h *featureHandlers) GetFeature() GetFeatureHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, featureID api.FeatureID) (models.NamespacedID, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return models.NamespacedID{}, err
			}

			return models.NamespacedID{
				Namespace: ns,
				ID:        featureID,
			}, nil
		},
		func(ctx context.Context, featureId models.NamespacedID) (productcatalog.Feature, error) {
			return h.connector.GetFeature(ctx, featureId)
		},
		commonhttp.JSONResponseEncoder,
		httptransport.AppendOptions(
			h.options,
			httptransport.WithErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter) bool {
				if _, ok := err.(*productcatalog.FeatureNotFoundError); ok {
					models.NewStatusProblem(ctx, err, http.StatusNotFound).Respond(w)
					return true
				}
				return false
			}),
			httptransport.WithOperationName("getFeature"),
		)...,
	)
}

func (h *featureHandlers) CreateFeature() CreateFeatureHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (productcatalog.CreateFeatureInputs, error) {
			featureIn := productcatalog.CreateFeatureInputs{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &featureIn); err != nil {
				return featureIn, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return featureIn, err
			}

			featureIn.Namespace = ns

			return featureIn, nil
		},
		func(ctx context.Context, feature productcatalog.CreateFeatureInputs) (productcatalog.Feature, error) {
			return h.connector.CreateFeature(ctx, feature)
		},
		commonhttp.JSONResponseEncoderWithStatus[productcatalog.Feature](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createFeature"),
			httptransport.WithErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter) bool {
				if _, ok := err.(*productcatalog.FeatureInvalidFiltersError); ok {
					models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w)
					return true
				}
				if _, ok := err.(*productcatalog.FeatureInvalidMeterAggregationError); ok {
					models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w)
					return true
				}
				if _, ok := err.(*models.MeterNotFoundError); ok {
					models.NewStatusProblem(ctx, err, http.StatusNotFound).Respond(w)
					return true
				}
				if _, ok := err.(*productcatalog.FeatureWithNameAlreadyExistsError); ok {
					models.NewStatusProblem(ctx, err, http.StatusConflict).Respond(w)
					return true
				}
				return false
			}),
		)...,
	)
}

func (h *featureHandlers) ListFeatures() ListFeaturesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, apiParams api.ListFeaturesParams) (productcatalog.ListFeaturesParams, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return productcatalog.ListFeaturesParams{}, err
			}
			params := productcatalog.ListFeaturesParams{
				Namespace:       ns,
				IncludeArchived: defaultx.WithDefault(apiParams.IncludeArchived, false),
				Offset:          defaultx.WithDefault(apiParams.Offset, 0),
				Limit:           defaultx.WithDefault(apiParams.Limit, 0),
				OrderBy:         defaultx.WithDefault((*productcatalog.FeatureOrderBy)(apiParams.OrderBy), productcatalog.FeatureOrderByUpdatedAt),
			}

			// TODO: standardize
			if params.Limit > 1000 {
				return params, commonhttp.NewHTTPError(
					http.StatusBadRequest,
					fmt.Errorf("limit must be less than or equal to %d", 1000),
				)
			}

			return params, nil
		},
		func(ctx context.Context, params productcatalog.ListFeaturesParams) ([]productcatalog.Feature, error) {
			return h.connector.ListFeatures(ctx, params)
		},
		commonhttp.JSONResponseEncoder,
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listFeatures"),
		)...,
	)
}

func (h *featureHandlers) DeleteFeature() DeleteFeatureHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, featureID api.FeatureID) (models.NamespacedID, error) {
			id := models.NamespacedID{
				ID: featureID,
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return id, err
			}

			id.Namespace = ns

			return id, nil
		},
		operation.AsNoResponseOperation(h.connector.ArchiveFeature),
		func(ctx context.Context, w http.ResponseWriter, response any) error {
			w.WriteHeader(http.StatusNoContent)
			return nil
		},
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("deleteFeature"),
			httptransport.WithErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter) bool {
				if _, ok := err.(*productcatalog.FeatureNotFoundError); ok {
					models.NewStatusProblem(ctx, err, http.StatusNotFound).Respond(w)
					return true
				}
				return false
			}),
		)...,
	)
}

func (h *featureHandlers) resolveNamespace(ctx context.Context) (string, error) {
	ns, ok := h.namespaceDecoder.GetNamespace(ctx)
	if !ok {
		return "", commonhttp.NewHTTPError(http.StatusInternalServerError, errors.New("internal server error"))
	}

	return ns, nil
}
