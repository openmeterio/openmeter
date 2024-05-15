package postgres_connector

import (
	"context"
	"log/slog"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestLedgerCreation(t *testing.T) {

	driver := initDB(t)
	databaseClient := db.NewClient(db.Driver(driver))
	defer databaseClient.Close()

	meterRepository := meter.NewInMemoryRepository([]models.Meter{})

	streamingConnector := newMockStreamingConnector()
	connector := NewPostgresConnector(slog.Default(), databaseClient, streamingConnector, meterRepository)

	ledgerSubject := ulid.Make().String() // ~ random string
	namespace := "default"
	existingLedgerID := ulid.ULID{}

	t.Run("CreateLedger", func(t *testing.T) {
		// let's provision a ledger
		ledger, err := connector.CreateLedger(context.Background(), credit.Ledger{
			Namespace: namespace,
			Subject:   ledgerSubject,
		})

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
		assert.Equal(t, &credit.LedgerAlreadyExistsError{
			Ledger: credit.Ledger{
				Subject: ledgerSubject,
				ID:      existingLedgerID,
			},
		}, details)
	})

}
