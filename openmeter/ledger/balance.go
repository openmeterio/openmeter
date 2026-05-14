package ledger

import "context"

type BalanceQuerier interface {
	GetAccountBalance(ctx context.Context, account Account, query RouteFilter, after *TransactionCursor) (Balance, error)
	GetSubAccountBalance(ctx context.Context, subAccount SubAccount, after *TransactionCursor) (Balance, error)
}
