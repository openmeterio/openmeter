package postgres_connector

import (
	"context"
	"log/slog"
	"testing"
	"time"

	om_tetsutils "github.com/openmeterio/openmeter/internal/testutils"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/testutils"
	meter_model "github.com/openmeterio/openmeter/internal/meter"
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

	oldSetup := func(streamingConnector *testutils.MockStreamingConnector, connector credit.Connector) (credit.Ledger, error) {
		// Initialize streaming connector with data points at time.Zero
		streamingConnector.AddRow(meter1.Slug, models.MeterQueryRow{})
		streamingConnector.AddRow(meter2.Slug, models.MeterQueryRow{})

		// let's provision a ledger
		ledger, err := connector.CreateLedger(context.Background(), credit.Ledger{
			Namespace: namespace,
			Subject:   ulid.Make().String(),
		})

		return ledger, err
	}

	tt := []struct {
		name        string
		description string
		test        func(t *testing.T, connector credit.Connector, streamingConnector *testutils.MockStreamingConnector, db_client *db.Client)
	}{
		{
			name:        "GetBalance",
			description: "Should return balance",
			test: func(t *testing.T, connector credit.Connector, streamingConnector *testutils.MockStreamingConnector, db_client *db.Client) {
				t.Parallel()
				ledger, err := oldSetup(streamingConnector, connector)
				assert.NoError(t, err)
				ctx := context.Background()
				feature := testutils.CreateFeature(t, connector, featureIn1)
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

				streamingConnector.AddRow(meter1.Slug, models.MeterQueryRow{
					Value:       1,
					WindowStart: t1,
					WindowEnd:   t2,
					GroupBy:     map[string]*string{},
				})

				// Get balance
				balance, err := connector.GetBalance(ctx, credit.NewNamespacedLedgerID(namespace, ledger.ID), t2)
				assert.NoError(t, err)

				// Assert balance
				assert.Equal(t,
					testutils.RemoveTimestampsFromBalance(credit.Balance{
						LedgerID: ledger.ID,
						Subject:  ledger.Subject,
						FeatureBalances: []credit.FeatureBalance{
							{
								Feature: feature,
								Balance: 99,
								Usage:   1,
							},
						},
						GrantBalances: []credit.GrantBalance{
							{
								Grant:   grant,
								Balance: 99,
							},
						},
					}),
					testutils.RemoveTimestampsFromBalance(balance),
				)
			},
		},
		{
			name:        "GetBalanceWithReset",
			description: "Should return balance after reset",
			test: func(t *testing.T, connector credit.Connector, streamingConnector *testutils.MockStreamingConnector, db_client *db.Client) {
				t.Parallel()
				ledger, err := oldSetup(streamingConnector, connector)
				assert.NoError(t, err)
				ctx := context.Background()
				feature := testutils.CreateFeature(t, connector, featureIn1)
				t1, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:01:00Z", time.UTC)
				t2, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:02:00Z", time.UTC)
				t3, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:03:00Z", time.UTC)

				reset := credit.Reset{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					EffectiveAt: t1,
				}
				_, _, err = connector.Reset(ctx, reset)
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

				streamingConnector.AddRow(meter1.Slug, models.MeterQueryRow{
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
				assert.Equal(t,
					testutils.RemoveTimestampsFromBalance(
						credit.Balance{
							LedgerID: ledger.ID,
							Subject:  ledger.Subject,
							FeatureBalances: []credit.FeatureBalance{
								{
									Feature: feature,
									Balance: 99,
									Usage:   1,
								},
							},
							GrantBalances: []credit.GrantBalance{
								{
									Grant:   grant,
									Balance: 99,
								},
							},
						}),
					testutils.RemoveTimestampsFromBalance(balance),
				)
			},
		},
		{
			name:        "GetBalanceWithVoidGrant",
			description: "Should exclude voided grant from balance",
			test: func(t *testing.T, connector credit.Connector, streamingConnector *testutils.MockStreamingConnector, db_client *db.Client) {
				t.Parallel()
				ledger, err := oldSetup(streamingConnector, connector)
				assert.NoError(t, err)
				ctx := context.Background()
				feature := testutils.CreateFeature(t, connector, featureIn1)
				t1, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:01:00Z", time.UTC)
				t2, _ := time.ParseInLocation(time.RFC3339, "2024-01-01T00:02:00Z", time.UTC)

				reset := credit.Reset{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					EffectiveAt: t1,
				}
				_, _, err = connector.Reset(ctx, reset)
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

				streamingConnector.AddRow(meter1.Slug, models.MeterQueryRow{
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
			test: func(t *testing.T, connector credit.Connector, streamingConnector *testutils.MockStreamingConnector, db_client *db.Client) {
				t.Parallel()
				ledger, err := oldSetup(streamingConnector, connector)
				assert.NoError(t, err)
				ctx := context.Background()
				feature := testutils.CreateFeature(t, connector, featureIn1)
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

				streamingConnector.AddRow(meter1.Slug, models.MeterQueryRow{
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
				assert.Equal(t,
					testutils.RemoveTimestampsFromBalance(
						credit.Balance{
							LedgerID: ledger.ID,
							Subject:  ledger.Subject,
							FeatureBalances: []credit.FeatureBalance{
								{
									Feature: feature,
									Balance: 90,
									Usage:   20,
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
						}),
					testutils.RemoveTimestampsFromBalance(balance),
				)
			},
		},
		{
			name:        "GetBalanceWithDifferentGrantExpiration",
			description: "Should burn down grant that expires first",
			test: func(t *testing.T, connector credit.Connector, streamingConnector *testutils.MockStreamingConnector, db_client *db.Client) {
				t.Parallel()
				ledger, err := oldSetup(streamingConnector, connector)
				assert.NoError(t, err)
				ctx := context.Background()
				feature := testutils.CreateFeature(t, connector, featureIn1)
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

				streamingConnector.AddRow(meter1.Slug, models.MeterQueryRow{
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
				assert.Equal(t,
					testutils.RemoveTimestampsFromBalance(
						credit.Balance{
							LedgerID: ledger.ID,
							Subject:  ledger.Subject,
							FeatureBalances: []credit.FeatureBalance{
								{
									Feature: feature,
									Balance: 90,
									Usage:   20,
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
						}),
					testutils.RemoveTimestampsFromBalance(balance),
				)
			},
		},
		{
			name:        "GetBalanceWithMultipleFeatures",
			description: "Should burn down the right feature",
			test: func(t *testing.T, connector credit.Connector, streamingConnector *testutils.MockStreamingConnector, db_client *db.Client) {
				t.Parallel()
				ledger, err := oldSetup(streamingConnector, connector)
				assert.NoError(t, err)
				ctx := context.Background()
				feature1 := testutils.CreateFeature(t, connector, featureIn1)
				feature2 := testutils.CreateFeature(t, connector, featureIn2)
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

				streamingConnector.AddRow(meter1.Slug, models.MeterQueryRow{
					Value:       1,
					WindowStart: t1,
					WindowEnd:   t2,
					GroupBy:     map[string]*string{},
				})
				streamingConnector.AddRow(meter2.Slug, models.MeterQueryRow{
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
				assert.ElementsMatch(t,
					testutils.RemoveTimestampsFromFeatureBalances(
						[]credit.FeatureBalance{
							{
								Feature: feature1,
								Balance: 99,
								Usage:   1,
							},
							{
								Feature: feature2,
								Balance: 90,
								Usage:   10,
							},
						}),
					testutils.RemoveTimestampsFromFeatureBalances(balance.FeatureBalances),
				)

				assert.ElementsMatch(t,
					testutils.RemoveTimestampsFromGrantBalances(
						[]credit.GrantBalance{
							{
								Grant:   grant1,
								Balance: 99,
							},
							{
								Grant:   grant2,
								Balance: 90,
							},
						}),
					testutils.RemoveTimestampsFromGrantBalances(balance.GrantBalances),
				)
			},
		},
		{
			name:        "Should include usage between ledger creation and first grant",
			description: `The ledger can exist before the first grant is created so we should account for that usage.`,
			test: func(t *testing.T, connector credit.Connector, streamingConnector *testutils.MockStreamingConnector, db_client *db.Client) {
				t.Skip(`
                FIXME
                This test would fail as currently this is not really possible to do.

                1. Taking the angle that what we're trying to do is to calculate the balance, if no balanace has been granted then it doesn't make sense to account for usage in that period.

                2. From an enforcement standpoint (based on balance), if no balance's been granted then enforcement will work even if usage happened.

                3. From a history standpoint this is an issue as this is usage reported to an existing ledger that's neither displayed, neither accounted for in the balance.
                `)
				start, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
				assert.NoError(t, err)
				subject := ulid.Make().String()
				ledgerID := credit.LedgerID(ulid.Make().String())

				// Create Ledger

				ledger, err := connector.CreateLedger(context.Background(), credit.Ledger{
					Namespace: namespace,
					ID:        ledgerID,
					Subject:   subject,
					CreatedAt: start,
				})
				assert.NoError(t, err)

				// Register Usage
				usage1 := models.MeterQueryRow{
					Value:       1,
					WindowStart: start.Add(time.Minute),
					WindowEnd:   start.Add(time.Minute * 2),
					GroupBy:     map[string]*string{},
				}
				streamingConnector.AddRow(meter1.Slug, usage1)

				at3m := start.Add(time.Minute * 3)
				at4m := start.Add(time.Minute * 4)

				// Create Feature & Grant
				feature := testutils.CreateFeature(t, connector, featureIn1)
				grant, err := connector.CreateGrant(context.Background(), credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: at3m,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationYear,
						Count:    1,
					},
					CreatedAt: &at3m,
					UpdatedAt: &at3m,
				})

				assert.NoError(t, err)

				// Get Balance
				balance, err := connector.GetBalance(context.Background(), credit.NewNamespacedLedgerID(namespace, ledger.ID), at4m)
				assert.NoError(t, err)
				assert.Equal(t, 1, len(balance.FeatureBalances))

				featureBalance := balance.FeatureBalances[0].Balance
				grantedAmount := grant.Amount
				usedAmount := usage1.Value

				assert.Equal(t, grantedAmount-usedAmount, featureBalance)
			},
		},
		{
			name:        "Should not include usage before ledger creation",
			description: `The meters can exist before the ledger is created but we should not account for that usage.`,
			test: func(t *testing.T, connector credit.Connector, streamingConnector *testutils.MockStreamingConnector, db_client *db.Client) {
				start, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
				assert.NoError(t, err)
				subject := ulid.Make().String()
				ledgerID := credit.LedgerID(ulid.Make().String())

				// Create Ledger

				ledger, err := connector.CreateLedger(context.Background(), credit.Ledger{
					Namespace: namespace,
					ID:        ledgerID,
					Subject:   subject,
					CreatedAt: start.Add(time.Minute * 3),
				})
				assert.NoError(t, err)

				// Register Usage
				streamingConnector.AddResponse(meter1.Slug, 1, ledger.CreatedAt.Add(-time.Second))

				at3m := start.Add(time.Minute * 3)
				at4m := start.Add(time.Minute * 4)

				// Create Feature & Grant
				feature := testutils.CreateFeature(t, connector, featureIn1)
				grant, err := connector.CreateGrant(context.Background(), credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: at3m,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationYear,
						Count:    1,
					},
					CreatedAt: &at3m,
					UpdatedAt: &at3m,
				})

				assert.NoError(t, err)

				// Get Balance
				balance, err := connector.GetBalance(context.Background(), credit.NewNamespacedLedgerID(namespace, ledger.ID), at4m)
				assert.NoError(t, err)

				featureBalance := balance.FeatureBalances[0].Balance
				grantedAmount := grant.Amount
				usedAmount := 0.0

				assert.Equal(t, grantedAmount-usedAmount, featureBalance)
			},
		},
		{
			name:        "Should not calculate usage twice",
			description: `We should not calculate usage twice when multiple grants were issued for the same period.`,
			test: func(t *testing.T, connector credit.Connector, streamingConnector *testutils.MockStreamingConnector, db_client *db.Client) {
				start, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
				assert.NoError(t, err)
				subject := ulid.Make().String()
				ledgerID := credit.LedgerID(ulid.Make().String())

				// Create Ledger
				ledger, err := connector.CreateLedger(context.Background(), credit.Ledger{
					Namespace: namespace,
					ID:        ledgerID,
					Subject:   subject,
					CreatedAt: start,
				})
				assert.NoError(t, err)

				at10s := start.Add(time.Second * 10)
				at20s := start.Add(time.Second * 20)
				at1m := start.Add(time.Minute)
				at1m10s := start.Add(time.Minute).Add(time.Second * 10)

				at3m := start.Add(time.Minute * 3)

				// Create Feature & Grant
				feature := testutils.CreateFeature(t, connector, featureIn1)
				grant1, err := connector.CreateGrant(context.Background(), credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: at10s,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationYear,
						Count:    1,
					},
					CreatedAt: &at10s,
					UpdatedAt: &at10s,
				})

				assert.NoError(t, err)
				grant2, err := connector.CreateGrant(context.Background(), credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: at20s,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationYear,
						Count:    1,
					},
					CreatedAt: &at20s,
					UpdatedAt: &at20s,
				})

				assert.NoError(t, err)
				grant3, err := connector.CreateGrant(context.Background(), credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: at1m,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationYear,
						Count:    1,
					},
					CreatedAt: &at1m,
					UpdatedAt: &at1m,
				})

				assert.NoError(t, err)
				grant4, err := connector.CreateGrant(context.Background(), credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: at1m10s,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationYear,
						Count:    1,
					},
					CreatedAt: &at10s,
					UpdatedAt: &at10s,
				})

				assert.NoError(t, err)

				streamingConnector.SetResponses(meter1.Slug, func(_ []testutils.SimpleEvent) []testutils.SimpleEvent {
					// 1. value=1 for window [start, start+1m]
					// 2. value=1 for window [start+1m, start+2m]
					return []testutils.SimpleEvent{
						{Value: 1, Time: start.Add(time.Minute).Add(-time.Second)},
						{Value: 1, Time: start.Add(time.Minute * 2).Add(-time.Second)},
					}
				})

				// Get Balance
				balance, err := connector.GetBalance(context.Background(), credit.NewNamespacedLedgerID(namespace, ledger.ID), at3m)
				assert.NoError(t, err)

				featureBalance := balance.FeatureBalances[0].Balance
				grantedAmount := grant1.Amount + grant2.Amount + grant3.Amount + grant4.Amount
				usedAmount := 2.0

				assert.Equal(t, grantedAmount-usedAmount, featureBalance)
			},
		},
		{
			name:        "Balance should be consistent accross usage periods",
			description: `Usage numbers read should be consistent accross resets. If you add up the calculated usage for each period it should be equal to the sum of the usage reported.`,
			test: func(t *testing.T, connector credit.Connector, streamingConnector *testutils.MockStreamingConnector, db_client *db.Client) {
				start, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
				assert.NoError(t, err)
				subject := ulid.Make().String()
				ledgerID := credit.LedgerID(ulid.Make().String())

				// Create Ledger
				ledger, err := connector.CreateLedger(context.Background(), credit.Ledger{
					Namespace: namespace,
					ID:        ledgerID,
					Subject:   subject,
					CreatedAt: start,
				})
				assert.NoError(t, err)

				feature := testutils.CreateFeature(t, connector, featureIn1)

				resetTime := start.Add(time.Hour).Add(time.Second * 30)

				grant1Time := resetTime.Add(-time.Minute)
				// Register Usage & Grant for First Usage Period
				grant1, err := connector.CreateGrant(context.Background(), credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: grant1Time,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationYear,
						Count:    1,
					},
					CreatedAt: &grant1Time,
					UpdatedAt: &grant1Time,
				})
				assert.NoError(t, err)

				streamingConnector.AddResponse(meter1.Slug, 1, resetTime.Add(-time.Second))

				// Get Balance at very end of period one
				balance, err := connector.GetBalance(context.Background(), credit.NewNamespacedLedgerID(namespace, ledger.ID), resetTime)
				assert.NoError(t, err)

				periodOneBalance := balance.FeatureBalances[0].Balance
				periodOneGranted := grant1.Amount
				periodOneUsage := periodOneGranted - periodOneBalance

				assert.Equal(t, periodOneUsage, 1.0)

				//
				//
				// Start of Second Usage Period
				//
				//

				_, _, err = connector.Reset(context.Background(), credit.Reset{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					EffectiveAt: resetTime,
				})
				assert.NoError(t, err)

				// Register Usage & Grant for Second Usage Period
				grant2Time := resetTime.Add(time.Second * 5)
				grant3Time := resetTime.Add(time.Minute).Add(time.Second * 5)

				streamingConnector.AddResponse(meter1.Slug, 1, resetTime.Add(time.Second))

				grant2, err := connector.CreateGrant(context.Background(), credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: grant2Time,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationYear,
						Count:    1,
					},
					CreatedAt: &grant2Time,
					UpdatedAt: &grant2Time,
				})
				assert.NoError(t, err)

				streamingConnector.AddResponse(meter1.Slug, 1, resetTime.Add(time.Minute).Add(time.Second))

				grant3, err := connector.CreateGrant(context.Background(), credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: grant3Time,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationYear,
						Count:    1,
					},
					CreatedAt: &grant3Time,
					UpdatedAt: &grant3Time,
				})
				assert.NoError(t, err)

				// Get Balance well into period two
				balance, err = connector.GetBalance(context.Background(), credit.NewNamespacedLedgerID(namespace, ledger.ID), resetTime.Add(time.Hour))
				assert.NoError(t, err)

				periodTwoBalance := balance.FeatureBalances[0].Balance
				periodTwoGranted := grant2.Amount + grant3.Amount
				periodTwoUsage := periodTwoGranted - periodTwoBalance

				assert.Equal(t, periodTwoUsage, 2.0)

				// Total Usage should be equal to the sum of the two periods
				totalUsage := periodOneUsage + periodTwoUsage
				assert.Equal(t, totalUsage, 3.0)
			},
		},
		{
			name:        "Should burn down grant created in same minute as usage was reported",
			description: `If usage is reported within the same minute as the grant is created, the grant should be burned down. Testing with priority.`,
			test: func(t *testing.T, connector credit.Connector, streamingConnector *testutils.MockStreamingConnector, db_client *db.Client) {
				t.Skip(`
                    It's not clear if we want this behavior.

                    We don't have point in time accuracy for the usage data only windowed minutes.
                    We could either take the first valid period for the grant
                        - the window which the grant falls into: in this case usage before the grant would burn it down
                        - the window right after: in this case the grant doesnt really take effect until window end
                    Another option could be to explicitly truncate or round the grant effectiveAt and expiration times to the minute.
                `)

				start, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
				assert.NoError(t, err)
				subject := ulid.Make().String()
				ledgerID := credit.LedgerID(ulid.Make().String())

				// Create Ledger

				ledger, err := connector.CreateLedger(context.Background(), credit.Ledger{
					Namespace: namespace,
					ID:        ledgerID,
					Subject:   subject,
					CreatedAt: start.Add(time.Minute),
				})
				assert.NoError(t, err)

				at3m := start.Add(time.Minute * 3)
				at3m30s := start.Add(time.Minute * 3).Add(time.Second * 30)

				// Create Feature
				feature := testutils.CreateFeature(t, connector, featureIn1)
				// Create two grants, later with higher prio
				grant1, err := connector.CreateGrant(context.Background(), credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: at3m,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationYear,
						Count:    1,
					},
					CreatedAt: &at3m,
					UpdatedAt: &at3m,
				})
				assert.NoError(t, err)

				grant2, err := connector.CreateGrant(context.Background(), credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    0,
					EffectiveAt: at3m30s,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationYear,
						Count:    1,
					},
					CreatedAt: &at3m30s,
					UpdatedAt: &at3m30s,
				})
				assert.NoError(t, err)

				// Register Usage
				streamingConnector.AddResponse(meter1.Slug, 1, at3m30s.Add(time.Second))

				// Get Balance
				balance, err := connector.GetBalance(context.Background(), credit.NewNamespacedLedgerID(namespace, ledger.ID), start.Add(time.Minute*5))
				assert.NoError(t, err)

				var grant2FromBalance *credit.GrantBalance
				for _, gb := range balance.GrantBalances {

					if *gb.Grant.ID == *grant2.ID {
						grant2FromBalance = &gb
					}
				}
				assert.NotNil(t, grant2FromBalance)

				var grant1FromBalance *credit.GrantBalance
				for _, gb := range balance.GrantBalances {
					if *gb.Grant.ID == *grant1.ID {
						grant1FromBalance = &gb
					}
				}
				assert.NotNil(t, grant1FromBalance)

				usedAmount := 1.0

				// grant2 should be burnt down
				assert.Equal(t, grant2.Amount-usedAmount, grant2FromBalance.Balance)
				// grant1 should not be burnt down
				assert.Equal(t, grant1.Amount, grant1FromBalance.Balance)
			},
		},
		{
			name:        "Should burn down only grant created in same minute as usage was reported",
			description: `If usage is reported within the same minute as the grant is created, the grant should be burned down.`,
			test: func(t *testing.T, connector credit.Connector, streamingConnector *testutils.MockStreamingConnector, db_client *db.Client) {
				t.Skip(`
                    It's not clear if we want this behavior.

                    We don't have point in time accuracy for the usage data only windowed minutes.
                    We could either take the first valid period for the grant
                        - the window which the grant falls into: in this case usage before the grant would burn it down
                        - the window right after: in this case the grant doesnt really take effect until window end
                    Another option could be to explicitly truncate or round the grant effectiveAt and expiration times to the minute.

                    This case is separate from the previous as it might make sense functionality wise to account for the initial period
                    between ledger creation and the first grant being granted.
                `)
				start, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
				assert.NoError(t, err)
				subject := ulid.Make().String()
				ledgerID := credit.LedgerID(ulid.Make().String())

				// Create Ledger

				ledger, err := connector.CreateLedger(context.Background(), credit.Ledger{
					Namespace: namespace,
					ID:        ledgerID,
					Subject:   subject,
					CreatedAt: start.Add(time.Minute),
				})
				assert.NoError(t, err)

				at4m30s := start.Add(time.Minute * 4).Add(time.Second * 30)

				// Create Feature & Grant
				feature := testutils.CreateFeature(t, connector, featureIn1)

				grant, err := connector.CreateGrant(context.Background(), credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    0,
					EffectiveAt: at4m30s,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationYear,
						Count:    1,
					},
					CreatedAt: &at4m30s,
					UpdatedAt: &at4m30s,
				})
				assert.NoError(t, err)

				streamingConnector.AddResponse(meter1.Slug, 1, at4m30s.Add(-time.Second))

				// Get Balance
				balance, err := connector.GetBalance(context.Background(), credit.NewNamespacedLedgerID(namespace, ledger.ID), start.Add(time.Minute*5))
				assert.NoError(t, err)

				var grantFromBalance *credit.GrantBalance
				for _, gb := range balance.GrantBalances {

					if *gb.Grant.ID == *grant.ID {
						grantFromBalance = &gb
					}
				}
				assert.NotNil(t, grantFromBalance)

				grantBalance := grantFromBalance.Balance
				usedAmount := 1.0

				// grant should be burnt down
				assert.Equal(t, grant.Amount-usedAmount, grantBalance)
			},
		},
		{
			name:        "Should not burn down higher priority grant with usage before it's effective at date",
			description: `Testing on longer timescale`,
			test: func(t *testing.T, connector credit.Connector, streamingConnector *testutils.MockStreamingConnector, db_client *db.Client) {
				start, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
				assert.NoError(t, err)
				subject := ulid.Make().String()
				ledgerID := credit.LedgerID(ulid.Make().String())

				// Create Ledger

				ledger, err := connector.CreateLedger(context.Background(), credit.Ledger{
					Namespace: namespace,
					ID:        ledgerID,
					Subject:   subject,
					CreatedAt: start.Add(time.Minute),
				})
				assert.NoError(t, err)

				at3m := start.Add(time.Minute * 3)
				at1h := start.Add(time.Hour)

				// Create Feature
				feature := testutils.CreateFeature(t, connector, featureIn1)
				// Create two grants, later with higher prio
				grant1, err := connector.CreateGrant(context.Background(), credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: at3m,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationYear,
						Count:    1,
					},
					CreatedAt: &at3m,
					UpdatedAt: &at3m,
				})
				assert.NoError(t, err)

				grant2, err := connector.CreateGrant(context.Background(), credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    0,
					EffectiveAt: at1h,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationYear,
						Count:    1,
					},
					CreatedAt: &at3m,
					UpdatedAt: &at3m,
				})
				assert.NoError(t, err)

				// Register Usage
				streamingConnector.AddResponse(meter1.Slug, 1, start.Add(time.Minute*3).Add(time.Second))

				// Get Balance
				balance, err := connector.GetBalance(context.Background(), credit.NewNamespacedLedgerID(namespace, ledger.ID), at1h.Add(time.Minute))
				assert.NoError(t, err)

				var grant2FromBalance *credit.GrantBalance
				for _, gb := range balance.GrantBalances {

					if *gb.Grant.ID == *grant2.ID {
						grant2FromBalance = &gb
					}
				}
				assert.NotNil(t, grant2FromBalance)

				var grant1FromBalance *credit.GrantBalance
				for _, gb := range balance.GrantBalances {
					if *gb.Grant.ID == *grant1.ID {
						grant1FromBalance = &gb
					}
				}
				assert.NotNil(t, grant1FromBalance)

				usedAmount := 1.0

				// grant2 should be burnt down
				assert.Equal(t, grant2.Amount-usedAmount, grant2FromBalance.Balance)
				// grant1 should not be burnt down
				assert.Equal(t, grant1.Amount, grant1FromBalance.Balance)
			},
		},
		{
			name:        "Should not burn down higher priority grant with usage reported in minute before it's effective at date",
			description: `Testing in adjacent minutes`,
			test: func(t *testing.T, connector credit.Connector, streamingConnector *testutils.MockStreamingConnector, db_client *db.Client) {
				start, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
				assert.NoError(t, err)
				subject := ulid.Make().String()
				ledgerID := credit.LedgerID(ulid.Make().String())

				// Create Ledger

				ledger, err := connector.CreateLedger(context.Background(), credit.Ledger{
					Namespace: namespace,
					ID:        ledgerID,
					Subject:   subject,
					CreatedAt: start.Add(time.Minute),
				})
				assert.NoError(t, err)

				at3m := start.Add(time.Minute * 3)
				at4m30s := start.Add(time.Minute * 4).Add(time.Second * 30)

				// Create Feature
				feature := testutils.CreateFeature(t, connector, featureIn1)
				// Create two grants, later with higher prio
				grant1, err := connector.CreateGrant(context.Background(), credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: at3m,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationYear,
						Count:    1,
					},
					CreatedAt: &at3m,
					UpdatedAt: &at3m,
				})
				assert.NoError(t, err)

				grant2, err := connector.CreateGrant(context.Background(), credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   feature.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    0,
					EffectiveAt: at4m30s,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationYear,
						Count:    1,
					},
					CreatedAt: &at4m30s,
					UpdatedAt: &at4m30s,
				})
				assert.NoError(t, err)

				// Register Usage
				streamingConnector.AddResponse(meter1.Slug, 1, at4m30s.Add(-time.Second))

				// Get Balance
				balance, err := connector.GetBalance(context.Background(), credit.NewNamespacedLedgerID(namespace, ledger.ID), start.Add(time.Minute*10))
				assert.NoError(t, err)

				var grant2FromBalance *credit.GrantBalance
				for _, gb := range balance.GrantBalances {

					if *gb.Grant.ID == *grant2.ID {
						grant2FromBalance = &gb
					}
				}
				assert.NotNil(t, grant2FromBalance)

				var grant1FromBalance *credit.GrantBalance
				for _, gb := range balance.GrantBalances {
					if *gb.Grant.ID == *grant1.ID {
						grant1FromBalance = &gb
					}
				}
				assert.NotNil(t, grant1FromBalance)

				usedAmount := 1.0

				// grant1 should be burnt down
				assert.Equal(t, grant1.Amount-usedAmount, grant1FromBalance.Balance)
				// grant2 should not be burnt down
				assert.Equal(t, grant2.Amount, grant2FromBalance.Balance)
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Log(tc.description)
			driver := om_tetsutils.InitPostgresDB(t)
			databaseClient := db.NewClient(db.Driver(driver))
			defer databaseClient.Close()

			old, err := time.Parse(time.RFC3339, "2020-01-01T00:00:00Z")
			assert.NoError(t, err)

			// Note: lock manager cannot be shared between tests as these parallel tests write the same ledger
			streamingConnector := testutils.NewMockStreamingConnector(t, testutils.MockStreamingConnectorParams{DefaultHighwatermark: old})
			connector := NewPostgresConnector(slog.Default(), databaseClient, streamingConnector, meterRepository)

			tc.test(t, connector, streamingConnector, databaseClient)
		})
	}
}
