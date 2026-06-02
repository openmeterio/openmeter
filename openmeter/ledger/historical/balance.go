package historical

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

type Balance struct {
	settled alpacadecimal.Decimal
	pending alpacadecimal.Decimal
}

var _ ledger.Balance = (*Balance)(nil)

func (b *Balance) Settled() alpacadecimal.Decimal {
	return b.settled
}

func (b *Balance) Pending() alpacadecimal.Decimal {
	return b.pending
}

var _ ledger.BalanceQuerier = (*Ledger)(nil)

func (l *Ledger) GetAccountBalance(ctx context.Context, account ledger.Account, route ledger.RouteFilter, query ledger.BalanceQuery) (ledger.Balance, error) {
	if account == nil {
		return nil, fmt.Errorf("account is required")
	}

	res, err := l.sumEntries(ctx, ledger.Query{
		Namespace: account.ID().Namespace,
		Filters: ledger.Filters{
			After:     query.After,
			AsOf:      query.AsOf,
			AccountID: lo.ToPtr(account.ID().ID),
			Route:     route,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to sum entries for account %s/%s: %w", account.ID().Namespace, account.ID().ID, err)
	}

	return &Balance{
		settled: res.SettledSum,
		pending: res.PendingSum,
	}, nil
}

func (l *Ledger) GetSubAccountBalance(ctx context.Context, subAccount ledger.SubAccount, query ledger.BalanceQuery) (ledger.Balance, error) {
	if subAccount == nil {
		return nil, fmt.Errorf("sub-account is required")
	}

	account, err := l.accountCatalog.GetAccountByID(ctx, subAccount.AccountID())
	if err != nil {
		return nil, fmt.Errorf("failed to get parent account for sub-account %s: %w", subAccount.Address().SubAccountID(), err)
	}

	return l.GetAccountBalance(ctx, account, subAccount.Route().Filter(), query)
}

func (l *Ledger) sumEntries(ctx context.Context, query ledger.Query) (ledger.QuerySummedResult, error) {
	if err := query.Validate(); err != nil {
		return ledger.QuerySummedResult{}, fmt.Errorf("failed to validate query: %w", err)
	}

	total, err := l.repo.SumEntries(ctx, query)
	if err != nil {
		return ledger.QuerySummedResult{}, fmt.Errorf("failed to sum ledger entries: %w", err)
	}

	// Historical ledger currently has no settled/pending separation.
	return ledger.QuerySummedResult{
		SettledSum: total,
		PendingSum: total,
	}, nil
}
