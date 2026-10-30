package customerscredits

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/samber/lo"
	"github.com/samber/mo"
	"golang.org/x/sync/errgroup"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

const (
	// maxListAllCreditBalancesPageSize bounds the work of one request: balances are
	// computed on read, one customer at a time.
	maxListAllCreditBalancesPageSize = 100
	// listAllCreditBalancesConcurrency bounds how many customer balances one request
	// computes in parallel.
	listAllCreditBalancesConcurrency = 8
)

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
					UpdatedAt:  f.UpdatedAt,
				}
				apiCustomerID = f.CustomerId
			}

			req, err := fromAPIListCreditGrantsParams(ctx, grantParams)
			if err != nil {
				return ListAllCreditGrantsRequest{}, err
			}

			req.Namespace = ns
			req.CustomerIDs, err = fromAPICustomerIDULIDFieldFilter(ctx, apiCustomerID)
			if err != nil {
				return ListAllCreditGrantsRequest{}, err
			}

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
		Namespace      string
		Page           pagination.Page
		CustomerIDs    []string
		Currencies     customerbalance.CurrencyFilter
		FeatureFilter  mo.Option[creditpurchase.FeatureFilters]
		RetrievedAt    time.Time
		HistoricalAsOf *time.Time
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

			err = page.Validate()
			if err == nil && page.PageSize > maxListAllCreditBalancesPageSize {
				err = fmt.Errorf("page size must be at most %d", maxListAllCreditBalancesPageSize)
			}

			if err != nil {
				return ListAllCreditBalancesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{Field: "page", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
				})
			}

			req := ListAllCreditBalancesRequest{
				Namespace:      ns,
				Page:           page,
				RetrievedAt:    lo.FromPtrOr(params.Timestamp, clock.Now()),
				HistoricalAsOf: params.Timestamp,
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
			// Customer IDs are ULIDs, so ID order is creation order: customers created
			// during a walk land on later pages instead of shifting earlier ones.
			// Historical reads list the customers that existed at that time, so a
			// rebuilt close keeps customers deleted since then.
			customers, err := h.customerService.ListCustomers(ctx, customer.ListCustomersInput{
				Namespace:   request.Namespace,
				Page:        request.Page,
				OrderBy:     "id",
				Order:       sortx.OrderAsc,
				CustomerIDs: request.CustomerIDs,
				ExistingAt:  request.HistoricalAsOf,
			})
			if err != nil {
				return ListAllCreditBalancesResponse{}, fmt.Errorf("list customers: %w", err)
			}

			items := make([]api.BillingCustomerCreditBalances, len(customers.Items))

			g, gctx := errgroup.WithContext(ctx)
			g.SetLimit(listAllCreditBalancesConcurrency)

			for idx, cust := range customers.Items {
				g.Go(func() error {
					balancesByCurrency, err := h.balanceFacade.GetBalances(gctx, customerbalance.GetBalancesInput{
						CustomerID:    cust.GetID(),
						Currencies:    request.Currencies,
						FeatureFilter: request.FeatureFilter,
						AsOf:          request.HistoricalAsOf,
					})
					// Customers created before credits were enabled have no ledger accounts and
					// therefore no credit activity; they are listed with no balances.
					if err != nil && !hasValidationIssueCode(err, ledger.ErrCodeCustomerAccountMissing) {
						return fmt.Errorf("get balances for customer %s: %w", cust.ID, err)
					}

					balances := make([]api.CreditBalance, 0, len(balancesByCurrency))
					for _, item := range balancesByCurrency {
						balances = append(balances, toAPICreditBalance(item.CurrencyReference(), item.Balance))
					}

					items[idx] = api.BillingCustomerCreditBalances{
						Customer:    api.CustomerReference{Id: cust.ID},
						RetrievedAt: request.RetrievedAt,
						Balances:    balances,
					}

					return nil
				})
			}

			if err := g.Wait(); err != nil {
				return ListAllCreditBalancesResponse{}, err
			}

			return response.NewPagePaginationResponse(items, response.PageMetaPage{
				Size:   request.Page.PageSize,
				Number: request.Page.PageNumber,
				Total:  lo.ToPtr(customers.TotalCount),
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

func hasValidationIssueCode(err error, code models.ErrorCode) bool {
	issues, issueErr := models.AsValidationIssues(err)
	if issueErr != nil {
		return false
	}

	return lo.ContainsBy(issues, func(issue models.ValidationIssue) bool {
		return issue.Code() == code
	})
}
