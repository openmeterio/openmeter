package productcatalogdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/operation"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
	"github.com/openmeterio/openmeter/pkg/strcase"
)

type FeatureHandler interface {
	GetFeature() GetFeatureHandler
	CreateFeature() CreateFeatureHandler
	UpdateFeature() UpdateFeatureHandler
	ListFeatures() ListFeaturesHandler
	DeleteFeature() DeleteFeatureHandler
}

type featureHandlers struct {
	namespaceDecoder namespacedriver.NamespaceDecoder
	options          []httptransport.HandlerOption
	connector        feature.FeatureConnector
	meterService     meter.Service
	llmcostService   llmcost.Service
}

func NewFeatureHandler(
	connector feature.FeatureConnector,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	meterService meter.Service,
	llmcostService llmcost.Service,
	options ...httptransport.HandlerOption,
) FeatureHandler {
	return &featureHandlers{
		namespaceDecoder: namespaceDecoder,
		options:          options,
		connector:        connector,
		meterService:     meterService,
		llmcostService:   llmcostService,
	}
}

type (
	GetFeatureHandlerRequest  = models.NamespacedID
	GetFeatureHandlerResponse = api.Feature
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
			feat, err := h.connector.GetFeature(ctx, featureId.Namespace, featureId.ID, feature.IncludeArchivedFeatureFalse)
			if err != nil {
				return api.Feature{}, err
			}

			resp, err := MapFeatureToResponse(*feat)
			if err != nil {
				return api.Feature{}, err
			}

			// Resolve LLM pricing if the feature has LLM unit cost
			if feat.UnitCost != nil && feat.UnitCost.Type == feature.UnitCostTypeLLM && h.llmcostService != nil {
				pricing := resolveLLMPricing(ctx, h.llmcostService, feat)
				if pricing != nil {
					enrichFeatureResponseWithPricing(&resp, pricing)
				}
			}

			return resp, nil
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
	CreateFeatureHandlerRequest  = feature.CreateFeatureInputs
	CreateFeatureHandlerResponse = api.Feature
)

type CreateFeatureHandler httptransport.Handler[CreateFeatureHandlerRequest, CreateFeatureHandlerResponse]

func (h *featureHandlers) CreateFeature() CreateFeatureHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (feature.CreateFeatureInputs, error) {
			parsedBody := api.CreateFeatureJSONRequestBody{}
			emptyFeature := feature.CreateFeatureInputs{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &parsedBody); err != nil {
				return emptyFeature, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return emptyFeature, err
			}

			// Resolve meter slug to meter ID
			var meterID *string
			if parsedBody.MeterSlug != nil {
				m, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
					Namespace: ns,
					IDOrSlug:  *parsedBody.MeterSlug,
				})
				if err != nil {
					return emptyFeature, err
				}
				meterID = &m.ID
			}

			return MapFeatureCreateInputsRequest(ns, parsedBody, meterID)
		},
		func(ctx context.Context, feature feature.CreateFeatureInputs) (api.Feature, error) {
			createdFeature, err := h.connector.CreateFeature(ctx, feature)
			if err != nil {
				return api.Feature{}, err
			}
			return MapFeatureToResponse(createdFeature)
		},
		commonhttp.JSONResponseEncoderWithStatus[api.Feature](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createFeature"),
			httptransport.WithErrorEncoder(getErrorEncoder()),
		)...,
	)
}

type (
	UpdateFeatureHandlerRequest  = feature.UpdateFeatureInputs
	UpdateFeatureHandlerResponse = api.Feature
	UpdateFeatureHandlerParams   = string
)

type UpdateFeatureHandler httptransport.HandlerWithArgs[UpdateFeatureHandlerRequest, UpdateFeatureHandlerResponse, UpdateFeatureHandlerParams]

func (h *featureHandlers) UpdateFeature() UpdateFeatureHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, featureID string) (UpdateFeatureHandlerRequest, error) {
			parsedBody := api.UpdateFeatureJSONRequestBody{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &parsedBody); err != nil {
				return feature.UpdateFeatureInputs{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return feature.UpdateFeatureInputs{}, err
			}

			return MapFeatureUpdateInputsRequest(ns, featureID, parsedBody)
		},
		func(ctx context.Context, input UpdateFeatureHandlerRequest) (UpdateFeatureHandlerResponse, error) {
			updatedFeature, err := h.connector.UpdateFeature(ctx, input)
			if err != nil {
				return api.Feature{}, err
			}

			resp, err := MapFeatureToResponse(updatedFeature)
			if err != nil {
				return api.Feature{}, err
			}

			// Resolve LLM pricing if the feature has LLM unit cost
			if updatedFeature.UnitCost != nil && updatedFeature.UnitCost.Type == feature.UnitCostTypeLLM && h.llmcostService != nil {
				pricing := resolveLLMPricing(ctx, h.llmcostService, &updatedFeature)
				if pricing != nil {
					enrichFeatureResponseWithPricing(&resp, pricing)
				}
			}

			return resp, nil
		},
		commonhttp.JSONResponseEncoder,
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("updateFeature"),
			httptransport.WithErrorEncoder(getErrorEncoder()),
		)...,
	)
}

type (
	ListFeaturesHandlerRequest  = feature.ListFeaturesParams
	ListFeaturesHandlerResponse = commonhttp.Union[[]api.Feature, pagination.Result[api.Feature]]
	ListFeaturesHandlerParams   = api.ListFeaturesParams
)

type ListFeaturesHandler httptransport.HandlerWithArgs[ListFeaturesHandlerRequest, ListFeaturesHandlerResponse, ListFeaturesHandlerParams]

func (h *featureHandlers) ListFeatures() ListFeaturesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, apiParams ListFeaturesHandlerParams) (ListFeaturesHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return feature.ListFeaturesParams{}, err
			}

			params := feature.ListFeaturesParams{
				Namespace:       ns,
				IncludeArchived: defaultx.WithDefault(apiParams.IncludeArchived, false),
				Page: pagination.Page{
					PageSize:   defaultx.WithDefault(apiParams.PageSize, 0),
					PageNumber: defaultx.WithDefault(apiParams.Page, 0),
				},
				Limit:  defaultx.WithDefault(apiParams.Limit, commonhttp.DefaultPageSize),
				Offset: defaultx.WithDefault(apiParams.Offset, 0),
				OrderBy: feature.FeatureOrderBy(
					// Go enum value has a snake_case name, so we need to convert it
					strcase.CamelToSnake(string(lo.FromPtrOr(apiParams.OrderBy, api.FeatureOrderByKey))),
				),
				Order:      sortx.Order(lo.FromPtrOr(apiParams.Order, api.SortOrderASC)),
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
				Option2: &pagination.Result[api.Feature]{},
			}

			paged, err := h.connector.ListFeatures(ctx, params)
			if err != nil {
				return response, err
			}

			mapped := make([]api.Feature, 0, len(paged.Items))
			for _, f := range paged.Items {
				resp, err := MapFeatureToResponse(f)
				if err != nil {
					return response, err
				}
				mapped = append(mapped, resp)
			}

			if params.Page.IsZero() {
				response.Option1 = &mapped
			} else {
				response.Option1 = nil
				response.Option2 = &pagination.Result[api.Feature]{
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
		commonhttp.EmptyResponseEncoder[DeleteFeatureHandlerResponse](http.StatusNoContent),
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
