package ledger

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Adapter interface {
	LedgerAdapter
	SubledgerAdapter
	TransactionAdapter
	entutils.TxCreator
}

type LedgerAdapter interface {
	WithLockedLedger(context.Context, WithLockedLedgerAdapterInput) error
	GetBalance(context.Context, LedgerID) (GetBalanceAdapterResult, error)
	GetLedger(context.Context, LedgerRef) (Ledger, error)
}

type SubledgerAdapter interface {
	UpsertSubledger(context.Context, UpsertSubledgerAdapterInput) (Subledger, error)
}

type TransactionAdapter interface {
	CreateTransaction(context.Context, CreateTransactionInput) (Transaction, error)
}
