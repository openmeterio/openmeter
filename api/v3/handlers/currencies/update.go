package currencies

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	v3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	UpdateCurrencyRequest  = currencies.UpdateCurrencyInput
	UpdateCurrencyResponse = v3.BillingCurrency
	UpdateCurrencyParams   = string
	UpdateCurrencyHandler  = httptransport.HandlerWithArgs[UpdateCurrencyRequest, UpdateCurrencyResponse, UpdateCurrencyParams]
)

func (h *handler) UpdateCurrency() UpdateCurrencyHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params UpdateCurrencyParams) (UpdateCurrencyRequest, error) {
			if !h.creditsEnabled {
				return UpdateCurrencyRequest{}, models.NewGenericValidationError(errCustomCurrenciesDisabled)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateCurrencyRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			body := v3.BillingCurrencyCustomUpdate{}
			if err = request.ParseBody(r, &body); err != nil {
				return UpdateCurrencyRequest{}, fmt.Errorf("failed to parse update custom currency request: %w", err)
			}

			return UpdateCurrencyRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        params,
				},
				Name:               body.Name,
				Symbol:             lo.FromPtr(body.Symbol),
				DecimalMark:        body.DecimalMark,
				ThousandsSeparator: body.ThousandSeparator,
			}, nil
		},
		func(ctx context.Context, request UpdateCurrencyRequest) (UpdateCurrencyResponse, error) {
			resp, err := h.service.UpdateCurrency(ctx, request)
			if err != nil {
				return UpdateCurrencyResponse{}, err
			}

			return ToAPIBillingCurrency(resp)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateCurrencyResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-custom-currency"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
