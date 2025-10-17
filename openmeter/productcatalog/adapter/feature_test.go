package adapter_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	db_feature "github.com/openmeterio/openmeter/openmeter/ent/db/feature"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestCreateFeature(t *testing.T) {
	namespace := "default"
	meter := meter.Meter{
		ManagedResource: models.ManagedResource{
			ID: "meter-1",
			NamespacedModel: models.NamespacedModel{
				Namespace: namespace,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Name: "Test meter",
		},
		Key:         "meter-1",
		GroupBy:     map[string]string{"key": "$.path"},
		Aggregation: meter.MeterAggregationCount,
		EventType:   "test",
	}

	testFeature := feature.CreateFeatureInputs{
		Namespace: namespace,
		Name:      "feature-1",
		Key:       "feature-1",
		MeterSlug: &meter.Key,
		MeterGroupByFilters: feature.MeterGroupByFilters{
			"key": filter.FilterString{
				Eq: lo.ToPtr("value"),
			},
		},
	}

	tt := []struct {
		name string
		run  func(t *testing.T, connector feature.FeatureRepo)
	}{
		{
			name: "Should create a feature and return the created feature with defaults",
			run: func(t *testing.T, connector feature.FeatureRepo) {
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
				assert.NotEmpty(t, feature.Namespace)
				assert.NotEmpty(t, feature.ID)
				assert.NotEmpty(t, feature.CreatedAt)
				assert.NotEmpty(t, feature.UpdatedAt)
				assert.Nil(t, feature.ArchivedAt)
				assert.NotEmpty(t, createFeatureOut.ID)
			},
		},
		{
			name: "Should archive a feature that exists and error on a feature that doesnt",
			run: func(t *testing.T, connector feature.FeatureRepo) {
				ctx := context.Background()
				featureIn := testFeature

				createFeatureOut, err := connector.CreateFeature(ctx, featureIn)
				assert.NoError(t, err)

				// archives the feature
				err = connector.ArchiveFeature(ctx, feature.ArchiveFeatureInput{
					Namespace: featureIn.Namespace,
					ID:        createFeatureOut.ID,
				})
				assert.NoError(t, err)

				// errors on a feature that doesn't exist
				fakeID := ulid.Make().String()
				err = connector.ArchiveFeature(ctx, feature.ArchiveFeatureInput{
					Namespace: featureIn.Namespace,
					ID:        fakeID,
				})
				assert.Error(t, err)
			},
		},
		{
			name: "Should search and order",
			run: func(t *testing.T, connector feature.FeatureRepo) {
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

				features, err := connector.ListFeatures(ctx, feature.ListFeaturesParams{
					Namespace: namespace,
				})
				assert.NoError(t, err)

				assert.Len(t, features.Items, 2)
				assert.Equal(t, "feature-3", features.Items[0].Name)

				features, err = connector.ListFeatures(ctx, feature.ListFeaturesParams{
					Namespace: namespace,
					Page: pagination.Page{
						PageSize:   1,
						PageNumber: 1,
					},
				})
				assert.NoError(t, err)

				assert.Len(t, features.Items, 1)
				assert.Equal(t, "feature-3", features.Items[0].Name)

				features, err = connector.ListFeatures(ctx, feature.ListFeaturesParams{
					Namespace: namespace,
					Page: pagination.Page{
						PageSize:   1,
						PageNumber: 2,
					},
				})
				assert.NoError(t, err)

				assert.Len(t, features.Items, 1)
				assert.Equal(t, "feature-2", features.Items[0].Name)

				err = connector.ArchiveFeature(ctx, feature.ArchiveFeatureInput{
					Namespace: namespace,
					ID:        features.Items[0].ID,
				})
				assert.NoError(t, err)

				features, err = connector.ListFeatures(ctx, feature.ListFeaturesParams{
					Namespace:       namespace,
					IncludeArchived: true,
				})
				assert.NoError(t, err)

				assert.Len(t, features.Items, 2)

				features, err = connector.ListFeatures(ctx, feature.ListFeaturesParams{
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
			run: func(t *testing.T, connector feature.FeatureRepo) {
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

				err = connector.ArchiveFeature(ctx, feature.ArchiveFeatureInput{
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

			if err := dbClient.Schema.Create(context.Background()); err != nil {
				t.Fatalf("failed to create schema: %v", err)
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

		if err := dbClient.Schema.Create(context.Background()); err != nil {
			t.Fatalf("failed to create schema: %v", err)
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

func TestArchiveFeature(t *testing.T) {
	namespace := "default"
	meter := meter.Meter{
		ManagedResource: models.ManagedResource{
			ID: "meter-1",
			NamespacedModel: models.NamespacedModel{
				Namespace: namespace,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Name: "Test meter",
		},
		Key:         "meter-1",
		GroupBy:     map[string]string{"key": "$.path"},
		Aggregation: meter.MeterAggregationCount,
		EventType:   "test",
	}

	testFeature := feature.CreateFeatureInputs{
		Namespace: namespace,
		Name:      "feature-1",
		Key:       "feature-1",
		MeterSlug: &meter.Key,
		MeterGroupByFilters: feature.MeterGroupByFilters{
			"key": filter.FilterString{
				Eq: lo.ToPtr("value"),
			},
		},
	}

	t.Run("Should allow archiving feature", func(t *testing.T) {
		testdb := testutils.InitPostgresDB(t)
		defer testdb.PGDriver.Close()
		dbClient := testdb.EntDriver.Client()
		defer dbClient.Close()

		if err := dbClient.Schema.Create(context.Background()); err != nil {
			t.Fatalf("failed to create schema: %v", err)
		}

		ctx := context.Background()

		// Let's set up any plan with phases and ratecards
		p, err := dbClient.Plan.Create().
			SetName("default").
			SetKey("default").
			SetVersion(1).
			SetEffectiveFrom(time.Now()).
			SetNamespace(testFeature.Namespace).
			SetBillingCadence("P1M").
			SetProRatingConfig(productcatalog.ProRatingConfig{
				Enabled: true,
				Mode:    productcatalog.ProRatingModeProratePrices,
			}).
			Save(ctx)
		assert.NoError(t, err)

		pp, err := dbClient.PlanPhase.Create().
			SetName("default").
			SetKey("default").
			SetNamespace(testFeature.Namespace).
			SetPlanID(p.ID).
			SetIndex(0).
			Save(ctx)
		assert.NoError(t, err)

		_, err = dbClient.PlanRateCard.Create().
			SetKey("default").
			SetName("default").
			SetType(productcatalog.FlatFeeRateCardType).
			SetNamespace(testFeature.Namespace).
			SetPhaseID(pp.ID).
			Save(ctx)
		assert.NoError(t, err)

		connector := adapter.NewPostgresFeatureRepo(dbClient, testutils.NewLogger(t))

		featureIn := testFeature

		createFeatureOut, err := connector.CreateFeature(ctx, featureIn)
		assert.NoError(t, err)

		err = connector.ArchiveFeature(ctx, feature.ArchiveFeatureInput{
			Namespace: createFeatureOut.Namespace,
			ID:        createFeatureOut.ID,
		})
		assert.NoError(t, err)
	})
}

func TestFetchingArchivedFeature(t *testing.T) {
	namespace := "default"
	meter := meter.Meter{
		ManagedResource: models.ManagedResource{
			ID: "meter-1",
			NamespacedModel: models.NamespacedModel{
				Namespace: namespace,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Name: "Test meter",
		},
		Key:         "meter-1",
		GroupBy:     map[string]string{"key": "$.path"},
		Aggregation: meter.MeterAggregationCount,
		EventType:   "test",
	}

	testFeature := feature.CreateFeatureInputs{
		Namespace: namespace,
		Name:      "feature-1",
		Key:       "feature-1",
		MeterSlug: &meter.Key,
		MeterGroupByFilters: feature.MeterGroupByFilters{
			"key": filter.FilterString{
				Eq: lo.ToPtr("value"),
			},
		},
	}

	t.Run("Should allow archiving feature", func(t *testing.T) {
		testdb := testutils.InitPostgresDB(t)
		defer testdb.PGDriver.Close()
		dbClient := testdb.EntDriver.Client()
		defer dbClient.Close()

		if err := dbClient.Schema.Create(context.Background()); err != nil {
			t.Fatalf("failed to create schema: %v", err)
		}

		ctx := context.Background()
		connector := adapter.NewPostgresFeatureRepo(dbClient, testutils.NewLogger(t))

		featureIn := testFeature

		createFeatureOutArchived, err := connector.CreateFeature(ctx, featureIn)
		assert.NoError(t, err)

		err = connector.ArchiveFeature(ctx, feature.ArchiveFeatureInput{
			Namespace: createFeatureOutArchived.Namespace,
			ID:        createFeatureOutArchived.ID,
		})
		assert.NoError(t, err)

		createFeatureOut, err := connector.CreateFeature(ctx, featureIn)
		assert.NoError(t, err)

		assert.NotEqual(t, createFeatureOutArchived.ID, createFeatureOut.ID)

		featchedFeature, err := connector.GetByIdOrKey(ctx, namespace, createFeatureOut.Key, true)
		assert.NoError(t, err)

		assert.Equal(t, createFeatureOut.ID, featchedFeature.ID)
	})
}
