package productcatalogdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/operation"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/strcase"
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

type (
	GetFeatureHandlerRequest  = models.NamespacedID
	GetFeatureHandlerResponse = *productcatalog.Feature
	GetFeatureHandlerParams   = string
)

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
			return h.connector.GetFeature(ctx, featureId.Namespace, featureId.ID, productcatalog.IncludeArchivedFeatureFalse)
		},
		commonhttp.JSONResponseEncoder,
		httptransport.AppendOptions(
			h.options,
			httptransport.WithErrorEncoder(getErrorEncoder()),
			httptransport.WithOperationName("getFeature"),
		)...,
	)
}

type (
	CreateFeatureHandlerRequest  = productcatalog.CreateFeatureInputs
	CreateFeatureHandlerResponse = productcatalog.Feature
)

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
				MeterSlug:           parsedBody.MeterSlug,
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

type (
	ListFeaturesHandlerRequest  = productcatalog.ListFeaturesParams
	ListFeaturesHandlerResponse = commonhttp.Union[[]api.Feature, pagination.PagedResponse[api.Feature]]
	ListFeaturesHandlerParams   = api.ListFeaturesParams
)

type ListFeaturesHandler httptransport.HandlerWithArgs[ListFeaturesHandlerRequest, ListFeaturesHandlerResponse, ListFeaturesHandlerParams]

func (h *featureHandlers) ListFeatures() ListFeaturesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, apiParams ListFeaturesHandlerParams) (ListFeaturesHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return productcatalog.ListFeaturesParams{}, err
			}

			// validate OrderBy
			if apiParams.OrderBy != nil {
				if !slices.Contains(productcatalog.FeatureOrderBy("").StrValues(), strcase.CamelToSnake(string(*apiParams.OrderBy))) {
					return productcatalog.ListFeaturesParams{}, commonhttp.NewHTTPError(http.StatusBadRequest, errors.New("invalid order by"))
				}
			}

			params := productcatalog.ListFeaturesParams{
				Namespace:       ns,
				IncludeArchived: defaultx.WithDefault(apiParams.IncludeArchived, false),
				Page: pagination.Page{
					PageSize:   defaultx.WithDefault(apiParams.PageSize, 0),
					PageNumber: defaultx.WithDefault(apiParams.Page, 0),
				},
				Limit:  defaultx.WithDefault(apiParams.Limit, commonhttp.DefaultPageSize),
				Offset: defaultx.WithDefault(apiParams.Offset, 0),
				OrderBy: productcatalog.FeatureOrderBy(
					strcase.CamelToSnake(defaultx.WithDefault((*string)(apiParams.OrderBy), string(productcatalog.FeatureOrderByUpdatedAt))),
				),
				Order:      commonhttp.GetSortOrder(api.ListFeaturesParamsOrderSortOrderASC, apiParams.Order),
				MeterSlugs: convert.DerefHeaderPtr[string](apiParams.MeterSlug),
			}

			if !params.Page.IsZero() {
				params.Page.PageNumber = defaultx.IfZero(params.Page.PageNumber, commonhttp.DefaultPage)
				params.Page.PageSize = defaultx.IfZero(params.Page.PageSize, commonhttp.DefaultPageSize)
			}

			// TODO: standardize
			if params.Page.PageSize > 1000 {
				return params, commonhttp.NewHTTPError(
					http.StatusBadRequest,
					fmt.Errorf("limit must be less than or equal to %d", 1000),
				)
			}

			return params, nil
		},
		func(ctx context.Context, params ListFeaturesHandlerRequest) (ListFeaturesHandlerResponse, error) {
			response := ListFeaturesHandlerResponse{
				Option1: &[]api.Feature{},
				Option2: &pagination.PagedResponse[api.Feature]{},
			}

			paged, err := h.connector.ListFeatures(ctx, params)
			if err != nil {
				return response, err
			}

			mapped := make([]api.Feature, 0, len(paged.Items))
			for _, f := range paged.Items {
				mapped = append(mapped, MapFeatureToResponse(f))
			}

			if params.Page.IsZero() {
				response.Option1 = &mapped
			} else {
				response.Option1 = nil
				response.Option2 = &pagination.PagedResponse[api.Feature]{
					Items:      mapped,
					TotalCount: paged.TotalCount,
					Page:       paged.Page,
				}
			}

			return response, err
		},
		commonhttp.JSONResponseEncoder,
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listFeatures"),
		)...,
	)
}

type (
	DeleteFeatureHandlerRequest  = models.NamespacedID
	DeleteFeatureHandlerResponse = interface{}
	DeleteFeatureHandlerParams   = string
)

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
