package postgres_connector

import (
	"context"
	"fmt"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	db_ledger "github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/ledger"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/pgulid"
	"github.com/openmeterio/openmeter/pkg/convertx"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (c *PostgresConnector) CreateLedger(ctx context.Context, namespace string, ledgerIn credit.Ledger) (credit.Ledger, error) {
	ledger, err := transaction(ctx, c, func(tx *db.Tx) (*credit.Ledger, error) {
		return c.createLedger(ctx, tx, namespace, ledgerIn)
	})

	if err != nil {
		return credit.Ledger{}, err
	}

	return *ledger, nil
}

func (c *PostgresConnector) createLedger(ctx context.Context, tx *db.Tx, namespace string, ledgerIn credit.Ledger) (*credit.Ledger, error) {
	entity, err := tx.Ledger.Create().
		SetNamespace(namespace).
		SetMetadata(ledgerIn.Metadata).
		SetSubject(ledgerIn.Subject).
		SetHighwatermark(defaultHighwatermark).
		Save(ctx)

	if db.IsConstraintError(err) {
		// This cannot happen in the same transaction as the previous Create
		// as the transaction is aborted at this stage
		existingLedgerEntity, err := c.db.Ledger.Query().
			Where(db_ledger.Namespace(namespace)).
			Where(db_ledger.Subject(ledgerIn.Subject)).
			Only(ctx)

		if err != nil {
			return nil, fmt.Errorf("cannot query existing ledger: %w", err)
		}
		return nil, &credit.LedgerAlreadyExistsError{
			Namespace: namespace,
			LedgerID:  existingLedgerEntity.ID.ULID,
			Subject:   ledgerIn.Subject,
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create ledger: %w", err)
	}

	return convertx.ToPointer(mapDBLedgerToModel(entity)), nil
}

func (c *PostgresConnector) UpsertLedger(ctx context.Context, namespace string, ledgerIn credit.UpsertLedger) (credit.Ledger, error) {
	ledger, err := transaction(ctx, c, func(tx *db.Tx) (*credit.Ledger, error) {
		ledger, err := tx.Ledger.Query().
			Where(db_ledger.Namespace(namespace)).
			Where(db_ledger.Subject(ledgerIn.Subject)).
			First(ctx)

		if err != nil {
			if db.IsNotFound(err) {
				newLedger, err := c.createLedger(ctx, tx, namespace, ledgerIn.ToLedger())
				if err != nil {
					return nil, err
				}
				return newLedger, err
			}

			return nil, err
		}

		if ledgerIn.Metadata != nil {
			err := tx.Ledger.Update().
				Where(db_ledger.Namespace(namespace)).
				Where(db_ledger.Subject(ledger.Subject)).
				SetMetadata(*ledgerIn.Metadata).Exec(ctx)

			if err != nil {
				return nil, err
			}

			ledger, err = tx.Ledger.Query().
				Where(db_ledger.Namespace(namespace)).
				Where(db_ledger.Subject(ledgerIn.Subject)).
				First(ctx)
			if err != nil {
				return nil, err
			}
		}

		return convertx.ToPointer(mapDBLedgerToModel(ledger)), nil

	})

	if err != nil {
		return credit.Ledger{}, err
	}

	return *ledger, nil
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
