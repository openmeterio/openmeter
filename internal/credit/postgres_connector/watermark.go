package postgres_connector

import (
	"context"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	db_ledger "github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/ledger"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/pgulid"
)

var defaultHighwatermark, _ = time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")

// GetHighWatermark returns the high watermark for the given credit and subject pair.
func (c *PostgresConnector) GetHighWatermark(ctx context.Context, namespace string, ledgerID ulid.ULID) (credit.HighWatermark, error) {
	ledgerEntity, err := c.db.Ledger.Query().
		Where(
			db_ledger.ID(pgulid.Wrap(ledgerID)),
			db_ledger.Namespace(namespace),
		).
		Only(ctx)

	if err != nil {
		if db.IsNotFound(err) {
			return credit.HighWatermark{
				LedgerID: ledgerID,
				Time:     defaultHighwatermark,
			}, nil
		}

		return credit.HighWatermark{}, fmt.Errorf("failed to get high watermark: %w", err)
	}

	return credit.HighWatermark{
		LedgerID: ledgerID,
		Time:     ledgerEntity.Highwatermark.In(time.UTC),
	}, nil
}

func checkAfterHighWatermark(t time.Time, ledger *db.Ledger) error {
	if !t.After(ledger.Highwatermark) {
		return &credit.HighWatermarBeforeError{
			Namespace:     ledger.Namespace,
			LedgerID:      ledger.ID.ULID,
			HighWatermark: ledger.Highwatermark,
		}
	}

	return nil
}
