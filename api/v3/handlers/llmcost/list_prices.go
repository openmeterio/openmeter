package llmcost

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
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

var listPricesAuthorizedFilters = map[string]request.AIPFilterOption{
	"provider": {
		Filters: []request.QueryFilterOp{
			request.QueryFilterEQ,
			request.QueryFilterNEQ,
			request.QueryFilterContains,
			request.QueryFilterOrContains,
		},
	},
	"model_id": {
		Filters: []request.QueryFilterOp{
			request.QueryFilterEQ,
			request.QueryFilterNEQ,
			request.QueryFilterContains,
		},
	},
	"model_name": {
		Filters: []request.QueryFilterOp{
			request.QueryFilterEQ,
			request.QueryFilterNEQ,
			request.QueryFilterContains,
		},
	},
	"currency": {
		Filters: []request.QueryFilterOp{
			request.QueryFilterEQ,
			request.QueryFilterNEQ,
			request.QueryFilterContains,
		},
	},
}

var listPricesAuthorizedSorts = []string{
	"id", "provider.id", "model.id", "effective_from", "effective_to",
}


func (h *handler) ListPrices() ListPricesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, _ ListPricesParams) (ListPricesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListPricesRequest{}, err
			}

			attrs, err := request.GetAipAttributes(r,
				request.WithDefaultPageSizeDefault(20),
				request.WithMaxPageSize(100),
				request.WithAuthorizedSorts(listPricesAuthorizedSorts),
				request.WithAuthorizedFilters(listPricesAuthorizedFilters),
			)
			if err != nil {
				return ListPricesRequest{}, err
			}

			pageNumber := attrs.Pagination.Number
			if pageNumber < 1 {
				pageNumber = 1
			}

			req := ListPricesRequest{
				Namespace: ns,
				Page:      pagination.NewPage(pageNumber, attrs.Pagination.Size),
				Provider:  request.FilterStringFromAip(attrs.Filters, "provider"),
				ModelID:   request.FilterStringFromAip(attrs.Filters, "model_id"),
				ModelName: request.FilterStringFromAip(attrs.Filters, "model_name"),
				Currency:  request.FilterStringFromAip(attrs.Filters, "currency"),
			}

			if len(attrs.Sorts) > 0 {
				req.OrderBy = attrs.Sorts[0].Field
				req.Order = attrs.Sorts[0].Order.ToSortxOrder()
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
