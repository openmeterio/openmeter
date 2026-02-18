package currencies

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	GetCostBasisRequest  = currencies.GetCostBasisInput
	GetCostBasisResponse = *currencies.CostBasis
	GetCostBasisHandler  = httptransport.HandlerWithArgs[GetCostBasisRequest, GetCostBasisResponse, string]
)

func (h *handler) GetCostBasis() GetCostBasisHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, costBasisID string) (GetCostBasisRequest, error) {
			if costBasisID == "" {
				return GetCostBasisRequest{}, fmt.Errorf("cost basis id is required")
			}
			return GetCostBasisRequest{
				ID: costBasisID,
			}, nil
		}, func(ctx context.Context, request GetCostBasisRequest) (GetCostBasisResponse, error) {
			return h.currencyService.GetCostBasis(ctx, request.ID)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetCostBasisResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getCostBasis"),
		)...,
	)
}
