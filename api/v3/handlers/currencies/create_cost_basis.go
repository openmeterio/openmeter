package currencies

import (
	"context"
	"fmt"
	"net/http"

	v3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	CreateCostBasisRequest  = currencies.CreateCostBasisInput
	CreateCostBasisResponse = v3.BillingCostBasis
	CreateCostBasisHandler  = httptransport.HandlerWithArgs[CreateCostBasisRequest, CreateCostBasisResponse, string]
)

func (h *handler) CreateCostBasis() CreateCostBasisHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, currencyID string) (CreateCostBasisRequest, error) {
			ns, ok := h.namespaceDecoder.GetNamespace(ctx)
			if !ok {
				return CreateCostBasisRequest{}, apierrors.NewInternalError(ctx, fmt.Errorf("failed to resolve namespace"))
			}

			body := &CreateCostBasisRequest{}
			if err := commonhttp.JSONRequestBodyDecoder(r, body); err != nil {
				return CreateCostBasisRequest{}, err
			}

			body.Namespace = ns
			body.CurrencyID = currencyID

			return *body, nil
		},
		func(ctx context.Context, request CreateCostBasisRequest) (CreateCostBasisResponse, error) {
			resp, err := h.currencyService.CreateCostBasis(ctx, request)
			if err != nil {
				return CreateCostBasisResponse{}, err
			}
			return CostBasisToAPI(resp), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateCostBasisResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-cost-basis"),
		)...,
	)
}
