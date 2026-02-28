package productcatalogdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/apiconverter"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/filter"
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
	ListFeatures() ListFeaturesHandler
	DeleteFeature() DeleteFeatureHandler
	QueryFeatureCost() QueryFeatureCostHandler
}

type featureHandlers struct {
	namespaceDecoder namespacedriver.NamespaceDecoder
	options          []httptransport.HandlerOption
	connector        feature.FeatureConnector
	llmcostService   llmcost.Service
	streaming        streaming.Connector
	meterService     meter.Service
	customerService  customer.Service
}

func NewFeatureHandler(
	connector feature.FeatureConnector,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	llmcostService llmcost.Service,
	streamingConnector streaming.Connector,
	meterService meter.Service,
	customerService customer.Service,
	options ...httptransport.HandlerOption,
) FeatureHandler {
	return &featureHandlers{
		namespaceDecoder: namespaceDecoder,
		options:          options,
		connector:        connector,
		llmcostService:   llmcostService,
		streaming:        streamingConnector,
		meterService:     meterService,
		customerService:  customerService,
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

			resp := MapFeatureToResponse(*feat)

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

			return MapFeatureCreateInputsRequest(ns, parsedBody), nil
		},
		func(ctx context.Context, feature feature.CreateFeatureInputs) (api.Feature, error) {
			createdFeature, err := h.connector.CreateFeature(ctx, feature)
			if err != nil {
				return api.Feature{}, err
			}
			return MapFeatureToResponse(createdFeature), nil
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
				mapped = append(mapped, MapFeatureToResponse(f))
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

type QueryFeatureCostHandlerRequest struct {
	Namespace string
	FeatureID string
	Params    api.QueryFeatureCostParams
}

type (
	QueryFeatureCostHandlerResponse = *api.FeatureCostQueryResult
	QueryFeatureCostHandlerParams   struct {
		FeatureID string
		api.QueryFeatureCostParams
	}
)

type QueryFeatureCostHandler httptransport.HandlerWithArgs[QueryFeatureCostHandlerRequest, QueryFeatureCostHandlerResponse, QueryFeatureCostHandlerParams]

func (h *featureHandlers) QueryFeatureCost() QueryFeatureCostHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, handlerParams QueryFeatureCostHandlerParams) (QueryFeatureCostHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return QueryFeatureCostHandlerRequest{}, err
			}

			return QueryFeatureCostHandlerRequest{
				Namespace: ns,
				FeatureID: handlerParams.FeatureID,
				Params:    handlerParams.QueryFeatureCostParams,
			}, nil
		},
		func(ctx context.Context, req QueryFeatureCostHandlerRequest) (QueryFeatureCostHandlerResponse, error) {
			// Get feature
			feat, err := h.connector.GetFeature(ctx, req.Namespace, req.FeatureID, feature.IncludeArchivedFeatureFalse)
			if err != nil {
				return nil, err
			}

			// Validate feature has meter and unit cost
			if feat.MeterSlug == nil {
				return nil, commonhttp.NewHTTPError(
					http.StatusBadRequest,
					fmt.Errorf("feature %s has no meter associated", feat.Key),
				)
			}

			if feat.UnitCost == nil {
				return nil, commonhttp.NewHTTPError(
					http.StatusBadRequest,
					fmt.Errorf("feature %s has no unit cost configured", feat.Key),
				)
			}

			// Get meter
			m, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
				Namespace: req.Namespace,
				IDOrSlug:  *feat.MeterSlug,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get meter: %w", err)
			}

			// Build streaming query params
			params, err := h.buildCostQueryParams(ctx, m, req)
			if err != nil {
				return nil, err
			}

			// Merge feature's MeterGroupByFilters into query
			if feat.MeterGroupByFilters != nil {
				if params.FilterGroupBy == nil {
					params.FilterGroupBy = map[string]filter.FilterString{}
				}
				for k, v := range feat.MeterGroupByFilters {
					// Only apply feature filters if not already specified by the user
					if _, exists := params.FilterGroupBy[k]; !exists {
						params.FilterGroupBy[k] = v
					}
				}
			}

			// For LLM unit cost: ensure dynamic properties are in GroupBy
			if feat.UnitCost.Type == feature.UnitCostTypeLLM && feat.UnitCost.LLM != nil {
				llmConf := feat.UnitCost.LLM
				var props []string
				if llmConf.ProviderProperty != "" {
					props = append(props, llmConf.ProviderProperty)
				}
				if llmConf.ModelProperty != "" {
					props = append(props, llmConf.ModelProperty)
				}
				if llmConf.TokenTypeProperty != "" {
					props = append(props, llmConf.TokenTypeProperty)
				}
				for _, prop := range props {
					if !slices.Contains(params.GroupBy, prop) {
						params.GroupBy = append(params.GroupBy, prop)
					}
				}
			}

			// Query meter for usage rows
			rows, err := h.streaming.QueryMeter(ctx, req.Namespace, m, params)
			if err != nil {
				return nil, fmt.Errorf("failed to query meter: %w", err)
			}

			// Compute cost for each row
			type unitCostResult struct {
				resolved   *feature.ResolvedUnitCost
				costDetail string
			}
			unitCostCache := make(map[string]unitCostResult)

			costRows := make([]api.FeatureCostQueryRow, 0, len(rows))
			var currency string
			for _, row := range rows {
				groupByValues := make(map[string]string, len(row.GroupBy))
				for k, v := range row.GroupBy {
					if v != nil {
						groupByValues[k] = *v
					}
				}

				// Build cache key from sorted group-by values
				cacheKey := fmt.Sprint(groupByValues)

				var resolved *feature.ResolvedUnitCost
				var costDetail string

				if cached, ok := unitCostCache[cacheKey]; ok {
					resolved = cached.resolved
					costDetail = cached.costDetail
				} else {
					resolved, err = h.connector.ResolveUnitCost(ctx, feature.ResolveUnitCostInput{
						Namespace:      req.Namespace,
						FeatureIDOrKey: req.FeatureID,
						GroupByValues:  groupByValues,
					})
					// If the LLM cost price is not found, emit a row with null amount and detail
					if err != nil {
						var vi models.ValidationIssue
						if errors.As(err, &vi) && vi.Code() == llmcost.ErrCodePriceNotFound {
							costDetail = vi.Error()
						} else {
							return nil, fmt.Errorf("failed to resolve unit cost: %w", err)
						}
					}

					unitCostCache[cacheKey] = unitCostResult{resolved: resolved, costDetail: costDetail}
				}

				if resolved == nil && costDetail == "" {
					continue
				}

				usage := alpacadecimal.NewFromFloat(row.Value)

				costRow := api.FeatureCostQueryRow{
					Usage:       usage.String(),
					WindowStart: row.WindowStart,
					WindowEnd:   row.WindowEnd,
					Subject:     row.Subject,
					CustomerId:  row.CustomerID,
				}

				if costDetail != "" {
					costRow.Detail = lo.ToPtr(costDetail)
				}

				if resolved != nil {
					currency = resolved.Currency
					costRow.Currency = resolved.Currency

					unitCost := resolved.Amount.String()
					costRow.UnitCost = &unitCost

					// cost = usage × unit cost
					cost := usage.Mul(resolved.Amount)
					amount := cost.String()
					costRow.Cost = &amount
				}
				if len(row.GroupBy) > 0 {
					costRow.GroupBy = &row.GroupBy
				}

				costRows = append(costRows, costRow)
			}

			if currency == "" {
				currency = "USD"
			}

			result := api.FeatureCostQueryResult{
				From:       req.Params.From,
				To:         req.Params.To,
				WindowSize: req.Params.WindowSize,
				Currency:   currency,
				Data:       costRows,
			}

			return &result, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[QueryFeatureCostHandlerResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("queryFeatureCost"),
			httptransport.WithErrorEncoder(getErrorEncoder()),
		)...,
	)
}

// buildCostQueryParams converts API query params to streaming.QueryParams.
// Follows the same pattern as meter/httphandler/mapping.go:toQueryParamsFromRequest.
func (h *featureHandlers) buildCostQueryParams(ctx context.Context, m meter.Meter, req QueryFeatureCostHandlerRequest) (streaming.QueryParams, error) {
	qp := req.Params

	params := streaming.QueryParams{
		ClientID: qp.ClientId,
		From:     qp.From,
		To:       qp.To,
	}

	if qp.WindowSize != nil {
		params.WindowSize = lo.ToPtr(meter.WindowSize(*qp.WindowSize))
	}

	if qp.GroupBy != nil {
		for _, groupBy := range *qp.GroupBy {
			if ok := groupBy == "subject" || groupBy == "customer_id" || m.GroupBy[groupBy] != ""; !ok {
				return params, models.NewGenericValidationError(fmt.Errorf("invalid group by: %s", groupBy))
			}
			params.GroupBy = append(params.GroupBy, groupBy)
		}
	}

	// Subject is a special query parameter which both filters and groups by subject(s)
	if qp.Subject != nil {
		params.FilterSubject = *qp.Subject
		if !slices.Contains(params.GroupBy, "subject") {
			params.GroupBy = append(params.GroupBy, "subject")
		}
	}

	// Resolve filter customer IDs
	if qp.FilterCustomerId != nil {
		filterCustomer, err := h.resolveFilterCustomers(ctx, req.Namespace, *qp.FilterCustomerId)
		if err != nil {
			return params, err
		}
		params.FilterCustomer = filterCustomer
		if len(filterCustomer) > 0 && !slices.Contains(params.GroupBy, "customer_id") {
			params.GroupBy = append(params.GroupBy, "customer_id")
		}
	}

	if qp.WindowTimeZone != nil {
		tz, err := time.LoadLocation(*qp.WindowTimeZone)
		if err != nil {
			return params, models.NewGenericValidationError(fmt.Errorf("invalid time zone: %w", err))
		}
		params.WindowTimeZone = tz
	}

	if qp.AdvancedMeterGroupByFilters != nil && len(*qp.AdvancedMeterGroupByFilters) > 0 {
		params.FilterGroupBy = apiconverter.ConvertStringMap(*qp.AdvancedMeterGroupByFilters)
	}

	if qp.FilterGroupBy != nil {
		filterGroupBy := map[string]string(*qp.FilterGroupBy)
		if len(filterGroupBy) > 0 {
			if len(params.FilterGroupBy) > 0 {
				return params, models.NewGenericValidationError(errors.New("advanced meter group by filters and filter group by cannot be used together"))
			}

			params.FilterGroupBy = map[string]filter.FilterString{}
			for k, v := range filterGroupBy {
				if _, ok := m.GroupBy[k]; ok {
					params.FilterGroupBy[k] = filter.FilterString{
						Eq: lo.ToPtr(v),
					}
					continue
				}
				return params, models.NewGenericValidationError(fmt.Errorf("invalid group by filter: %s", k))
			}
		}
	}

	return params, nil
}

// resolveFilterCustomers resolves customer IDs to streaming.Customer.
func (h *featureHandlers) resolveFilterCustomers(ctx context.Context, namespace string, customerIDs []string) ([]streaming.Customer, error) {
	if len(customerIDs) == 0 {
		return nil, nil
	}

	customers, err := h.customerService.ListCustomers(ctx, customer.ListCustomersInput{
		Namespace:   namespace,
		CustomerIDs: customerIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list customers: %w", err)
	}

	customersById := lo.KeyBy(customers.Items, func(c customer.Customer) string {
		return c.ID
	})

	for _, id := range customerIDs {
		if _, ok := customersById[id]; !ok {
			return nil, models.NewGenericNotFoundError(fmt.Errorf("customer with id %s not found", id))
		}
	}

	return lo.Map(customers.Items, func(c customer.Customer, _ int) streaming.Customer {
		return c
	}), nil
}

func (h *featureHandlers) resolveNamespace(ctx context.Context) (string, error) {
	ns, ok := h.namespaceDecoder.GetNamespace(ctx)
	if !ok {
		return "", commonhttp.NewHTTPError(http.StatusInternalServerError, errors.New("internal server error"))
	}

	return ns, nil
}
