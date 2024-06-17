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
	"github.com/openmeterio/openmeter/internal/meter"
	om_testutils "github.com/openmeterio/openmeter/internal/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestLedgerCreation(t *testing.T) {

	driver := om_testutils.InitPostgresDB(t)
	databaseClient := db.NewClient(db.Driver(driver))
	defer databaseClient.Close()
	namespace := "default"

	testMeter := models.Meter{
		Namespace: namespace,
		ID:        "meter-1",
		Slug:      "meter-1",
		GroupBy:   map[string]string{"key": "$.path"},
	}

	meterRepository := meter.NewInMemoryRepository([]models.Meter{testMeter})

	old, err := time.Parse(time.RFC3339, "2020-01-01T00:00:00Z")
	assert.NoError(t, err)

	streamingConnector := testutils.NewMockStreamingConnector(t, testutils.MockStreamingConnectorParams{DefaultHighwatermark: old})

	connector := NewPostgresConnector(slog.Default(), databaseClient, streamingConnector, meterRepository, PostgresConnectorConfig{
		WindowSize: time.Minute,
	})

	ledgerSubject := ulid.Make().String() // ~ random string

	existingLedgerID := credit.LedgerID("")

	t.Run("CreateLedger", func(t *testing.T) {
		// let's provision a ledger
		ledger, err := connector.CreateLedger(context.Background(), credit.Ledger{
			Namespace: namespace,
			Subject:   ledgerSubject,
		})

		assert.NotEmpty(t, ledger.CreatedAt)
		assert.NoError(t, err)
		assert.Equal(t, ledger.Subject, ledgerSubject)
		existingLedgerID = ledger.ID
	})

	t.Run("CreateDuplicateLedger", func(t *testing.T) {
		_, err := connector.CreateLedger(context.Background(), credit.Ledger{
			Namespace: namespace,
			Subject:   ledgerSubject,
		})

		assert.Error(t, err)

		details, ok := err.(*credit.LedgerAlreadyExistsError)
		assert.True(t, ok, "We got an already exists error")
		assert.NotEmpty(t, details.Ledger.CreatedAt)
		assert.Equal(t, &credit.LedgerAlreadyExistsError{
			Ledger: credit.Ledger{
				Namespace: "default",
				Subject:   ledgerSubject,
				ID:        existingLedgerID,
				CreatedAt: details.Ledger.CreatedAt,
			},
		}, details)
	})

	t.Run("GetLedgerAffectedByMeterSubject", func(t *testing.T) {
		assert := assert.New(t)

		ledgerSubject := ulid.Make().String()

		l1, err := connector.CreateLedger(context.Background(), credit.Ledger{
			Namespace: namespace,
			Subject:   ledgerSubject,
		})
		assert.NoError(err)

		f1, err := connector.CreateFeature(context.Background(), credit.Feature{
			Namespace: namespace,
			Name:      "feature1",
			MeterSlug: "meter-1",
		})
		assert.NoError(err)

		ledger, err := connector.GetLedgerAffectedByMeterSubject(context.Background(), namespace, "meter-1", ledgerSubject)
		assert.NoError(err)
		assert.Nil(ledger)

		previousGrantEffAt, _ := time.Parse(time.RFC3339, "2024-03-01T00:00:00Z")
		resetAt, _ := time.Parse(time.RFC3339, "2024-03-02T00:00:00Z")

		// This ensures that during Reset the balance calculation can succeed
		streamingConnector.AddRow("meter-1", models.MeterQueryRow{
			Value:       0,
			WindowStart: time.Now(),
			WindowEnd:   time.Now(),
		})

		_, err = connector.CreateGrant(context.Background(), credit.Grant{
			Namespace:   namespace,
			FeatureID:   f1.ID,
			LedgerID:    l1.ID,
			EffectiveAt: previousGrantEffAt,
			Type:        credit.GrantTypeUsage,
			Expiration: credit.ExpirationPeriod{
				Count:    1,
				Duration: credit.ExpirationPeriodDurationDay,
			},
			Amount: 100,
		})
		assert.NoError(err)

		ledger, err = connector.GetLedgerAffectedByMeterSubject(context.Background(), namespace, "meter-1", ledgerSubject)
		assert.NoError(err)
		assert.NotNil(ledger)
		assert.Equal(l1.ID, ledger.ID)

		_, _, err = connector.Reset(context.Background(), credit.Reset{
			Namespace:   namespace,
			LedgerID:    l1.ID,
			EffectiveAt: resetAt,
		})
		assert.NoError(err)

		ledger, err = connector.GetLedgerAffectedByMeterSubject(context.Background(), namespace, "meter-1", ledgerSubject)
		assert.NoError(err)
		assert.Nil(ledger)

	})

}
