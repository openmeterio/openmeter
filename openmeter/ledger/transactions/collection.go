package transactions

import (
	"context"
	"fmt"
	"sort"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type fboSource struct {
	subAccount ledger.SubAccount
	priority   int
	balance    alpacadecimal.Decimal
}

type fboCollection struct {
	subAccount ledger.SubAccount
	amount     alpacadecimal.Decimal
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
) ([]fboCollection, error) {
	if deps.SubAccountService == nil {
		return nil, fmt.Errorf("sub-account service is required")
	}

	customerAccounts, err := deps.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("get customer accounts: %w", err)
	}

	fboAccountWithID, ok := customerAccounts.FBOAccount.(accountIdentifier)
	if !ok {
		return nil, fmt.Errorf("customer FBO account does not expose an ID")
	}

	subAccounts, err := deps.SubAccountService.ListSubAccounts(ctx, ledgeraccount.ListSubAccountsInput{
		Namespace: fboAccountWithID.ID().Namespace,
		AccountID: fboAccountWithID.ID().ID,
	})
	if err != nil {
		return nil, fmt.Errorf("list sub-accounts: %w", err)
	}

	sources := make([]fboSource, 0, len(subAccounts))
	for _, subAccount := range subAccounts {
		route := subAccount.Route()
		if route.Currency != currency {
			continue
		}

		balance, err := settledBalanceForSubAccount(ctx, subAccount)
		if err != nil {
			return nil, err
		}

		priority := ledger.DefaultCustomerFBOPriority
		if route.CreditPriority != nil {
			priority = *route.CreditPriority
		}

		sources = append(sources, fboSource{
			subAccount: subAccount,
			priority:   priority,
			balance:    balance,
		})
	}

	sort.Slice(sources, func(i, j int) bool {
		if sources[i].priority == sources[j].priority {
			return sources[i].subAccount.Address().SubAccountID() < sources[j].subAccount.Address().SubAccountID()
		}

		return sources[i].priority < sources[j].priority
	})

	return collectFromPrioritizedSources(sources, target), nil
}

func collectFromPrioritizedSources(sources []fboSource, target alpacadecimal.Decimal) []fboCollection {
	remaining := target
	out := make([]fboCollection, 0, len(sources))

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

		out = append(out, fboCollection{
			subAccount: source.subAccount,
			amount:     amount,
		})
		remaining = remaining.Sub(amount)
	}

	return out
}

func settledBalanceForSubAccount(ctx context.Context, subAccount ledger.SubAccount) (alpacadecimal.Decimal, error) {
	balance, err := subAccount.GetBalance(ctx)
	if err != nil {
		return alpacadecimal.Decimal{}, fmt.Errorf("get balance for sub-account %s: %w", subAccount.Address().SubAccountID(), err)
	}

	return balance.Settled(), nil
}
