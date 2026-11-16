package features

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	UpdateFeatureRequest  = feature.UpdateFeatureInputs
	UpdateFeatureResponse = apiv3models.Feature
	UpdateFeatureParams   = string
	UpdateFeatureHandler  httptransport.HandlerWithArgs[UpdateFeatureRequest, UpdateFeatureResponse, UpdateFeatureParams]
)

func (h *handler) UpdateFeature() UpdateFeatureHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, featureID UpdateFeatureParams) (UpdateFeatureRequest, error) {
			body := apiv3models.UpdateFeatureRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return UpdateFeatureRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateFeatureRequest{}, err
			}

			return convertUpdateRequestToDomain(ns, featureID, body)
		},
		func(ctx context.Context, req UpdateFeatureRequest) (UpdateFeatureResponse, error) {
			updated, err := h.connector.UpdateFeature(ctx, req)
			if err != nil {
				return UpdateFeatureResponse{}, err
			}

			resp, err := ConvertFeatureToAPI(updated)
			if err != nil {
				return UpdateFeatureResponse{}, err
			}

			// Resolve LLM pricing if applicable
			if updated.UnitCost != nil && updated.UnitCost.Type == feature.UnitCostTypeLLM && h.llmcostService != nil {
				pricing := resolveLLMPricing(ctx, h.llmcostService, &updated)
				if pricing != nil {
					enrichFeatureResponseWithPricing(&resp, pricing)
				}
			}

			return resp, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateFeatureResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-feature"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
