package customerbalance

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"
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
	AsOf          *time.Time
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

	if i.AsOf != nil && i.AsOf.IsZero() {
		errs = append(errs, errors.New("asOf must not be zero"))
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

// ListCustomerBalances returns one row per customer, in customer ID order.
//
// IDs are ULIDs, so new customers land on later pages instead of shifting
// earlier ones. Deleting a customer during a live walk still shifts later rows
// up a page; an AsOf walk keeps deleted customers, so its pages stay put.
//
// With AsOf, the customers are the ones that existed at AsOf. A customer
// created later but holding a grant backdated before AsOf is left out.
//
// A customer without an FBO account has never held credit and is listed with no
// balances. Any other failure, including a partly provisioned customer, fails
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
		ExistingAt:  input.AsOf,
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
				AsOf:          input.AsOf,
			})
			if err != nil && !isFBOAccountMissing(err) {
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

// The FBO account holds every credit balance, so without it the customer has
// none. A missing receivable or accrued account next to an FBO account is a
// partly provisioned customer whose balance we can't read.
func isFBOAccountMissing(err error) bool {
	issues, issueErr := models.AsValidationIssues(err)
	if issueErr != nil {
		return false
	}

	return lo.ContainsBy(issues, func(issue models.ValidationIssue) bool {
		return issue.Code() == ledger.ErrCodeCustomerAccountMissing &&
			issue.Attributes()["account_type"] == ledger.AccountTypeCustomerFBO
	})
}
