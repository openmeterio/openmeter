package featurecost

import (
	"context"
	"fmt"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/handlers/query"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/cost"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	QueryFeatureCostRequest struct {
		Namespace string
		FeatureID string
		Body      api.MeterQueryRequest
	}
	QueryFeatureCostResponse = api.FeatureCostQueryResult
	QueryFeatureCostParams   = string
	QueryFeatureCostHandler  httptransport.HandlerWithArgs[QueryFeatureCostRequest, QueryFeatureCostResponse, QueryFeatureCostParams]
)

func (h *handler) QueryFeatureCost() QueryFeatureCostHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, featureID QueryFeatureCostParams) (QueryFeatureCostRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return QueryFeatureCostRequest{}, err
			}

			var body api.MeterQueryRequest
			if err := request.ParseOptionalBody(r, &body); err != nil {
				return QueryFeatureCostRequest{}, err
			}

			return QueryFeatureCostRequest{
				Namespace: ns,
				FeatureID: featureID,
				Body:      body,
			}, nil
		},
		func(ctx context.Context, req QueryFeatureCostRequest) (QueryFeatureCostResponse, error) {
			// Get the feature to find its meter
			feat, err := h.featureConnector.GetFeature(ctx, req.Namespace, req.FeatureID, feature.IncludeArchivedFeatureFalse)
			if err != nil {
				return QueryFeatureCostResponse{}, err
			}

			if feat.MeterSlug == nil {
				return QueryFeatureCostResponse{}, models.NewGenericValidationError(
					fmt.Errorf("feature %s has no meter associated", feat.Key),
				)
			}

			// Get the meter for query param validation
			m, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
				Namespace: req.Namespace,
				IDOrSlug:  *feat.MeterSlug,
			})
			if err != nil {
				return QueryFeatureCostResponse{}, fmt.Errorf("failed to get meter: %w", err)
			}

			// Build streaming query params using shared logic
			params, err := query.BuildQueryParams(ctx, m, req.Body, query.NewCustomerResolver(h.customerService))
			if err != nil {
				return QueryFeatureCostResponse{}, err
			}

			// Query feature cost
			result, err := h.costService.QueryFeatureCost(ctx, cost.QueryFeatureCostInput{
				Namespace:   req.Namespace,
				FeatureID:   req.FeatureID,
				QueryParams: params,
			})
			if err != nil {
				return QueryFeatureCostResponse{}, err
			}

			return ConvertCostQueryResultToAPI(result, req.Body), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[QueryFeatureCostResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("query-feature-cost"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
