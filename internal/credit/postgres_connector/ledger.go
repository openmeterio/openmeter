package postgres_connector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	credit_model "github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	db_ledger "github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/ledger"
)

// https://www.postgresql.org/docs/current/errcodes-appendix.html
const pgLockNotAvailableErrorCode = "55P03"

// lockLedger locks the ledger for the given namespace and subject to avoid concurrent updates (grant, grant void and reset).
func (a *PostgresConnector) LockLedger(tx *db.Tx, ctx context.Context, namespace string, subject string) (*db.Ledger, error) {
	return LockLedger(tx, ctx, namespace, subject)
}

// LockLedger locks the ledger for the given namespace and subject to avoid concurrent updates (grant, grant void and reset).
func LockLedger(tx *db.Tx, ctx context.Context, namespace string, subject string) (*db.Ledger, error) {
	// Lock ledger for the subject with pessimistic update
	ledgerEntity, err := tx.Ledger.Query().
		Where(db_ledger.Namespace(namespace)).
		Where(db_ledger.Subject(subject)).

		// We use the ForUpdate method to tell ent to ask our DB to lock
		// the returned records for update.
		ForUpdate(
			// We specify that the query should not wait for the lock to be
			// released and instead fail immediately if the record is locked.
			sql.WithLockAction(sql.NoWait),
		).
		Only(ctx)
	if err != nil {
		if strings.Contains(err.Error(), pgLockNotAvailableErrorCode) {
			return nil, &credit_model.LockErrNotObtainedError{Namespace: namespace, Subject: subject}

		}
		return nil, fmt.Errorf("failed to lock ledger: %w", err)
	}

	return ledgerEntity.Update().SetUpdatedAt(time.Now()).Save(ctx)
}
