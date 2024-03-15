// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package adapter_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"

	db_feature "github.com/openmeterio/openmeter/openmeter/ent/db/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func TestCreateFeature(t *testing.T) {
	namespace := "default"
	meter := models.Meter{
		Namespace: namespace,
		ID:        "meter-1",
		Slug:      "meter-1",
		GroupBy:   map[string]string{"key": "$.path"},
	}

	testFeature := productcatalog.CreateFeatureInputs{
		Namespace: namespace,
		Name:      "feature-1",
		Key:       "feature-1",
		MeterSlug: &meter.Slug,
		MeterGroupByFilters: map[string]string{
			"key": "value",
		},
	}

	tt := []struct {
		name string
		run  func(t *testing.T, connector productcatalog.FeatureRepo)
	}{
		{
			name: "Should create a feature and return the created feature with defaults",
			run: func(t *testing.T, connector productcatalog.FeatureRepo) {
				ctx := context.Background()
				featureIn := testFeature

				createFeatureOut, err := connector.CreateFeature(ctx, featureIn)
				assert.NoError(t, err)

				feature, err := connector.GetByIdOrKey(ctx, namespace, createFeatureOut.ID, false)
				assert.NoError(t, err)

				// truncate times due to CI errors
				createFeatureOut.CreatedAt = createFeatureOut.CreatedAt.Truncate(time.Millisecond)
				feature.CreatedAt = feature.CreatedAt.Truncate(time.Millisecond)
				createFeatureOut.UpdatedAt = createFeatureOut.UpdatedAt.Truncate(time.Millisecond)
				feature.UpdatedAt = feature.UpdatedAt.Truncate(time.Millisecond)

				assert.Equal(t, createFeatureOut, *feature)
				assert.NotEmpty(t, feature.ID)
				assert.NotEmpty(t, feature.CreatedAt)
				assert.NotEmpty(t, feature.UpdatedAt)
				assert.Nil(t, feature.ArchivedAt)
				assert.NotEmpty(t, createFeatureOut.ID)
			},
		},
		{
			name: "Should archive a feature that exists and error on a feature that doesnt",
			run: func(t *testing.T, connector productcatalog.FeatureRepo) {
				ctx := context.Background()
				featureIn := testFeature

				createFeatureOut, err := connector.CreateFeature(ctx, featureIn)
				assert.NoError(t, err)

				// archives the feature
				err = connector.ArchiveFeature(ctx, models.NamespacedID{
					Namespace: featureIn.Namespace,
					ID:        createFeatureOut.ID,
				})
				assert.NoError(t, err)

				// errors on a feature that doesn't exist
				fakeID := ulid.Make().String()
				err = connector.ArchiveFeature(ctx, models.NamespacedID{
					Namespace: featureIn.Namespace,
					ID:        fakeID,
				})
				assert.Error(t, err)
			},
		},
		{
			name: "Should search and order",
			run: func(t *testing.T, connector productcatalog.FeatureRepo) {
				ctx := context.Background()
				featureIn1 := testFeature
				featureIn1.Name = "feature-3"
				featureIn1.Key = "feature-3"
				featureIn2 := testFeature
				featureIn2.Name = "feature-2"
				featureIn2.Key = "feature-2"

				_, err := connector.CreateFeature(ctx, featureIn1)
				assert.NoError(t, err)

				time.Sleep(100 * time.Millisecond)

				_, err = connector.CreateFeature(ctx, featureIn2)
				assert.NoError(t, err)

				features, err := connector.ListFeatures(ctx, productcatalog.ListFeaturesParams{
					Namespace: namespace,
				})
				assert.NoError(t, err)

				assert.Len(t, features.Items, 2)
				assert.Equal(t, "feature-3", features.Items[0].Name)

				features, err = connector.ListFeatures(ctx, productcatalog.ListFeaturesParams{
					Namespace: namespace,
					Page: pagination.Page{
						PageSize:   1,
						PageNumber: 1,
					},
				})
				assert.NoError(t, err)

				assert.Len(t, features.Items, 1)
				assert.Equal(t, "feature-3", features.Items[0].Name)

				features, err = connector.ListFeatures(ctx, productcatalog.ListFeaturesParams{
					Namespace: namespace,
					Page: pagination.Page{
						PageSize:   1,
						PageNumber: 2,
					},
				})
				assert.NoError(t, err)

				assert.Len(t, features.Items, 1)
				assert.Equal(t, "feature-2", features.Items[0].Name)

				err = connector.ArchiveFeature(ctx, models.NamespacedID{
					Namespace: namespace,
					ID:        features.Items[0].ID,
				})
				assert.NoError(t, err)

				features, err = connector.ListFeatures(ctx, productcatalog.ListFeaturesParams{
					Namespace:       namespace,
					IncludeArchived: true,
				})
				assert.NoError(t, err)

				assert.Len(t, features.Items, 2)

				features, err = connector.ListFeatures(ctx, productcatalog.ListFeaturesParams{
					Namespace:       namespace,
					IncludeArchived: false,
				})
				assert.NoError(t, err)

				assert.Len(t, features.Items, 1)
				assert.Equal(t, "feature-3", features.Items[0].Name)
			},
		},
		{
			name: "Should find by name",
			run: func(t *testing.T, connector productcatalog.FeatureRepo) {
				ctx := context.Background()
				featureIn1 := testFeature
				featureIn1.Name = "feature-1"
				featureIn1.Key = "feature-1"
				featureIn2 := testFeature
				featureIn2.Name = "feature-2"
				featureIn2.Key = "feature-2"

				_, err := connector.CreateFeature(ctx, featureIn1)
				assert.NoError(t, err)

				_, err = connector.CreateFeature(ctx, featureIn2)
				assert.NoError(t, err)

				foundFeature, err := connector.GetByIdOrKey(ctx, namespace, "feature-1", false)
				assert.NoError(t, err)

				assert.Equal(t, "feature-1", foundFeature.Name)

				err = connector.ArchiveFeature(ctx, models.NamespacedID{
					Namespace: namespace,
					ID:        foundFeature.ID,
				})
				assert.NoError(t, err)

				_, err = connector.GetByIdOrKey(ctx, namespace, "feature-1", false)
				assert.Error(t, err)

				foundFeature, err = connector.GetByIdOrKey(ctx, namespace, "feature-1", true)
				assert.NoError(t, err)

				assert.Equal(t, "feature-1", foundFeature.Name)
			},
		},
	}

	var m sync.Mutex

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m.Lock()
			defer m.Unlock()

			testdb := testutils.InitPostgresDB(t)
			defer testdb.PGDriver.Close()
			dbClient := testdb.EntDriver.Client()
			defer dbClient.Close()

			if err := migrate.Up(testdb.URL); err != nil {
				t.Fatalf("failed to migrate db: %s", err.Error())
			}

			dbConnector := adapter.NewPostgresFeatureRepo(dbClient, testutils.NewLogger(t))
			tc.run(t, dbConnector)
		})
	}

	t.Run("Should actually use the pg driver and write through that", func(t *testing.T) {
		t.Parallel()
		m.Lock()
		defer m.Unlock()

		testdb := testutils.InitPostgresDB(t)
		defer testdb.PGDriver.Close()
		dbClient := testdb.EntDriver.Client()
		defer dbClient.Close()

		if err := migrate.Up(testdb.URL); err != nil {
			t.Fatalf("failed to migrate db: %s", err.Error())
		}

		dbConnector := adapter.NewPostgresFeatureRepo(dbClient, testutils.NewLogger(t))
		ctx := context.Background()
		featureIn := testFeature

		createFeatureOut, err := dbConnector.CreateFeature(ctx, featureIn)
		assert.NoError(t, err)

		feature, err := dbClient.Feature.Query().Where(db_feature.ID(createFeatureOut.ID)).Only(ctx)
		assert.NoError(t, err)

		assert.Equal(t, featureIn.Name, feature.Name)
	})
}
