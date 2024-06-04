package postgres_connector

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/testutils"
	meter_model "github.com/openmeterio/openmeter/internal/meter"
	om_testutils "github.com/openmeterio/openmeter/internal/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestWatermark(t *testing.T) {
	namespace := "default"
	subject := "subject-1"

	tt := []struct {
		name        string
		description string
		test        func(t *testing.T, connector credit.Connector, streamingConnector *testutils.MockStreamingConnector, db_client *db.Client)
	}{
		{
			name:        "GetHighWatermark",
			description: "Should return highwatermark",
			test: func(t *testing.T, connector credit.Connector, streamingConnector *testutils.MockStreamingConnector, db_client *db.Client) {
				ctx := context.Background()
				t1, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:01:00Z", time.UTC)

				ledgerID := ulid.Make().String()

				_, err := db_client.Ledger.
					Create().
					SetID(string(ledgerID)).
					SetSubject(ulid.Make().String()).
					SetNamespace(namespace).
					SetHighwatermark(t1).
					Save(ctx)
				assert.NoError(t, err)

				hw, err := connector.GetHighWatermark(ctx, credit.NamespacedLedgerID{
					Namespace: namespace,
					ID:        credit.LedgerID(ledgerID),
				})
				assert.NoError(t, err)

				expected := credit.HighWatermark{
					LedgerID: credit.LedgerID(ledgerID),
					Time:     t1,
				}
				assert.Equal(t, expected.LedgerID, hw.LedgerID)
				assert.Equal(t, expected.Time.Unix(), hw.Time.Unix())
			},
		},
		{
			name:        "GetDefaultHighWatermark",
			description: "Should return default highwatermark",
			test: func(t *testing.T, connector credit.Connector, streamingConnector *testutils.MockStreamingConnector, db_client *db.Client) {
				ctx := context.Background()

				ledger, err := connector.CreateLedger(ctx, credit.Ledger{
					Namespace: namespace,
					Subject:   subject,
				})
				assert.NoError(t, err)

				hw, err := connector.GetHighWatermark(ctx, credit.NewNamespacedLedgerID(namespace, ledger.ID))
				assert.NoError(t, err)

				expected := credit.HighWatermark{
					LedgerID: ledger.ID,
					Time:     defaultHighwatermark,
				}
				assert.Equal(t, expected, hw)
			},
		},
		{
			name:        "checkAfterHighWatermark",
			description: "Should check if time is after high watermark",
			test: func(t *testing.T, connector credit.Connector, streamingConnector *testutils.MockStreamingConnector, db_client *db.Client) {
				t1, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:01:00Z", time.UTC)
				t2, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:02:00Z", time.UTC)
				t3, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:03:00Z", time.UTC)

				ledgerID := ulid.Make().String()

				ledger := db.Ledger{
					ID:            ledgerID,
					Namespace:     namespace,
					Highwatermark: t2,
				}

				err := checkAfterHighWatermark(t1, &ledger)
				expected := &credit.HighWatermarBeforeError{
					Namespace:     namespace,
					LedgerID:      credit.LedgerID(ledgerID),
					HighWatermark: t2,
				}
				assert.Equal(t, expected, err, "should return error when time is before high watermark")

				err = checkAfterHighWatermark(t2, &ledger)
				assert.Equal(t, expected, err, "should return error when time is equal to high watermark")

				err = checkAfterHighWatermark(t3, &ledger)
				assert.NoError(t, err, "should not return error when time is after high watermark")
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			t.Log(tc.description)
			driver := om_testutils.InitPostgresDB(t)
			databaseClient := db.NewClient(db.Driver(driver))
			defer databaseClient.Close()

			old, err := time.Parse(time.RFC3339, "2020-01-01T00:00:00Z")
			assert.NoError(t, err)

			// Note: lock manager cannot be shared between tests as these parallel tests write the same ledger
			streamingConnector := testutils.NewMockStreamingConnector(t, testutils.MockStreamingConnectorParams{DefaultHighwatermark: old})
			meterRepository := meter_model.NewInMemoryRepository([]models.Meter{})
			connector := NewPostgresConnector(slog.Default(), databaseClient, streamingConnector, meterRepository, PostgresConnectorConfig{
				WindowSize: time.Minute,
			})

			tc.test(t, connector, streamingConnector, databaseClient)
		})
	}
}
