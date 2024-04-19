package postgres_connector

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/peterldowns/pgtestdb"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/credit"
	inmemory_lock "github.com/openmeterio/openmeter/internal/credit/inmemory_lock"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	meter_model "github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/internal/streaming"
	credit_model "github.com/openmeterio/openmeter/pkg/credit"
	"github.com/openmeterio/openmeter/pkg/models"
	product_model "github.com/openmeterio/openmeter/pkg/product"
)

func TestPostgresConnectorBalances(t *testing.T) {
	namespace := "default"
	subject := "subject-1"
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
	productIn1 := product_model.Product{
		Namespace: namespace,
		MeterSlug: meter1.Slug,
		Name:      "product-1",
	}
	productIn2 := product_model.Product{
		Namespace: namespace,
		MeterSlug: meter2.Slug,
		Name:      "product-2",
	}

	tt := []struct {
		name        string
		description string
		test        func(t *testing.T, connector credit.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client)
	}{
		{
			name:        "GetBalance",
			description: "Should return balance",
			test: func(t *testing.T, connector credit.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client) {
				ctx := context.Background()
				product := createProduct(t, connector, namespace, productIn1)
				// We need to truncate the time to workaround pgx driver timezone issue
				t1 := time.Now().Truncate(time.Hour * 24)
				t2 := t1.Add(time.Hour).Truncate(0)

				grant, err := connector.CreateGrant(ctx, namespace, credit_model.Grant{
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

				streamingConnector.addRow(meter1.Slug, models.MeterQueryRow{
					Value:       1,
					WindowStart: t1,
					WindowEnd:   t2,
					GroupBy:     map[string]*string{},
				})

				// Get balance
				balance, err := connector.GetBalance(ctx, namespace, subject, t2)
				assert.NoError(t, err)

				// Assert balance
				assert.Equal(t, credit_model.Balance{
					Subject: subject,
					ProductBalances: []credit_model.ProductBalance{
						{
							Product: product,
							Balance: 99,
						},
					},
					GrantBalances: []credit_model.GrantBalance{
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
			test: func(t *testing.T, connector credit.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client) {
				ctx := context.Background()
				product := createProduct(t, connector, namespace, productIn1)
				t1, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:00:00Z", time.UTC)
				t2, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:01:00Z", time.UTC)
				t3, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:02:00Z", time.UTC)

				reset := credit_model.Reset{
					Subject:     subject,
					EffectiveAt: t1,
				}
				_, _, err := connector.Reset(ctx, namespace, reset)
				assert.NoError(t, err)

				grant, err := connector.CreateGrant(ctx, namespace, credit_model.Grant{
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
				})
				assert.NoError(t, err)

				streamingConnector.addRow(meter1.Slug, models.MeterQueryRow{
					Value:       1,
					WindowStart: t2,
					WindowEnd:   t3,
					GroupBy:     map[string]*string{},
				})

				// Get balance
				balance, err := connector.GetBalance(ctx, namespace, subject, t3)
				assert.NoError(t, err)

				// FIXME
				balance.GrantBalances[0].Grant.EffectiveAt = grant.EffectiveAt

				// Assert balance
				assert.Equal(t, credit_model.Balance{
					Subject: subject,
					ProductBalances: []credit_model.ProductBalance{
						{
							Product: product,
							Balance: 99,
						},
					},
					GrantBalances: []credit_model.GrantBalance{
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
			test: func(t *testing.T, connector credit.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client) {
				ctx := context.Background()
				product := createProduct(t, connector, namespace, productIn1)
				t1, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:00:00Z", time.UTC)
				t2, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:01:00Z", time.UTC)

				reset := credit_model.Reset{
					Subject:     subject,
					EffectiveAt: t1,
				}
				_, _, err := connector.Reset(ctx, namespace, reset)
				assert.NoError(t, err)

				grant, err := connector.CreateGrant(ctx, namespace, credit_model.Grant{
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
				})
				assert.NoError(t, err)

				_, err = connector.VoidGrant(ctx, namespace, grant)
				assert.NoError(t, err)

				streamingConnector.addRow(meter1.Slug, models.MeterQueryRow{
					Value:       1,
					WindowStart: t1,
					WindowEnd:   t2,
					GroupBy:     map[string]*string{},
				})

				// Get balance
				balance, err := connector.GetBalance(ctx, namespace, subject, t2)
				assert.NoError(t, err)

				// Assert balance
				assert.Equal(t, credit_model.Balance{
					Subject:         subject,
					ProductBalances: []credit_model.ProductBalance{},
					GrantBalances:   []credit_model.GrantBalance{},
				}, balance)
			},
		},
		{
			name:        "GetBalanceWithPiorities",
			description: "Should burn down grant with highest priority first",
			test: func(t *testing.T, connector credit.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client) {
				ctx := context.Background()
				product := createProduct(t, connector, namespace, productIn1)
				t1, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:00:00Z", time.UTC)
				t2, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:01:00Z", time.UTC)

				grant1, err := connector.CreateGrant(ctx, namespace, credit_model.Grant{
					Subject:     subject,
					ProductID:   product.ID,
					Type:        credit_model.GrantTypeUsage,
					Amount:      10,
					Priority:    1,
					EffectiveAt: t1,
					Expiration: credit_model.ExpirationPeriod{
						Duration: credit_model.ExpirationPeriodDurationMonth,
						Count:    1,
					},
				})
				assert.NoError(t, err)

				grant2, err := connector.CreateGrant(ctx, namespace, credit_model.Grant{
					Subject:     subject,
					ProductID:   product.ID,
					Type:        credit_model.GrantTypeUsage,
					Amount:      100,
					Priority:    2,
					EffectiveAt: t1,
					Expiration: credit_model.ExpirationPeriod{
						Duration: credit_model.ExpirationPeriodDurationMonth,
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
				balance, err := connector.GetBalance(ctx, namespace, subject, t2)
				assert.NoError(t, err)

				// FIXME
				balance.GrantBalances[0].Grant.EffectiveAt = grant1.EffectiveAt
				balance.GrantBalances[1].Grant.EffectiveAt = grant2.EffectiveAt

				// Assert balance
				assert.Equal(t, credit_model.Balance{
					Subject: subject,
					ProductBalances: []credit_model.ProductBalance{
						{
							Product: product,
							Balance: 90,
						},
					},
					GrantBalances: []credit_model.GrantBalance{
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
			test: func(t *testing.T, connector credit.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client) {
				ctx := context.Background()
				product := createProduct(t, connector, namespace, productIn1)
				t1, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:00:00Z", time.UTC)
				t2, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:01:00Z", time.UTC)

				grant1, err := connector.CreateGrant(ctx, namespace, credit_model.Grant{
					Subject:     subject,
					ProductID:   product.ID,
					Type:        credit_model.GrantTypeUsage,
					Amount:      10,
					Priority:    1,
					EffectiveAt: t1,
					Expiration: credit_model.ExpirationPeriod{
						Duration: credit_model.ExpirationPeriodDurationHour,
						Count:    1,
					},
				})
				assert.NoError(t, err)

				grant2, err := connector.CreateGrant(ctx, namespace, credit_model.Grant{
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

				streamingConnector.addRow(meter1.Slug, models.MeterQueryRow{
					Value:       20,
					WindowStart: t1,
					WindowEnd:   t2,
					GroupBy:     map[string]*string{},
				})

				// Get balance
				balance, err := connector.GetBalance(ctx, namespace, subject, t2)
				assert.NoError(t, err)

				// FIXME
				balance.GrantBalances[0].Grant.EffectiveAt = grant1.EffectiveAt
				balance.GrantBalances[1].Grant.EffectiveAt = grant2.EffectiveAt

				// Assert balance
				assert.Equal(t, credit_model.Balance{
					Subject: subject,
					ProductBalances: []credit_model.ProductBalance{
						{
							Product: product,
							Balance: 90,
						},
					},
					GrantBalances: []credit_model.GrantBalance{
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
			name:        "GetBalanceWithMultipleProducts",
			description: "Should burn down the right product",
			test: func(t *testing.T, connector credit.Connector, streamingConnector *mockStreamingConnector, db_client *db.Client) {
				ctx := context.Background()
				product1 := createProduct(t, connector, namespace, productIn1)
				product2 := createProduct(t, connector, namespace, productIn2)
				t1, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:00:00Z", time.UTC)
				t2, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:01:00Z", time.UTC)

				grant1, err := connector.CreateGrant(ctx, namespace, credit_model.Grant{
					Subject:     subject,
					ProductID:   product1.ID,
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

				grant2, err := connector.CreateGrant(ctx, namespace, credit_model.Grant{
					Subject:     subject,
					ProductID:   product2.ID,
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
				balance, err := connector.GetBalance(ctx, namespace, subject, t2)
				assert.NoError(t, err)

				// FIXME
				balance.GrantBalances[0].Grant.EffectiveAt = grant1.EffectiveAt
				balance.GrantBalances[1].Grant.EffectiveAt = grant2.EffectiveAt

				// Assert balance
				assert.ElementsMatch(t, []credit_model.ProductBalance{
					{
						Product: product1,
						Balance: 99,
					},
					{
						Product: product2,
						Balance: 90,
					},
				}, balance.ProductBalances)

				assert.ElementsMatch(t, []credit_model.GrantBalance{
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
			driver := entsql.OpenDB(dialect.Postgres, pgtestdb.New(t, pgtestdb.Config{
				DriverName: "pgx",
				User:       "postgres",
				Password:   "postgres",
				Host:       "localhost",
				Port:       "5432",
				Options:    "sslmode=disable",
			}, &EntMigrator{}))
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
