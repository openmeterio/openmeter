package postgres_connector

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	db_ledger "github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/ledger"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (c *PostgresConnector) CreateLedger(ctx context.Context, ledgerIn credit.Ledger) (credit.Ledger, error) {
	entity, err := c.db.Ledger.Create().
		SetNamespace(ledgerIn.Namespace).
		SetMetadata(ledgerIn.Metadata).
		SetSubject(ledgerIn.Subject).
		SetHighwatermark(defaultHighwatermark).
		Save(ctx)

	if db.IsConstraintError(err) {
		// This cannot happen in the same transaction as the previous Create
		// as the transaction is aborted at this stage
		existingLedgerEntity, err := c.db.Ledger.Query().
			Where(db_ledger.Namespace(ledgerIn.Namespace)).
			Where(db_ledger.Subject(ledgerIn.Subject)).
			Only(ctx)

		if err != nil {
			return credit.Ledger{}, fmt.Errorf("cannot query existing ledger: %w", err)
		}
		return credit.Ledger{}, &credit.LedgerAlreadyExistsError{
			Ledger: mapDBLedgerToModel(existingLedgerEntity),
		}
	}

	if err != nil {
		return credit.Ledger{}, fmt.Errorf("failed to create ledger: %w", err)
	}

	return mapDBLedgerToModel(entity), nil

}

func (c *PostgresConnector) ListLedgers(ctx context.Context, params credit.ListLedgersParams) ([]credit.Ledger, error) {
	query := c.db.Ledger.Query().
		Order(
			db_ledger.ByCreatedAt(),
		).
		Where(db_ledger.Namespace(params.Namespace))

	if len(params.Subjects) > 0 {
		query = query.Where(
			db_ledger.SubjectIn(params.Subjects...),
		)
	}

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}

	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	dbLedgers, err := query.All(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return []credit.Ledger{}, nil
		}
		return nil, err
	}

	return slicesx.Map(dbLedgers, mapDBLedgerToModel), nil
}

func (c *PostgresConnector) getLedger(ctx context.Context, ledgerID credit.NamespacedLedgerID) (*db.Ledger, error) {
	return c.db.Ledger.Query().
		Where(db_ledger.Namespace(ledgerID.Namespace)).
		Where(db_ledger.ID(string(ledgerID.ID))).
		Only(ctx)
}

func mapDBLedgerToModel(ledger *db.Ledger) credit.Ledger {
	return credit.Ledger{
		Namespace: ledger.Namespace,
		ID:        credit.LedgerID(ledger.ID),
		Subject:   ledger.Subject,
		Metadata:  ledger.Metadata,
	}
}
