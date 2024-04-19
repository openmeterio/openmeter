package postgres_connector

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	db_credit "github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/creditentry"
	credit_model "github.com/openmeterio/openmeter/pkg/credit"
)

// GetHighWatermark returns the high watermark for the given credit and subject pair.
func (c *PostgresConnector) GetHighWatermark(ctx context.Context, namespace string, subject string) (credit_model.HighWatermark, error) {
	lastReset, err := c.db.CreditEntry.Query().
		Where(
			db_credit.Namespace(namespace),
			db_credit.Subject(subject),
			db_credit.EntryTypeEQ(credit_model.EntryTypeReset),
		).
		Order(
			db_credit.ByEffectiveAt(sql.OrderDesc()),
		).
		Select(db_credit.FieldEffectiveAt).
		First(ctx)

	if err != nil {
		if db.IsNotFound(err) {
			return credit_model.HighWatermark{
				Subject: subject,
				Time:    time.Time{},
			}, nil
		}

		return credit_model.HighWatermark{}, fmt.Errorf("failed to get high watermark: %w", err)
	}

	return credit_model.HighWatermark{
		Subject: subject,
		Time:    lastReset.EffectiveAt,
	}, nil
}
