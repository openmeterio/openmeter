package customerscredits

import (
	"context"
	"errors"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

var errUnsupportedFeatureFilter = errors.New("feature filter is not supported for this balance endpoint")

type (
	GetCustomerCreditBalanceRequest struct {
		CustomerID customer.CustomerID
		Currencies customerbalance.CurrencyFilter
	}
	GetCustomerCreditBalanceResponse = api.BillingCreditBalances
	GetCustomerCreditBalanceParams   struct {
		CustomerID api.ULID
		Params     api.GetCustomerCreditBalanceParams
	}
	GetCustomerCreditBalanceHandler httptransport.HandlerWithArgs[GetCustomerCreditBalanceRequest, GetCustomerCreditBalanceResponse, GetCustomerCreditBalanceParams]
)

func (h *handler) GetCustomerCreditBalance() GetCustomerCreditBalanceHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, args GetCustomerCreditBalanceParams) (GetCustomerCreditBalanceRequest, error) {
			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetCustomerCreditBalanceRequest{}, err
			}

			if args.Params.Filter != nil && args.Params.Filter.Feature != nil {
				return GetCustomerCreditBalanceRequest{}, apierrors.NewBadRequestError(
					ctx,
					models.NewGenericValidationError(errUnsupportedFeatureFilter),
					apierrors.InvalidParameters{
						{
							Field:  "filter.feature",
							Reason: errUnsupportedFeatureFilter.Error(),
							Source: apierrors.InvalidParamSourceQuery,
						},
					},
				)
			}

			request := GetCustomerCreditBalanceRequest{
				CustomerID: customer.CustomerID{
					Namespace: namespace,
					ID:        args.CustomerID,
				},
			}

			if args.Params.Filter != nil && args.Params.Filter.Currency != nil {
				currency := currencyx.Code(*args.Params.Filter.Currency)
				request.Currencies = customerbalance.CurrencyFilter{
					Codes: []currencyx.Code{currency},
				}
			}

			return request, nil
		},
		func(ctx context.Context, request GetCustomerCreditBalanceRequest) (GetCustomerCreditBalanceResponse, error) {
			_, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerID: &request.CustomerID,
			})
			if err != nil {
				return GetCustomerCreditBalanceResponse{}, err
			}

			balancesByCurrency, err := h.balanceFacade.GetBalances(ctx, customerbalance.GetBalancesInput{
				CustomerID: request.CustomerID,
				Currencies: request.Currencies,
			})
			if err != nil {
				return GetCustomerCreditBalanceResponse{}, err
			}

			balances := make([]api.CreditBalance, 0, len(balancesByCurrency))
			for _, item := range balancesByCurrency {
				if len(request.Currencies.Codes) == 0 && item.Balance.Settled().IsZero() && item.Balance.Pending().IsZero() {
					continue
				}

				balances = append(balances, convertBalance(item.Currency, item.Balance))
			}

			return GetCustomerCreditBalanceResponse{
				RetrievedAt: clock.Now(),
				Balances:    balances,
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[GetCustomerCreditBalanceResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-customer-credit-balance"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
