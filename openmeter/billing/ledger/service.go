package ledger

import (
	"context"
)

type Service interface {
	LedgerService
}

type LedgerService interface {
	WithLockedLedger(context.Context, WithLockedLedgerInput) error
	GetBalance(context.Context, GetBalanceInput) (GetBalanceResult, error)
}

type LedgerMutationService interface {
	Ledger() Ledger
	UpsertSubledger(context.Context, UpsertSubledgerInput) (Subledger, error)
	CreateTransaction(context.Context, CreateTransactionInput) (Transaction, error)

	Withdraw(context.Context, WithdrawInput) (WithdrawalResults, error)
}
