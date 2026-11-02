package customerbalance

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/mo"
	"golang.org/x/sync/errgroup"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

const (
	// MaxListCustomerBalancesPageSize bounds the work of one request: balances
	// are computed on read, one customer at a time.
	MaxListCustomerBalancesPageSize = 100
	// listCustomerBalancesConcurrency bounds how many customer balances one page
	// computes in parallel.
	listCustomerBalancesConcurrency = 8
)

type customerLister interface {
	ListCustomers(ctx context.Context, input customer.ListCustomersInput) (pagination.Result[customer.Customer], error)
}

type balancesGetter interface {
	GetBalances(ctx context.Context, input GetBalancesInput) ([]BalanceByCurrency, error)
}

type ListCustomerBalancesInput struct {
	Namespace string
	Page      pagination.Page
	// CustomerIDs limits the listing; empty lists every customer in the namespace.
	CustomerIDs   []string
	Currencies    CurrencyFilter
	FeatureFilter mo.Option[creditpurchase.FeatureFilters]
}

func (i ListCustomerBalancesInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if err := i.Page.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("page: %w", err))
	}

	// A zero page size would remove the query limit, so it is rejected too.
	if i.Page.PageSize < 1 || i.Page.PageSize > MaxListCustomerBalancesPageSize {
		errs = append(errs, fmt.Errorf("page size must be between 1 and %d", MaxListCustomerBalancesPageSize))
	}

	for _, id := range i.CustomerIDs {
		if id == "" {
			errs = append(errs, errors.New("customer ID must not be empty"))
		}
	}

	if err := i.Currencies.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currencies: %w", err))
	}

	if err := ValidateFeatureFilter(i.FeatureFilter); err != nil {
		errs = append(errs, fmt.Errorf("feature filter: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CustomerBalances struct {
	CustomerID customer.CustomerID
	Balances   []BalanceByCurrency
}

// Lister reads credit balances for a page of customers.
type Lister struct {
	customers customerLister
	balances  balancesGetter
}

func NewLister(customers customerLister, balances balancesGetter) (*Lister, error) {
	if customers == nil {
		return nil, errors.New("customer lister is required")
	}

	if balances == nil {
		return nil, errors.New("balances getter is required")
	}

	return &Lister{customers: customers, balances: balances}, nil
}

// ListCustomerBalances returns current balances in customer ID order.
// Each customer is read independently; concurrent changes can affect balances
// and move customers between pages.
//
// A customer without any ledger accounts has never held credit and is listed with
// no balances. Any other failure, including a partly provisioned customer, fails
// the page so no balance goes missing silently.
func (l *Lister) ListCustomerBalances(ctx context.Context, input ListCustomerBalancesInput) (pagination.Result[CustomerBalances], error) {
	if err := input.Validate(); err != nil {
		return pagination.Result[CustomerBalances]{}, err
	}

	customers, err := l.customers.ListCustomers(ctx, customer.ListCustomersInput{
		Namespace:   input.Namespace,
		Page:        input.Page,
		OrderBy:     "id",
		Order:       sortx.OrderAsc,
		CustomerIDs: input.CustomerIDs,
	})
	if err != nil {
		return pagination.Result[CustomerBalances]{}, fmt.Errorf("list customers: %w", err)
	}

	items := make([]CustomerBalances, len(customers.Items))

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(listCustomerBalancesConcurrency)

	for idx, cust := range customers.Items {
		g.Go(func() error {
			balances, err := l.balances.GetBalances(gctx, GetBalancesInput{
				CustomerID:    cust.GetID(),
				Currencies:    input.Currencies,
				FeatureFilter: input.FeatureFilter,
			})
			if err != nil && !ledger.IsCustomerWithoutAccounts(err) {
				return fmt.Errorf("get balances for customer %s: %w", cust.ID, err)
			}

			items[idx] = CustomerBalances{CustomerID: cust.GetID(), Balances: balances}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return pagination.Result[CustomerBalances]{}, err
	}

	return pagination.Result[CustomerBalances]{
		Page:       customers.Page,
		TotalCount: customers.TotalCount,
		Items:      items,
	}, nil
}
