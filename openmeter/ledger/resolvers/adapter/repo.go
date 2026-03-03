package adapter

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	ledgercustomeraccountdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgercustomeraccount"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/resolvers"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type repo struct {
	db *entdb.Client
}

var _ resolvers.Repo = (*repo)(nil)

func NewRepo(db *entdb.Client) resolvers.Repo {
	return &repo{db: db}
}

func (r *repo) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := r.db.HijackTx(ctx, &sql.TxOptions{ReadOnly: false})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}

	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

var _ entutils.TxUser[*repo] = (*repo)(nil)

func (r *repo) WithTx(ctx context.Context, tx *entutils.TxDriver) *repo {
	return &repo{
		db: entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client(),
	}
}

func (r *repo) Self() *repo {
	return r
}

func (r *repo) CreateCustomerAccount(ctx context.Context, input resolvers.CreateCustomerAccountInput) error {
	return entutils.TransactingRepoWithNoValue(ctx, r, func(ctx context.Context, tx *repo) error {
		_, err := tx.db.LedgerCustomerAccount.Create().
			SetNamespace(input.CustomerID.Namespace).
			SetCustomerID(input.CustomerID.ID).
			SetAccountType(input.AccountType).
			SetAccountID(input.AccountID).
			Save(ctx)
		if err != nil {
			if entdb.IsConstraintError(err) {
				existing, getErr := tx.db.LedgerCustomerAccount.Query().
					Where(
						ledgercustomeraccountdb.Namespace(input.CustomerID.Namespace),
						ledgercustomeraccountdb.CustomerID(input.CustomerID.ID),
						ledgercustomeraccountdb.AccountType(input.AccountType),
					).
					Only(ctx)
				if getErr != nil {
					return fmt.Errorf("failed to fetch existing ledger customer account mapping: %w", getErr)
				}

				return &resolvers.CustomerAccountAlreadyExistsError{
					CustomerID:  input.CustomerID,
					AccountType: input.AccountType,
					AccountID:   existing.AccountID,
				}
			}

			return fmt.Errorf("failed to create ledger customer account: %w", err)
		}

		return nil
	})
}

func (r *repo) GetCustomerAccountIDs(ctx context.Context, customerID customer.CustomerID) (map[ledger.AccountType]string, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (map[ledger.AccountType]string, error) {
		entities, err := tx.db.LedgerCustomerAccount.Query().
			Where(
				ledgercustomeraccountdb.Namespace(customerID.Namespace),
				ledgercustomeraccountdb.CustomerID(customerID.ID),
			).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get ledger customer accounts: %w", err)
		}

		result := make(map[ledger.AccountType]string, len(entities))
		for _, entity := range entities {
			result[entity.AccountType] = entity.AccountID
		}

		return result, nil
	})
}
