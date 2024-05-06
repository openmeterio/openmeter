package postgres_connector

import (
	"context"
	"fmt"

	"github.com/oklog/ulid/v2"
	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	db_ledger "github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/ledger"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/pgulid"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (c *PostgresConnector) CreateLedger(ctx context.Context, namespace string, ledgerIn credit.Ledger, upsert bool) (credit.Ledger, error) {
	dbLedger, err := transaction(ctx, c, func(tx *db.Tx) (*db.Ledger, error) {
		existingLedgerEntity, err := tx.Ledger.Query().
			Where(db_ledger.Namespace(namespace)).
			Where(db_ledger.Subject(ledgerIn.Subject)).
			Only(ctx)

		if !db.IsNotFound(err) {
			return nil, err
		}

		// ledger already exists
		if err == nil {
			if upsert {
				return c.upsertLedger(ctx, tx, existingLedgerEntity, ledgerIn.Metadata)
			}
			return nil, &credit.LedgerAlreadyExistsError{
				Namespace: namespace,
				LedgerID:  existingLedgerEntity.ID.ULID,
				Subject:   ledgerIn.Subject,
			}
		}

		entity, err := tx.Ledger.Create().
			SetNamespace(namespace).
			SetMetadata(ledgerIn.Metadata).
			SetSubject(ledgerIn.Subject).
			SetHighwatermark(defaultHighwatermark).
			Save(ctx)

		if err != nil {
			return nil, fmt.Errorf("failed to create ledger: %w", err)
		}

		return entity, nil
	})

	if err != nil {
		return credit.Ledger{}, err
	}

	return mapDBLedgerToModel(dbLedger), nil
}

func (a *PostgresConnector) upsertLedger(ctx context.Context, tx *db.Tx, entity *db.Ledger, metadata map[string]string) (*db.Ledger, error) {
	return tx.Ledger.UpdateOneID(entity.ID).
		SetMetadata(metadata).
		Save(ctx)
}

func (c *PostgresConnector) ListLedgers(ctx context.Context, namespace string, params credit.ListLedgersParams) ([]credit.Ledger, error) {
	query := c.db.Ledger.Query().
		Order(
			db_ledger.ByCreatedAt(),
		)

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
			return nil, nil
		}
		return nil, err
	}

	return slicesx.Map(dbLedgers, mapDBLedgerToModel), nil
}

func (c *PostgresConnector) getLedger(ctx context.Context, namespace string, ledgerID ulid.ULID) (*db.Ledger, error) {
	return c.db.Ledger.Query().
		Where(db_ledger.Namespace(namespace)).
		Where(db_ledger.ID(pgulid.Wrap(ledgerID))).
		Only(ctx)
}

func mapDBLedgerToModel(ledger *db.Ledger) credit.Ledger {
	return credit.Ledger{
		ID:       ledger.ID.ULID,
		Subject:  ledger.Subject,
		Metadata: ledger.Metadata,
	}
}
