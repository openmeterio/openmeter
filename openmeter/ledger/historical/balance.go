package historical

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

var _ ledger.BalanceQuerier = (*Ledger)(nil)

func (l *Ledger) GetAccountBalance(ctx context.Context, account ledger.Account, route ledger.RouteFilter, query ledger.BalanceQuery) (ledger.Balance, error) {
	if account == nil {
		return alpacadecimal.Zero, fmt.Errorf("account is required")
	}

	balance, err := l.sumEntries(ctx, ledger.Query{
		Namespace: account.ID().Namespace,
		Filters: ledger.Filters{
			After:     query.After,
			AsOf:      query.AsOf,
			AccountID: lo.ToPtr(account.ID().ID),
			Route:     route,
		},
	})
	if err != nil {
		return alpacadecimal.Zero, fmt.Errorf("failed to sum entries for account %s/%s: %w", account.ID().Namespace, account.ID().ID, err)
	}

	return balance, nil
}

func (l *Ledger) GetSubAccountBalance(ctx context.Context, subAccount ledger.SubAccount, query ledger.BalanceQuery) (ledger.Balance, error) {
	if subAccount == nil {
		return alpacadecimal.Zero, fmt.Errorf("sub-account is required")
	}

	account, err := l.accountCatalog.GetAccountByID(ctx, subAccount.AccountID())
	if err != nil {
		return alpacadecimal.Zero, fmt.Errorf("failed to get parent account for sub-account %s: %w", subAccount.Address().SubAccountID(), err)
	}

	return l.GetAccountBalance(ctx, account, subAccount.Route().Filter(), query)
}

func (l *Ledger) sumEntries(ctx context.Context, query ledger.Query) (alpacadecimal.Decimal, error) {
	if err := query.Validate(); err != nil {
		return alpacadecimal.Zero, fmt.Errorf("failed to validate query: %w", err)
	}

	total, err := l.repo.SumEntries(ctx, query)
	if err != nil {
		return alpacadecimal.Zero, fmt.Errorf("failed to sum ledger entries: %w", err)
	}

	return total, nil
}
