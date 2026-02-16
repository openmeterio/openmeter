package historical

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

// Ledger represents a historical ledger for settled balances.
type Ledger struct {
	accountService account.Service
	repo           Repo
}

// ----------------------------------------------------------------------------
// Let's implement ledger.Ledger interface
// ----------------------------------------------------------------------------

var _ ledger.Ledger = (*Ledger)(nil)

func (l *Ledger) ListTransactions(ctx context.Context, params ledger.ListTransactionsInput) (pagination.Result[ledger.Transaction], error) {
	panic("not implemented")
}

// SetUpTransactionInput sets up a transaction input and runs validations
func (l *Ledger) SetUpTransactionInput(ctx context.Context, at time.Time, entries []ledger.EntryInput) (ledger.TransactionInput, error) {
	if len(entries) < 2 {
		return nil, errors.New("at least two entries are required")
	}

	// Let's validate the entries
	for idx, entry := range entries {
		if err := ledger.ValidateEntryInput(ctx, entry); err != nil {
			return nil, fmt.Errorf("invalid entry at index %d: %w", idx, err)
		}
	}

	entryInputs, err := slicesx.MapWithErr(entries, func(e ledger.EntryInput) (*EntryInput, error) {
		return &EntryInput{
			amount:  e.Amount(),
			address: e.PostingAddress(),
		}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to map entry inputs: %w", err)
	}

	tx := &TransactionInput{
		bookedAt:    at,
		entryInputs: entryInputs,
	}

	if err := tx.Validate(ctx); err != nil {
		return nil, fmt.Errorf("failed to validate transaction input: %w", err)
	}

	return tx, nil
}

func (l *Ledger) CommitGroup(ctx context.Context, group ledger.TransactionGroupInput) (ledger.TransactionGroup, error) {
	txInputs := make([]*TransactionInput, 0, len(group.Transactions()))
	for idx, tx := range group.Transactions() {
		// Let's validate the input transactions use the same implementation
		inp, err := l.requirePreparedTransactionInput(tx)
		if err != nil {
			return nil, fmt.Errorf("transaction %d: %w", idx, err)
		}

		txInputs = append(txInputs, inp)
	}

	if len(txInputs) == 0 {
		return nil, errors.New("no transactions to commit")
	}

	// 1. Validate each transaction sequentially
	for idx, txInput := range txInputs {
		if err := txInput.Validate(ctx); err != nil {
			return nil, fmt.Errorf("failed to validate transaction at index %d in group: %w", idx, err)
		}
	}

	// TODO: accounts should be locked for this, note: later we can be more granular
	return transaction.Run(ctx, l.repo, func(ctx context.Context) (*TransactionGroup, error) {
		// 2. Validate account balances after the transactions (lock everything preemptively, not by sub-txs)
		for _, txInput := range txInputs {
			if err := l.validateAccountBalancesForTransaction(ctx, txInput); err != nil {
				return nil, fmt.Errorf("failed to validate account balances for transaction: %w", err)
			}
		}

		// 3. Create the transactions & the group
		txG, err := l.repo.CreateTransactionGroup(ctx, CreateTransactionGroupInput{
			Namespace:   group.Namespace(),
			Annotations: group.Annotations(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create transaction group: %w", err)
		}

		txGroup := &TransactionGroup{
			data: txG,
		}

		for _, txInput := range txInputs {
			tx, err := l.repo.BookTransaction(ctx, models.NamespacedID{Namespace: group.Namespace(), ID: txG.ID}, txInput)
			if err != nil {
				return nil, fmt.Errorf("failed to create transaction: %w", err)
			}

			txGroup.transactions = append(txGroup.transactions, tx)
		}

		return txGroup, nil
	})
}

func (l *Ledger) requirePreparedTransactionInput(tx ledger.TransactionInput) (*TransactionInput, error) {
	inp, ok := tx.(*TransactionInput)
	if !ok {
		return nil, errors.New("transaction input is not a *historical.TransactionInput")
	}

	return inp, nil
}

func (l *Ledger) validateAccountBalancesForTransaction(_ context.Context, _ *TransactionInput) error {
	// TODO: implement this
	return nil
}

// ----------------------------------------------------------------------------
// Let's implement ledger.Querier interface
// ----------------------------------------------------------------------------

var _ ledger.Querier = (*Ledger)(nil)

func (l *Ledger) SumEntries(ctx context.Context, query ledger.Query) (ledger.QuerySummedResult, error) {
	panic("not implemented")
}
