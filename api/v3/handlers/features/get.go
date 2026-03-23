package features

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	GetFeatureRequest struct {
		Namespace string
		IDOrKey   string
	}
	GetFeatureResponse = api.Feature
	GetFeatureParams   = string
	GetFeatureHandler  httptransport.HandlerWithArgs[GetFeatureRequest, GetFeatureResponse, GetFeatureParams]
)

func (h *handler) GetFeature() GetFeatureHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, featureID GetFeatureParams) (GetFeatureRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetFeatureRequest{}, err
			}

			return GetFeatureRequest{
				Namespace: ns,
				IDOrKey:   featureID,
			}, nil
		},
		func(ctx context.Context, req GetFeatureRequest) (GetFeatureResponse, error) {
			feat, err := h.connector.GetFeature(ctx, req.Namespace, req.IDOrKey, feature.IncludeArchivedFeatureFalse)
			if err != nil {
				return GetFeatureResponse{}, err
			}

			resp, err := convertFeatureToAPI(*feat)
			if err != nil {
				return GetFeatureResponse{}, err
			}

			// Resolve LLM pricing if applicable
			if feat.UnitCost != nil && feat.UnitCost.Type == feature.UnitCostTypeLLM && h.llmcostService != nil {
				pricing := resolveLLMPricing(ctx, h.llmcostService, feat)
				if pricing != nil {
					enrichFeatureResponseWithPricing(&resp, pricing)
				}
			}

			return resp, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[GetFeatureResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-feature"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
