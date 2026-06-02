package historical

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Ledger represents a historical ledger for settled balances.
type Ledger struct {
	accountCatalog ledger.AccountCatalog
	accountLocker  ledger.AccountLocker
	repo           Repo

	routingValidator ledger.RoutingValidator
}

// NewLedger constructs a Ledger with the given repo and account collaborators.
func NewLedger(repo Repo, accountCatalog ledger.AccountCatalog, accountLocker ledger.AccountLocker, routingValidator ledger.RoutingValidator) *Ledger {
	return &Ledger{
		accountCatalog:   accountCatalog,
		accountLocker:    accountLocker,
		repo:             repo,
		routingValidator: routingValidator,
	}
}

// ----------------------------------------------------------------------------
// Let's implement ledger.Ledger interface
// ----------------------------------------------------------------------------

var _ ledger.Ledger = (*Ledger)(nil)

func (l *Ledger) ListTransactions(ctx context.Context, params ledger.ListTransactionsInput) (ledger.ListTransactionsResult, error) {
	if err := params.Validate(); err != nil {
		return ledger.ListTransactionsResult{}, fmt.Errorf("failed to validate list transactions input: %w", err)
	}

	res, err := l.repo.ListTransactions(ctx, params)
	if err != nil {
		return ledger.ListTransactionsResult{}, fmt.Errorf("failed to list transactions: %w", err)
	}

	return ledger.ListTransactionsResult{
		Items:      res.Items,
		NextCursor: res.NextCursor,
	}, nil
}

func (l *Ledger) GetTransactionGroup(ctx context.Context, id models.NamespacedID) (ledger.TransactionGroup, error) {
	group, err := l.repo.GetTransactionGroup(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction group: %w", err)
	}

	return group, nil
}

func (l *Ledger) CommitGroup(ctx context.Context, group ledger.TransactionGroupInput) (ledger.TransactionGroup, error) {
	txInputs := group.Transactions()

	if len(txInputs) == 0 {
		return nil, ledger.ErrTransactionGroupEmpty
	}

	// 1. Validate each transaction sequentially
	for idx, txInput := range txInputs {
		if err := ledger.ValidateTransactionInputWith(ctx, txInput, l.routingValidator); err != nil {
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

	subAccounts := make([]ledger.SubAccount, 0, len(subAccountIDs))
	for subAccountID := range subAccountIDs {
		subAccount, err := l.accountCatalog.GetSubAccountByID(ctx, models.NamespacedID{Namespace: namespace, ID: subAccountID})
		if err != nil {
			return fmt.Errorf("failed to get sub-account: %w", err)
		}
		subAccounts = append(subAccounts, subAccount)
	}

	accounts := make(map[models.NamespacedID]ledger.Account, len(subAccountIDs))
	for _, subAccount := range subAccounts {
		accountID := subAccount.AccountID()

		_, ok := accounts[accountID]
		if !ok {
			account, err := l.accountCatalog.GetAccountByID(ctx, accountID)
			if err != nil {
				return fmt.Errorf("failed to get account: %w", err)
			}

			accounts[accountID] = account
		}
	}

	affectedAccounts := make([]ledger.Account, 0, len(accounts))
	for _, acc := range accounts {
		affectedAccounts = append(affectedAccounts, acc)
	}

	return l.accountLocker.LockAccountsForPosting(ctx, affectedAccounts)
}

func (l *Ledger) validateAccountBalancesForTransaction(_ context.Context, _ ledger.TransactionInput) error {
	// TODO: implement this
	return nil
}
