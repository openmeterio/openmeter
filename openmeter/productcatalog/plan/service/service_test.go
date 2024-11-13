package service

import (
	"context"
	"crypto/rand"
	"slices"
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
	productcatalogadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan/adapter"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func TestPlanService(t *testing.T) {
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
	env.Meter.ReplaceMeters(ctx, NewTestMeters(t, namespace))

	meters, err := env.Meter.ListMeters(ctx, namespace)
	require.NoErrorf(t, err, "listing Meters must not fail")
	require.NotEmptyf(t, meters, "list of Meters must not be empty")

	// Set Feature for each Meter
	features := make(map[string]feature.Feature, len(meters))
	for _, m := range meters {
		input := feature.CreateFeatureInputs{
			Name:                m.Slug,
			Key:                 m.Slug,
			Namespace:           namespace,
			MeterSlug:           lo.ToPtr(m.Slug),
			MeterGroupByFilters: m.GroupBy,
			Metadata:            map[string]string{},
		}

		feat, err := env.Feature.CreateFeature(ctx, input)
		require.NoErrorf(t, err, "creating Feature must not fail")
		require.NotNil(t, feat, "Feature must not be empty")

		features[feat.Key] = feat
	}

	t.Run("Plan", func(t *testing.T) {
		t.Run("Create", func(t *testing.T) {
			planInput := NewProPlan(t, namespace)

			draftPlan, err := env.Plan.CreatePlan(ctx, planInput)
			require.NoErrorf(t, err, "creating Plan must not fail")
			require.NotNil(t, draftPlan, "Plan must not be empty")

			plan.AssertPlanCreateInputEqual(t, planInput, *draftPlan)
			assert.Equalf(t, plan.DraftStatus, draftPlan.Status(), "Plan Status mismatch: expected=%s, actual=%s", plan.DraftStatus, draftPlan.Status())

			t.Run("Get draft plan", func(t *testing.T) {
				getPlan, err := env.Plan.GetPlan(ctx, plan.GetPlanInput{
					NamespacedID: models.NamespacedID{
						Namespace: planInput.Namespace,
					},
					Key:           planInput.Key,
					IncludeLatest: true,
				})
				require.NoErrorf(t, err, "getting draft Plan must not fail")
				require.NotNil(t, getPlan, "draft Plan must not be empty")

				assert.Equalf(t, draftPlan.ID, getPlan.ID, "Plan ID mismatch: %s = %s", draftPlan.ID, getPlan.ID)
				assert.Equalf(t, draftPlan.Key, getPlan.Key, "Plan Key mismatch: %s = %s", draftPlan.Key, getPlan.Key)
				assert.Equalf(t, draftPlan.Version, getPlan.Version, "Plan Version mismatch: %d = %d", draftPlan.Version, getPlan.Version)
				assert.Equalf(t, plan.DraftStatus, getPlan.Status(), "Plan Status mismatch: expected=%s, actual=%s", plan.DraftStatus, getPlan.Status())
			})

			t.Run("NewPhase", func(t *testing.T) {
				updatedPhases := slices.Clone(draftPlan.Phases)
				updatedPhases = append(updatedPhases, plan.Phase{
					NamespacedID: models.NamespacedID{
						Namespace: namespace,
					},
					Key:         "pro-2",
					Name:        "Pro-2",
					Description: lo.ToPtr("Pro-2 phase"),
					Metadata:    map[string]string{"name": "pro-2"},
					StartAfter:  ThreeMonthPeriod,
					PlanID:      draftPlan.ID,
					RateCards: []plan.RateCard{
						plan.NewRateCardFrom(plan.UsageBasedRateCard{
							RateCardMeta: plan.RateCardMeta{
								NamespacedID: models.NamespacedID{
									Namespace: namespace,
								},
								Key:         "pro-2-ratecard-1",
								Type:        plan.UsageBasedRateCardType,
								Name:        "Pro-2 RateCard 1",
								Description: lo.ToPtr("Pro-2 RateCard 1"),
								Metadata:    map[string]string{"name": "pro-2-ratecard-1"},
								Feature: &feature.Feature{
									Namespace: namespace,
									Key:       "api_requests_total",
								},
								EntitlementTemplate: lo.ToPtr(plan.NewEntitlementTemplateFrom(plan.MeteredEntitlementTemplate{
									Metadata:                nil,
									IsSoftLimit:             true,
									IssueAfterReset:         lo.ToPtr(500.0),
									IssueAfterResetPriority: lo.ToPtr[uint8](1),
									PreserveOverageAtReset:  lo.ToPtr(true),
									UsagePeriod:             MonthPeriod,
								})),
								TaxConfig: &plan.TaxConfig{
									Stripe: &plan.StripeTaxConfig{
										Code: "txcd_10000000",
									},
								},
							},
							BillingCadence: MonthPeriod,
							Price: lo.ToPtr(plan.NewPriceFrom(plan.TieredPrice{
								Mode: plan.VolumeTieredPrice,
								Tiers: []plan.PriceTier{
									{
										UpToAmount: lo.ToPtr(decimal.NewFromInt(1000)),
										FlatPrice: &plan.PriceTierFlatPrice{
											Amount: decimal.NewFromInt(100),
										},
										UnitPrice: &plan.PriceTierUnitPrice{
											Amount: decimal.NewFromInt(50),
										},
									},
								},
								MinimumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
								MaximumAmount: nil,
							})),
						}),
					},
				})

				updateInput := plan.UpdatePlanInput{
					NamespacedID: models.NamespacedID{
						Namespace: planInput.Namespace,
						ID:        draftPlan.ID,
					},
					Phases: lo.ToPtr(updatedPhases),
				}

				updatedPlan, err := env.Plan.UpdatePlan(ctx, updateInput)
				require.NoErrorf(t, err, "updating draft Plan must not fail")
				require.NotNil(t, updatedPlan, "updated draft Plan must not be empty")
				require.NotNil(t, updatedPlan, "updated draft Plan must not be empty")

				assert.Equalf(t, plan.DraftStatus, updatedPlan.Status(), "Plan Status mismatch: expected=%s, actual=%s", plan.DraftStatus, updatedPlan.Status())

				plan.AssertPlanPhasesEqual(t, updatedPhases, updatedPlan.Phases)

				t.Run("Update", func(t *testing.T) {
					updatedPhases = slices.Clone(draftPlan.Phases)
					updatedPhases = append(updatedPhases, plan.Phase{
						NamespacedID: models.NamespacedID{
							Namespace: namespace,
						},
						Key:         "pro-2",
						Name:        "Pro-2",
						Description: lo.ToPtr("Pro-2 phase"),
						Metadata:    map[string]string{"name": "pro-2"},
						StartAfter:  SixMonthPeriod,
						PlanID:      draftPlan.ID,
						RateCards: []plan.RateCard{
							plan.NewRateCardFrom(plan.UsageBasedRateCard{
								RateCardMeta: plan.RateCardMeta{
									NamespacedID: models.NamespacedID{
										Namespace: namespace,
									},
									Key:                 "pro-2-ratecard-1",
									Type:                plan.UsageBasedRateCardType,
									Name:                "Pro-2 RateCard 1",
									Description:         lo.ToPtr("Pro-2 RateCard 1"),
									Metadata:            map[string]string{"name": "pro-ratecard-1"},
									Feature:             nil,
									EntitlementTemplate: nil,
									TaxConfig: &plan.TaxConfig{
										Stripe: &plan.StripeTaxConfig{
											Code: "txcd_10000000",
										},
									},
								},
								BillingCadence: MonthPeriod,
								Price: lo.ToPtr(plan.NewPriceFrom(plan.TieredPrice{
									Mode: plan.VolumeTieredPrice,
									Tiers: []plan.PriceTier{
										{
											UpToAmount: lo.ToPtr(decimal.NewFromInt(1000)),
											FlatPrice: &plan.PriceTierFlatPrice{
												Amount: decimal.NewFromInt(50),
											},
											UnitPrice: &plan.PriceTierUnitPrice{
												Amount: decimal.NewFromInt(25),
											},
										},
									},
									MinimumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
									MaximumAmount: nil,
								})),
							}),
						},
					})

					updateInput = plan.UpdatePlanInput{
						NamespacedID: models.NamespacedID{
							Namespace: planInput.Namespace,
							ID:        draftPlan.ID,
						},
						Phases: lo.ToPtr(updatedPhases),
					}

					updatedPlan, err = env.Plan.UpdatePlan(ctx, updateInput)
					require.NoErrorf(t, err, "updating draft Plan must not fail")
					require.NotNil(t, updatedPlan, "updated draft Plan must not be empty")

					plan.AssertPlanPhasesEqual(t, updatedPhases, updatedPlan.Phases)
				})

				t.Run("Delete", func(t *testing.T) {
					updateInput = plan.UpdatePlanInput{
						NamespacedID: models.NamespacedID{
							Namespace: planInput.Namespace,
							ID:        draftPlan.ID,
						},
						Phases: lo.ToPtr(draftPlan.Phases),
					}

					updatedPlan, err = env.Plan.UpdatePlan(ctx, updateInput)
					require.NoErrorf(t, err, "updating draft Plan must not fail")
					require.NotNil(t, updatedPlan, "updated draft Plan must not be empty")

					plan.AssertPlanEqual(t, *updatedPlan, *draftPlan)
				})
			})

			var publishedPlan *plan.Plan
			t.Run("Publish", func(t *testing.T) {
				publishAt := time.Now().Truncate(time.Microsecond)

				publishInput := plan.PublishPlanInput{
					NamespacedID: models.NamespacedID{
						Namespace: draftPlan.Namespace,
						ID:        draftPlan.ID,
					},
					EffectivePeriod: plan.EffectivePeriod{
						EffectiveFrom: &publishAt,
						EffectiveTo:   nil,
					},
				}

				publishedPlan, err = env.Plan.PublishPlan(ctx, publishInput)
				require.NoErrorf(t, err, "publishing draft Plan must not fail")
				require.NotNil(t, publishedPlan, "published Plan must not be empty")
				require.NotNil(t, publishedPlan.EffectiveFrom, "EffectiveFrom for published Plan must not be empty")

				assert.Equalf(t, publishAt, *publishedPlan.EffectiveFrom, "EffectiveFrom for published Plan mismatch: expected=%s, actual=%s", publishAt, *publishedPlan.EffectiveFrom)
				assert.Equalf(t, plan.ActiveStatus, publishedPlan.Status(), "Plan Status mismatch: expected=%s, actual=%s", plan.ActiveStatus, publishedPlan.Status())

				t.Run("Update", func(t *testing.T) {
					updateInput := plan.UpdatePlanInput{
						NamespacedID: models.NamespacedID{
							Namespace: draftPlan.Namespace,
							ID:        draftPlan.ID,
						},
						Name: lo.ToPtr("Invalid Update"),
					}

					_, err = env.Plan.UpdatePlan(ctx, updateInput)
					require.Errorf(t, err, "updating active Plan must fail")
				})
			})

			var nextPlan *plan.Plan
			t.Run("NewVersion", func(t *testing.T) {
				nextInput := plan.NextPlanInput{
					NamespacedID: models.NamespacedID{
						Namespace: publishedPlan.Namespace,
					},
					Key: publishedPlan.Key,
				}

				nextPlan, err = env.Plan.NextPlan(ctx, nextInput)
				require.NoErrorf(t, err, "creating a new draft Plan from active must not fail")
				require.NotNil(t, nextPlan, "new draft Plan must not be empty")

				assert.Equalf(t, publishedPlan.Version+1, nextPlan.Version, "new draft Plan must have higher version number")
				assert.Equalf(t, plan.DraftStatus, nextPlan.Status(), "Plan Status mismatch: expected=%s, actual=%s", plan.DraftStatus, nextPlan.Status())

				var publishedNextPlan *plan.Plan
				t.Run("Publish", func(t *testing.T) {
					publishAt := time.Now().Truncate(time.Microsecond)

					publishInput := plan.PublishPlanInput{
						NamespacedID: models.NamespacedID{
							Namespace: nextPlan.Namespace,
							ID:        nextPlan.ID,
						},
						EffectivePeriod: plan.EffectivePeriod{
							EffectiveFrom: &publishAt,
							EffectiveTo:   nil,
						},
					}

					publishedNextPlan, err = env.Plan.PublishPlan(ctx, publishInput)
					require.NoErrorf(t, err, "publishing draft Plan must not fail")
					require.NotNil(t, publishedNextPlan, "published Plan must not be empty")
					require.NotNil(t, publishedNextPlan.EffectiveFrom, "EffectiveFrom for published Plan must not be empty")

					assert.Equalf(t, publishAt, *publishedNextPlan.EffectiveFrom, "EffectiveFrom for published Plan mismatch: expected=%s, actual=%s", publishAt, *publishedPlan.EffectiveFrom)
					assert.Equalf(t, plan.ActiveStatus, publishedNextPlan.Status(), "Plan Status mismatch: expected=%s, actual=%s", plan.ActiveStatus, publishedNextPlan.Status())

					prevPlan, err := env.Plan.GetPlan(ctx, plan.GetPlanInput{
						NamespacedID: models.NamespacedID{
							Namespace: publishedPlan.Namespace,
							ID:        publishedPlan.ID,
						},
					})
					require.NoErrorf(t, err, "getting previous Plan version must not fail")
					require.NotNil(t, prevPlan, "previous Plan version must not be empty")

					assert.Equalf(t, plan.ArchivedStatus, prevPlan.Status(), "Plan Status mismatch: expected=%s, actual=%s", plan.ArchivedStatus, prevPlan.Status())
				})

				var archivedPlan *plan.Plan
				t.Run("Archive", func(t *testing.T) {
					archiveAt := time.Now().Truncate(time.Microsecond)

					archiveInput := plan.ArchivePlanInput{
						NamespacedID: models.NamespacedID{
							Namespace: nextPlan.Namespace,
							ID:        nextPlan.ID,
						},
						EffectiveTo: archiveAt,
					}

					archivedPlan, err = env.Plan.ArchivePlan(ctx, archiveInput)
					require.NoErrorf(t, err, "archiving Plan must not fail")
					require.NotNil(t, archivedPlan, "archived Plan must not be empty")
					require.NotNil(t, archivedPlan.EffectiveTo, "EffectiveFrom for archived Plan must not be empty")

					assert.Equalf(t, archiveAt, *archivedPlan.EffectiveTo, "EffectiveTo for published Plan mismatch: expected=%s, actual=%s", archiveAt, *archivedPlan.EffectiveTo)
					assert.Equalf(t, plan.ArchivedStatus, archivedPlan.Status(), "Status mismatch for archived Plan: expected=%s, actual=%s", plan.ArchivedStatus, archivedPlan.Status())
				})

				t.Run("Delete", func(t *testing.T) {
					deleteInput := plan.DeletePlanInput{
						NamespacedID: models.NamespacedID{
							Namespace: archivedPlan.Namespace,
							ID:        archivedPlan.ID,
						},
					}

					err = env.Plan.DeletePlan(ctx, deleteInput)
					require.NoErrorf(t, err, "deleting Plan must not fail")
					require.NotNil(t, archivedPlan, "archived Plan must not be empty")
					require.NotNil(t, archivedPlan.EffectiveTo, "EffectiveFrom for archived Plan must not be empty")

					deletedPlan, err := env.Plan.GetPlan(ctx, plan.GetPlanInput{
						NamespacedID: models.NamespacedID{
							Namespace: archivedPlan.Namespace,
							ID:        archivedPlan.ID,
						},
					})
					require.NoErrorf(t, err, "getting deleted Plan version must not fail")
					require.NotNil(t, deletedPlan, "deleted Plan version must not be empty")

					assert.NotNilf(t, deletedPlan.DeletedAt, "deletedAt must not be empty")
				})
			})
		})
	})
}

var (
	MonthPeriod      = datex.FromDuration(30 * 24 * time.Hour)
	TwoMonthPeriod   = datex.FromDuration(60 * 24 * time.Hour)
	ThreeMonthPeriod = datex.FromDuration(90 * 24 * time.Hour)
	SixMonthPeriod   = datex.FromDuration(180 * 24 * time.Hour)
)

func NewProPlan(t *testing.T, namespace string) plan.CreatePlanInput {
	t.Helper()

	return plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Key:         "pro",
		Name:        "Pro",
		Description: lo.ToPtr("Pro plan v1"),
		Metadata:    map[string]string{"name": "pro"},
		Currency:    currency.USD,
		Phases: []plan.Phase{
			{
				NamespacedID: models.NamespacedID{
					Namespace: namespace,
				},
				Key:         "trial",
				Name:        "Trial",
				Description: lo.ToPtr("Trial phase"),
				Metadata:    map[string]string{"name": "trial"},
				StartAfter:  MonthPeriod,
				RateCards: []plan.RateCard{
					plan.NewRateCardFrom(plan.FlatFeeRateCard{
						RateCardMeta: plan.RateCardMeta{
							NamespacedID: models.NamespacedID{
								Namespace: namespace,
							},
							Key:                 "trial-ratecard-1",
							Type:                plan.FlatFeeRateCardType,
							Name:                "Trial RateCard 1",
							Description:         lo.ToPtr("Trial RateCard 1"),
							Metadata:            map[string]string{"name": "trial-ratecard-1"},
							Feature:             nil,
							EntitlementTemplate: nil,
							TaxConfig: &plan.TaxConfig{
								Stripe: &plan.StripeTaxConfig{
									Code: "txcd_10000000",
								},
							},
						},
						BillingCadence: &MonthPeriod,
						Price: plan.NewPriceFrom(plan.FlatPrice{
							Amount:      decimal.NewFromInt(0),
							PaymentTerm: plan.InArrearsPaymentTerm,
						}),
					}),
				},
			},
			{
				NamespacedID: models.NamespacedID{
					Namespace: namespace,
				},
				Key:         "pro",
				Name:        "Pro",
				Description: lo.ToPtr("Pro phase"),
				Metadata:    map[string]string{"name": "pro"},
				StartAfter:  TwoMonthPeriod,
				RateCards: []plan.RateCard{
					plan.NewRateCardFrom(plan.UsageBasedRateCard{
						RateCardMeta: plan.RateCardMeta{
							NamespacedID: models.NamespacedID{
								Namespace: namespace,
							},
							Key:                 "pro-ratecard-1",
							Type:                plan.UsageBasedRateCardType,
							Name:                "Pro RateCard 1",
							Description:         lo.ToPtr("Pro RateCard 1"),
							Metadata:            map[string]string{"name": "pro-ratecard-1"},
							Feature:             nil,
							EntitlementTemplate: nil,
							TaxConfig: &plan.TaxConfig{
								Stripe: &plan.StripeTaxConfig{
									Code: "txcd_10000000",
								},
							},
						},
						BillingCadence: MonthPeriod,
						Price: lo.ToPtr(plan.NewPriceFrom(plan.TieredPrice{
							Mode: plan.VolumeTieredPrice,
							Tiers: []plan.PriceTier{
								{
									UpToAmount: lo.ToPtr(decimal.NewFromInt(1000)),
									FlatPrice: &plan.PriceTierFlatPrice{
										Amount: decimal.NewFromInt(100),
									},
									UnitPrice: &plan.PriceTierUnitPrice{
										Amount: decimal.NewFromInt(50),
									},
								},
							},
							MinimumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
							MaximumAmount: nil,
						})),
					}),
				},
			},
		},
	}
}

func NewTestULID(t *testing.T) string {
	t.Helper()

	return ulid.MustNew(ulid.Timestamp(time.Now().UTC()), rand.Reader).String()
}

var NewTestNamespace = NewTestULID

func NewTestMeters(t *testing.T, namespace string) []models.Meter {
	t.Helper()

	return []models.Meter{
		{
			Namespace:   namespace,
			ID:          NewTestULID(t),
			Slug:        "api_requests_total",
			Aggregation: models.MeterAggregationCount,
			EventType:   "request",
			GroupBy: map[string]string{
				"method": "$.method",
				"path":   "$.path",
			},
			WindowSize: "MINUTE",
		},
		{
			Namespace:     namespace,
			ID:            NewTestULID(t),
			Slug:          "tokens_total",
			Aggregation:   models.MeterAggregationSum,
			EventType:     "prompt",
			ValueProperty: "$.tokens",
			GroupBy: map[string]string{
				"model": "$.model",
				"type":  "$.type",
			},
			WindowSize: "MINUTE",
		},
		{
			Namespace:     namespace,
			ID:            NewTestULID(t),
			Slug:          "workload_runtime_duration_seconds",
			Aggregation:   models.MeterAggregationSum,
			EventType:     "workload",
			ValueProperty: "$.duration_seconds",
			GroupBy: map[string]string{
				"region":        "$.region",
				"zone":          "$.zone",
				"instance_type": "$.instance_type",
			},
			WindowSize: "MINUTE",
		},
	}
}

type testEnv struct {
	Meter   *meter.InMemoryRepository
	Feature feature.FeatureConnector
	Plan    plan.Service

	db     *testutils.TestDB
	client *entdb.Client

	close sync.Once
}

func (e *testEnv) DBSchemaMigrate(t *testing.T) {
	require.NotNilf(t, e.db, "database must be initialized")

	err := migrate.Up(e.db.URL)
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

	meterRepository := meter.NewInMemoryRepository(nil)

	featureAdapter := productcatalogadapter.NewPostgresFeatureRepo(client, logger)
	featureService := feature.NewFeatureConnector(featureAdapter, meterRepository)

	planAdapter, err := adapter.New(adapter.Config{
		Client: client,
		Logger: logger,
	})
	require.NoErrorf(t, err, "initializing Plan adapter must not fail")
	require.NotNilf(t, planAdapter, "Plan adapter must not be nil")

	config := Config{
		Feature: featureService,
		Adapter: planAdapter,
		Logger:  logger,
	}

	planService, err := New(config)
	require.NoErrorf(t, err, "initializing Plan service must not fail")
	require.NotNilf(t, planService, "Plan service must not be nil")

	return &testEnv{
		Meter:   meterRepository,
		Feature: featureService,
		Plan:    planService,
		db:      db,
		client:  client,
		close:   sync.Once{},
	}
}
