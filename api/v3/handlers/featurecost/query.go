package featurecost

import (
	"context"
	"fmt"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/handlers/meters/query"
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
		h.decodeQueryFeatureCostRequest,
		func(ctx context.Context, req QueryFeatureCostRequest) (QueryFeatureCostResponse, error) {
			result, err := h.queryFeatureCost(ctx, req)
			if err != nil {
				return QueryFeatureCostResponse{}, err
			}

			return ToAPIFeatureCostQueryResult(result.result, req.Body), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[QueryFeatureCostResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("query-feature-cost"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

func (h *handler) decodeQueryFeatureCostRequest(ctx context.Context, r *http.Request, featureID QueryFeatureCostParams) (QueryFeatureCostRequest, error) {
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
}

type featureCostQueryResult struct {
	featureKey string
	groupBy    []string
	result     *cost.CostQueryResult
}

func (h *handler) queryFeatureCost(ctx context.Context, req QueryFeatureCostRequest) (featureCostQueryResult, error) {
	// Get the feature to find its meter.
	feat, err := h.featureConnector.GetFeature(ctx, req.Namespace, req.FeatureID, feature.IncludeArchivedFeatureFalse)
	if err != nil {
		return featureCostQueryResult{}, err
	}

	if feat.MeterID == nil {
		return featureCostQueryResult{}, models.NewGenericValidationError(
			fmt.Errorf("feature %s has no meter associated", feat.Key),
		)
	}

	// Get the meter for query param validation.
	m, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
		Namespace: req.Namespace,
		IDOrSlug:  *feat.MeterID,
	})
	if err != nil {
		return featureCostQueryResult{}, fmt.Errorf("failed to get meter: %w", err)
	}

	params, err := query.BuildQueryParams(ctx, m, req.Body, query.NewCustomerResolver(h.customerService))
	if err != nil {
		return featureCostQueryResult{}, err
	}

	result, err := h.costService.QueryFeatureCost(ctx, cost.QueryFeatureCostInput{
		Namespace:   req.Namespace,
		FeatureID:   feat.ID,
		QueryParams: params,
	})
	if err != nil {
		return featureCostQueryResult{}, err
	}

	return featureCostQueryResult{
		featureKey: feat.Key,
		groupBy:    params.GroupBy,
		result:     result,
	}, nil
}
