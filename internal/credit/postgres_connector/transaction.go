package postgres_connector

import (
	"context"
	"fmt"
	"strings"

	credit_model "github.com/openmeterio/openmeter/internal/credit"
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
			panic(err)
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
func mutationTransaction[R any](ctx context.Context, connector *PostgresConnector, namespace, subject string, callback func(tx *db.Tx, ledgerEntity *db.Ledger) (*R, error)) (*R, error) {
	// Start a transaction and lock the ledger for the subject
	return transaction(ctx, connector, func(tx *db.Tx) (*R, error) {
		ledgerEntity, err := lockLedger(tx, ctx, namespace, subject)
		if err != nil {
			return nil, err
		}

		return callback(tx, ledgerEntity)
	})
}

// lockLedger locks the ledger for the given namespace and subject to avoid concurrent updates
func lockLedger(tx *db.Tx, ctx context.Context, namespace string, subject string) (*db.Ledger, error) {
	// Lock ledger for the subject with pessimistic update
	ledgerEntity, err := tx.Ledger.Query().
		Where(db_ledger.Namespace(namespace)).
		Where(db_ledger.Subject(subject)).

		// We use the ForUpdate method to tell ent to ask our DB to lock
		// the returned records for update.
		ForUpdate().
		Only(ctx)
	if err != nil {
		// If the ledger does not exist, we create it and try to lock it again
		if db.IsNotFound(err) {
			err := upsertLedger(tx, ctx, namespace, subject)
			if err != nil {
				return nil, fmt.Errorf("failed to upsert ledger: %w", err)
			}

			return lockLedger(tx, ctx, namespace, subject)
		}

		if strings.Contains(err.Error(), pgLockNotAvailableErrorCode) {
			return nil, &credit_model.LockErrNotObtainedError{Namespace: namespace, Subject: subject}
		}
		return nil, fmt.Errorf("failed to lock ledger: %w", err)
	}

	return ledgerEntity, nil
}

// Upsert ledger for the subject
// Ledger is created the first time when a mutation like granting, void or reset is performed.
func upsertLedger(tx *db.Tx, ctx context.Context, namespace, subject string) error {
	err := tx.Ledger.
		Create().
		SetNamespace(namespace).
		SetSubject(subject).
		OnConflictColumns(db_ledger.FieldNamespace, db_ledger.FieldSubject).
		Ignore().
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to upsert ledger: %w", err)
	}

	return nil
}
