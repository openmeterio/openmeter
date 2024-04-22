package postgres_connector

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/credit"
	inmemory_lock "github.com/openmeterio/openmeter/internal/credit/inmemory_lock"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	meter_model "github.com/openmeterio/openmeter/internal/meter"
	credit_model "github.com/openmeterio/openmeter/pkg/credit"
	"github.com/openmeterio/openmeter/pkg/models"
	product_model "github.com/openmeterio/openmeter/pkg/product"
)

func TestPostgresConnectorLedger(t *testing.T) {
	namespace := "default"
	subject := "subject-1"
	meter := models.Meter{
		Namespace:   namespace,
		ID:          "meter-1",
		Slug:        "meter-1",
		Aggregation: models.MeterAggregationSum,
	}
	meterRepository := meter_model.NewInMemoryRepository([]models.Meter{meter})
	productIn := product_model.Product{
		Namespace: namespace,
		MeterSlug: meter.Slug,
		Name:      "product-1",
	}

	tt := []struct {
		name        string
		description string
		test        func(t *testing.T, connector credit.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client)
	}{
		{
			name:        "GetLedger",
			description: "Should return ledger entries",
			test: func(t *testing.T, connector credit.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client) {
				ctx := context.Background()
				product := createProduct(t, connector, namespace, productIn)
				// We need to truncate the time to workaround pgx driver timezone issue
				// We also move it to the past to avoid timezone issues
				t1 := time.Now().Truncate(time.Hour * 24).Add(-time.Hour * 24)
				t2 := t1.Add(time.Hour).Truncate(0)
				t3 := t2.Add(time.Hour).Truncate(0)
				t4 := t3.Add(time.Hour).Truncate(0)

				grant1, err := connector.CreateGrant(ctx, namespace, credit_model.Grant{
					Subject:     subject,
					ProductID:   product.ID,
					Type:        credit_model.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: t1,
					Expiration: credit_model.ExpirationPeriod{
						Duration: credit_model.ExpirationPeriodDurationMonth,
						Count:    1,
					},
					Rollover: &credit_model.GrantRollover{
						Type: credit_model.GrantRolloverTypeRemainingAmount,
					},
				})
				assert.NoError(t, err)

				grant2, err := connector.CreateGrant(ctx, namespace, credit_model.Grant{
					Subject:     subject,
					ProductID:   product.ID,
					Type:        credit_model.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: t2,
					Expiration: credit_model.ExpirationPeriod{
						Duration: credit_model.ExpirationPeriodDurationMonth,
						Count:    1,
					},
					Rollover: &credit_model.GrantRollover{
						Type: credit_model.GrantRolloverTypeOriginalAmount,
					},
				})
				assert.NoError(t, err)

				// Void grant2
				_, err = connector.VoidGrant(ctx, namespace, grant2)
				assert.NoError(t, err)

				usage := 1.0
				streamingConnector.addRow(meter.Slug, models.MeterQueryRow{
					Value: usage,
					// Grant 1's effective time is t1, so usage starts from t1
					WindowStart: t1,
					// Reset time is t3, so usage ends at t3
					WindowEnd: t3,
					GroupBy:   map[string]*string{},
				})

				reset, rolloverGrants, err := connector.Reset(ctx, namespace, credit_model.Reset{
					Subject:     subject,
					EffectiveAt: t3,
				})
				assert.NoError(t, err)

				// Get ledger
				ledgerList, err := connector.GetLedger(ctx, namespace, subject, t1, t4)
				assert.NoError(t, err)

				// Expected
				ledgerEntries := ledgerList.GetEntries()
				ledgerUsage := -1 * usage
				reamingAmount := grant1.Amount - usage

				// Assert balance
				assert.Equal(t, []credit_model.LedgerEntry{
					// Original grant
					{
						ID:        grant1.ID,
						Type:      credit_model.LedgerEntryTypeGrant,
						Time:      t1,
						ProductID: product.ID,
						Amount:    &grant1.Amount,
					},
					// Void
					{
						ID:        grant2.ID,
						Type:      credit_model.LedgerEntryTypeVoid,
						Time:      t2,
						ProductID: product.ID,
						Amount:    &grant2.Amount,
					},
					// Usage
					{
						ID:        grant1.ID,
						Type:      credit_model.LedgerEntryTypeGrantUsage,
						Time:      t3,
						ProductID: product.ID,
						Amount:    &ledgerUsage,
						From:      &t1,
						To:        &t3,
					},
					// Reset
					{
						ID:   reset.ID,
						Type: credit_model.LedgerEntryTypeReset,
						Time: t3,
					},
					// Rolled over grant
					{
						ID:        rolloverGrants[0].ID,
						Type:      credit_model.LedgerEntryTypeGrant,
						Time:      t3,
						ProductID: product.ID,
						Amount:    &reamingAmount,
					},
				}, ledgerEntries)
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
			lockManager := inmemory_lock.NewLockManager(time.Second * 10)
			streamingConnector := newMockStreamingConnector()
			connector := NewPostgresConnector(slog.Default(), databaseClient, streamingConnector, meterRepository, lockManager)

			tc.test(t, connector, streamingConnector, databaseClient)
		})
	}
}
