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

			var body CreateCostBasisRequest
			if err := request.ParseBody(r, &body); err != nil {
				return CreateCostBasisRequest{}, err
			}

			body.Namespace = ns
			body.CurrencyID = currencyID

			return body, nil
		},
		func(ctx context.Context, request CreateCostBasisRequest) (CreateCostBasisResponse, error) {
			resp, err := h.service.CreateCostBasis(ctx, request)
			if err != nil {
				return CreateCostBasisResponse{}, err
			}
			slog.InfoContext(ctx, "created currency cost basis",
				slog.String("operation", "create-cost-basis"),
				slog.String("namespace", resp.Namespace),
				slog.String("currency_id", resp.CurrencyID),
				slog.String("cost_basis_id", resp.ID),
				slog.String("fiat_code", resp.FiatCode),
			)
			return ToAPIBillingCostBasis(resp), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateCostBasisResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-cost-basis"),
		)...,
	)
}
