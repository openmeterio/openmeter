package currencies

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	ListCostBasesRequest  = struct{}
	ListCostBasesResponse = currencies.CostBases
	ListCostBasesHandler  = httptransport.Handler[ListCostBasesRequest, ListCostBasesResponse]
)

func (h *handler) ListCostBases() ListCostBasesHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (ListCostBasesRequest, error) {
			return ListCostBasesRequest{}, nil
		},
		func(ctx context.Context, request ListCostBasesRequest) (ListCostBasesResponse, error) {
			return h.currencyService.ListCostBases(ctx)
		},
		commonhttp.JSONResponseEncoderWithStatus[ListCostBasesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getCostBases"),
		)...,
	)
}
