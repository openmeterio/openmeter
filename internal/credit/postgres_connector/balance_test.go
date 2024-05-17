package postgres_connector

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	meter_model "github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestPostgresConnectorBalances(t *testing.T) {
	namespace := "default"
	meter1 := models.Meter{
		Namespace:   namespace,
		ID:          "meter-1",
		Slug:        "meter-1",
		Aggregation: models.MeterAggregationSum,
	}
	meter2 := models.Meter{
		Namespace:   namespace,
		ID:          "meter-2",
		Slug:        "meter-2",
		Aggregation: models.MeterAggregationSum,
	}
	meterRepository := meter_model.NewInMemoryRepository([]models.Meter{meter1, meter2})
	featureIn1 := credit.Feature{
		Namespace: namespace,
		MeterSlug: meter1.Slug,
		Name:      "feature-1",
	}
	featureIn2 := credit.Feature{
		Namespace: namespace,
		MeterSlug: meter2.Slug,
		Name:      "feature-2",
	}

	tt := []struct {
		name        string
		description string
		test        func(t *testing.T, connector credit.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client, ledger credit.Ledger)
	}{
		{
			name:        "GetBalance",
			description: "Should return balance",
			test: func(t *testing.T, connector credit.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client, ledger credit.Ledger) {
				ctx := context.Background()
				feature := createFeature(t, connector, featureIn1)
				// We need to truncate the time to workaround pgx driver timezone issue
				// We also move it to the past to avoid timezone issues
				t1 := time.Now().Truncate(time.Hour * 24).Add(-time.Hour * 24)
				t2 := t1.Add(time.Hour).Truncate(0)

				grant, err := connector.CreateGrant(ctx, credit.Grant{
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

				streamingConnector.addRow(meter1.Slug, models.MeterQueryRow{
					Value:       1,
					WindowStart: t1,
					WindowEnd:   t2,
					GroupBy:     map[string]*string{},
				})

				// Get balance
				balance, err := connector.GetBalance(ctx, credit.NewNamespacedLedgerID(namespace, ledger.ID), t2)
				assert.NoError(t, err)

				// Assert balance
				assert.Equal(t, credit.Balance{
					LedgerID: ledger.ID,
					Subject:  ledger.Subject,
					FeatureBalances: []credit.FeatureBalance{
						{
							Feature: feature,
							Balance: 99,
						},
					},
					GrantBalances: []credit.GrantBalance{
						{
							Grant:   grant,
							Balance: 99,
						},
					},
				}, balance)
			},
		},
		{
			name:        "GetBalanceWithReset",
			description: "Should return balance after reset",
			test: func(t *testing.T, connector credit.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client, ledger credit.Ledger) {
				ctx := context.Background()
				feature := createFeature(t, connector, featureIn1)
				t1, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:01:00Z", time.UTC)
				t2, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:02:00Z", time.UTC)
				t3, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:03:00Z", time.UTC)

				reset := credit.Reset{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					EffectiveAt: t1,
				}
				_, _, err := connector.Reset(ctx, reset)
				assert.NoError(t, err)

				grant, err := connector.CreateGrant(ctx, credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: t2,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationMonth,
						Count:    1,
					},
				})
				assert.NoError(t, err)

				streamingConnector.addRow(meter1.Slug, models.MeterQueryRow{
					Value:       1,
					WindowStart: t2,
					WindowEnd:   t3,
					GroupBy:     map[string]*string{},
				})

				// Get balance
				balance, err := connector.GetBalance(ctx, credit.NewNamespacedLedgerID(ledger.Namespace, ledger.ID), t3)
				assert.NoError(t, err)

				// FIXME
				balance.GrantBalances[0].Grant.EffectiveAt = grant.EffectiveAt

				// Assert balance
				assert.Equal(t, credit.Balance{
					LedgerID: ledger.ID,
					Subject:  ledger.Subject,
					FeatureBalances: []credit.FeatureBalance{
						{
							Feature: feature,
							Balance: 99,
						},
					},
					GrantBalances: []credit.GrantBalance{
						{
							Grant:   grant,
							Balance: 99,
						},
					},
				}, balance)
			},
		},
		{
			name:        "GetBalanceWithVoidGrant",
			description: "Should exclude voided grant from balance",
			test: func(t *testing.T, connector credit.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client, ledger credit.Ledger) {
				ctx := context.Background()
				feature := createFeature(t, connector, featureIn1)
				t1, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:01:00Z", time.UTC)
				t2, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:02:00Z", time.UTC)

				reset := credit.Reset{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					EffectiveAt: t1,
				}
				_, _, err := connector.Reset(ctx, reset)
				assert.NoError(t, err)

				grant, err := connector.CreateGrant(ctx, credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: t2,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationMonth,
						Count:    1,
					},
				})
				assert.NoError(t, err)

				_, err = connector.VoidGrant(ctx, grant)
				assert.NoError(t, err)

				streamingConnector.addRow(meter1.Slug, models.MeterQueryRow{
					Value:       1,
					WindowStart: t1,
					WindowEnd:   t2,
					GroupBy:     map[string]*string{},
				})

				// Get balance
				balance, err := connector.GetBalance(ctx, credit.NewNamespacedLedgerID(namespace, ledger.ID), t2)
				assert.NoError(t, err)

				// Assert balance
				assert.Equal(t, credit.Balance{
					LedgerID:        ledger.ID,
					Subject:         ledger.Subject,
					FeatureBalances: []credit.FeatureBalance{},
					GrantBalances:   []credit.GrantBalance{},
				}, balance)
			},
		},
		{
			name:        "GetBalanceWithPiorities",
			description: "Should burn down grant with highest priority first",
			test: func(t *testing.T, connector credit.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client, ledger credit.Ledger) {
				ctx := context.Background()
				feature := createFeature(t, connector, featureIn1)
				t1, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:01:00Z", time.UTC)
				t2, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:02:00Z", time.UTC)

				grant1, err := connector.CreateGrant(ctx, credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      10,
					Priority:    1,
					EffectiveAt: t1,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationMonth,
						Count:    1,
					},
				})
				assert.NoError(t, err)

				grant2, err := connector.CreateGrant(ctx, credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    2,
					EffectiveAt: t1,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationMonth,
						Count:    1,
					},
				})
				assert.NoError(t, err)

				streamingConnector.addRow(meter1.Slug, models.MeterQueryRow{
					Value:       20,
					WindowStart: t1,
					WindowEnd:   t2,
					GroupBy:     map[string]*string{},
				})

				// Get balance
				balance, err := connector.GetBalance(ctx, credit.NewNamespacedLedgerID(namespace, ledger.ID), t2)
				assert.NoError(t, err)

				// FIXME
				balance.GrantBalances[0].Grant.EffectiveAt = grant1.EffectiveAt
				balance.GrantBalances[1].Grant.EffectiveAt = grant2.EffectiveAt

				// Assert balance
				assert.Equal(t, credit.Balance{
					LedgerID: ledger.ID,
					Subject:  ledger.Subject,
					FeatureBalances: []credit.FeatureBalance{
						{
							Feature: feature,
							Balance: 90,
						},
					},
					GrantBalances: []credit.GrantBalance{
						{
							Grant:   grant1,
							Balance: 0,
						},
						{
							Grant:   grant2,
							Balance: 90,
						},
					},
				}, balance)
			},
		},
		{
			name:        "GetBalanceWithDifferentGrantExpiration",
			description: "Should burn down grant that expires first",
			test: func(t *testing.T, connector credit.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client, ledger credit.Ledger) {
				ctx := context.Background()
				feature := createFeature(t, connector, featureIn1)
				t1, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:01:00Z", time.UTC)
				t2, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:02:00Z", time.UTC)

				grant1, err := connector.CreateGrant(ctx, credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      10,
					Priority:    1,
					EffectiveAt: t1,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationHour,
						Count:    1,
					},
				})
				assert.NoError(t, err)

				grant2, err := connector.CreateGrant(ctx, credit.Grant{
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

				streamingConnector.addRow(meter1.Slug, models.MeterQueryRow{
					Value:       20,
					WindowStart: t1,
					WindowEnd:   t2,
					GroupBy:     map[string]*string{},
				})

				// Get balance
				balance, err := connector.GetBalance(ctx, credit.NewNamespacedLedgerID(namespace, ledger.ID), t2)
				assert.NoError(t, err)

				// FIXME
				balance.GrantBalances[0].Grant.EffectiveAt = grant1.EffectiveAt
				balance.GrantBalances[1].Grant.EffectiveAt = grant2.EffectiveAt

				// Assert balance
				assert.Equal(t, credit.Balance{
					LedgerID: ledger.ID,
					Subject:  ledger.Subject,
					FeatureBalances: []credit.FeatureBalance{
						{
							Feature: feature,
							Balance: 90,
						},
					},
					GrantBalances: []credit.GrantBalance{
						{
							Grant:   grant1,
							Balance: 0,
						},
						{
							Grant:   grant2,
							Balance: 90,
						},
					},
				}, balance)
			},
		},
		{
			name:        "GetBalanceWithMultipleFeatures",
			description: "Should burn down the right feature",
			test: func(t *testing.T, connector credit.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client, ledger credit.Ledger) {
				ctx := context.Background()
				feature1 := createFeature(t, connector, featureIn1)
				feature2 := createFeature(t, connector, featureIn2)
				t1, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:01:00Z", time.UTC)
				t2, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:02:00Z", time.UTC)

				grant1, err := connector.CreateGrant(ctx, credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature1.ID,
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

				grant2, err := connector.CreateGrant(ctx, credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature2.ID,
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

				streamingConnector.addRow(meter1.Slug, models.MeterQueryRow{
					Value:       1,
					WindowStart: t1,
					WindowEnd:   t2,
					GroupBy:     map[string]*string{},
				})
				streamingConnector.addRow(meter2.Slug, models.MeterQueryRow{
					Value:       10,
					WindowStart: t1,
					WindowEnd:   t2,
					GroupBy:     map[string]*string{},
				})

				// Get balance
				balance, err := connector.GetBalance(ctx, credit.NewNamespacedLedgerID(namespace, ledger.ID), t2)
				assert.NoError(t, err)

				// FIXME
				balance.GrantBalances[0].Grant.EffectiveAt = grant1.EffectiveAt
				balance.GrantBalances[1].Grant.EffectiveAt = grant2.EffectiveAt

				// Assert balance
				assert.ElementsMatch(t, []credit.FeatureBalance{
					{
						Feature: feature1,
						Balance: 99,
					},
					{
						Feature: feature2,
						Balance: 90,
					},
				}, balance.FeatureBalances)

				assert.ElementsMatch(t, []credit.GrantBalance{
					{
						Grant:   grant1,
						Balance: 99,
					},
					{
						Grant:   grant2,
						Balance: 90,
					},
				}, balance.GrantBalances)
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
			// Initialize streaming connector with data points at time.Zero
			streamingConnector.addRow(meter1.Slug, models.MeterQueryRow{})
			streamingConnector.addRow(meter2.Slug, models.MeterQueryRow{})
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

func newMockStreamingConnector() *mockStreamingConnector {
	return &mockStreamingConnector{
		rows: map[string][]models.MeterQueryRow{},
	}
}

type mockStreamingConnector struct {
	rows map[string][]models.MeterQueryRow
}

// TODO: ideally we would use github.com/stretchr/testify/mock for this
func (m *mockStreamingConnector) addRow(meterSlug string, row models.MeterQueryRow) {
	m.rows[meterSlug] = append(m.rows[meterSlug], row)
}

func (m *mockStreamingConnector) ListEvents(ctx context.Context, namespace string, params streaming.ListEventsParams) ([]api.IngestedEvent, error) {
	return []api.IngestedEvent{}, nil
}

func (m *mockStreamingConnector) CreateMeter(ctx context.Context, namespace string, meter *models.Meter) error {
	return nil
}

func (m *mockStreamingConnector) DeleteMeter(ctx context.Context, namespace string, meterSlug string) error {
	return nil
}

func (m *mockStreamingConnector) QueryMeter(ctx context.Context, namespace string, meterSlug string, params *streaming.QueryParams) ([]models.MeterQueryRow, error) {
	rows := []models.MeterQueryRow{}
	if _, ok := m.rows[meterSlug]; !ok {
		return rows, &models.MeterNotFoundError{MeterSlug: meterSlug}
	}

	for _, row := range m.rows[meterSlug] {
		if row.WindowStart.Equal(*params.From) && row.WindowEnd.Equal(*params.To) {
			rows = append(rows, row)
		}
	}

	return rows, nil
}

func (m *mockStreamingConnector) ListMeterSubjects(ctx context.Context, namespace string, meterSlug string, from *time.Time, to *time.Time) ([]string, error) {
	return []string{}, nil
}
