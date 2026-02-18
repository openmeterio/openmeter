package currencies

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	CreateCostBasisRequest  = currencies.CreateCostBasisInput
	CreateCostBasisResponse = *currencies.CostBasis
	CreateCostBasisHandler  = httptransport.Handler[CreateCostBasisRequest, CreateCostBasisResponse]
)

func (h *handler) CreateCostBasis() CreateCostBasisHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateCostBasisRequest, error) {
			body := &CreateCostBasisRequest{}
			if err := commonhttp.JSONRequestBodyDecoder(r, body); err != nil {
				return CreateCostBasisRequest{}, err
			}

			return *body, nil
		},
		func(ctx context.Context, request CreateCostBasisRequest) (CreateCostBasisResponse, error) {
			_, err := h.currencyService.CreateCostBasis(ctx, request)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateCostBasisResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createCostBasis"),
		)...,
	)
}
