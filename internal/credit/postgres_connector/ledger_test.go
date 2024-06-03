package postgres_connector

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"

	om_testutils "github.com/openmeterio/openmeter/internal/testutils"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/testutils"
	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestLedgerCreation(t *testing.T) {

	driver := om_testutils.InitPostgresDB(t)
	databaseClient := db.NewClient(db.Driver(driver))
	defer databaseClient.Close()

	meterRepository := meter.NewInMemoryRepository([]models.Meter{})

	old, err := time.Parse(time.RFC3339, "2020-01-01T00:00:00Z")
	assert.NoError(t, err)

	streamingConnector := testutils.NewMockStreamingConnector(t, testutils.MockStreamingConnectorParams{DefaultHighwatermark: old})
	connector := NewPostgresConnector(slog.Default(), databaseClient, streamingConnector, meterRepository)

	ledgerSubject := ulid.Make().String() // ~ random string
	namespace := "default"
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

}
