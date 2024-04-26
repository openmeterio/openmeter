package postgres_connector

import (
	"context"
	"log/slog"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"

	credit_model "github.com/openmeterio/openmeter/internal/credit"
	inmemory_lock "github.com/openmeterio/openmeter/internal/credit/inmemory_lock"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	meter_internal "github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestFeature(t *testing.T) {
	namespace := "default"
	meter := models.Meter{
		Namespace: namespace,
		ID:        "meter-1",
		Slug:      "meter-1",
		GroupBy:   map[string]string{"key": "$.path"},
	}
	meters := []models.Meter{meter}
	meterRepository := meter_internal.NewInMemoryRepository(meters)
	lockManager := inmemory_lock.NewLockManager(time.Second * 10)

	testFeature := credit_model.Feature{
		Namespace: namespace,
		Name:      "feature-1",
		MeterSlug: meter.Slug,
		MeterGroupByFilters: &map[string]string{
			"key": "value",
		},
	}

	tt := []struct {
		name        string
		description string
		test        func(t *testing.T, connector credit_model.Connector, db_client *db.Client)
	}{
		{
			name:        "CreateFeature",
			description: "Create a feature in the database",
			test: func(t *testing.T, connector credit_model.Connector, db_client *db.Client) {
				ctx := context.Background()
				featureIn := testFeature
				featureOut, err := connector.CreateFeature(ctx, namespace, featureIn)
				assert.NoError(t, err)
				// assert count
				assert.Equal(t, 1, db_client.Feature.Query().CountX(ctx))
				// assert fields
				assert.NotNil(t, featureOut.ID)
				featureIn.ID = featureOut.ID
				expected := featureIn
				archived := false
				expected.Archived = &archived
				assert.Equal(t, expected, featureOut)
			},
		},
		{
			name:        "GetFeature",
			description: "Get a feature from the database",
			test: func(t *testing.T, connector credit_model.Connector, db_client *db.Client) {
				ctx := context.Background()
				featureIn, err := connector.CreateFeature(ctx, namespace, testFeature)
				assert.NoError(t, err)

				featureOut, err := connector.GetFeature(ctx, namespace, *featureIn.ID)
				assert.NoError(t, err)

				expected := featureIn
				archived := false
				expected.Archived = &archived
				assert.Equal(t, expected, featureOut)
			},
		},
		{
			name:        "DeleteFeature",
			description: "Delete a feature in the database",
			test: func(t *testing.T, connector credit_model.Connector, db_client *db.Client) {
				ctx := context.Background()
				p, err := connector.CreateFeature(ctx, namespace, testFeature)
				assert.NoError(t, err)

				err = connector.DeleteFeature(ctx, namespace, *p.ID)
				assert.NoError(t, err)

				// assert
				p, err = connector.GetFeature(ctx, namespace, *p.ID)
				assert.NoError(t, err)
				assert.True(t, *p.Archived)
			},
		},
		{
			name:        "ListFeatures",
			description: "List features in the database",
			test: func(t *testing.T, connector credit_model.Connector, db_client *db.Client) {
				ctx := context.Background()
				feature, err := connector.CreateFeature(ctx, namespace, testFeature)
				assert.NoError(t, err)

				features, err := connector.ListFeatures(ctx, namespace, credit_model.ListFeaturesParams{})
				assert.NoError(t, err)
				assert.Len(t, features, 1)

				expected := feature
				archived := false
				expected.Archived = &archived
				assert.Equal(t, []credit_model.Feature{expected}, features)
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
			connector := NewPostgresConnector(slog.Default(), databaseClient, nil, meterRepository, lockManager)
			tc.test(t, connector, databaseClient)
		})
	}
}
