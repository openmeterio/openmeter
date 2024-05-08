package postgres_connector

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestPostgresConnectorGrants(t *testing.T) {
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
			name:        "CreateGrant",
			description: "Create a grant in the database",
			test: func(t *testing.T, connector credit.Connector, db_client *db.Client, ledger credit.Ledger) {
				ctx := context.Background()
				p := createFeature(t, connector, namespace, features[0])
				grant := credit.Grant{
					LedgerID:    ledger.ID,
					FeatureID:   p.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      decimal.NewFromFloat(100),
					Priority:    1,
					EffectiveAt: effectiveTime,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationDay,
						Count:    1,
					},
				}
				g, err := connector.CreateGrant(ctx, namespace, grant)
				assert.NoError(t, err)
				// assert count
				assert.Equal(t, 1, db_client.CreditEntry.Query().CountX(ctx))
				// assert fields
				assert.NotNil(t, g.ID)
				// remove additional fields
				g.ID = nil
				assert.Equal(t, g, grant)
			},
		},
		{
			name:        "VoidGrant",
			description: "Void a grant in the database and get the latest grant for an ID",
			test: func(t *testing.T, connector credit.Connector, db_client *db.Client, ledger credit.Ledger) {
				ctx := context.Background()
				p := createFeature(t, connector, namespace, features[0])
				grant := credit.Grant{
					LedgerID:    ledger.ID,
					FeatureID:   p.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      decimal.NewFromFloat(100),
					Priority:    1,
					EffectiveAt: effectiveTime,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationDay,
						Count:    1,
					},
				}
				g, err := connector.CreateGrant(ctx, namespace, grant)
				assert.NoError(t, err)
				// should return the grant
				g2, err := connector.GetGrant(ctx, namespace, *g.ID)
				assert.NoError(t, err)
				assert.Equal(t, g, g2)
				v, err := connector.VoidGrant(ctx, namespace, g)
				assert.NoError(t, err)
				// should return the void grant
				g3, err := connector.GetGrant(ctx, namespace, *g.ID)
				assert.NoError(t, err)
				assert.Equal(t, v, g3)
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
				assert.Equal(t, v, grant)
			},
		},
		{
			name:        "VoidGrantNotFound",
			description: "Void a grant that does not exist",
			test: func(t *testing.T, connector credit.Connector, db_client *db.Client, ledger credit.Ledger) {
				ctx := context.Background()
				p := createFeature(t, connector, namespace, features[0])
				id := ulid.MustNew(ulid.Now(), nil)
				grant := credit.Grant{
					ID:          &id,
					LedgerID:    ledger.ID,
					FeatureID:   p.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      decimal.NewFromFloat(100),
					Priority:    1,
					EffectiveAt: effectiveTime,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationDay,
						Count:    1,
					},
				}
				_, err := connector.VoidGrant(ctx, namespace, grant)
				assert.Error(t, err)
				assert.Equal(t, &credit.GrantNotFoundError{GrantID: id}, err)
			},
		},
		{
			name:        "ListGrants",
			description: "List grants for ledgers",
			test: func(t *testing.T, connector credit.Connector, db_client *db.Client, ledger1 credit.Ledger) {
				ledger2, err := connector.CreateLedger(context.Background(), namespace, credit.Ledger{
					Subject: ulid.Make().String(),
				})

				assert.NoError(t, err)

				ctx := context.Background()
				p := createFeature(t, connector, namespace, features[0])
				grant_s1_1 := credit.Grant{
					LedgerID:    ledger1.ID,
					FeatureID:   p.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      decimal.NewFromFloat(100),
					Priority:    1,
					EffectiveAt: effectiveTime,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationDay,
						Count:    1,
					},
				}
				grant_s1_2 := credit.Grant{
					LedgerID:    ledger1.ID,
					FeatureID:   p.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      decimal.NewFromFloat(200),
					Priority:    2,
					EffectiveAt: effectiveTime,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationDay,
						Count:    1,
					},
				}
				grant_s2_1 := credit.Grant{
					LedgerID:    ledger2.ID,
					FeatureID:   p.ID,
					Type:        credit.GrantTypeUsage,
					Amount:      decimal.NewFromFloat(300),
					Priority:    1,
					EffectiveAt: effectiveTime,
					Expiration: credit.ExpirationPeriod{
						Duration: credit.ExpirationPeriodDurationDay,
						Count:    1,
					},
				}
				grant_s1_1, err = connector.CreateGrant(ctx, namespace, grant_s1_1)
				assert.NoError(t, err)
				grant_s1_2, err = connector.CreateGrant(ctx, namespace, grant_s1_2)
				assert.NoError(t, err)
				grant_s2_1, err = connector.CreateGrant(ctx, namespace, grant_s2_1)
				assert.NoError(t, err)
				void_grant_s1_1, err := connector.VoidGrant(ctx, namespace, grant_s1_1)
				assert.NoError(t, err)
				// assert count
				assert.Equal(t, 4, db_client.CreditEntry.Query().CountX(ctx))
				// all ledgers' non-void grants
				gs, err := connector.ListGrants(ctx, namespace, credit.ListGrantsParams{})
				assert.NoError(t, err)
				assert.ElementsMatch(t, []credit.Grant{grant_s1_2, grant_s2_1}, gs)
				// ledger-1's non-void grants
				gs, err = connector.ListGrants(ctx, namespace, credit.ListGrantsParams{
					LedgerIDs: []ulid.ULID{ledger1.ID},
				})
				assert.NoError(t, err)
				assert.ElementsMatch(t, []credit.Grant{grant_s1_2}, gs)
				// all ledger' grants, including void grants
				gs, err = connector.ListGrants(ctx, namespace, credit.ListGrantsParams{IncludeVoid: true})
				assert.NoError(t, err)
				assert.ElementsMatch(t, []credit.Grant{grant_s1_2, grant_s2_1, void_grant_s1_1}, gs)
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
			connector := NewPostgresConnector(slog.Default(), databaseClient, nil, meterRepository)

			// let's provision a ledger
			ledger, err := connector.CreateLedger(context.Background(), namespace, credit.Ledger{
				Subject: ulid.Make().String(),
			})

			assert.NoError(t, err)

			tc.test(t, connector, databaseClient, ledger)
		})
	}
}

func createFeature(t *testing.T, connector credit.Connector, namespace string, feature credit.Feature) credit.Feature {
	ctx := context.Background()
	p, err := connector.CreateFeature(ctx, namespace, feature)
	if err != nil {
		t.Error(err)
	}
	return p
}
