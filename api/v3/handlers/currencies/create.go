package currencies

import (
	"context"
	"log/slog"
	"net/http"

	v3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	CreateCurrencyRequest  = currencies.CreateCurrencyInput
	CreateCurrencyResponse = v3.BillingCurrency
	CreateCurrencyHandler  = httptransport.Handler[CreateCurrencyRequest, CreateCurrencyResponse]
)

func (h *handler) CreateCurrency() CreateCurrencyHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateCurrencyRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateCurrencyRequest{}, err
			}

			var body CreateCurrencyRequest
			if err := request.ParseBody(r, &body); err != nil {
				return CreateCurrencyRequest{}, err
			}

			body.Namespace = ns
			return body, nil
		},
		func(ctx context.Context, request CreateCurrencyRequest) (CreateCurrencyResponse, error) {
			resp, err := h.currencyService.CreateCurrency(ctx, request)
			if err != nil {
				return CreateCurrencyResponse{}, err
			}
			slog.InfoContext(ctx, "created custom currency",
				slog.String("operation", "create-custom-currency"),
				slog.String("namespace", resp.Namespace),
				slog.String("currency_id", resp.ID),
				slog.String("currency_code", resp.Code),
			)
			return ToAPIBillingCurrency(resp)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateCurrencyResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-custom-currency"),
		)...,
	)
}
