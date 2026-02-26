package historical

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
)

// Ledger represents a historical ledger for settled balances.
type Ledger struct {
	accountService account.Service
	repo           Repo

	locker *lockr.Locker
}

// NewLedger constructs a Ledger with the given repo, account service and locker.
func NewLedger(repo Repo, accountService account.Service, locker *lockr.Locker) *Ledger {
	return &Ledger{
		repo:           repo,
		accountService: accountService,
		locker:         locker,
	}
}

// ----------------------------------------------------------------------------
// Let's implement ledger.Ledger interface
// ----------------------------------------------------------------------------

var _ ledger.Ledger = (*Ledger)(nil)

func (l *Ledger) ListTransactions(ctx context.Context, params ledger.ListTransactionsInput) (pagination.Result[ledger.Transaction], error) {
	if err := params.Validate(); err != nil {
		return pagination.Result[ledger.Transaction]{}, fmt.Errorf("failed to validate list transactions input: %w", err)
	}

	res, err := l.repo.ListTransactions(ctx, params)
	if err != nil {
		return pagination.Result[ledger.Transaction]{}, fmt.Errorf("failed to list transactions: %w", err)
	}

	return pagination.Result[ledger.Transaction]{
		Items: lo.Map(res.Items, func(item *Transaction, _ int) ledger.Transaction {
			return item
		}),
		NextCursor: res.NextCursor,
	}, nil
}

func (l *Ledger) CommitGroup(ctx context.Context, group ledger.TransactionGroupInput) (ledger.TransactionGroup, error) {
	txInputs := group.Transactions()

	if len(txInputs) == 0 {
		return nil, errors.New("no transactions to commit")
	}

	// 1. Validate each transaction sequentially
	for idx, txInput := range txInputs {
		if err := ledger.ValidateTransactionInput(ctx, txInput); err != nil {
			return nil, fmt.Errorf("failed to validate transaction at index %d in group: %w", idx, err)
		}
	}

	return transaction.Run(ctx, l.repo, func(ctx context.Context) (*TransactionGroup, error) {
		// 1.1  (lock everything preemptively, not by sub-txs)
		if err := l.lockAccountsForTransactionInputs(ctx, group.Namespace(), txInputs); err != nil {
			return nil, fmt.Errorf("failed to lock accounts for transaction inputs: %w", err)
		}

		// 2. Validate account balances after the transactions
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

func (l *Ledger) lockAccountsForTransactionInputs(ctx context.Context, namespace string, txInputs []ledger.TransactionInput) error {
	// 1. Let's collect all accounts
	subAccountIDs := make(map[string]struct{}, len(txInputs))

	for _, txInput := range txInputs {
		for _, entryInput := range txInput.EntryInputs() {
			subAccountID := entryInput.PostingAddress().SubAccountID()

			subAccountIDs[subAccountID] = struct{}{}
		}
	}

	subAccounts := make([]*account.SubAccount, 0, len(subAccountIDs))
	for subAccountID := range subAccountIDs {
		subAccount, err := l.accountService.GetSubAccountByID(ctx, models.NamespacedID{Namespace: namespace, ID: subAccountID})
		if err != nil {
			return fmt.Errorf("failed to get sub-account: %w", err)
		}
		subAccounts = append(subAccounts, subAccount)
	}

	accounts := make(map[string]*account.Account, len(subAccountIDs))
	for _, subAccount := range subAccounts {
		accountID := subAccount.AccountID()

		_, ok := accounts[accountID]
		if !ok {
			account, err := l.accountService.GetAccountByID(ctx, models.NamespacedID{Namespace: namespace, ID: accountID})
			if err != nil {
				return fmt.Errorf("failed to get account: %w", err)
			}

			accounts[accountID] = account
		}
	}

	// 2. Let's lock all customer accounts affected
	for _, acc := range accounts {
		switch acc.Type() {
		case ledger.AccountTypeCustomerFBO, ledger.AccountTypeCustomerReceivable:
			cSvc, err := acc.AsCustomerAccount()
			if err != nil {
				return fmt.Errorf("failed to convert account to customer account: %w", err)
			}

			if err := cSvc.Lock(ctx); err != nil {
				return fmt.Errorf("failed to lock customer account: %w", err)
			}
		}
	}

	return nil
}

func (l *Ledger) validateAccountBalancesForTransaction(_ context.Context, _ ledger.TransactionInput) error {
	// TODO: implement this
	return nil
}

// ----------------------------------------------------------------------------
// Let's implement ledger.Querier interface
// ----------------------------------------------------------------------------

var _ ledger.Querier = (*Ledger)(nil)

func (l *Ledger) SumEntries(ctx context.Context, query ledger.Query) (ledger.QuerySummedResult, error) {
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
