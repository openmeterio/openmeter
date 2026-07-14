package currencies

import (
	"context"
	"fmt"
	"net/http"

	v3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	GetCurrencyRequest  = currencies.GetCurrencyInput
	GetCurrencyResponse = v3.BillingCurrency
	GetCurrencyHandler  = httptransport.HandlerWithArgs[GetCurrencyRequest, GetCurrencyResponse, GetCurrencyParams]
	GetCurrencyParams   = string
)

func (h *handler) GetCurrency() GetCurrencyHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetCurrencyParams) (GetCurrencyRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetCurrencyRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return GetCurrencyRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        params,
				},
				ExpandOptions: currencies.ExpandOptions{
					CostBasis: true,
				},
			}, nil
		},
		func(ctx context.Context, request GetCurrencyRequest) (GetCurrencyResponse, error) {
			resp, err := h.service.GetCurrency(ctx, request)
			if err != nil {
				return GetCurrencyResponse{}, err
			}

			return ToAPIBillingCurrency(resp)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetCurrencyResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-custom-currency"),
		)...,
	)
}
