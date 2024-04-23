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

func checkAfterHighWatermark(t time.Time, ledger *db.Ledger) error {
	if !t.After(ledger.Highwatermark) {
		return &credit_model.HighWatermarBeforeError{Namespace: ledger.Namespace, Subject: ledger.Subject, HighWatermark: ledger.Highwatermark}
	}

	return nil
}
