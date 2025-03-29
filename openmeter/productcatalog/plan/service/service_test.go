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
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/mockadapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	productcatalogadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan/adapter"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
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
	err := env.Meter.ReplaceMeters(ctx, NewTestMeters(t, namespace))
	require.NoError(t, err, "replacing Meters must not fail")

	result, err := env.Meter.ListMeters(ctx, meter.ListMetersParams{
		Namespace: namespace,
		Page:      pagination.NewPage(1, 100),
	})
	meters := result.Items
	require.NoErrorf(t, err, "listing Meters must not fail")
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

	t.Run("Plan", func(t *testing.T) {
		t.Run("Create", func(t *testing.T) {
			planInput := NewProPlan(t, namespace)

			draftPlan, err := env.Plan.CreatePlan(ctx, planInput)
			require.NoErrorf(t, err, "creating Plan must not fail")
			require.NotNil(t, draftPlan, "Plan must not be empty")

			plan.AssertPlanCreateInputEqual(t, planInput, *draftPlan)
			assert.Equalf(t, productcatalog.DraftStatus, draftPlan.Status(),
				"Plan Status mismatch: expected=%s, actual=%s", productcatalog.DraftStatus, draftPlan.Status())

			t.Run("Get", func(t *testing.T) {
				getPlan, err := env.Plan.GetPlan(ctx, plan.GetPlanInput{
					NamespacedID: models.NamespacedID{
						Namespace: planInput.Namespace,
					},
					Key:           planInput.Key,
					IncludeLatest: true,
				})
				require.NoErrorf(t, err, "getting draft Plan must not fail")
				require.NotNil(t, getPlan, "draft Plan must not be empty")

				assert.Equalf(t, draftPlan.ID, getPlan.ID,
					"Plan ID mismatch: %s = %s", draftPlan.ID, getPlan.ID)
				assert.Equalf(t, draftPlan.Key, getPlan.Key,
					"Plan Key mismatch: %s = %s", draftPlan.Key, getPlan.Key)
				assert.Equalf(t, draftPlan.Version, getPlan.Version,
					"Plan Version mismatch: %d = %d", draftPlan.Version, getPlan.Version)
				assert.Equalf(t, productcatalog.DraftStatus, getPlan.Status(),
					"Plan Status mismatch: expected=%s, actual=%s", productcatalog.DraftStatus, getPlan.Status())
			})

			t.Run("NewPhase", func(t *testing.T) {
				updatedPhases := lo.Map(slices.Clone(draftPlan.Phases), func(p plan.Phase, idx int) plan.Phase {
					if idx == len(draftPlan.Phases)-1 {
						p.Duration = &SixMonthPeriod
					}

					return p
				})
				updatedPhases = append(updatedPhases, plan.Phase{
					PhaseManagedFields: plan.PhaseManagedFields{
						NamespacedID: models.NamespacedID{
							Namespace: namespace,
						},
						PlanID: draftPlan.ID,
					},
					Phase: productcatalog.Phase{
						PhaseMeta: productcatalog.PhaseMeta{
							Key:         "pro-2",
							Name:        "Pro-2",
							Description: lo.ToPtr("Pro-2 phase"),
							Metadata:    models.Metadata{"name": "pro-2"},
							Duration:    nil,
						},
						RateCards: []productcatalog.RateCard{
							&productcatalog.FlatFeeRateCard{
								RateCardMeta: productcatalog.RateCardMeta{
									Key:         "api_requests_total",
									Name:        "Pro-2 RateCard 1",
									Description: lo.ToPtr("Pro-2 RateCard 1"),
									Metadata:    models.Metadata{"name": "pro-2-ratecard-1"},
									Feature: &feature.Feature{
										Namespace: namespace,
										Key:       "api_requests_total",
									},
									EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(
										productcatalog.MeteredEntitlementTemplate{
											Metadata:                nil,
											IsSoftLimit:             true,
											IssueAfterReset:         lo.ToPtr(500.0),
											IssueAfterResetPriority: lo.ToPtr[uint8](1),
											PreserveOverageAtReset:  lo.ToPtr(true),
											UsagePeriod:             MonthPeriod,
										}),
									TaxConfig: &productcatalog.TaxConfig{
										Stripe: &productcatalog.StripeTaxConfig{
											Code: "txcd_10000000",
										},
									},
									Price: productcatalog.NewPriceFrom(productcatalog.TieredPrice{
										Mode: productcatalog.VolumeTieredPrice,
										Tiers: []productcatalog.PriceTier{
											{
												UpToAmount: lo.ToPtr(decimal.NewFromInt(1000)),
												FlatPrice: &productcatalog.PriceTierFlatPrice{
													Amount: decimal.NewFromInt(100),
												},
												UnitPrice: &productcatalog.PriceTierUnitPrice{
													Amount: decimal.NewFromInt(50),
												},
											},
											{
												UpToAmount: nil,
												FlatPrice: &productcatalog.PriceTierFlatPrice{
													Amount: decimal.NewFromInt(75),
												},
												UnitPrice: &productcatalog.PriceTierUnitPrice{
													Amount: decimal.NewFromInt(25),
												},
											},
										},
										Commitments: productcatalog.Commitments{
											MinimumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
											MaximumAmount: nil,
										},
									}),
								},
								BillingCadence: nil,
							},
						},
					},
				})

				updateInput := plan.UpdatePlanInput{
					NamespacedID: models.NamespacedID{
						Namespace: planInput.Namespace,
						ID:        draftPlan.ID,
					},
					Phases: func(p []plan.Phase) *[]productcatalog.Phase {
						if len(p) == 0 {
							return nil
						}

						var phases []productcatalog.Phase

						for _, phase := range p {
							phases = append(phases, productcatalog.Phase{
								PhaseMeta: phase.PhaseMeta,
								RateCards: phase.RateCards,
							})
						}

						return &phases
					}(updatedPhases),
				}

				updatedPlan, err := env.Plan.UpdatePlan(ctx, updateInput)
				require.NoErrorf(t, err, "updating draft Plan must not fail")
				require.NotNil(t, updatedPlan, "updated draft Plan must not be empty")
				require.NotNil(t, updatedPlan, "updated draft Plan must not be empty")

				assert.Equalf(t, productcatalog.DraftStatus, updatedPlan.Status(),
					"Plan Status mismatch: expected=%s, actual=%s", productcatalog.DraftStatus, updatedPlan.Status())

				plan.AssertPlanPhasesEqual(t, updatedPhases, updatedPlan.Phases)

				t.Run("Update", func(t *testing.T) {
					t.Run("PhaseAndRateCards", func(t *testing.T) {
						updatedPhases := lo.Map(slices.Clone(draftPlan.Phases), func(p plan.Phase, idx int) plan.Phase {
							if idx == len(draftPlan.Phases)-1 {
								p.Duration = &SixMonthPeriod
							}

							return p
						})
						updatedPhases = append(updatedPhases, plan.Phase{
							PhaseManagedFields: plan.PhaseManagedFields{
								ManagedModel: models.ManagedModel{},
								NamespacedID: models.NamespacedID{
									Namespace: namespace,
								},
								PlanID: draftPlan.ID,
							},
							Phase: productcatalog.Phase{
								PhaseMeta: productcatalog.PhaseMeta{
									Key:         "pro-2",
									Name:        "Pro-2",
									Description: lo.ToPtr("Pro-2 phase"),
									Metadata:    models.Metadata{"name": "pro-2"},
									Duration:    nil,
								},
								RateCards: []productcatalog.RateCard{
									&plan.UsageBasedRateCard{
										RateCardManagedFields: plan.RateCardManagedFields{
											ManagedModel: models.ManagedModel{
												CreatedAt: time.Time{},
												UpdatedAt: time.Time{},
												DeletedAt: &time.Time{},
											},
											NamespacedID: models.NamespacedID{
												Namespace: namespace,
												ID:        "",
											},
											PhaseID: "",
										},
										UsageBasedRateCard: productcatalog.UsageBasedRateCard{
											RateCardMeta: productcatalog.RateCardMeta{
												Key:                 "pro-2-ratecard-1",
												Name:                "Pro-2 RateCard 1",
												Description:         lo.ToPtr("Pro-2 RateCard 1"),
												Metadata:            models.Metadata{"name": "pro-ratecard-1"},
												Feature:             nil,
												EntitlementTemplate: nil,
												TaxConfig: &productcatalog.TaxConfig{
													Stripe: &productcatalog.StripeTaxConfig{
														Code: "txcd_10000000",
													},
												},
												Price: productcatalog.NewPriceFrom(
													productcatalog.TieredPrice{
														Mode: productcatalog.VolumeTieredPrice,
														Tiers: []productcatalog.PriceTier{
															{
																UpToAmount: lo.ToPtr(decimal.NewFromInt(1000)),
																FlatPrice: &productcatalog.PriceTierFlatPrice{
																	Amount: decimal.NewFromInt(50),
																},
																UnitPrice: &productcatalog.PriceTierUnitPrice{
																	Amount: decimal.NewFromInt(25),
																},
															},
															{
																UpToAmount: nil,
																FlatPrice: &productcatalog.PriceTierFlatPrice{
																	Amount: decimal.NewFromInt(25),
																},
																UnitPrice: &productcatalog.PriceTierUnitPrice{
																	Amount: decimal.NewFromInt(12),
																},
															},
														},
														Commitments: productcatalog.Commitments{
															MinimumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
															MaximumAmount: nil,
														},
													}),
											},
											BillingCadence: MonthPeriod,
										},
									},
								},
							},
						})

						updateInput = plan.UpdatePlanInput{
							NamespacedID: models.NamespacedID{
								Namespace: planInput.Namespace,
								ID:        draftPlan.ID,
							},
							Phases: lo.ToPtr(lo.Map(updatedPhases, func(p plan.Phase, idx int) productcatalog.Phase {
								return productcatalog.Phase{
									PhaseMeta: p.PhaseMeta,
									RateCards: p.RateCards,
								}
							})),
						}

						updatedPlan, err = env.Plan.UpdatePlan(ctx, updateInput)
						require.NoErrorf(t, err, "updating draft Plan must not fail")
						require.NotNil(t, updatedPlan, "updated draft Plan must not be empty")

						plan.AssertPlanPhasesEqual(t, updatedPhases, updatedPlan.Phases)
					})

					t.Run("EmptyRateCards", func(t *testing.T) {
						updatedPhases := lo.Map(slices.Clone(draftPlan.Phases), func(p plan.Phase, idx int) plan.Phase {
							if idx == len(draftPlan.Phases)-1 {
								p.Duration = &SixMonthPeriod
							}

							return p
						})
						updatedPhases = append(updatedPhases, plan.Phase{
							PhaseManagedFields: plan.PhaseManagedFields{
								NamespacedID: models.NamespacedID{
									Namespace: namespace,
								},
								PlanID: draftPlan.ID,
							},
							Phase: productcatalog.Phase{
								PhaseMeta: productcatalog.PhaseMeta{
									Key:         "pro-2",
									Name:        "Pro-2",
									Description: lo.ToPtr("Pro-2 phase"),
									Metadata:    models.Metadata{"name": "pro-2"},
									Duration:    nil,
								},
								RateCards: []productcatalog.RateCard{},
							},
						})

						updateInput = plan.UpdatePlanInput{
							NamespacedID: models.NamespacedID{
								Namespace: planInput.Namespace,
								ID:        draftPlan.ID,
							},
							Phases: lo.ToPtr(lo.Map(updatedPhases, func(p plan.Phase, _ int) productcatalog.Phase {
								return productcatalog.Phase{
									PhaseMeta: p.PhaseMeta,
									RateCards: p.RateCards,
								}
							})),
						}

						updatedPlan, err = env.Plan.UpdatePlan(ctx, updateInput)
						require.NoErrorf(t, err, "updating draft Plan must not fail")
						require.NotNil(t, updatedPlan, "updated draft Plan must not be empty")

						plan.AssertPlanPhasesEqual(t, updatedPhases, updatedPlan.Phases)
					})
				})

				t.Run("Delete", func(t *testing.T) {
					updateInput = plan.UpdatePlanInput{
						NamespacedID: models.NamespacedID{
							Namespace: planInput.Namespace,
							ID:        draftPlan.ID,
						},
						Phases: lo.ToPtr(lo.Map(draftPlan.Phases, func(p plan.Phase, _ int) productcatalog.Phase {
							return productcatalog.Phase{
								PhaseMeta: p.PhaseMeta,
								RateCards: p.RateCards,
							}
						})),
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
					EffectivePeriod: productcatalog.EffectivePeriod{
						EffectiveFrom: &publishAt,
						EffectiveTo:   nil,
					},
				}

				publishedPlan, err = env.Plan.PublishPlan(ctx, publishInput)
				require.NoErrorf(t, err, "publishing draft Plan must not fail")
				require.NotNil(t, publishedPlan, "published Plan must not be empty")
				require.NotNil(t, publishedPlan.EffectiveFrom, "EffectiveFrom for published Plan must not be empty")

				assert.Equalf(t, publishAt, *publishedPlan.EffectiveFrom,
					"EffectiveFrom for published Plan mismatch: expected=%s, actual=%s", publishAt, *publishedPlan.EffectiveFrom)
				assert.Equalf(t, productcatalog.ActiveStatus, publishedPlan.Status(),
					"Plan Status mismatch: expected=%s, actual=%s", productcatalog.ActiveStatus, publishedPlan.Status())

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
				nextPlan, err = env.Plan.CreatePlan(ctx, planInput)
				require.NoErrorf(t, err, "creating a new draft Plan from active must not fail")
				require.NotNil(t, nextPlan, "new draft Plan must not be empty")

				assert.Equalf(t, publishedPlan.Version+1, nextPlan.Version,
					"new draft Plan must have higher version number")
				assert.Equalf(t, productcatalog.DraftStatus, nextPlan.Status(),
					"Plan Status mismatch: expected=%s, actual=%s", productcatalog.DraftStatus, nextPlan.Status())

				// Let's update the plan to enforce alignment
				t.Run("Update to enforce alignment", func(t *testing.T) {
					updateInput := plan.UpdatePlanInput{
						AlignmentUpdate: productcatalog.AlignmentUpdate{
							BillablesMustAlign: lo.ToPtr(true),
						},
						NamespacedID: nextPlan.NamespacedID,
					}

					_, err := env.Plan.UpdatePlan(ctx, updateInput)
					require.NoError(t, err)

					// Get the updated plan
					updatedPlan, err := env.Plan.GetPlan(ctx, plan.GetPlanInput{
						NamespacedID: nextPlan.NamespacedID,
					})
					require.NoError(t, err)

					assert.Equal(t, true, updatedPlan.Alignment.BillablesMustAlign)
				})

				t.Run("Should not allow publishing draft plan with alignment issues", func(t *testing.T) {
					// Let's update the plan to have a misaligned phase
					oldPhases := lo.Map(nextPlan.Phases, func(p plan.Phase, idx int) productcatalog.Phase {
						return productcatalog.Phase{
							PhaseMeta: p.PhaseMeta,
							RateCards: p.RateCards,
						}
					})
					updateInput := plan.UpdatePlanInput{
						NamespacedID: nextPlan.NamespacedID,
						Phases: lo.ToPtr(append(oldPhases, productcatalog.Phase{
							PhaseMeta: productcatalog.PhaseMeta{
								Key:  "misaligned",
								Name: "Misaligned",
							},
							RateCards: []productcatalog.RateCard{
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
						})),
					}

					_, err := env.Plan.UpdatePlan(ctx, updateInput)
					require.NoError(t, err)

					// Get the updated plan
					_, err = env.Plan.GetPlan(ctx, plan.GetPlanInput{
						NamespacedID: nextPlan.NamespacedID,
					})
					require.NoError(t, err)

					// Let's try to publish the plan
					publishAt := time.Now().Truncate(time.Microsecond)

					publishInput := plan.PublishPlanInput{
						NamespacedID: models.NamespacedID{
							Namespace: nextPlan.Namespace,
							ID:        nextPlan.ID,
						},
						EffectivePeriod: productcatalog.EffectivePeriod{
							EffectiveFrom: &publishAt,
							EffectiveTo:   nil,
						},
					}
					_, err = env.Plan.PublishPlan(ctx, publishInput)
					require.Error(t, err, "publishing draft Plan with alignment issues must fail")

					// Let's update the plan to fix the alignment issue by removing the last phase
					_, err = env.Plan.UpdatePlan(ctx, plan.UpdatePlanInput{
						NamespacedID: nextPlan.NamespacedID,
						Phases:       lo.ToPtr(oldPhases),
					})
					require.NoError(t, err)
				})

				t.Run("Publish", func(t *testing.T) {
					publishAt := time.Now().Truncate(time.Microsecond)

					publishInput := plan.PublishPlanInput{
						NamespacedID: models.NamespacedID{
							Namespace: nextPlan.Namespace,
							ID:        nextPlan.ID,
						},
						EffectivePeriod: productcatalog.EffectivePeriod{
							EffectiveFrom: &publishAt,
							EffectiveTo:   nil,
						},
					}

					publishedNextPlan, err := env.Plan.PublishPlan(ctx, publishInput)
					require.NoErrorf(t, err, "publishing draft Plan must not fail")
					require.NotNil(t, publishedNextPlan, "published Plan must not be empty")
					require.NotNil(t, publishedNextPlan.EffectiveFrom, "EffectiveFrom for published Plan must not be empty")

					assert.Equalf(t, publishAt, *publishedNextPlan.EffectiveFrom,
						"EffectiveFrom for published Plan mismatch: expected=%s, actual=%s", publishAt, *publishedPlan.EffectiveFrom)
					assert.Equalf(t, productcatalog.ActiveStatus, publishedNextPlan.Status(),
						"Plan Status mismatch: expected=%s, actual=%s", productcatalog.ActiveStatus, publishedNextPlan.Status())

					prevPlan, err := env.Plan.GetPlan(ctx, plan.GetPlanInput{
						NamespacedID: models.NamespacedID{
							Namespace: publishedPlan.Namespace,
							ID:        publishedPlan.ID,
						},
					})
					require.NoErrorf(t, err, "getting previous Plan version must not fail")
					require.NotNil(t, prevPlan, "previous Plan version must not be empty")

					assert.Equalf(t, productcatalog.ArchivedStatus, prevPlan.Status(),
						"Plan Status mismatch: expected=%s, actual=%s", productcatalog.ArchivedStatus, prevPlan.Status())

					t.Run("Archive", func(t *testing.T) {
						archiveAt := time.Now().Truncate(time.Microsecond)

						archiveInput := plan.ArchivePlanInput{
							NamespacedID: models.NamespacedID{
								Namespace: nextPlan.Namespace,
								ID:        nextPlan.ID,
							},
							EffectiveTo: archiveAt,
						}

						archivedPlan, err := env.Plan.ArchivePlan(ctx, archiveInput)
						require.NoErrorf(t, err, "archiving Plan must not fail")
						require.NotNil(t, archivedPlan, "archived Plan must not be empty")
						require.NotNil(t, archivedPlan.EffectiveTo, "EffectiveFrom for archived Plan must not be empty")

						assert.Equalf(t, archiveAt, *archivedPlan.EffectiveTo,
							"EffectiveTo for published Plan mismatch: expected=%s, actual=%s", archiveAt, *archivedPlan.EffectiveTo)
						assert.Equalf(t, productcatalog.ArchivedStatus, archivedPlan.Status(),
							"Status mismatch for archived Plan: expected=%s, actual=%s", productcatalog.ArchivedStatus, archivedPlan.Status())
					})
				})

				t.Run("Delete", func(t *testing.T) {
					deleteInput := plan.DeletePlanInput{
						NamespacedID: models.NamespacedID{
							Namespace: nextPlan.Namespace,
							ID:        nextPlan.ID,
						},
					}

					err = env.Plan.DeletePlan(ctx, deleteInput)
					require.NoErrorf(t, err, "deleting Plan must not fail")

					deletedPlan, err := env.Plan.GetPlan(ctx, plan.GetPlanInput{
						NamespacedID: models.NamespacedID{
							Namespace: nextPlan.Namespace,
							ID:        nextPlan.ID,
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
	MonthPeriod      = isodate.FromDuration(30 * 24 * time.Hour)
	TwoMonthPeriod   = isodate.FromDuration(60 * 24 * time.Hour)
	ThreeMonthPeriod = isodate.FromDuration(90 * 24 * time.Hour)
	SixMonthPeriod   = isodate.FromDuration(180 * 24 * time.Hour)
)

func NewProPlan(t *testing.T, namespace string) plan.CreatePlanInput {
	t.Helper()

	return plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Key:         "pro",
				Name:        "Pro",
				Description: lo.ToPtr("Pro plan v1"),
				Metadata:    models.Metadata{"name": "pro"},
				Currency:    currency.USD,
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:         "trial",
						Name:        "Trial",
						Description: lo.ToPtr("Trial phase"),
						Metadata:    map[string]string{"name": "trial"},
						Duration:    &TwoMonthPeriod,
					},
					RateCards: []productcatalog.RateCard{
						&plan.FlatFeeRateCard{
							RateCardManagedFields: plan.RateCardManagedFields{
								ManagedModel: models.ManagedModel{
									CreatedAt: time.Time{},
									UpdatedAt: time.Time{},
									DeletedAt: &time.Time{},
								},
								NamespacedID: models.NamespacedID{
									Namespace: "",
									ID:        "",
								},
								PhaseID: "",
							},
							FlatFeeRateCard: productcatalog.FlatFeeRateCard{
								RateCardMeta: productcatalog.RateCardMeta{
									Key:                 "trial-ratecard-1",
									Name:                "Trial RateCard 1",
									Description:         lo.ToPtr("Trial RateCard 1"),
									Metadata:            models.Metadata{"name": "trial-ratecard-1"},
									Feature:             nil,
									EntitlementTemplate: nil,
									TaxConfig: &productcatalog.TaxConfig{
										Stripe: &productcatalog.StripeTaxConfig{
											Code: "txcd_10000000",
										},
									},
									Price: productcatalog.NewPriceFrom(
										productcatalog.FlatPrice{
											Amount:      decimal.NewFromInt(0),
											PaymentTerm: productcatalog.InArrearsPaymentTerm,
										}),
								},
								BillingCadence: &MonthPeriod,
							},
						},
					},
				},
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:         "pro",
						Name:        "Pro",
						Description: lo.ToPtr("Pro phase"),
						Metadata:    models.Metadata{"name": "pro"},
					},
					RateCards: []productcatalog.RateCard{
						&plan.UsageBasedRateCard{
							RateCardManagedFields: plan.RateCardManagedFields{
								ManagedModel: models.ManagedModel{
									CreatedAt: time.Time{},
									UpdatedAt: time.Time{},
									DeletedAt: &time.Time{},
								},
								NamespacedID: models.NamespacedID{
									Namespace: namespace,
									ID:        "",
								},
								PhaseID: "",
							},
							UsageBasedRateCard: productcatalog.UsageBasedRateCard{
								RateCardMeta: productcatalog.RateCardMeta{
									Key:                 "pro-ratecard-1",
									Name:                "Pro RateCard 1",
									Description:         lo.ToPtr("Pro RateCard 1"),
									Metadata:            models.Metadata{"name": "pro-ratecard-1"},
									Feature:             nil,
									EntitlementTemplate: nil,
									TaxConfig: &productcatalog.TaxConfig{
										Stripe: &productcatalog.StripeTaxConfig{
											Code: "txcd_10000000",
										},
									},
									Price: productcatalog.NewPriceFrom(
										productcatalog.TieredPrice{
											Mode: productcatalog.VolumeTieredPrice,
											Tiers: []productcatalog.PriceTier{
												{
													UpToAmount: lo.ToPtr(decimal.NewFromInt(1000)),
													FlatPrice: &productcatalog.PriceTierFlatPrice{
														Amount: decimal.NewFromInt(100),
													},
													UnitPrice: &productcatalog.PriceTierUnitPrice{
														Amount: decimal.NewFromInt(50),
													},
												},
												{
													UpToAmount: nil,
													FlatPrice: &productcatalog.PriceTierFlatPrice{
														Amount: decimal.NewFromInt(75),
													},
													UnitPrice: &productcatalog.PriceTierUnitPrice{
														Amount: decimal.NewFromInt(25),
													},
												},
											},
											Commitments: productcatalog.Commitments{
												MinimumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
												MaximumAmount: nil,
											},
										}),
								},
								BillingCadence: MonthPeriod,
							},
						},
					},
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
	Plan    plan.Service

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

	meterAdapter, err := meteradapter.New(nil)
	require.NoErrorf(t, err, "initializing Meter adapter must not fail")
	require.NotNilf(t, meterAdapter, "Meter adapter must not be nil")

	featureAdapter := productcatalogadapter.NewPostgresFeatureRepo(client, logger)
	featureService := feature.NewFeatureConnector(featureAdapter, meterAdapter, eventbus.NewMock(t))

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
		Meter:   meterAdapter,
		Feature: featureService,
		Plan:    planService,
		db:      db,
		client:  client,
		close:   sync.Once{},
	}
}
