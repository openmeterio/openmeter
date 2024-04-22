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

func TestPostgresConnectorReset(t *testing.T) {
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
		test        func(t *testing.T, connector credit.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client, lockManager credit_model.LockManager)
	}{
		{
			name:        "Reset",
			description: "Should move high watermark ahead",
			test: func(t *testing.T, connector credit.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client, lockManager credit_model.LockManager) {
				ctx := context.Background()
				product := createProduct(t, connector, namespace, productIn)
				// We need to truncate the time to workaround pgx driver timezone issue
				// We also move it to the past to avoid timezone issues
				t1 := time.Now().Truncate(time.Hour * 24).Add(-time.Hour * 24)
				t2 := t1.Add(time.Hour).Truncate(0)
				t3 := t2.Add(time.Hour).Truncate(0)

				_, err := connector.CreateGrant(ctx, namespace, credit_model.Grant{
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
				})
				assert.NoError(t, err)

				// We need to add a row to the streaming connector as we call balance in the reset
				// even though there is no grant to rollover
				streamingConnector.addRow(meter.Slug, models.MeterQueryRow{})

				// Reset
				reset, rolloverGrants, err := connector.Reset(ctx, namespace, credit_model.Reset{
					Subject:     subject,
					EffectiveAt: t3,
				})
				assert.NoError(t, err)
				assert.NotNil(t, reset.ID)

				// Get high watermark
				highWatermark, err := connector.GetHighWatermark(ctx, namespace, subject)
				assert.NoError(t, err)
				assert.Equal(t, credit_model.HighWatermark{
					Subject: subject,
					Time:    t3,
				}, highWatermark)

				// Get grants
				grants, err := connector.ListGrants(ctx, namespace, credit.ListGrantsParams{
					Subjects:          []string{subject},
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
			test: func(t *testing.T, connector credit.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client, lockManager credit_model.LockManager) {
				ctx := context.Background()
				product := createProduct(t, connector, namespace, productIn)
				// We need to truncate the time to workaround pgx driver timezone issue
				t1 := time.Now().Truncate(time.Hour * 24)
				t2 := t1.Add(time.Hour).Truncate(0)
				t3 := t2.Add(time.Hour).Truncate(0)

				_, err := connector.CreateGrant(ctx, namespace, credit_model.Grant{
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
						Type: credit_model.GrantRolloverTypeOriginalAmount,
					},
				})
				assert.NoError(t, err)

				// We need to add a row to the streaming connector as we call balance in the reset
				// even though rollover grant is original amount
				streamingConnector.addRow(meter.Slug, models.MeterQueryRow{})

				// Reset
				_, rolloverGrants, err := connector.Reset(ctx, namespace, credit_model.Reset{
					Subject:     subject,
					EffectiveAt: t3,
				})
				assert.NoError(t, err)

				// Get grants
				grants, err := connector.ListGrants(ctx, namespace, credit.ListGrantsParams{
					Subjects:          []string{subject},
					FromHighWatermark: true,
				})
				assert.NoError(t, err)

				// Grants after reset should be the same as rollover grants
				assert.Equal(t, rolloverGrants, grants)
			},
		},
		{
			name:        "ResetWithRemainingRollover",
			description: "Should rollover grants with remaining amount",
			test: func(t *testing.T, connector credit.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client, lockManager credit_model.LockManager) {
				ctx := context.Background()
				product := createProduct(t, connector, namespace, productIn)
				// We need to truncate the time to workaround pgx driver timezone issue
				t1 := time.Now().Truncate(time.Hour * 24)
				t2 := t1.Add(time.Hour).Truncate(0)
				t3 := t2.Add(time.Hour).Truncate(0)

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

				usage := 1.0
				streamingConnector.addRow(meter.Slug, models.MeterQueryRow{
					Value: usage,
					// Grant 1's effective time is t1, so usage starts from t1
					WindowStart: t1,
					// Reset time is t3, so usage ends at t3
					WindowEnd: t3,
					GroupBy:   map[string]*string{},
				})

				_, rolloverGrants, err := connector.Reset(ctx, namespace, credit_model.Reset{
					Subject:     subject,
					EffectiveAt: t3,
				})
				assert.NoError(t, err)

				// Get grants
				grants, err := connector.ListGrants(ctx, namespace, credit.ListGrantsParams{
					Subjects:          []string{subject},
					FromHighWatermark: true,
				})
				assert.NoError(t, err)

				// Assert remaining amount
				reamingAmount := grant1.Amount - usage
				assert.Equal(t, reamingAmount, rolloverGrants[0].Amount)

				// Assert: grants after reset should be the same as rollover grants
				assert.Equal(t, rolloverGrants, grants)
			},
		},
		{
			name:        "ResetLock",
			description: "Should manage locks correctly",
			test: func(t *testing.T, connector credit.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client, lockManager credit_model.LockManager) {
				ctx := context.Background()
				// We need to add a row to the streaming connector as we call balance in the reset
				// even though there is no grant to rollover
				streamingConnector.addRow(meter.Slug, models.MeterQueryRow{})

				// Reset
				resetIn := credit_model.Reset{
					Subject:     subject,
					EffectiveAt: time.Now(),
				}

				// 1. Lock ledger
				lock, err := lockManager.Obtain(ctx, namespace, subject)
				assert.NoError(t, err)

				// 2. Should fail to obtain lock
				_, _, err = connector.Reset(ctx, namespace, resetIn)
				expectedErr := credit_model.LockErrNotObtained{Namespace: namespace, Subject: subject}
				assert.Error(t, err, expectedErr.Error())

				// 3. Lock should remain locked
				_, err = lockManager.Obtain(ctx, namespace, subject)
				assert.Error(t, err, expectedErr.Error())

				err = lockManager.Release(ctx, lock)
				assert.NoError(t, err)

				// 4. Should succeed to obtain lock
				_, _, err = connector.Reset(ctx, namespace, resetIn)
				assert.NoError(t, err)

				// 5. Lock should be released (check via obtaining lock again)
				lock, err = lockManager.Obtain(ctx, namespace, subject)
				assert.NoError(t, err)
				err = lockManager.Release(ctx, lock)
				assert.NoError(t, err)
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

			tc.test(t, connector, streamingConnector, databaseClient, lockManager)
		})
	}
}
