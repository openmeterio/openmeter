package service_test

import (
	"context"
	"slices"
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	pctestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestPlanService(t *testing.T) {
	MonthPeriod := datetime.MustParseDuration(t, "P1M")
	TwoMonthPeriod := datetime.MustParseDuration(t, "P2M")
	SixMonthPeriod := datetime.MustParseDuration(t, "P6M")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup test environment
	env := pctestutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	// Run database migrations
	env.DBSchemaMigrate(t)

	// Get new namespace ID
	namespace := pctestutils.NewTestNamespace(t)

	// Setup meter repository
	err := env.Meter.ReplaceMeters(ctx, pctestutils.NewTestMeters(t, namespace))
	require.NoError(t, err, "replacing meters must not fail")

	result, err := env.Meter.ListMeters(ctx, meter.ListMetersParams{
		Page: pagination.Page{
			PageSize:   1000,
			PageNumber: 1,
		},
		Namespace: namespace,
	})
	require.NoErrorf(t, err, "listing meters must not fail")

	meters := result.Items
	require.NotEmptyf(t, meters, "list of Meters must not be empty")

	// Set a feature for each meter
	features := make([]feature.Feature, 0, len(meters))
	for _, m := range meters {
		input := pctestutils.NewTestFeatureFromMeter(t, &m)

		feat, err := env.Feature.CreateFeature(ctx, input)
		require.NoErrorf(t, err, "creating feature must not fail")
		require.NotNil(t, feat, "feature must not be empty")

		features = append(features, feat)
	}

	addonV1Input := pctestutils.NewTestAddon(t, namespace)

	addonV1Input.RateCards = productcatalog.RateCards{
		&productcatalog.UsageBasedRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:                 features[0].Key,
				Name:                features[0].Name,
				Description:         lo.ToPtr("RateCard 1"),
				Metadata:            models.Metadata{"name": features[0].Name},
				FeatureKey:          lo.ToPtr(features[0].Key),
				FeatureID:           lo.ToPtr(features[0].ID),
				EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.BooleanEntitlementTemplate{}),
				TaxConfig: &productcatalog.TaxConfig{
					Stripe: &productcatalog.StripeTaxConfig{
						Code: "txcd_10000000",
					},
				},
				Price: nil, // This would match with a TieredPrice, which is not supported for add-ons
			},
			BillingCadence: MonthPeriod,
		},
	}

	addonV1, err := env.Addon.CreateAddon(ctx, addonV1Input)
	require.NoErrorf(t, err, "creating add-on must not fail")

	addonV1, err = env.Addon.PublishAddon(ctx, addon.PublishAddonInput{
		NamespacedID: models.NamespacedID{
			Namespace: namespace,
			ID:        addonV1.ID,
		},
		EffectivePeriod: productcatalog.EffectivePeriod{
			EffectiveFrom: lo.ToPtr(time.Now()),
			EffectiveTo:   nil,
		},
	})
	require.NoErrorf(t, err, "publishing add-on must not fail")

	t.Run("Plan", func(t *testing.T) {
		t.Run("Create", func(t *testing.T) {
			planInput := pctestutils.NewTestPlan(t, namespace, []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:         "trial",
						Name:        "Trial",
						Description: lo.ToPtr("Trial phase"),
						Metadata:    map[string]string{"name": "trial"},
						Duration:    &TwoMonthPeriod,
					},
					RateCards: []productcatalog.RateCard{
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:         features[0].Key,
								Name:        features[0].Name,
								Description: lo.ToPtr("RateCard 1"),
								Metadata:    models.Metadata{"name": features[0].Name},
								FeatureKey:  lo.ToPtr(features[0].Key),
								FeatureID:   lo.ToPtr(features[0].ID),
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
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:         "pro",
						Name:        "Pro",
						Description: lo.ToPtr("Pro phase"),
						Metadata:    models.Metadata{"name": "pro"},
					},
					RateCards: []productcatalog.RateCard{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:         features[0].Key,
								Name:        features[0].Name,
								Description: lo.ToPtr("RateCard 1"),
								Metadata:    models.Metadata{"name": features[0].Name},
								FeatureKey:  lo.ToPtr(features[0].Key),
								FeatureID:   lo.ToPtr(features[0].ID),
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
			}...)

			var draftPlanV1 *plan.Plan

			draftPlanV1, err = env.Plan.CreatePlan(ctx, planInput)
			require.NoErrorf(t, err, "creating Plan must not fail")
			require.NotNil(t, draftPlanV1, "Plan must not be empty")

			plan.AssertPlanCreateInputEqual(t, planInput, *draftPlanV1)

			assert.Equalf(t, productcatalog.PlanStatusDraft, draftPlanV1.Status(),
				"Plan Status mismatch: expected=%s, actual=%s", productcatalog.PlanStatusDraft, draftPlanV1.Status())

			t.Run("Get", func(t *testing.T) {
				var getPlanV1 *plan.Plan

				getPlanV1, err = env.Plan.GetPlan(ctx, plan.GetPlanInput{
					NamespacedID: models.NamespacedID{
						Namespace: planInput.Namespace,
					},
					Key:           planInput.Key,
					IncludeLatest: true,
				})
				require.NoErrorf(t, err, "getting draft Plan must not fail")
				require.NotNil(t, getPlanV1, "draft Plan must not be empty")

				assert.Equalf(t, draftPlanV1.ID, getPlanV1.ID, "Plan ID mismatch: %s = %s", draftPlanV1.ID, getPlanV1.ID)

				assert.Equalf(t, draftPlanV1.Key, getPlanV1.Key, "Plan Key mismatch: %s = %s", draftPlanV1.Key, getPlanV1.Key)

				assert.Equalf(t, draftPlanV1.Version, getPlanV1.Version, "Plan Version mismatch: %d = %d",
					draftPlanV1.Version, getPlanV1.Version)

				assert.Equalf(t, productcatalog.PlanStatusDraft, getPlanV1.Status(), "Plan Status mismatch: expected=%s, actual=%s",
					productcatalog.PlanStatusDraft, getPlanV1.Status())
			})

			t.Run("NewPhase", func(t *testing.T) {
				updatedPhases := lo.Map(slices.Clone(draftPlanV1.Phases), func(p plan.Phase, idx int) plan.Phase {
					if idx == len(draftPlanV1.Phases)-1 {
						p.Duration = &SixMonthPeriod
					}

					return p
				})

				updatedPhases = append(updatedPhases, plan.Phase{
					PhaseManagedFields: plan.PhaseManagedFields{
						NamespacedID: models.NamespacedID{
							Namespace: namespace,
						},
						PlanID: draftPlanV1.ID,
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
							&productcatalog.UsageBasedRateCard{
								RateCardMeta: productcatalog.RateCardMeta{
									Key:         features[0].Key,
									Name:        features[0].Name,
									Description: lo.ToPtr("RateCard 1"),
									Metadata:    models.Metadata{"name": features[0].Name},
									FeatureKey:  lo.ToPtr(features[0].Key),
									FeatureID:   lo.ToPtr(features[0].ID),
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
								BillingCadence: MonthPeriod,
							},
						},
					},
				})

				updateInput := plan.UpdatePlanInput{
					NamespacedID: models.NamespacedID{
						Namespace: planInput.Namespace,
						ID:        draftPlanV1.ID,
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

				var updatedPlanV1 *plan.Plan

				updatedPlanV1, err = env.Plan.UpdatePlan(ctx, updateInput)
				require.NoErrorf(t, err, "updating draft Plan must not fail")
				require.NotNil(t, updatedPlanV1, "updated draft Plan must not be empty")
				require.NotNil(t, updatedPlanV1, "updated draft Plan must not be empty")

				assert.Equalf(t, productcatalog.PlanStatusDraft, updatedPlanV1.Status(),
					"Plan Status mismatch: expected=%s, actual=%s", productcatalog.PlanStatusDraft, updatedPlanV1.Status())

				plan.AssertPlanPhasesEqual(t, updatedPhases, updatedPlanV1.Phases)

				t.Run("Update", func(t *testing.T) {
					t.Run("PhaseAndRateCards", func(t *testing.T) {
						updatedPhases = lo.Map(slices.Clone(draftPlanV1.Phases), func(p plan.Phase, idx int) plan.Phase {
							if idx == len(draftPlanV1.Phases)-1 {
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
								PlanID: draftPlanV1.ID,
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
									&productcatalog.UsageBasedRateCard{
										RateCardMeta: productcatalog.RateCardMeta{
											Key:         features[0].Key,
											Name:        features[0].Name,
											Description: lo.ToPtr("RateCard 1"),
											Metadata:    models.Metadata{"name": features[0].Name},
											FeatureKey:  lo.ToPtr(features[0].Key),
											FeatureID:   lo.ToPtr(features[0].ID),
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
						})

						updateInput = plan.UpdatePlanInput{
							NamespacedID: models.NamespacedID{
								Namespace: planInput.Namespace,
								ID:        draftPlanV1.ID,
							},
							Phases: lo.ToPtr(lo.Map(updatedPhases, func(p plan.Phase, idx int) productcatalog.Phase {
								return productcatalog.Phase{
									PhaseMeta: p.PhaseMeta,
									RateCards: p.RateCards,
								}
							})),
						}

						updatedPlanV1, err = env.Plan.UpdatePlan(ctx, updateInput)
						require.NoErrorf(t, err, "updating draft Plan must not fail")
						require.NotNil(t, updatedPlanV1, "updated draft Plan must not be empty")

						plan.AssertPlanPhasesEqual(t, updatedPhases, updatedPlanV1.Phases)
					})

					t.Run("EmptyRateCards", func(t *testing.T) {
						updatedPhases = lo.Map(slices.Clone(draftPlanV1.Phases), func(p plan.Phase, idx int) plan.Phase {
							if idx == len(draftPlanV1.Phases)-1 {
								p.Duration = &SixMonthPeriod
							}

							return p
						})

						updatedPhases = append(updatedPhases, plan.Phase{
							PhaseManagedFields: plan.PhaseManagedFields{
								NamespacedID: models.NamespacedID{
									Namespace: namespace,
								},
								PlanID: draftPlanV1.ID,
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
								ID:        draftPlanV1.ID,
							},
							Phases: lo.ToPtr(lo.Map(updatedPhases, func(p plan.Phase, _ int) productcatalog.Phase {
								return productcatalog.Phase{
									PhaseMeta: p.PhaseMeta,
									RateCards: p.RateCards,
								}
							})),
						}

						updateInput.IgnoreNonCriticalIssues = true

						updatedPlanV1, err = env.Plan.UpdatePlan(ctx, updateInput)
						require.NoErrorf(t, err, "updating draft Plan must not fail")
						require.NotNil(t, updatedPlanV1, "updated draft Plan must not be empty")

						plan.AssertPlanPhasesEqual(t, updatedPhases, updatedPlanV1.Phases)
					})
				})

				t.Run("Delete", func(t *testing.T) {
					updateInput = plan.UpdatePlanInput{
						NamespacedID: models.NamespacedID{
							Namespace: planInput.Namespace,
							ID:        draftPlanV1.ID,
						},
						Phases: lo.ToPtr(lo.Map(draftPlanV1.Phases, func(p plan.Phase, _ int) productcatalog.Phase {
							return productcatalog.Phase{
								PhaseMeta: p.PhaseMeta,
								RateCards: p.RateCards,
							}
						})),
					}

					updatedPlanV1, err = env.Plan.UpdatePlan(ctx, updateInput)
					require.NoErrorf(t, err, "updating draft Plan must not fail")
					require.NotNil(t, updatedPlanV1, "updated draft Plan must not be empty")

					plan.AssertPlanEqual(t, *updatedPlanV1, *draftPlanV1)
				})
			})

			var publishedPlanV1 *plan.Plan

			t.Run("Publish", func(t *testing.T) {
				publishAt := time.Now().Truncate(time.Microsecond)

				publishInput := plan.PublishPlanInput{
					NamespacedID: draftPlanV1.NamespacedID,
					EffectivePeriod: productcatalog.EffectivePeriod{
						EffectiveFrom: &publishAt,
						EffectiveTo:   nil,
					},
				}

				publishedPlanV1, err = env.Plan.PublishPlan(ctx, publishInput)
				require.NoErrorf(t, err, "publishing draft Plan must not fail")
				require.NotNil(t, publishedPlanV1, "published Plan must not be empty")

				require.NotNil(t, publishedPlanV1.EffectiveFrom, "EffectiveFrom for published Plan must not be empty")

				assert.Equalf(t, publishAt, *publishedPlanV1.EffectiveFrom,
					"EffectiveFrom for published Plan mismatch: expected=%s, actual=%s", publishAt, *publishedPlanV1.EffectiveFrom)

				assert.Equalf(t, productcatalog.PlanStatusActive, publishedPlanV1.Status(),
					"Plan Status mismatch: expected=%s, actual=%s", productcatalog.PlanStatusActive, publishedPlanV1.Status())

				t.Run("Update", func(t *testing.T) {
					updateInput := plan.UpdatePlanInput{
						NamespacedID: models.NamespacedID{
							Namespace: draftPlanV1.Namespace,
							ID:        draftPlanV1.ID,
						},
						Name: lo.ToPtr("Invalid Update"),
					}

					_, err = env.Plan.UpdatePlan(ctx, updateInput)
					require.Errorf(t, err, "updating active Plan must fail")
				})
			})

			var (
				planV2          *plan.Plan
				publishedPlanV2 *plan.Plan
			)

			t.Run("V2", func(t *testing.T) {
				planV2, err = env.Plan.CreatePlan(ctx, planInput)
				require.NoErrorf(t, err, "creating a new draft Plan from active must not fail")
				require.NotNil(t, planV2, "new draft Plan must not be empty")

				assert.Equalf(t, publishedPlanV1.Version+1, planV2.Version,
					"new draft Plan must have higher version number")

				assert.Equalf(t, productcatalog.PlanStatusDraft, planV2.Status(),
					"Plan Status mismatch: expected=%s, actual=%s", productcatalog.PlanStatusDraft, planV2.Status())

				t.Run("Should not allow publishing draft plan with alignment issues", func(t *testing.T) {
					// Let's update the plan to have a misaligned phase
					oldPhases := lo.Map(planV2.Phases, func(p plan.Phase, idx int) productcatalog.Phase {
						return productcatalog.Phase{
							PhaseMeta: p.PhaseMeta,
							RateCards: p.RateCards,
						}
					})

					updateInput := plan.UpdatePlanInput{
						NamespacedID: planV2.NamespacedID,
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
									BillingCadence: lo.ToPtr(datetime.MustParseDuration(t, "P1W")),
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
									BillingCadence: lo.ToPtr(datetime.MustParseDuration(t, "P1M")),
								},
							},
						})),
					}

					updateInput.IgnoreNonCriticalIssues = true

					_, err = env.Plan.UpdatePlan(ctx, updateInput)
					require.NoError(t, err)

					// Get the updated plan
					_, err = env.Plan.GetPlan(ctx, plan.GetPlanInput{
						NamespacedID: planV2.NamespacedID,
					})
					require.NoError(t, err)

					// Let's try to publish the plan
					publishAt := time.Now().Truncate(time.Microsecond)

					publishInput := plan.PublishPlanInput{
						NamespacedID: planV2.NamespacedID,
						EffectivePeriod: productcatalog.EffectivePeriod{
							EffectiveFrom: &publishAt,
							EffectiveTo:   nil,
						},
					}

					_, err = env.Plan.PublishPlan(ctx, publishInput)
					require.Error(t, err, "publishing draft Plan with alignment issues must fail")

					// Let's update the plan to fix the alignment issue by removing the last phase
					_, err = env.Plan.UpdatePlan(ctx, plan.UpdatePlanInput{
						NamespacedID: planV2.NamespacedID,
						Phases:       lo.ToPtr(oldPhases),
					})
					require.NoError(t, err)
				})

				t.Run("Publish", func(t *testing.T) {
					publishAt := time.Now().Truncate(time.Microsecond)

					publishPlanV2Input := plan.PublishPlanInput{
						NamespacedID: planV2.NamespacedID,
						EffectivePeriod: productcatalog.EffectivePeriod{
							EffectiveFrom: &publishAt,
							EffectiveTo:   nil,
						},
					}

					publishedPlanV2, err = env.Plan.PublishPlan(ctx, publishPlanV2Input)
					require.NoErrorf(t, err, "publishing draft Plan must not fail")
					require.NotNil(t, publishedPlanV2, "published Plan must not be empty")
					require.NotNil(t, publishedPlanV2.EffectiveFrom, "EffectiveFrom for published Plan must not be empty")

					assert.Equalf(t, publishAt, *publishedPlanV2.EffectiveFrom,
						"EffectiveFrom for published Plan mismatch: expected=%s, actual=%s", publishAt, *publishedPlanV1.EffectiveFrom)

					assert.Equalf(t, productcatalog.PlanStatusActive, publishedPlanV2.Status(),
						"Plan Status mismatch: expected=%s, actual=%s", productcatalog.PlanStatusActive, publishedPlanV2.Status())

					prevPlanV1, err := env.Plan.GetPlan(ctx, plan.GetPlanInput{
						NamespacedID: publishedPlanV1.NamespacedID,
					})
					require.NoErrorf(t, err, "getting previous Plan version must not fail")
					require.NotNil(t, prevPlanV1, "previous Plan version must not be empty")

					assert.Equalf(t, productcatalog.PlanStatusArchived, prevPlanV1.Status(),
						"Plan Status mismatch: expected=%s, actual=%s", productcatalog.PlanStatusArchived, prevPlanV1.Status())
				})
			})

			var planV3 *plan.Plan

			t.Run("V3", func(t *testing.T) {
				planV3, err = env.Plan.CreatePlan(ctx, planInput)
				require.NoErrorf(t, err, "creating a new draft Plan from active must not fail")
				require.NotNil(t, planV3, "new draft Plan must not be empty")

				assert.Equalf(t, publishedPlanV2.Version+1, planV3.Version, "new draft Plan must have higher version number")

				assert.Equalf(t, productcatalog.PlanStatusDraft, planV3.Status(),
					"Plan Status mismatch: expected=%s, actual=%s", productcatalog.PlanStatusDraft, planV3.Status())

				t.Run("Addon", func(t *testing.T) {
					var planAddonV3 *planaddon.PlanAddon

					t.Run("Assign", func(t *testing.T) {
						planAddonV3, err = env.PlanAddon.CreatePlanAddon(ctx, planaddon.CreatePlanAddonInput{
							NamespacedModel: models.NamespacedModel{
								Namespace: namespace,
							},
							PlanID:        planV3.ID,
							AddonID:       addonV1.ID,
							FromPlanPhase: planV3.Phases[1].Key,
						})

						require.NoErrorf(t, err, "creating a new PlanAddon from active must not fail")
						require.NotNil(t, planAddonV3, "new PlanAddon must not be empty")

						assert.Equalf(t, addonV1.ID, planAddonV3.Addon.ID, "Addon ID mismatch: expected=%s, actual=%s", addonV1.ID, planAddonV3.Addon.ID)
					})

					t.Run("Publish", func(t *testing.T) {
						publishAt := time.Now().Truncate(time.Microsecond)

						publishPlanV3Input := plan.PublishPlanInput{
							NamespacedID: planV3.NamespacedID,
							EffectivePeriod: productcatalog.EffectivePeriod{
								EffectiveFrom: &publishAt,
								EffectiveTo:   nil,
							},
						}

						publishedPlanV3, err := env.Plan.PublishPlan(ctx, publishPlanV3Input)
						require.NoErrorf(t, err, "publishing draft Plan must not fail")
						require.NotNil(t, publishedPlanV3, "published Plan must not be empty")
						require.NotNil(t, publishedPlanV3.EffectiveFrom, "EffectiveFrom for published Plan must not be empty")

						assert.Equalf(t, publishAt, *publishedPlanV3.EffectiveFrom,
							"EffectiveFrom for published Plan mismatch: expected=%s, actual=%s", publishAt, *publishedPlanV1.EffectiveFrom)

						assert.Equalf(t, productcatalog.PlanStatusActive, publishedPlanV3.Status(),
							"Plan Status mismatch: expected=%s, actual=%s", productcatalog.PlanStatusActive, publishedPlanV3.Status())

						prevPlanV2, err := env.Plan.GetPlan(ctx, plan.GetPlanInput{
							NamespacedID: publishedPlanV2.NamespacedID,
						})
						require.NoErrorf(t, err, "getting previous Plan version must not fail")
						require.NotNil(t, prevPlanV2, "previous Plan version must not be empty")

						assert.Equalf(t, productcatalog.PlanStatusArchived, prevPlanV2.Status(),
							"Plan Status mismatch: expected=%s, actual=%s", productcatalog.PlanStatusArchived, prevPlanV2.Status())

						t.Run("Archive", func(t *testing.T) {
							archiveAt := time.Now().Truncate(time.Microsecond)

							archivePlanV3Input := plan.ArchivePlanInput{
								NamespacedID: planV3.NamespacedID,
								EffectiveTo:  archiveAt,
							}

							var archivedPlanV3 *plan.Plan

							archivedPlanV3, err = env.Plan.ArchivePlan(ctx, archivePlanV3Input)
							require.NoErrorf(t, err, "archiving Plan must not fail")
							require.NotNil(t, archivedPlanV3, "archived Plan must not be empty")
							require.NotNil(t, archivedPlanV3.EffectiveTo, "EffectiveFrom for archived Plan must not be empty")

							assert.Equalf(t, archiveAt, *archivedPlanV3.EffectiveTo,
								"EffectiveTo for published Plan mismatch: expected=%s, actual=%s", archiveAt, *archivedPlanV3.EffectiveTo)

							assert.Equalf(t, productcatalog.PlanStatusArchived, archivedPlanV3.Status(),
								"Status mismatch for archived Plan: expected=%s, actual=%s", productcatalog.PlanStatusArchived, archivedPlanV3.Status())
						})
					})
				})

				t.Run("Delete", func(t *testing.T) {
					deleteInput := plan.DeletePlanInput{
						NamespacedID: planV3.NamespacedID,
					}

					err = env.Plan.DeletePlan(ctx, deleteInput)
					require.NoErrorf(t, err, "deleting Plan must not fail")

					deletedPlanV3, err := env.Plan.GetPlan(ctx, plan.GetPlanInput{
						NamespacedID: planV3.NamespacedID,
					})
					require.NoErrorf(t, err, "getting deleted Plan version must not fail")
					require.NotNil(t, deletedPlanV3, "deleted Plan version must not be empty")

					assert.NotNilf(t, deletedPlanV3.DeletedAt, "deletedAt must not be empty")
				})
			})
		})
	})
}
