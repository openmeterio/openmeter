package postgres_connector

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	credit_model "github.com/openmeterio/openmeter/internal/credit"
	db_ledger "github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/ledger"
)

var defaultHighwatermark, _ = time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")

// GetHighWatermark returns the high watermark for the given credit and subject pair.
func (c *PostgresConnector) GetHighWatermark(ctx context.Context, namespace string, subject string) (credit_model.HighWatermark, error) {
	ledgerEntity, err := c.db.Ledger.Query().
		Where(
			db_ledger.Subject(subject),
			db_ledger.Namespace(namespace),
		).
		Only(ctx)
	if err != nil {
		// TODO: pointer return?
		return credit_model.HighWatermark{}, err
	}

	return credit_model.HighWatermark{
		Subject: subject,
		Time:    ledgerEntity.Highwatermark,
	}, nil
}

func (c *PostgresConnector) getHighWatermarkWithTrnsLock(ctx context.Context, namespace, subject string) (time.Time, error) {
	ledgerEntity, err := c.db.Ledger.Query().
		Where(
			db_ledger.Subject(subject),
			db_ledger.Namespace(namespace),
		).
		ForUpdate().
		Only(ctx)

	if err != nil {
		return time.Time{}, nil
	}

	return ledgerEntity.Highwatermark, nil
}

func (c *PostgresConnector) upsertSubjectLedger(ctx context.Context, namespace, subject string) error {
	// TODO: check if we are not returning an error

	// TODO: let's create an inmemory singleton lru cache e.g. with github.com/maypok86/otter and skip upsert
	// for anything we have seen
	err := c.db.Ledger.
		Create().
		SetNamespace(namespace).
		SetSubject(subject).
		SetHighwatermark(defaultHighwatermark).
		OnConflict(sql.DoNothing()).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to upsert ledger: %w", err)
	}

	return nil
}

// GetHighWatermark returns the high watermark for the given credit and subject pair.
func (c *PostgresConnector) checkHighWatermark(ctx context.Context, namespace string, subject string, t time.Time) (credit_model.HighWatermark, error) {
	hw, err := c.GetHighWatermark(ctx, namespace, subject)
	if err != nil {
		return credit_model.HighWatermark{}, err
	}

	if !t.After(hw.Time) {
		return credit_model.HighWatermark{}, &credit_model.HighWatermarBeforeError{Namespace: namespace, Subject: subject, HighWatermark: hw.Time}
	}

	return hw, nil
}
