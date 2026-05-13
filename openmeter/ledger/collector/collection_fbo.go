package collector

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/cmpx"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type fboCollectionSource struct {
	address        ledger.PostingAddress
	available      alpacadecimal.Decimal
	creditPriority int
	cursor         string
}

var _ cmpx.Comparable[fboCollectionSource] = fboCollectionSource{}

type accountIdentifier interface {
	ID() models.NamespacedID
}

func (c *accrualCollector) collectCustomerFBO(
	ctx context.Context,
	customerID customer.CustomerID,
	currency currencyx.Code,
	target alpacadecimal.Decimal,
) ([]transactions.PostingAmount, error) {
	sources, err := c.listCustomerFBOSources(ctx, customerID, currency)
	if err != nil {
		return nil, err
	}

	slices.SortStableFunc(sources, cmpx.Compare[fboCollectionSource])

	return selectFBOPostingAmounts(sources, target), nil
}

func (s fboCollectionSource) Compare(other fboCollectionSource) int {
	if c := cmp.Compare(s.creditPriority, other.creditPriority); c != 0 {
		return c
	}

	return cmp.Compare(s.cursor, other.cursor)
}

func (c *accrualCollector) listCustomerFBOSources(
	ctx context.Context,
	customerID customer.CustomerID,
	currency currencyx.Code,
) ([]fboCollectionSource, error) {
	if c.deps.AccountCatalog == nil {
		return nil, fmt.Errorf("account catalog is required")
	}

	if c.deps.BalanceQuerier == nil {
		return nil, fmt.Errorf("balance querier is required")
	}

	customerAccounts, err := c.deps.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("get customer accounts: %w", err)
	}

	fboAccountWithID, ok := customerAccounts.FBOAccount.(accountIdentifier)
	if !ok {
		return nil, fmt.Errorf("customer FBO account does not expose an ID")
	}

	subAccounts, err := c.deps.AccountCatalog.ListSubAccounts(ctx, ledger.ListSubAccountsInput{
		Namespace: fboAccountWithID.ID().Namespace,
		AccountID: fboAccountWithID.ID().ID,
	})
	if err != nil {
		return nil, fmt.Errorf("list sub-accounts: %w", err)
	}

	sources := make([]fboCollectionSource, 0, len(subAccounts))
	for _, subAccount := range subAccounts {
		route := subAccount.Route()
		if route.Currency != currency {
			continue
		}

		balance, err := c.settledSubAccountBalance(ctx, subAccount)
		if err != nil {
			return nil, err
		}

		sources = append(sources, fboCollectionSource{
			address:        subAccount.Address(),
			available:      balance,
			creditPriority: customerFBOPriority(route),
			cursor:         subAccount.Address().SubAccountID(),
		})
	}

	return sources, nil
}

func customerFBOPriority(route ledger.Route) int {
	if route.CreditPriority == nil {
		return ledger.DefaultCustomerFBOPriority
	}

	return *route.CreditPriority
}

func (c *accrualCollector) settledSubAccountBalance(ctx context.Context, subAccount ledger.SubAccount) (alpacadecimal.Decimal, error) {
	balance, err := c.deps.BalanceQuerier.GetSubAccountBalance(ctx, subAccount, nil)
	if err != nil {
		return alpacadecimal.Decimal{}, fmt.Errorf("get balance for sub-account %s: %w", subAccount.Address().SubAccountID(), err)
	}

	return balance.Settled(), nil
}

func selectFBOPostingAmounts(sources []fboCollectionSource, target alpacadecimal.Decimal) []transactions.PostingAmount {
	remaining := target
	out := make([]transactions.PostingAmount, 0, len(sources))

	for _, source := range sources {
		if !remaining.IsPositive() {
			break
		}

		if !source.available.IsPositive() {
			continue
		}

		amount := source.available
		if source.available.GreaterThan(remaining) {
			amount = remaining
		}

		out = append(out, transactions.PostingAmount{
			Address: source.address,
			Amount:  amount,
		})
		remaining = remaining.Sub(amount)
	}

	return out
}
