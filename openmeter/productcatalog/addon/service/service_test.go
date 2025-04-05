package service

import (
	"context"
	"crypto/rand"
	"sync"
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/meter"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/mockadapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	productcatalogadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestAddonService(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup test environment
	env := newTestEnv(t)
	defer env.Close(t)

	// Run database migrations
	env.DBSchemaMigrate(t)

	// Get new namespace ID
	namespace := NewTestNamespace(t)

	// Setup meter repository
	err := env.Meter.ReplaceMeters(ctx, NewTestMeters(t, namespace))
	require.NoError(t, err, "replacing Meters must not fail")

	result, err := env.Meter.ListMeters(ctx, meter.ListMetersParams{
		Namespace: namespace,
		Page:      pagination.NewPage(1, 100),
	})
	require.NoErrorf(t, err, "listing Meters must not fail")

	meters := result.Items
	require.NotEmptyf(t, meters, "list of Meters must not be empty")

	// Set Feature for each Meter
	features := make(map[string]feature.Feature, len(meters))
	for _, m := range meters {
		input := feature.CreateFeatureInputs{
			Name:                m.Key,
			Key:                 m.Key,
			Namespace:           namespace,
			MeterSlug:           lo.ToPtr(m.Key),
			MeterGroupByFilters: m.GroupBy,
			Metadata:            map[string]string{},
		}

		feat, err := env.Feature.CreateFeature(ctx, input)
		require.NoErrorf(t, err, "creating Feature must not fail")
		require.NotNil(t, feat, "Feature must not be empty")

		features[feat.Key] = feat
	}

	t.Run("Addon", func(t *testing.T) {
		t.Run("Create", func(t *testing.T) {
			addonInput := NewAddon(t, namespace)

			draftAddon, err := env.Addon.CreateAddon(ctx, addonInput)
			require.NoErrorf(t, err, "creating add-on must not fail")
			require.NotNil(t, draftAddon, "add-on must not be empty")

			addon.AssertAddonCreateInputEqual(t, addonInput, *draftAddon)
			assert.Equalf(t, productcatalog.AddonStatusDraft, draftAddon.Status(),
				"add-on status mismatch: expected=%s, actual=%s", productcatalog.AddonStatusDraft, draftAddon.Status())

			t.Run("Get", func(t *testing.T) {
				getAddon, err := env.Addon.GetAddon(ctx, addon.GetAddonInput{
					NamespacedID: models.NamespacedID{
						Namespace: addonInput.Namespace,
					},
					Key:           addonInput.Key,
					IncludeLatest: true,
				})
				require.NoErrorf(t, err, "getting draft add-on must not fail")
				require.NotNil(t, getAddon, "draft add-on must not be empty")

				assert.Equalf(t, draftAddon.ID, getAddon.ID,
					"Plan ID mismatch: %s = %s", draftAddon.ID, getAddon.ID)
				assert.Equalf(t, draftAddon.Key, getAddon.Key,
					"Plan Key mismatch: %s = %s", draftAddon.Key, getAddon.Key)
				assert.Equalf(t, draftAddon.Version, getAddon.Version,
					"Plan Version mismatch: %d = %d", draftAddon.Version, getAddon.Version)
				assert.Equalf(t, productcatalog.AddonStatusDraft, getAddon.Status(),
					"Plan Status mismatch: expected=%s, actual=%s", productcatalog.AddonStatusDraft, getAddon.Status())
			})

			t.Run("Update", func(t *testing.T) {
				updateInput := addon.UpdateAddonInput{
					NamespacedID: models.NamespacedID{
						Namespace: addonInput.Namespace,
						ID:        draftAddon.ID,
					},
					RateCards: &productcatalog.RateCards{},
				}

				updatedAddon, err := env.Addon.UpdateAddon(ctx, updateInput)
				require.NoErrorf(t, err, "updating draft add-on must not fail")
				require.NotNil(t, updatedAddon, "updated draft add-on must not be empty")

				addon.AssertAddonUpdateInputEqual(t, updateInput, *updatedAddon)
			})

			var publishedAddon *addon.Addon
			t.Run("Publish", func(t *testing.T) {
				publishAt := time.Now().Truncate(time.Microsecond)

				publishInput := addon.PublishAddonInput{
					NamespacedID: draftAddon.NamespacedID,
					EffectivePeriod: productcatalog.EffectivePeriod{
						EffectiveFrom: &publishAt,
						EffectiveTo:   nil,
					},
				}

				publishedAddon, err = env.Addon.PublishAddon(ctx, publishInput)
				require.NoErrorf(t, err, "publishing draft add-on must not fail")
				require.NotNil(t, publishedAddon, "published add-on must not be empty")
				require.NotNil(t, publishedAddon.EffectiveFrom, "EffectiveFrom for published add-on must not be empty")

				assert.Equalf(t, publishAt, *publishedAddon.EffectiveFrom,
					"EffectiveFrom for published add-on mismatch: expected=%s, actual=%s", publishAt, *publishedAddon.EffectiveFrom)
				assert.Equalf(t, productcatalog.AddonStatusActive, publishedAddon.Status(),
					"add-on Status mismatch: expected=%s, actual=%s", productcatalog.AddonStatusActive, publishedAddon.Status())

				t.Run("Update", func(t *testing.T) {
					updateInput := addon.UpdateAddonInput{
						NamespacedID: draftAddon.NamespacedID,
						Name:         lo.ToPtr("Invalid Update"),
					}

					_, err = env.Addon.UpdateAddon(ctx, updateInput)
					require.Errorf(t, err, "updating active add-on must fail")
				})
			})

			var nextAddon *addon.Addon
			t.Run("NewVersion", func(t *testing.T) {
				nextAddon, err = env.Addon.CreateAddon(ctx, addonInput)
				require.NoErrorf(t, err, "creating a new draft add-on from active must not fail")
				require.NotNil(t, nextAddon, "new draft add-on must not be empty")

				assert.Equalf(t, publishedAddon.Version+1, nextAddon.Version,
					"new draft add-on must have higher version number")
				assert.Equalf(t, productcatalog.AddonStatusDraft, nextAddon.Status(),
					"add-on Status mismatch: expected=%s, actual=%s", productcatalog.AddonStatusDraft, nextAddon.Status())

				t.Run("PublishUnaligned", func(t *testing.T) {
					updateInput := addon.UpdateAddonInput{
						NamespacedID: nextAddon.NamespacedID,
						RateCards: &productcatalog.RateCards{
							&productcatalog.FlatFeeRateCard{
								RateCardMeta: productcatalog.RateCardMeta{
									Key:  "misaligned1",
									Name: "Misaligned 1",
									Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
										Amount:      decimal.NewFromInt(100),
										PaymentTerm: productcatalog.DefaultPaymentTerm,
									}),
								},
								BillingCadence: lo.ToPtr(testutils.GetISODuration(t, "P1W")),
							},
							&productcatalog.FlatFeeRateCard{
								RateCardMeta: productcatalog.RateCardMeta{
									Key:  "misaligned2",
									Name: "Misaligned 2",
									Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
										Amount:      decimal.NewFromInt(10),
										PaymentTerm: productcatalog.DefaultPaymentTerm,
									}),
								},
								BillingCadence: lo.ToPtr(testutils.GetISODuration(t, "P1M")),
							},
						},
					}

					_, err := env.Addon.UpdateAddon(ctx, updateInput)
					require.NoError(t, err)

					// Get the updated add-on
					_, err = env.Addon.GetAddon(ctx, addon.GetAddonInput{
						NamespacedID: nextAddon.NamespacedID,
					})
					require.NoError(t, err)

					// Let's try to publish the add-on
					publishAt := time.Now().Truncate(time.Microsecond)

					publishInput := addon.PublishAddonInput{
						NamespacedID: nextAddon.NamespacedID,
						EffectivePeriod: productcatalog.EffectivePeriod{
							EffectiveFrom: &publishAt,
							EffectiveTo:   nil,
						},
					}

					_, err = env.Addon.PublishAddon(ctx, publishInput)
					require.Error(t, err, "publishing draft add-on with alignment issues must fail")

					// Let's update the plan to fix the alignment issue
					_, err = env.Addon.UpdateAddon(ctx, addon.UpdateAddonInput{
						NamespacedID: nextAddon.NamespacedID,
						RateCards:    &publishedAddon.RateCards,
					})
					require.NoError(t, err)
				})

				t.Run("Publish", func(t *testing.T) {
					publishAt := time.Now().Truncate(time.Microsecond)

					publishInput := addon.PublishAddonInput{
						NamespacedID: nextAddon.NamespacedID,
						EffectivePeriod: productcatalog.EffectivePeriod{
							EffectiveFrom: &publishAt,
							EffectiveTo:   nil,
						},
					}

					publishedNextAddon, err := env.Addon.PublishAddon(ctx, publishInput)
					require.NoErrorf(t, err, "publishing draft add-on must not fail")
					require.NotNil(t, publishedNextAddon, "published add-on must not be empty")
					require.NotNil(t, publishedNextAddon.EffectiveFrom, "EffectiveFrom for published add-on must not be empty")

					assert.Equalf(t, publishAt, *publishedNextAddon.EffectiveFrom,
						"EffectiveFrom for published add-on mismatch: expected=%s, actual=%s", publishAt, *publishedNextAddon.EffectiveFrom)
					assert.Equalf(t, productcatalog.AddonStatusActive, publishedNextAddon.Status(),
						"add-on Status mismatch: expected=%s, actual=%s", productcatalog.AddonStatusActive, publishedNextAddon.Status())

					prevAddon, err := env.Addon.GetAddon(ctx, addon.GetAddonInput{
						NamespacedID: publishedAddon.NamespacedID,
					})
					require.NoErrorf(t, err, "getting previous add-on version must not fail")
					require.NotNil(t, prevAddon, "previous add version must not be empty")

					assert.Equalf(t, productcatalog.AddonStatusArchived, prevAddon.Status(),
						"add Status mismatch: expected=%s, actual=%s", productcatalog.AddonStatusArchived, prevAddon.Status())

					t.Run("Archive", func(t *testing.T) {
						archiveAt := time.Now().Truncate(time.Microsecond)

						archiveInput := addon.ArchiveAddonInput{
							NamespacedID: nextAddon.NamespacedID,
							EffectiveTo:  archiveAt,
						}

						archivedAddon, err := env.Addon.ArchiveAddon(ctx, archiveInput)
						require.NoErrorf(t, err, "archiving add-on must not fail")
						require.NotNil(t, archivedAddon, "archived add-on must not be empty")
						require.NotNil(t, archivedAddon.EffectiveTo, "EffectiveFrom for archived add-on must not be empty")

						assert.Equalf(t, archiveAt, *archivedAddon.EffectiveTo,
							"EffectiveTo for published add-on mismatch: expected=%s, actual=%s", archiveAt, *archivedAddon.EffectiveTo)
						assert.Equalf(t, productcatalog.AddonStatusArchived, archivedAddon.Status(),
							"Status mismatch for archived add-on: expected=%s, actual=%s", productcatalog.AddonStatusArchived, archivedAddon.Status())
					})
				})

				t.Run("Delete", func(t *testing.T) {
					deleteInput := addon.DeleteAddonInput{
						NamespacedID: nextAddon.NamespacedID,
					}

					err = env.Addon.DeleteAddon(ctx, deleteInput)
					require.NoErrorf(t, err, "deleting add-on must not fail")

					deletedAddon, err := env.Addon.GetAddon(ctx, addon.GetAddonInput{
						NamespacedID: nextAddon.NamespacedID,
					})
					require.NoErrorf(t, err, "getting deleted add-on version must not fail")
					require.NotNil(t, deletedAddon, "deleted add-on version must not be empty")

					assert.NotNilf(t, deletedAddon.DeletedAt, "deletedAt must not be empty")
				})
			})
		})
	})
}

var (
	MonthPeriod      = isodate.FromDuration(30 * 24 * time.Hour)
	TwoMonthPeriod   = isodate.FromDuration(60 * 24 * time.Hour)
	ThreeMonthPeriod = isodate.FromDuration(90 * 24 * time.Hour)
	SixMonthPeriod   = isodate.FromDuration(180 * 24 * time.Hour)
)

func NewAddon(t *testing.T, namespace string) addon.CreateAddonInput {
	t.Helper()

	return addon.CreateAddonInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Addon: productcatalog.Addon{
			AddonMeta: productcatalog.AddonMeta{
				Key:          "security",
				Name:         "Security",
				Description:  lo.ToPtr("Security add-on"),
				InstanceType: productcatalog.AddonInstanceTypeSingle,
				Currency:     currency.USD,
				Metadata:     models.Metadata{"name": "security"},
				Annotations:  models.Annotations{"name": "security"},
			},
		},
	}
}

func NewTestULID(t *testing.T) string {
	t.Helper()

	return ulid.MustNew(ulid.Timestamp(time.Now().UTC()), rand.Reader).String()
}

var NewTestNamespace = NewTestULID

func NewTestMeters(t *testing.T, namespace string) []meter.Meter {
	t.Helper()

	return []meter.Meter{
		{
			ManagedResource: models.ManagedResource{
				ID: NewTestULID(t),
				NamespacedModel: models.NamespacedModel{
					Namespace: namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: "Test meter",
			},
			Key:         "api_requests_total",
			Aggregation: meter.MeterAggregationCount,
			EventType:   "request",
			GroupBy: map[string]string{
				"method": "$.method",
				"path":   "$.path",
			},
		},
		{
			ManagedResource: models.ManagedResource{
				ID: NewTestULID(t),
				NamespacedModel: models.NamespacedModel{
					Namespace: namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: "Test meter",
			},
			Key:           "tokens_total",
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "prompt",
			ValueProperty: lo.ToPtr("$.tokens"),
			GroupBy: map[string]string{
				"model": "$.model",
				"type":  "$.type",
			},
		},
		{
			ManagedResource: models.ManagedResource{
				ID: NewTestULID(t),
				NamespacedModel: models.NamespacedModel{
					Namespace: namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: "Test meter",
			},
			Key:           "workload_runtime_duration_seconds",
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "workload",
			ValueProperty: lo.ToPtr("$.duration_seconds"),
			GroupBy: map[string]string{
				"region":        "$.region",
				"zone":          "$.zone",
				"instance_type": "$.instance_type",
			},
		},
	}
}

type testEnv struct {
	Meter   *meteradapter.TestAdapter
	Feature feature.FeatureConnector
	Addon   addon.Service

	db     *testutils.TestDB
	client *entdb.Client

	close sync.Once
}

func (e *testEnv) DBSchemaMigrate(t *testing.T) {
	require.NotNilf(t, e.db, "database must be initialized")

	err := e.db.EntDriver.Client().Schema.Create(context.Background())
	require.NoErrorf(t, err, "schema migration must not fail")
}

func (e *testEnv) Close(t *testing.T) {
	t.Helper()

	e.close.Do(func() {
		if e.db != nil {
			if err := e.db.EntDriver.Close(); err != nil {
				t.Errorf("failed to close ent driver: %v", err)
			}

			if err := e.db.PGDriver.Close(); err != nil {
				t.Errorf("failed to postgres driver: %v", err)
			}
		}

		if e.client != nil {
			if err := e.client.Close(); err != nil {
				t.Errorf("failed to close ent client: %v", err)
			}
		}
	})
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	logger := testutils.NewLogger(t)

	db := testutils.InitPostgresDB(t)
	client := db.EntDriver.Client()

	publisher := eventbus.NewMock(t)

	meterAdapter, err := meteradapter.New(nil)
	require.NoErrorf(t, err, "initializing Meter adapter must not fail")
	require.NotNilf(t, meterAdapter, "Meter adapter must not be nil")

	featureAdapter := productcatalogadapter.NewPostgresFeatureRepo(client, logger)
	featureService := feature.NewFeatureConnector(featureAdapter, meterAdapter, publisher)

	addonAdapter, err := adapter.New(adapter.Config{
		Client: client,
		Logger: logger,
	})
	require.NoErrorf(t, err, "initializing add-on adapter must not fail")
	require.NotNilf(t, addonAdapter, "add-on adapter must not be nil")

	config := Config{
		Feature:   featureService,
		Adapter:   addonAdapter,
		Logger:    logger,
		Publisher: publisher,
	}

	addonService, err := New(config)
	require.NoErrorf(t, err, "initializing add-on service must not fail")
	require.NotNilf(t, addonService, "add-on service must not be nil")

	return &testEnv{
		Meter:   meterAdapter,
		Feature: featureService,
		Addon:   addonService,
		db:      db,
		client:  client,
		close:   sync.Once{},
	}
}
