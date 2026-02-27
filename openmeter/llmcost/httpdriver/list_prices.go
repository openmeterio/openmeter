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
	ListPricesRequest  = llmcost.ListPricesInput
	ListPricesResponse = response.PagePaginationResponse[api.LLMCostPrice]
	ListPricesParams   = api.ListLlmCostPricesParams
	ListPricesHandler  = httptransport.HandlerWithArgs[ListPricesRequest, ListPricesResponse, ListPricesParams]
)

func (h *handler) ListPrices() ListPricesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListPricesParams) (ListPricesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListPricesRequest{}, err
			}

			req := ListPricesRequest{
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
			}

			if params.At != nil {
				req.At = params.At
			}

			return req, nil
		},
		func(ctx context.Context, request ListPricesRequest) (ListPricesResponse, error) {
			result, err := h.service.ListPrices(ctx, request)
			if err != nil {
				return ListPricesResponse{}, fmt.Errorf("failed to list llm cost prices: %w", err)
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
		commonhttp.JSONResponseEncoderWithStatus[ListPricesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-llm-cost-prices"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
