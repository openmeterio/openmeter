package postgres_connector

import (
	"context"
	"fmt"
	"time"

	credit_model "github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
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
		if db.IsNotFound(err) {
			// Upsert ledger for the subject
			err := c.db.Ledger.
				Create().
				SetNamespace(namespace).
				SetSubject(subject).
				SetHighwatermark(defaultHighwatermark).
				OnConflictColumns(db_ledger.FieldNamespace, db_ledger.FieldSubject).
				Ignore().
				Exec(ctx)
			if err != nil {
				return credit_model.HighWatermark{}, fmt.Errorf("failed to upsert ledger: %w", err)
			}

			return credit_model.HighWatermark{
				Subject: subject,
				Time:    defaultHighwatermark,
			}, nil
		}

		return credit_model.HighWatermark{}, fmt.Errorf("failed to get high watermark: %w", err)
	}

	return credit_model.HighWatermark{
		Subject: subject,
		Time:    ledgerEntity.Highwatermark,
	}, nil
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
