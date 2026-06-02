package ledger

import (
	"context"
	"time"
)

type BalanceQuery struct {
	After *TransactionCursor
	AsOf  *time.Time
}

type BalanceQuerier interface {
	GetAccountBalance(ctx context.Context, account Account, route RouteFilter, query BalanceQuery) (Balance, error)
	GetSubAccountBalance(ctx context.Context, subAccount SubAccount, query BalanceQuery) (Balance, error)
}
