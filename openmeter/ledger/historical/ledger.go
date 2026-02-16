package historical

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/account"
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

func (l *Ledger) GetAccount(ctx context.Context, address ledger.Address) (ledger.Account, error) {
	account, err := l.accountService.GetAccount(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("failed to get ledger account for address %s: %w", address, err)
	}

	if account == nil {
		return nil, fmt.Errorf("returned nil account for address %s", address)
	}

	return account, nil
}

// SetUpTransactionIntent sets up a transaction intent and runs validations
func (l *Ledger) SetUpTransactionInput(ctx context.Context, at time.Time, entries []ledger.EntryInput) (ledger.TransactionInput, error) {
	if len(entries) < 2 {
		return nil, errors.New("at least two entries are required")
	}

	// Let's validate the addresses are correct by fetching the accounts
	uniqAccs := map[ledger.Address]*account.Account{}

	for _, entry := range entries {
		if lo.SomeBy(lo.Keys(uniqAccs), func(key ledger.Address) bool {
			return key.Equal(entry.Account())
		}) {
			continue
		}

		acc, err := l.accountService.GetAccount(ctx, entry.Account())
		if err != nil {
			return nil, fmt.Errorf("failed to get account for address %s: %w", entry.Account(), err)
		}

		uniqAccs[entry.Account()] = acc
	}

	entryInputs, err := slicesx.MapWithErr(entries, func(e ledger.EntryInput) (*EntryInput, error) {
		acc, ok := uniqAccs[e.Account()]
		if !ok {
			return nil, fmt.Errorf("account %s not found", e.Account())
		}

		addrData := acc.AddressData()
		addr := account.NewAddressFromData(addrData)

		return &EntryInput{
			input: CreateEntryInput{
				AccountID:   addr.ID().ID,
				AccountType: addr.Type(),
				Amount:      e.Amount(),
				Namespace:   addr.ID().Namespace,
				DimensionIDs: lo.MapToSlice(addrData.Dimensions, func(_ string, value *account.Dimension) string {
					return value.ID.ID
				}),
			},
			address: addr,
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
	namespace := ""
	for idx, txInput := range txInputs {
		if err := txInput.Validate(ctx); err != nil {
			return nil, fmt.Errorf("failed to validate transaction at index %d in group: %w", idx, err)
		}

		if idx == 0 {
			namespace = txInput.getNamespace()
		} else if txInput.getNamespace() != namespace {
			return nil, fmt.Errorf("transaction at index %d has a different namespace than the first transaction", idx)
		}
	}

	// 2. Validate account balances after the transactions (lock everything preemptively, not by sub-txs)
	for _, txInput := range txInputs {
		if err := l.validateAccountBalancesForTransaction(ctx, txInput); err != nil {
			return nil, fmt.Errorf("failed to validate account balances for transaction: %w", err)
		}
	}

	// 3. Create the transactions & the group
	txG, err := l.repo.CreateTransactionGroup(ctx, CreateTransactionGroupInput{
		Namespace:   namespace,
		Annotations: group.Annotations(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction group: %w", err)
	}

	txGroup := &TransactionGroup{
		data: txG,
	}

	for _, txInput := range txInputs {
		tx, err := l.createTransaction(ctx, txG.ID, txInput)
		if err != nil {
			return nil, fmt.Errorf("failed to create transaction: %w", err)
		}

		txGroup.transactions = append(txGroup.transactions, tx)
	}

	return txGroup, nil
}

func (l *Ledger) requirePreparedTransactionInput(tx ledger.TransactionInput) (*TransactionInput, error) {
	inp, ok := tx.(*TransactionInput)
	if !ok {
		return nil, errors.New("transaction input is not a *historical.TransactionInput")
	}

	return inp, nil
}

func (l *Ledger) createTransaction(ctx context.Context, groupID string, txInput *TransactionInput) (*Transaction, error) {
	tx, err := l.repo.CreateTransaction(ctx, CreateTransactionInput{
		Namespace: txInput.getNamespace(),
		GroupID:   groupID,
		BookedAt:  txInput.BookedAt(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// We need *EntryInputs for the already resolved DimensionIDs
	entryInps := make([]*EntryInput, len(txInput.EntryInputs()), 0)
	for idx, entry := range txInput.EntryInputs() {
		ei, ok := entry.(*EntryInput)
		if !ok {
			return nil, fmt.Errorf("entry at index %d is not a *EntryInput", idx)
		}

		entryInps = append(entryInps, ei)
	}

	entries, err := l.repo.CreateEntries(ctx, lo.Map(entryInps, func(e *EntryInput, _ int) CreateEntryInput {
		return CreateEntryInput{
			Namespace:     txInput.getNamespace(),
			AccountID:     e.Account().ID().ID,
			AccountType:   e.Account().Type(),
			Amount:        e.Amount(),
			TransactionID: tx.ID,
			DimensionIDs:  e.input.DimensionIDs,
		}
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to create entries: %w", err)
	}

	return &Transaction{
		data: tx,
		entries: lo.Map(entries, func(e EntryData, _ int) *Entry {
			return &Entry{
				data: e,
			}
		}),
	}, nil
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
