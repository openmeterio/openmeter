package adapter

import (
	"context"
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	MonthPeriod      = datex.FromDuration(30 * 24 * time.Hour)
	TwoMonthPeriod   = datex.FromDuration(60 * 24 * time.Hour)
	ThreeMonthPeriod = datex.FromDuration(90 * 24 * time.Hour)
)

var namespace = "01JBX0P4GQZCQY1WNGX3VT94P4"

var planV1Input = plan.CreatePlanInput{
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
					Metadata:    models.Metadata{"name": "trial"},
					StartAfter:  MonthPeriod,
				},
				Discounts: nil,
				RateCards: []productcatalog.RateCard{
					&plan.FlatFeeRateCard{
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
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
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
					StartAfter:  TwoMonthPeriod,
				},
				Discounts: nil,
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
												Amount: decimal.NewFromInt(5),
											},
											UnitPrice: &productcatalog.PriceTierUnitPrice{
												Amount: decimal.NewFromInt(25),
											},
										},
									},
									MinimumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
									MaximumAmount: nil,
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

func TestPostgresAdapter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pg := testutils.InitPostgresDB(t)
	defer func() {
		if err := pg.EntDriver.Close(); err != nil {
			t.Errorf("failed to close ent driver: %v", err)
		}

		if err := pg.PGDriver.Close(); err != nil {
			t.Errorf("failed to postgres driver: %v", err)
		}
	}()

	err := pg.EntDriver.Client().Schema.Create(context.Background())
	require.NoErrorf(t, err, "schema migration must not fail")

	entClient := pg.EntDriver.Client()
	defer func() {
		if err = entClient.Close(); err != nil {
			t.Errorf("failed to close ent client: %v", err)
		}
	}()

	repo, err := New(Config{
		Client: pg.EntDriver.Client(),
		Logger: testutils.NewDiscardLogger(t),
	})
	require.NoError(t, err)

	t.Run("Plan", func(t *testing.T) {
		var planV1 *plan.Plan

		t.Run("Create", func(t *testing.T) {
			planV1, err = repo.CreatePlan(ctx, planV1Input)
			require.NoErrorf(t, err, "creating new plan must not fail")

			require.NotNilf(t, planV1, "plan must not be nil")

			plan.AssertPlanCreateInputEqual(t, planV1Input, *planV1)
		})

		t.Run("Get", func(t *testing.T) {
			t.Run("ById", func(t *testing.T) {
				getPlanV1, err := repo.GetPlan(ctx, plan.GetPlanInput{
					NamespacedID: models.NamespacedID{
						Namespace: namespace,
						ID:        planV1.ID,
					},
				})
				assert.NoErrorf(t, err, "getting plan by id must not fail")

				require.NotNilf(t, planV1, "plan must not be nil")

				plan.AssertPlanEqual(t, *planV1, *getPlanV1)
			})

			t.Run("ByKey", func(t *testing.T) {
				getPlanV1, err := repo.GetPlan(ctx, plan.GetPlanInput{
					NamespacedID: models.NamespacedID{
						Namespace: namespace,
					},
					Key:           planV1Input.Key,
					IncludeLatest: true,
				})
				assert.NoErrorf(t, err, "getting plan by key must not fail")

				require.NotNilf(t, getPlanV1, "plan must not be nil")

				plan.AssertPlanEqual(t, *planV1, *getPlanV1)
			})

			t.Run("ByKeyVersion", func(t *testing.T) {
				getPlanV1, err := repo.GetPlan(ctx, plan.GetPlanInput{
					NamespacedID: models.NamespacedID{
						Namespace: namespace,
					},
					Key:     planV1Input.Key,
					Version: 1,
				})
				assert.NoErrorf(t, err, "getting plan by key and version must not fail")

				require.NotNilf(t, getPlanV1, "plan must not be nil")

				plan.AssertPlanEqual(t, *planV1, *getPlanV1)
			})
		})

		t.Run("List", func(t *testing.T) {
			t.Run("ById", func(t *testing.T) {
				listPlanV1, err := repo.ListPlans(ctx, plan.ListPlansInput{
					Namespaces: []string{namespace},
					IDs:        []string{planV1.ID},
				})
				assert.NoErrorf(t, err, "listing plan by id must not fail")

				require.Lenf(t, listPlanV1.Items, 1, "plans must not be empty")

				plan.AssertPlanEqual(t, *planV1, listPlanV1.Items[0])
			})

			t.Run("ByKey", func(t *testing.T) {
				listPlanV1, err := repo.ListPlans(ctx, plan.ListPlansInput{
					Namespaces: []string{namespace},
					Keys:       []string{planV1Input.Key},
				})
				assert.NoErrorf(t, err, "getting plan by key must not fail")

				require.Lenf(t, listPlanV1.Items, 1, "plans must not be empty")

				plan.AssertPlanEqual(t, *planV1, listPlanV1.Items[0])
			})

			t.Run("ByKeyVersion", func(t *testing.T) {
				listPlanV1, err := repo.ListPlans(ctx, plan.ListPlansInput{
					Namespaces:  []string{namespace},
					KeyVersions: map[string][]int{planV1Input.Key: {1}},
				})
				assert.NoErrorf(t, err, "getting plan by key and version must not fail")

				require.Lenf(t, listPlanV1.Items, 1, "plans must not be empty")

				plan.AssertPlanEqual(t, *planV1, listPlanV1.Items[0])
			})
		})

		t.Run("Update", func(t *testing.T) {
			now := time.Now()

			planV1Update := plan.UpdatePlanInput{
				NamespacedID: models.NamespacedID{
					Namespace: namespace,
					ID:        planV1.ID,
				},
				EffectivePeriod: productcatalog.EffectivePeriod{
					EffectiveFrom: lo.ToPtr(now.UTC()),
					EffectiveTo:   lo.ToPtr(now.Add(30 * 24 * time.Hour).UTC()),
				},
				Name:        lo.ToPtr("Pro Published"),
				Description: lo.ToPtr("Pro Published"),
				Metadata: &models.Metadata{
					"name":        "Pro Published",
					"description": "Pro Published",
				},
				Phases: nil,
			}

			planV1, err = repo.UpdatePlan(ctx, planV1Update)
			require.NoErrorf(t, err, "updating plan must not fail")

			require.NotNilf(t, planV1, "plan must not be nil")

			plan.AssertPlanUpdateInputEqual(t, planV1Update, *planV1)
		})

		t.Run("Delete", func(t *testing.T) {
			err = repo.DeletePlan(ctx, plan.DeletePlanInput{
				NamespacedID: models.NamespacedID{
					Namespace: planV1.Namespace,
					ID:        planV1.ID,
				},
			})
			require.NoErrorf(t, err, "deleting plan must not fail")

			getPlanV1, err := repo.GetPlan(ctx, plan.GetPlanInput{
				NamespacedID: models.NamespacedID{
					Namespace: namespace,
					ID:        planV1.ID,
				},
			})
			require.NoErrorf(t, err, "getting plan by id must not fail")

			require.NotNilf(t, getPlanV1, "plan must not be nil")

			plan.AssertPlanEqual(t, *planV1, *getPlanV1)
		})
	})

	t.Run("Phase", func(t *testing.T) {
		planV1, err := repo.CreatePlan(ctx, planV1Input)
		require.NoErrorf(t, err, "creating new plan must not fail")

		require.NotNilf(t, planV1, "plan must not be nil")

		plan.AssertPlanCreateInputEqual(t, planV1Input, *planV1)

		var phase *plan.Phase

		t.Run("Create", func(t *testing.T) {
			phaseInput := plan.CreatePhaseInput{
				NamespacedModel: models.NamespacedModel{
					Namespace: namespace,
				},
				PlanID: planV1.ID,
				Phase: productcatalog.Phase{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:         "team",
						Name:        "Team",
						Description: lo.ToPtr("Team"),
						Metadata:    models.Metadata{"name": "team"},
						StartAfter:  ThreeMonthPeriod,
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
									Key:                 "team-ratecard-1",
									Name:                "Team RateCard 1",
									Description:         lo.ToPtr("Team RateCard 1"),
									Metadata:            models.Metadata{"name": "team-ratecard-1"},
									Feature:             nil,
									EntitlementTemplate: nil,
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
										MinimumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
										MaximumAmount: nil,
									}),
								},
								BillingCadence: MonthPeriod,
							},
						},
					},
				},
			}

			phase, err = repo.CreatePhase(ctx, phaseInput)
			assert.NoErrorf(t, err, "creating phase must not fail")

			require.NotNilf(t, phase, "plan phase must not be nil")

			plan.AssertPhaseCreateInputEqual(t, phaseInput, *phase)
		})

		t.Run("Get", func(t *testing.T) {
			t.Run("ById", func(t *testing.T) {
				getPhase, err := repo.GetPhase(ctx, plan.GetPhaseInput{
					NamespacedID: models.NamespacedID{
						Namespace: namespace,
						ID:        phase.ID,
					},
				})
				require.NoErrorf(t, err, "getting Phase by id must not fail")

				require.NotNilf(t, getPhase, "Phase must not be nil")

				plan.AssertPlanPhaseEqual(t, *getPhase, *phase)
			})

			t.Run("ByKey", func(t *testing.T) {
				getPhase, err := repo.GetPhase(ctx, plan.GetPhaseInput{
					NamespacedID: models.NamespacedID{
						Namespace: namespace,
					},
					Key:    phase.Key,
					PlanID: planV1.ID,
				})
				require.NoErrorf(t, err, "getting Phase by id must not fail")

				require.NotNilf(t, getPhase, "Phase must not be nil")

				plan.AssertPlanPhaseEqual(t, *getPhase, *phase)
			})
		})

		t.Run("Update", func(t *testing.T) {
			phaseUpdate := plan.UpdatePhaseInput{
				NamespacedID: models.NamespacedID{
					Namespace: namespace,
					ID:        phase.ID,
				},
				Key:        phase.Key,
				PlanID:     planV1.ID,
				StartAfter: lo.ToPtr(ThreeMonthPeriod),
			}

			updatePhase, err := repo.UpdatePhase(ctx, phaseUpdate)
			require.NoErrorf(t, err, "updating phase must not fail")

			require.NotNilf(t, updatePhase, "phase must not be nil")

			plan.AssertPhaseUpdateInputEqual(t, phaseUpdate, *updatePhase)
		})

		t.Run("Delete", func(t *testing.T) {
			err = repo.DeletePhase(ctx, plan.DeletePhaseInput{
				NamespacedID: models.NamespacedID{
					Namespace: namespace,
					ID:        phase.ID,
				},
				Key:    phase.Key,
				PlanID: planV1.ID,
			})
			require.NoErrorf(t, err, "deleting Phase must not fail")

			getPhase, err := repo.GetPhase(ctx, plan.GetPhaseInput{
				NamespacedID: models.NamespacedID{
					Namespace: namespace,
					ID:        phase.ID,
				},
				Key:    phase.Key,
				PlanID: planV1.ID,
			})
			require.NoErrorf(t, err, "getting Phase by id must not fail")

			require.NotNilf(t, getPhase, "Phase must not be nil")

			plan.AssertPlanPhaseEqual(t, *getPhase, *phase)
		})
	})
}
