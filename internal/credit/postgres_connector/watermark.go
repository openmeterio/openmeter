package postgres_connector

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	db_ledger "github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/ledger"
)

var defaultHighwatermark, _ = time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")

// GetHighWatermark returns the high watermark for the given credit and subject pair.
func (c *PostgresConnector) GetHighWatermark(ctx context.Context, ledgerID credit.NamespacedLedgerID) (credit.HighWatermark, error) {
	ledgerEntity, err := c.db.Ledger.Query().
		Where(
			db_ledger.ID(string(ledgerID.ID)),
			db_ledger.Namespace(ledgerID.Namespace),
		).
		Only(ctx)

	if err != nil {
		if db.IsNotFound(err) {
			return credit.HighWatermark{
				LedgerID: ledgerID.ID,
				Time:     defaultHighwatermark,
			}, nil
		}

		return credit.HighWatermark{}, fmt.Errorf("failed to get high watermark: %w", err)
	}

	return credit.HighWatermark{
		LedgerID: ledgerID.ID,
		Time:     ledgerEntity.Highwatermark.In(time.UTC),
	}, nil
}

func checkAfterHighWatermark(t time.Time, ledger *db.Ledger) error {
	if !t.After(ledger.Highwatermark) {
		return &credit.HighWatermarBeforeError{
			Namespace:     ledger.Namespace,
			LedgerID:      credit.LedgerID(ledger.ID),
			HighWatermark: ledger.Highwatermark,
		}
	}

	return nil
}
