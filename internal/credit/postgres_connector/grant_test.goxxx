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
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestPostgresConnectorGrants(t *testing.T) {
	windowSize := time.Minute
	namespace := "default"

	meters := []models.Meter{
		{
			Namespace: namespace,
			ID:        "meter-1",
			Slug:      "meter-1",
		},
	}
	meterRepository := meter.NewInMemoryRepository(meters)
	features := []credit.Feature{{
		Namespace: namespace,
		MeterSlug: "meter-1",
		Name:      "feature-1",
	}}

	effectiveTime := time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC)

	tt := []struct {
		name        string
		description string
		test        func(t *testing.T, connector credit.Connector, db_client *db.Client, ledger credit.Ledger)
	}{
		{
			name:        "Should truncate grant effectiveAt to windowsize",
			description: "EffectiveAt should be the end date of the window the input falls into",
			test: func(t *testing.T, connector credit.Connector, db_client *db.Client, ledger credit.Ledger) {
				ctx := context.Background()
				p := testutils.CreateFeature(t, connector, features[0])

				effectiveTime := effectiveTime.Truncate(windowSize)
				inpEffectiveTime := effectiveTime.Add(windowSize / 2)

				grant := credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   p.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: inpEffectiveTime,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationDay,
						Count:    1,
					},
				}
				_, err := connector.CreateGrant(ctx, grant)
				assert.NoError(t, err)

				grants, err := connector.ListGrants(ctx, credit.ListGrantsParams{
					Namespace:         namespace,
					LedgerIDs:         []credit.LedgerID{ledger.ID},
					From:              convert.ToPointer(effectiveTime.Add(-windowSize * 2)),
					To:                convert.ToPointer(effectiveTime.Add(windowSize * 2)),
					FromHighWatermark: false,
					IncludeVoid:       true,
				})
				assert.NoError(t, err)

				assert.Equal(t, 1, len(grants))
				assert.Equal(t, effectiveTime.Add(windowSize), grants[0].EffectiveAt)
			},
		},
		{
			name:        "CreateGrant",
			description: "Create a grant in the database",
			test: func(t *testing.T, connector credit.Connector, db_client *db.Client, ledger credit.Ledger) {
				ctx := context.Background()
				p := testutils.CreateFeature(t, connector, features[0])
				effectiveTime := effectiveTime.Truncate(windowSize)
				grant := credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   p.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: effectiveTime,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationDay,
						Count:    1,
					},
				}
				g, err := connector.CreateGrant(ctx, grant)
				assert.NoError(t, err)
				// assert count
				assert.Equal(t, 1, db_client.CreditEntry.Query().CountX(ctx))
				// assert fields
				assert.NotNil(t, g.ID)
				// remove additional fields
				g.ID = nil
				assert.NotEmpty(t, *g.CreatedAt)
				assert.NotEmpty(t, *g.UpdatedAt)

				// Calculate ExpirationAt
				grant.ExpiresAt = grant.Expiration.GetExpiration(grant.EffectiveAt)

				assert.Equal(t, testutils.RemoveTimestampsFromGrant(g), grant)
			},
		},
		{
			name:        "VoidGrant",
			description: "Void a grant in the database and get the latest grant for an ID",
			test: func(t *testing.T, connector credit.Connector, db_client *db.Client, ledger credit.Ledger) {
				ctx := context.Background()
				p := testutils.CreateFeature(t, connector, features[0])
				grant := credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger.ID,
					FeatureID:   p.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: effectiveTime,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationDay,
						Count:    1,
					},
				}
				g, err := connector.CreateGrant(ctx, grant)
				assert.NoError(t, err)
				// should return the grant
				g2, err := connector.GetGrant(ctx, credit.NewNamespacedGrantID(namespace, *g.ID))
				assert.NoError(t, err)
				testutils.AssertGrantsEqual(t, g, g2)

				// So that in postgres the created_at and updated_at are different
				time.Sleep(1 * time.Millisecond)
				v, err := connector.VoidGrant(ctx, g)
				assert.NoError(t, err)
				// should return the void grant
				g3, err := connector.GetGrant(ctx, credit.NewNamespacedGrantID(namespace, *g.ID))
				assert.NoError(t, err)
				testutils.AssertGrantsEqual(t, v, g3)
				// assert count
				assert.Equal(t, 2, db_client.CreditEntry.Query().CountX(ctx))
				// assert fields
				assert.NotNil(t, v.ID)
				assert.Equal(t, v.ParentID, g.ID)
				assert.True(t, v.Void)
				// remove additional fields
				v.ID = nil
				v.ParentID = nil
				v.Void = false
				assert.NotEmpty(t, *v.CreatedAt)
				assert.NotEmpty(t, *v.UpdatedAt)

				testutils.AssertGrantsEqual(t, v, grant)
			},
		},
		{
			name:        "VoidGrantNotFound",
			description: "Void a grant that does not exist",
			test: func(t *testing.T, connector credit.Connector, db_client *db.Client, ledger credit.Ledger) {
				ctx := context.Background()
				p := testutils.CreateFeature(t, connector, features[0])
				id := credit.GrantID(ulid.MustNew(ulid.Now(), nil).String())
				grant := credit.Grant{
					Namespace:   namespace,
					ID:          &id,
					LedgerID:    ledger.ID,
					FeatureID:   p.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: effectiveTime,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationDay,
						Count:    1,
					},
				}
				_, err := connector.VoidGrant(ctx, grant)
				assert.Error(t, err)
				assert.Equal(t, &credit.GrantNotFoundError{GrantID: id}, err)
			},
		},
		{
			name:        "ListGrants",
			description: "List grants for ledgers",
			test: func(t *testing.T, connector credit.Connector, db_client *db.Client, ledger1 credit.Ledger) {
				ledger2, err := connector.CreateLedger(context.Background(), credit.Ledger{
					Namespace: namespace,
					Subject:   ulid.Make().String(),
				})

				assert.NoError(t, err)

				ctx := context.Background()
				p := testutils.CreateFeature(t, connector, features[0])
				grant_s1_1 := credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger1.ID,
					FeatureID:   p.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: effectiveTime,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationDay,
						Count:    1,
					},
				}
				grant_s1_2 := credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger1.ID,
					FeatureID:   p.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      200,
					Priority:    2,
					EffectiveAt: effectiveTime,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationDay,
						Count:    1,
					},
				}
				grant_s2_1 := credit.Grant{
					Namespace:   namespace,
					LedgerID:    ledger2.ID,
					FeatureID:   p.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      300,
					Priority:    1,
					EffectiveAt: effectiveTime,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationDay,
						Count:    1,
					},
				}
				grant_s1_1, err = connector.CreateGrant(ctx, grant_s1_1)
				assert.NoError(t, err)
				grant_s1_2, err = connector.CreateGrant(ctx, grant_s1_2)
				assert.NoError(t, err)
				grant_s2_1, err = connector.CreateGrant(ctx, grant_s2_1)
				assert.NoError(t, err)
				void_grant_s1_1, err := connector.VoidGrant(ctx, grant_s1_1)
				assert.NoError(t, err)
				// assert count
				assert.Equal(t, 4, db_client.CreditEntry.Query().CountX(ctx))
				// all ledgers' non-void grants
				gs, err := connector.ListGrants(ctx, credit.ListGrantsParams{
					Namespace: namespace,
				})
				assert.NoError(t, err)
				assert.ElementsMatch(t,
					testutils.RemoveTimestampsFromGrants([]credit.Grant{grant_s1_2, grant_s2_1}),
					testutils.RemoveTimestampsFromGrants(gs),
				)
				// ledger-1's non-void grants
				gs, err = connector.ListGrants(ctx, credit.ListGrantsParams{
					Namespace: namespace,
					LedgerIDs: []credit.LedgerID{ledger1.ID},
				})
				assert.NoError(t, err)
				assert.ElementsMatch(t,
					testutils.RemoveTimestampsFromGrants([]credit.Grant{grant_s1_2}),
					testutils.RemoveTimestampsFromGrants(gs),
				)
				// all ledger' grants, including void grants
				gs, err = connector.ListGrants(ctx, credit.ListGrantsParams{
					Namespace:   namespace,
					IncludeVoid: true,
				})
				assert.NoError(t, err)
				assert.ElementsMatch(t,
					testutils.RemoveTimestampsFromGrants([]credit.Grant{grant_s1_2, grant_s2_1, void_grant_s1_1}),
					testutils.RemoveTimestampsFromGrants(gs),
				)
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			t.Log(tc.description)
			driver := om_testutils.InitPostgresDB(t)
			databaseClient := db.NewClient(db.Driver(driver))
			defer databaseClient.Close()
			// Note: lock manager cannot be shared between tests as these parallel tests write the same ledger
			old, err := time.Parse(time.RFC3339, "2020-01-01T00:00:00Z")
			assert.NoError(t, err)
			streamingConnector := testutils.NewMockStreamingConnector(t, testutils.MockStreamingConnectorParams{DefaultHighwatermark: old})
			for _, meter := range meters {
				streamingConnector.AddRow(meter.Slug, models.MeterQueryRow{
					Value: 0,
				})
			}

			connector := NewPostgresConnector(slog.Default(), databaseClient, streamingConnector, meterRepository, PostgresConnectorConfig{
				WindowSize: windowSize,
			})

			// let's provision a ledger
			ledger, err := connector.CreateLedger(context.Background(), credit.Ledger{
				Namespace: namespace,
				Subject:   ulid.Make().String(),
			})

			assert.NoError(t, err)

			tc.test(t, connector, databaseClient, ledger)
		})
	}
}
