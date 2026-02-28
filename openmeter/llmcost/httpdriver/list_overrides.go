package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	ListOverridesRequest  = llmcost.ListOverridesInput
	ListOverridesResponse = response.PagePaginationResponse[api.LLMCostPrice]
	ListOverridesParams   = api.ListLlmCostOverridesParams
	ListOverridesHandler  = httptransport.HandlerWithArgs[ListOverridesRequest, ListOverridesResponse, ListOverridesParams]
)

func (h *handler) ListOverrides() ListOverridesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListOverridesParams) (ListOverridesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListOverridesRequest{}, err
			}

			req := ListOverridesRequest{
				Namespace: ns,
			}

			if params.Page != nil {
				req.Page = pagination.NewPage(
					lo.FromPtrOr(params.Page.Number, 1),
					lo.FromPtrOr(params.Page.Size, 20),
				)

				if err := req.Page.Validate(); err != nil {
					return req, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "page", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
			} else {
				req.Page = pagination.NewPage(1, 20)
			}

			if params.Filter != nil {
				if params.Filter.Provider != nil {
					p := llmcost.Provider(*params.Filter.Provider)
					req.Provider = &p
				}
				if params.Filter.ModelId != nil {
					req.ModelID = params.Filter.ModelId
				}
				if params.Filter.ModelName != nil {
					req.ModelName = params.Filter.ModelName
				}
			}

			return req, nil
		},
		func(ctx context.Context, request ListOverridesRequest) (ListOverridesResponse, error) {
			result, err := h.service.ListOverrides(ctx, request)
			if err != nil {
				return ListOverridesResponse{}, fmt.Errorf("failed to list llm cost overrides: %w", err)
			}

			items := lo.Map(result.Items, func(item llmcost.Price, _ int) api.LLMCostPrice {
				return domainPriceToAPI(item)
			})

			return response.NewPagePaginationResponse(items, response.PageMetaPage{
				Size:   request.Page.PageSize,
				Number: request.Page.PageNumber,
				Total:  lo.ToPtr(result.TotalCount),
			}), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListOverridesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-llm-cost-overrides"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
