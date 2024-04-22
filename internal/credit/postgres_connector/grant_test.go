package postgres_connector

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"

	credit "github.com/openmeterio/openmeter/internal/credit"
	inmemory_lock "github.com/openmeterio/openmeter/internal/credit/inmemory_lock"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	"github.com/openmeterio/openmeter/internal/meter"
	credit_model "github.com/openmeterio/openmeter/pkg/credit"
	"github.com/openmeterio/openmeter/pkg/models"
	product_model "github.com/openmeterio/openmeter/pkg/product"
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
	products := []product_model.Product{{
		Namespace: namespace,
		MeterSlug: "meter-1",
		Name:      "product-1",
	}}

	effectiveTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local)

	tt := []struct {
		name        string
		description string
		test        func(t *testing.T, connector credit.Connector, db_client *db.Client)
	}{
		{
			name:        "CreateGrant",
			description: "Create a grant in the database",
			test: func(t *testing.T, connector credit.Connector, db_client *db.Client) {
				ctx := context.Background()
				p := createProduct(t, connector, namespace, products[0])
				grant := credit_model.Grant{
					Subject:     "subject-1",
					ProductID:   p.ID,
					Type:        credit_model.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: effectiveTime,
					Expiration: credit_model.ExpirationPeriod{
						Duration: credit_model.ExpirationPeriodDurationDay,
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
			test: func(t *testing.T, connector credit.Connector, db_client *db.Client) {
				ctx := context.Background()
				p := createProduct(t, connector, namespace, products[0])
				grant := credit_model.Grant{
					Subject:     "subject-1",
					ProductID:   p.ID,
					Type:        credit_model.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: effectiveTime,
					Expiration: credit_model.ExpirationPeriod{
						Duration: credit_model.ExpirationPeriodDurationDay,
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
			test: func(t *testing.T, connector credit.Connector, db_client *db.Client) {
				ctx := context.Background()
				p := createProduct(t, connector, namespace, products[0])
				id := ulid.MustNew(ulid.Now(), nil).String()
				grant := credit_model.Grant{
					ID:          &id,
					Subject:     "subject-1",
					ProductID:   p.ID,
					Type:        credit_model.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: effectiveTime,
					Expiration: credit_model.ExpirationPeriod{
						Duration: credit_model.ExpirationPeriodDurationDay,
						Count:    1,
					},
				}
				_, err := connector.VoidGrant(ctx, namespace, grant)
				assert.Error(t, err)
				assert.Equal(t, &credit_model.GrantNotFoundError{GrantID: id}, err)
			},
		},
		{
			name:        "ListGrants",
			description: "List grants for subjects",
			test: func(t *testing.T, connector credit.Connector, db_client *db.Client) {
				ctx := context.Background()
				p := createProduct(t, connector, namespace, products[0])
				grant_s1_1 := credit_model.Grant{
					Subject:     "subject-1",
					ProductID:   p.ID,
					Type:        credit_model.GrantTypeUsage,
					Amount:      100,
					Priority:    1,
					EffectiveAt: effectiveTime,
					Expiration: credit_model.ExpirationPeriod{
						Duration: credit_model.ExpirationPeriodDurationDay,
						Count:    1,
					},
				}
				grant_s1_2 := credit_model.Grant{
					Subject:     "subject-1",
					ProductID:   p.ID,
					Type:        credit_model.GrantTypeUsage,
					Amount:      200,
					Priority:    2,
					EffectiveAt: effectiveTime,
					Expiration: credit_model.ExpirationPeriod{
						Duration: credit_model.ExpirationPeriodDurationDay,
						Count:    1,
					},
				}
				grant_s2_1 := credit_model.Grant{
					Subject:     "subject-2",
					ProductID:   p.ID,
					Type:        credit_model.GrantTypeUsage,
					Amount:      300,
					Priority:    1,
					EffectiveAt: effectiveTime,
					Expiration: credit_model.ExpirationPeriod{
						Duration: credit_model.ExpirationPeriodDurationDay,
						Count:    1,
					},
				}
				grant_s1_1, err := connector.CreateGrant(ctx, namespace, grant_s1_1)
				assert.NoError(t, err)
				grant_s1_2, err = connector.CreateGrant(ctx, namespace, grant_s1_2)
				assert.NoError(t, err)
				grant_s2_1, err = connector.CreateGrant(ctx, namespace, grant_s2_1)
				assert.NoError(t, err)
				void_grant_s1_1, err := connector.VoidGrant(ctx, namespace, grant_s1_1)
				assert.NoError(t, err)
				// assert count
				assert.Equal(t, 4, db_client.CreditEntry.Query().CountX(ctx))
				// all subjects' non-void grants
				gs, err := connector.ListGrants(ctx, namespace, credit.ListGrantsParams{})
				assert.NoError(t, err)
				assert.ElementsMatch(t, []credit_model.Grant{grant_s1_2, grant_s2_1}, gs)
				// subject-1's non-void grants
				gs, err = connector.ListGrants(ctx, namespace, credit.ListGrantsParams{Subjects: []string{"subject-1"}})
				assert.NoError(t, err)
				assert.ElementsMatch(t, []credit_model.Grant{grant_s1_2}, gs)
				// all subjects' grants, including void grants
				gs, err = connector.ListGrants(ctx, namespace, credit.ListGrantsParams{IncludeVoid: true})
				assert.NoError(t, err)
				assert.ElementsMatch(t, []credit_model.Grant{grant_s1_2, grant_s2_1, void_grant_s1_1}, gs)
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
			connector := NewPostgresConnector(slog.Default(), databaseClient, nil, meterRepository, lockManager)
			tc.test(t, connector, databaseClient)
		})
	}
}

func createProduct(t *testing.T, connector credit.Connector, namespace string, product product_model.Product) product_model.Product {
	ctx := context.Background()
	p, err := connector.CreateProduct(ctx, namespace, product)
	if err != nil {
		t.Error(err)
	}
	return p
}
