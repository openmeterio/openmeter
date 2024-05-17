package postgres_connector

import (
	"context"
	"fmt"
	"strings"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	db_ledger "github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/ledger"
)

// https://www.postgresql.org/docs/current/errcodes-appendix.html
const pgLockNotAvailableErrorCode = "55P03"

type TransactionManager interface {
	startTransaction(ctx context.Context) (*db.Tx, error)
}

// Start a transaction
func (c *PostgresConnector) startTransaction(ctx context.Context) (*db.Tx, error) {
	return c.db.Tx(ctx)
}

// Transaction generic
func transaction[R any](ctx context.Context, connector TransactionManager, callback func(tx *db.Tx) (*R, error)) (*R, error) {
	tx, err := connector.startTransaction(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		// This prevents pgsql connections being stuck in a transaction if a panic occurs
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	result, err := callback(tx)
	if err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			err = fmt.Errorf("%w: %v", err, rerr)
		}
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return result, nil
}

// mutationTransaction is a generic to perform atomic mutations on the ledger.
func mutationTransaction[R any](ctx context.Context, connector *PostgresConnector, ledgerID credit.NamespacedLedgerID, callback func(tx *db.Tx, ledgerEntity *db.Ledger) (*R, error)) (*R, error) {
	// Start a transaction and lock the ledger for the subject
	return transaction(ctx, connector, func(tx *db.Tx) (*R, error) {
		ledgerEntity, err := lockLedger(tx, ctx, ledgerID)
		if err != nil {
			return nil, err
		}

		return callback(tx, ledgerEntity)
	})
}

// lockLedger locks the ledger for the given namespace and subject to avoid concurrent updates
func lockLedger(tx *db.Tx, ctx context.Context, ledgerID credit.NamespacedLedgerID) (*db.Ledger, error) {
	// Lock ledger for the subject with pessimistic update
	ledgerEntity, err := tx.Ledger.Query().
		Where(db_ledger.Namespace(ledgerID.Namespace)).
		Where(db_ledger.ID(string(ledgerID.ID))).

		// We use the ForUpdate method to tell ent to ask our DB to lock
		// the returned records for update.
		ForUpdate().
		Only(ctx)

	if err != nil {
		if db.IsNotFound(err) {
			return nil, &credit.LedgerNotFoundError{
				LedgerID: ledgerID.ID,
			}
		}
		if strings.Contains(err.Error(), pgLockNotAvailableErrorCode) {
			return nil, &credit.LockErrNotObtainedError{
				ID: ledgerID.ID,
			}
		}
		return nil, err
	}

	return ledgerEntity, nil
}
