package currencies

import (
	"context"
	"net/http"

	v3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	CreateCurrencyRequest  = currencies.CreateCurrencyInput
	CreateCurrencyResponse = v3.BillingCurrencyCustom
	CreateCurrencyHandler  = httptransport.Handler[CreateCurrencyRequest, CreateCurrencyResponse]
)

func (h *handler) CreateCurrency() CreateCurrencyHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateCurrencyRequest, error) {
			body := &CreateCurrencyRequest{}
			if err := commonhttp.JSONRequestBodyDecoder(r, body); err != nil {
				return CreateCurrencyRequest{}, err
			}
			return *body, nil
		},
		func(ctx context.Context, request CreateCurrencyRequest) (CreateCurrencyResponse, error) {
			resp, err := h.currencyService.CreateCurrency(ctx, request)
			if err != nil {
				return CreateCurrencyResponse{}, apierrors.NewConflictError(ctx, err, "Currency already exists")
			}
			return resp, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateCurrencyResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-custom-currency"),
		)...,
	)
}
