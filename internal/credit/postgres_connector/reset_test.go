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
	meter_model "github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestPostgresConnectorReset(t *testing.T) {
	namespace := "default"
	meter := models.Meter{
		Namespace:   namespace,
		ID:          "meter-1",
		Slug:        "meter-1",
		Aggregation: models.MeterAggregationSum,
	}
	meterRepository := meter_model.NewInMemoryRepository([]models.Meter{meter})
	featureIn := credit.Feature{
		Namespace: namespace,
		MeterSlug: meter.Slug,
		Name:      "feature-1",
	}

	tt := []struct {
		name        string
		description string
		test        func(t *testing.T, connector credit.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client, ledger credit.Ledger)
	}{
		{
			name:        "Reset",
			description: "Should move high watermark ahead",
			test: func(t *testing.T, connector credit.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client, ledger credit.Ledger) {
				ctx := context.Background()
				feature := createFeature(t, connector, featureIn)
				// We need to truncate the time to workaround pgx driver timezone issue
				// We also move it to the past to avoid timezone issues
				t1 := time.Now().In(time.UTC).Truncate(time.Hour * 24).Add(-time.Hour * 24)
				t2 := t1.Add(time.Hour).Truncate(0)
				t3 := t2.Add(time.Hour).Truncate(0)

				_, err := connector.CreateGrant(ctx, credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: t1,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationMonth,
						Count:    1,
					},
				})
				assert.NoError(t, err)

				// We need to add a row to the streaming connector as we call balance in the reset
				// even though there is no grant to rollover
				streamingConnector.addRow(meter.Slug, models.MeterQueryRow{})

				// Reset
				reset, rolloverGrants, err := connector.Reset(ctx, credit.Reset{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					EffectiveAt: t3,
				})
				assert.NoError(t, err)
				assert.NotNil(t, reset.ID)

				// Get high watermark
				highWatermark, err := connector.GetHighWatermark(ctx, credit.NamespacedLedgerID{
					Namespace: namespace,
					ID:        ledger.ID,
				})
				assert.NoError(t, err)
				assert.Equal(t, credit.HighWatermark{
					LedgerID: ledger.ID,
					Time:     t3,
				}, highWatermark)

				// Get grants
				grants, err := connector.ListGrants(ctx, credit.ListGrantsParams{
					Namespace:         namespace,
					LedgerIDs:         []credit.LedgerID{ledger.ID},
					FromHighWatermark: true,
				})
				assert.NoError(t, err)

				// No rollover grants
				assert.Len(t, rolloverGrants, 0)
				assert.Len(t, grants, 0)
			},
		},
		{
			name:        "ResetWithFullRollover",
			description: "Should rollover grants with original amount",
			test: func(t *testing.T, connector credit.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client, ledger credit.Ledger) {
				ctx := context.Background()
				feature := createFeature(t, connector, featureIn)
				// We need to truncate the time to workaround pgx driver timezone issue
				t1 := time.Now().Truncate(time.Hour * 24)
				t2 := t1.Add(time.Hour).Truncate(0)
				t3 := t2.Add(time.Hour).Truncate(0)

				_, err := connector.CreateGrant(ctx, credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: t1,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationMonth,
						Count:    1,
					},
					Rollover: &credit.GrantRollover{
						Type: credit.GrantRolloverTypeOriginalAmount,
					},
				})
				assert.NoError(t, err)

				// We need to add a row to the streaming connector as we call balance in the reset
				// even though rollover grant is original amount
				streamingConnector.addRow(meter.Slug, models.MeterQueryRow{})

				// Reset
				_, rolloverGrants, err := connector.Reset(ctx, credit.Reset{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					EffectiveAt: t3,
				})
				assert.NoError(t, err)
				assert.Len(t, rolloverGrants, 1)

				// Get grants
				grants, err := connector.ListGrants(ctx, credit.ListGrantsParams{
					Namespace:         namespace,
					LedgerIDs:         []credit.LedgerID{ledger.ID},
					FromHighWatermark: true,
				})
				assert.NoError(t, err)
				assert.Len(t, rolloverGrants, 1)

				// Grants after reset should be the same as rollover grants
				assert.Equal(t,
					removeTimestampsFromGrants(rolloverGrants),
					removeTimestampsFromGrants(grants),
				)
			},
		},
		{
			name:        "ResetWithRemainingRollover",
			description: "Should rollover grants with remaining amount",
			test: func(t *testing.T, connector credit.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client, ledger credit.Ledger) {
				ctx := context.Background()
				feature := createFeature(t, connector, featureIn)
				// We need to truncate the time to workaround pgx driver timezone issue
				t1 := time.Now().Truncate(time.Hour * 24)
				t2 := t1.Add(time.Hour).Truncate(0)
				t3 := t2.Add(time.Hour).Truncate(0)

				grant1, err := connector.CreateGrant(ctx, credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: t1,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationMonth,
						Count:    1,
					},
					Rollover: &credit.GrantRollover{
						Type: credit.GrantRolloverTypeRemainingAmount,
					},
				})
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

				_, rolloverGrants, err := connector.Reset(ctx, credit.Reset{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					EffectiveAt: t3,
				})
				assert.NoError(t, err)

				// Get grants
				grants, err := connector.ListGrants(ctx, credit.ListGrantsParams{
					Namespace:         namespace,
					LedgerIDs:         []credit.LedgerID{ledger.ID},
					FromHighWatermark: true,
				})
				assert.NoError(t, err)

				// Assert remaining amount
				reamingAmount := grant1.Amount - usage
				assert.Equal(t, reamingAmount, rolloverGrants[0].Amount)

				// Assert: grants after reset should be the same as rollover grants
				assert.Equal(t,
					removeTimestampsFromGrants(rolloverGrants),
					removeTimestampsFromGrants(grants),
				)
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
			connector := NewPostgresConnector(slog.Default(), databaseClient, streamingConnector, meterRepository)

			// let's provision a ledger
			ledger, err := connector.CreateLedger(context.Background(), credit.Ledger{
				Namespace: namespace,
				Subject:   ulid.Make().String(),
			})

			assert.NoError(t, err)

			tc.test(t, connector, streamingConnector, databaseClient, ledger)
		})
	}
}
