package currencies

import (
	"context"
	"fmt"
	"net/http"

	"github.com/alpacahq/alpacadecimal"

	v3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
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
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateCostBasisRequest{}, err
			}

			var body v3.CreateCostBasisRequest
			if err = request.ParseBody(r, &body); err != nil {
				return CreateCostBasisRequest{}, err
			}

			rate, err := alpacadecimal.NewFromString(body.Rate)
			if err != nil {
				return CreateCostBasisRequest{}, fmt.Errorf("invalid rate: %w", err)
			}

			return CreateCostBasisRequest{
				Namespace:     ns,
				CurrencyID:    currencyID,
				FiatCode:      currencyx.Code(body.FiatCode),
				Rate:          rate,
				EffectiveFrom: body.EffectiveFrom,
				EffectiveTo:   body.EffectiveTo,
			}, nil
		},
		func(ctx context.Context, request CreateCostBasisRequest) (CreateCostBasisResponse, error) {
			resp, err := h.service.CreateCostBasis(ctx, request)
			if err != nil {
				return CreateCostBasisResponse{}, err
			}

			return ToAPIBillingCostBasis(resp), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateCostBasisResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-cost-basis"),
		)...,
	)
}
