package features

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	UpdateFeatureRequest  = feature.UpdateFeatureInputs
	UpdateFeatureResponse = api.Feature
	UpdateFeatureParams   = string
	UpdateFeatureHandler  httptransport.HandlerWithArgs[UpdateFeatureRequest, UpdateFeatureResponse, UpdateFeatureParams]
)

func (h *handler) UpdateFeature() UpdateFeatureHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, featureID UpdateFeatureParams) (UpdateFeatureRequest, error) {
			// Update is a patch operation, so we need to read the body to determine if the unit_cost field is explicit
			// set to null (clearing the cost) or omitted (don't change).
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				return UpdateFeatureRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{Reason: "unable to read body", Source: apierrors.InvalidParamSourceBody},
				})
			}

			// Detect explicit null vs omitted for unit_cost.
			// JSON null means "clear the cost", omitted means "don't change".
			var rawFields map[string]json.RawMessage
			err = json.Unmarshal(bodyBytes, &rawFields)
			if err != nil {
				return UpdateFeatureRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{Reason: "unable to parse body", Source: apierrors.InvalidParamSourceBody},
				})
			}

			// Extract the unit_cost field from the body.
			unitCostRaw, unitCostPresent := rawFields["unit_cost"]
			// If the unit_cost field is present and set to null, we need to clear the cost.
			clearUnitCost := unitCostPresent && string(unitCostRaw) == "null"

			// Parse the body into the API request struct.
			body := api.UpdateFeatureRequest{}
			if err := json.Unmarshal(bodyBytes, &body); err != nil {
				return UpdateFeatureRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{Reason: "unable to parse body", Source: apierrors.InvalidParamSourceBody},
				})
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateFeatureRequest{}, err
			}

			return convertUpdateRequestToDomain(ns, featureID, body, clearUnitCost)
		},
		func(ctx context.Context, req UpdateFeatureRequest) (UpdateFeatureResponse, error) {
			updated, err := h.connector.UpdateFeature(ctx, req)
			if err != nil {
				return UpdateFeatureResponse{}, err
			}

			resp, err := convertFeatureToAPI(updated)
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
