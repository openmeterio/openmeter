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

// PostingAmount is a preselected amount to post against an address.
type PostingAmount struct {
	Address     ledger.PostingAddress
	Amount      alpacadecimal.Decimal
	IdentityKey string
	Annotations models.Annotations
}

type accountIdentifier interface {
	ID() models.NamespacedID
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

	// Recognition correction sorts the original accrued source entries by
	// sub-account id and unwinds from the end. Keep forward recognition ordered
	// the same way so partial corrections are deterministic.
	//
	// There is no business requirement on the priority order of earning recognition.
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
	balance, err := deps.BalanceQuerier.GetSubAccountBalance(ctx, subAccount, ledger.BalanceQuery{})
	if err != nil {
		return alpacadecimal.Decimal{}, fmt.Errorf("get balance for sub-account %s: %w", subAccount.Address().SubAccountID(), err)
	}

	return balance.Settled(), nil
}
