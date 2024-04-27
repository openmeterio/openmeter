package postgres_connector

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	credit_model "github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	meter_model "github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestWatermark(t *testing.T) {
	namespace := "default"
	subject := "subject"

	tt := []struct {
		name        string
		description string
		test        func(t *testing.T, connector credit_model.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client)
	}{
		{
			name:        "GetHighWatermark",
			description: "Should return highwatermark",
			test: func(t *testing.T, connector credit_model.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client) {
				ctx := context.Background()
				t1, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:01:00Z", time.UTC)

				_, err := db_client.Ledger.
					Create().
					SetNamespace(namespace).
					SetSubject(subject).
					SetHighwatermark(t1).
					Save(ctx)
				assert.NoError(t, err)

				hw, err := connector.GetHighWatermark(ctx, namespace, subject)
				assert.NoError(t, err)

				expected := credit_model.HighWatermark{
					Subject: subject,
					Time:    t1,
				}
				assert.Equal(t, expected.Subject, hw.Subject)
				assert.Equal(t, expected.Time.Unix(), hw.Time.Unix())
			},
		},
		{
			name:        "GetDefaultHighWatermark",
			description: "Should return default highwatermark",
			test: func(t *testing.T, connector credit_model.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client) {
				ctx := context.Background()

				hw, err := connector.GetHighWatermark(ctx, namespace, subject)
				assert.NoError(t, err)

				expected := credit_model.HighWatermark{
					Subject: subject,
					Time:    defaultHighwatermark,
				}
				assert.Equal(t, expected, hw)
			},
		},
		{
			name:        "checkAfterHighWatermark",
			description: "Should check if time is after high watermark",
			test: func(t *testing.T, connector credit_model.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client) {
				t1, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:01:00Z", time.UTC)
				t2, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:02:00Z", time.UTC)
				t3, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:03:00Z", time.UTC)
				ledger := db.Ledger{Namespace: namespace, Subject: subject, Highwatermark: t2}

				err := checkAfterHighWatermark(t1, &ledger)
				expected := &credit_model.HighWatermarBeforeError{Namespace: namespace, Subject: subject, HighWatermark: t2}
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
			driver := initDB(t)
			databaseClient := db.NewClient(db.Driver(driver))
			defer databaseClient.Close()

			// Note: lock manager cannot be shared between tests as these parallel tests write the same ledger
			streamingConnector := newMockStreamingConnector()
			meterRepository := meter_model.NewInMemoryRepository([]models.Meter{})
			connector := NewPostgresConnector(slog.Default(), databaseClient, streamingConnector, meterRepository)

			tc.test(t, connector, streamingConnector, databaseClient)
		})
	}
}
