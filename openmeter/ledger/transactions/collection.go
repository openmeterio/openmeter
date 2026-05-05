package transactions

import (
	"context"
	"fmt"
	"sort"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type subAccountBalance struct {
	subAccount ledger.SubAccount
	balance    alpacadecimal.Decimal
}

type subAccountAmount struct {
	subAccount ledger.SubAccount
	amount     alpacadecimal.Decimal
}

type prioritizedSubAccountBalance struct {
	subAccountBalance
	priority int
}

type accountIdentifier interface {
	ID() models.NamespacedID
}

func collectFromPrioritizedCustomerFBO(
	ctx context.Context,
	customerID customer.CustomerID,
	currency currencyx.Code,
	target alpacadecimal.Decimal,
	deps ResolverDependencies,
) ([]subAccountAmount, error) {
	sources, err := listCustomerFBOSources(ctx, customerID, currency, deps)
	if err != nil {
		return nil, err
	}

	return collectFromPrioritizedSources(sources, target), nil
}

func listCustomerFBOSources(
	ctx context.Context,
	customerID customer.CustomerID,
	currency currencyx.Code,
	deps ResolverDependencies,
) ([]prioritizedSubAccountBalance, error) {
	if deps.AccountCatalog == nil {
		return nil, fmt.Errorf("account catalog is required")
	}

	if deps.BalanceQuerier == nil {
		return nil, fmt.Errorf("balance querier is required")
	}

	customerAccounts, err := deps.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("get customer accounts: %w", err)
	}

	fboAccountWithID, ok := customerAccounts.FBOAccount.(accountIdentifier)
	if !ok {
		return nil, fmt.Errorf("customer FBO account does not expose an ID")
	}

	subAccounts, err := deps.AccountCatalog.ListSubAccounts(ctx, ledger.ListSubAccountsInput{
		Namespace: fboAccountWithID.ID().Namespace,
		AccountID: fboAccountWithID.ID().ID,
	})
	if err != nil {
		return nil, fmt.Errorf("list sub-accounts: %w", err)
	}

	sources := make([]prioritizedSubAccountBalance, 0, len(subAccounts))
	for _, subAccount := range subAccounts {
		route := subAccount.Route()
		if route.Currency != currency {
			continue
		}

		balance, err := settledBalanceForSubAccount(ctx, deps, subAccount)
		if err != nil {
			return nil, err
		}

		priority := ledger.DefaultCustomerFBOPriority
		if route.CreditPriority != nil {
			priority = *route.CreditPriority
		}

		sources = append(sources, prioritizedSubAccountBalance{
			subAccountBalance: subAccountBalance{
				subAccount: subAccount,
				balance:    balance,
			},
			priority: priority,
		})
	}

	sort.Slice(sources, func(i, j int) bool {
		if sources[i].priority == sources[j].priority {
			return sources[i].subAccount.Address().SubAccountID() < sources[j].subAccount.Address().SubAccountID()
		}

		return sources[i].priority < sources[j].priority
	})

	return sources, nil
}

func collectFromPrioritizedSources(sources []prioritizedSubAccountBalance, target alpacadecimal.Decimal) []subAccountAmount {
	remaining := target
	out := make([]subAccountAmount, 0, len(sources))

	for _, source := range sources {
		if !remaining.IsPositive() {
			break
		}

		if !source.balance.IsPositive() {
			continue
		}

		amount := source.balance
		if source.balance.GreaterThan(remaining) {
			amount = remaining
		}

		out = append(out, subAccountAmount{
			subAccount: source.subAccount,
			amount:     amount,
		})
		remaining = remaining.Sub(amount)
	}

	return out
}

func collectFromAttributableCustomerAccrued(
	ctx context.Context,
	customerID customer.CustomerID,
	currency currencyx.Code,
	target alpacadecimal.Decimal,
	deps ResolverDependencies,
) ([]subAccountAmount, error) {
	customerAccounts, err := deps.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("get customer accounts: %w", err)
	}

	accruedAccountWithID, ok := customerAccounts.AccruedAccount.(accountIdentifier)
	if !ok {
		return nil, fmt.Errorf("customer accrued account does not expose an ID")
	}

	subAccounts, err := deps.AccountCatalog.ListSubAccounts(ctx, ledger.ListSubAccountsInput{
		Namespace: accruedAccountWithID.ID().Namespace,
		AccountID: accruedAccountWithID.ID().ID,
	})
	if err != nil {
		return nil, fmt.Errorf("list accrued sub-accounts: %w", err)
	}

	sources := make([]subAccountBalance, 0, len(subAccounts))
	for _, subAccount := range subAccounts {
		route := subAccount.Route()
		if route.Currency != currency || route.CostBasis == nil {
			continue
		}

		balance, err := settledBalanceForSubAccount(ctx, deps, subAccount)
		if err != nil {
			return nil, err
		}

		sources = append(sources, subAccountBalance{
			subAccount: subAccount,
			balance:    balance,
		})
	}

	sort.Slice(sources, func(i, j int) bool {
		return sources[i].subAccount.Address().SubAccountID() < sources[j].subAccount.Address().SubAccountID()
	})

	return collectFromBalanceSources(sources, target), nil
}

func collectFromBalanceSources(sources []subAccountBalance, target alpacadecimal.Decimal) []subAccountAmount {
	remaining := target
	out := make([]subAccountAmount, 0, len(sources))

	for _, source := range sources {
		if !remaining.IsPositive() {
			break
		}

		if !source.balance.IsPositive() {
			continue
		}

		amount := source.balance
		if source.balance.GreaterThan(remaining) {
			amount = remaining
		}

		out = append(out, subAccountAmount{
			subAccount: source.subAccount,
			amount:     amount,
		})
		remaining = remaining.Sub(amount)
	}

	return out
}

func decimalPointersEqual(left, right *alpacadecimal.Decimal) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return left.Equal(*right)
	}
}

func settledBalanceForSubAccount(ctx context.Context, deps ResolverDependencies, subAccount ledger.SubAccount) (alpacadecimal.Decimal, error) {
	balance, err := deps.BalanceQuerier.GetSubAccountBalance(ctx, subAccount, nil)
	if err != nil {
		return alpacadecimal.Decimal{}, fmt.Errorf("get balance for sub-account %s: %w", subAccount.Address().SubAccountID(), err)
	}

	return balance.Settled(), nil
}
