package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/pkg/convert"
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

type GetFeatureHandlerRequest = models.NamespacedID
type GetFeatureHandlerResponse = *productcatalog.Feature
type GetFeatureHandlerParams = string

type GetFeatureHandler httptransport.HandlerWithArgs[GetFeatureHandlerRequest, GetFeatureHandlerResponse, GetFeatureHandlerParams]

func (h *featureHandlers) GetFeature() GetFeatureHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, featureID string) (GetFeatureHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return models.NamespacedID{}, err
			}

			return models.NamespacedID{
				Namespace: ns,
				ID:        featureID,
			}, nil
		},
		func(ctx context.Context, featureId GetFeatureHandlerRequest) (GetFeatureHandlerResponse, error) {
			return h.connector.GetFeature(ctx, featureId.Namespace, featureId.ID)
		},
		commonhttp.JSONResponseEncoder,
		httptransport.AppendOptions(
			h.options,
			httptransport.WithErrorEncoder(getErrorEncoder()),
			httptransport.WithOperationName("getFeature"),
		)...,
	)
}

type CreateFeatureHandlerRequest = productcatalog.CreateFeatureInputs
type CreateFeatureHandlerResponse = productcatalog.Feature

type CreateFeatureHandler httptransport.Handler[CreateFeatureHandlerRequest, CreateFeatureHandlerResponse]

func (h *featureHandlers) CreateFeature() CreateFeatureHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (productcatalog.CreateFeatureInputs, error) {
			parsedBody := api.CreateFeatureJSONRequestBody{}
			emptyFeature := productcatalog.CreateFeatureInputs{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &parsedBody); err != nil {
				return emptyFeature, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return emptyFeature, err
			}

			return productcatalog.CreateFeatureInputs{
				Namespace:           ns,
				Name:                parsedBody.Name,
				Key:                 parsedBody.Key,
				MeterIdOrSlug:       parsedBody.MeterIdOrSlug,
				MeterGroupByFilters: convert.DerefHeaderPtr[string](parsedBody.MeterGroupByFilters),
				Metadata:            convert.DerefHeaderPtr[string](parsedBody.Metadata),
			}, nil
		},
		func(ctx context.Context, feature productcatalog.CreateFeatureInputs) (productcatalog.Feature, error) {
			return h.connector.CreateFeature(ctx, feature)
		},
		commonhttp.JSONResponseEncoderWithStatus[productcatalog.Feature](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createFeature"),
			httptransport.WithErrorEncoder(getErrorEncoder()),
		)...,
	)
}

type ListFeaturesHandlerRequest = productcatalog.ListFeaturesParams
type ListFeaturesHandlerResponse = []productcatalog.Feature
type ListFeaturesHandlerParams = api.ListFeaturesParams

type ListFeaturesHandler httptransport.HandlerWithArgs[ListFeaturesHandlerRequest, ListFeaturesHandlerResponse, ListFeaturesHandlerParams]

func (h *featureHandlers) ListFeatures() ListFeaturesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, apiParams ListFeaturesHandlerParams) (ListFeaturesHandlerRequest, error) {
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
		func(ctx context.Context, params ListFeaturesHandlerRequest) ([]productcatalog.Feature, error) {
			return h.connector.ListFeatures(ctx, params)
		},
		commonhttp.JSONResponseEncoder,
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listFeatures"),
		)...,
	)
}

type DeleteFeatureHandlerRequest = models.NamespacedID
type DeleteFeatureHandlerResponse = interface{}
type DeleteFeatureHandlerParams = string

type DeleteFeatureHandler httptransport.HandlerWithArgs[DeleteFeatureHandlerRequest, DeleteFeatureHandlerResponse, DeleteFeatureHandlerParams]

func (h *featureHandlers) DeleteFeature() DeleteFeatureHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, featureID DeleteFeatureHandlerParams) (DeleteFeatureHandlerRequest, error) {
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
			httptransport.WithErrorEncoder(getErrorEncoder()),
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
