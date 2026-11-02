package customerscredits

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// maxListAllCreditGrantsPageSize bounds one namespace-wide page: each grant
// loads its realizations.
const maxListAllCreditGrantsPageSize = 1000

type (
	ListAllCreditGrantsRequest = creditgrant.ListInput
	ListAllCreditGrantsHandler httptransport.HandlerWithArgs[ListAllCreditGrantsRequest, ListCreditGrantsResponse, api.ListAllCreditGrantsParams]
)

func (h *handler) ListAllCreditGrants() ListAllCreditGrantsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params api.ListAllCreditGrantsParams) (ListAllCreditGrantsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListAllCreditGrantsRequest{}, err
			}

			// The namespace-wide listing shares the per-customer query shape, plus the
			// customer filter mapped below.
			grantParams := api.ListCreditGrantsParams{Page: params.Page, Sort: params.Sort}
			var apiCustomerID *api.ULIDFieldFilter
			if f := params.Filter; f != nil {
				grantParams.Filter = &api.ListCreditGrantsParamsFilter{
					Status:     f.Status,
					Currency:   f.Currency,
					Key:        f.Key,
					FeatureKey: f.FeatureKey,
				}
				apiCustomerID = f.CustomerId
			}

			req, err := fromAPIListCreditGrantsParams(ctx, grantParams)
			if err != nil {
				return ListAllCreditGrantsRequest{}, err
			}

			// One customer bounds the per-customer route; this one spans the namespace,
			// and a zero page size would remove the query limit.
			if req.Page.PageSize < 1 || req.Page.PageSize > maxListAllCreditGrantsPageSize {
				err := fmt.Errorf("page size must be between 1 and %d", maxListAllCreditGrantsPageSize)

				return ListAllCreditGrantsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{Field: "page", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
				})
			}

			req.Namespace = ns
			req.CustomerIDs, err = fromAPICustomerIDULIDFieldFilter(ctx, apiCustomerID)
			if err != nil {
				return ListAllCreditGrantsRequest{}, err
			}

			req.AllCustomers = len(req.CustomerIDs) == 0

			return req, nil
		},
		h.listCreditGrants,
		commonhttp.JSONResponseEncoderWithStatus[ListCreditGrantsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-all-credit-grants"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

type (
	ListAllCreditBalancesRequest struct {
		customerbalance.ListCustomerBalancesInput
		RetrievedAt time.Time
	}
	ListAllCreditBalancesResponse = response.PagePaginationResponse[api.BillingCustomerCreditBalances]
	ListAllCreditBalancesHandler  httptransport.HandlerWithArgs[ListAllCreditBalancesRequest, ListAllCreditBalancesResponse, api.ListAllCreditBalancesParams]
)

func (h *handler) ListAllCreditBalances() ListAllCreditBalancesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params api.ListAllCreditBalancesParams) (ListAllCreditBalancesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListAllCreditBalancesRequest{}, err
			}

			page := pagination.NewPage(1, 20)
			if params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Page.Number, 1),
					lo.FromPtrOr(params.Page.Size, 20),
				)
			}

			req := ListAllCreditBalancesRequest{
				ListCustomerBalancesInput: customerbalance.ListCustomerBalancesInput{
					Namespace: ns,
					Page:      page,
				},
				RetrievedAt: clock.Now(),
			}

			if f := params.Filter; f != nil {
				req.CustomerIDs, err = fromAPICustomerIDULIDFieldFilter(ctx, f.CustomerId)
				if err != nil {
					return ListAllCreditBalancesRequest{}, err
				}

				req.Currencies, err = fromAPICurrencyStringFieldFilterExact(ctx, f.Currency)
				if err != nil {
					return ListAllCreditBalancesRequest{}, err
				}

				// A present currency filter that selects no code must not widen the read to
				// every currency. The per-customer route keeps its existing lenient behavior.
				if f.Currency != nil && (len(req.Currencies.Codes) == 0 || lo.Contains(req.Currencies.Codes, "")) {
					err := errors.New("currency must not be empty")

					return ListAllCreditBalancesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[currency]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}

				req.FeatureFilter, err = fromAPICustomerCreditFeatureFilter(f.FeatureKey)
				if err != nil {
					return ListAllCreditBalancesRequest{}, newFeatureKeyFilterBadRequest(ctx, err)
				}
			}

			return req, nil
		},
		func(ctx context.Context, request ListAllCreditBalancesRequest) (ListAllCreditBalancesResponse, error) {
			lister, err := customerbalance.NewLister(h.customerService, h.balanceFacade)
			if err != nil {
				return ListAllCreditBalancesResponse{}, err
			}

			listed, err := lister.ListCustomerBalances(ctx, request.ListCustomerBalancesInput)
			if err != nil {
				return ListAllCreditBalancesResponse{}, err
			}

			items := lo.Map(listed.Items, func(item customerbalance.CustomerBalances, _ int) api.BillingCustomerCreditBalances {
				return api.BillingCustomerCreditBalances{
					Customer:    api.CustomerReference{Id: item.CustomerID.ID},
					RetrievedAt: request.RetrievedAt,
					Balances: lo.Map(item.Balances, func(b customerbalance.BalanceByCurrency, _ int) api.CreditBalance {
						return toAPICreditBalance(b.CurrencyReference(), b.Balance)
					}),
				}
			})

			return response.NewPagePaginationResponse(items, response.PageMetaPage{
				Size:   request.Page.PageSize,
				Number: request.Page.PageNumber,
				Total:  lo.ToPtr(listed.TotalCount),
			}), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListAllCreditBalancesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-all-credit-balances"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
