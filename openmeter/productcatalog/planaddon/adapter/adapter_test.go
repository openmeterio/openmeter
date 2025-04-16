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
	featureadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	addonadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/addon/adapter"
	addonservice "github.com/openmeterio/openmeter/openmeter/productcatalog/addon/service"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	planadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/adapter"
	planservice "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/service"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	MonthPeriod = isodate.FromDuration(30 * 24 * time.Hour)
	namespace   = "01JBX0P4GQZCQY1WNGX3VT94P4"
)

var featureInput = feature.CreateFeatureInputs{
	Name:      "Feature 1",
	Key:       "feature1",
	Namespace: namespace,
}

var addonV1Input = addon.CreateAddonInput{
	NamespacedModel: models.NamespacedModel{
		Namespace: namespace,
	},
	Addon: productcatalog.Addon{
		AddonMeta: productcatalog.AddonMeta{
			Key:          "addon1",
			Name:         "Addon v1",
			Description:  lo.ToPtr("Addon v1"),
			Metadata:     models.Metadata{"name": "addon1"},
			Annotations:  models.Annotations{"key": "value"},
			Currency:     currency.USD,
			InstanceType: productcatalog.AddonInstanceTypeSingle,
		},
		RateCards: nil,
	},
}

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

	logger := testutils.NewDiscardLogger(t)

	publisher := eventbus.NewMock(t)

	featureSrv := feature.NewFeatureConnector(
		featureadapter.NewPostgresFeatureRepo(entClient, logger),
		nil,
		publisher,
	)

	planAdapter, err := planadapter.New(planadapter.Config{
		Client: entClient,
		Logger: logger,
	})
	require.NoErrorf(t, err, "creating plan repo must not fail")

	planSrv, err := planservice.New(planservice.Config{
		Adapter:   planAdapter,
		Feature:   featureSrv,
		Logger:    logger,
		Publisher: publisher,
	})

	addonAdapter, err := addonadapter.New(addonadapter.Config{
		Client: entClient,
		Logger: logger,
	})
	require.NoErrorf(t, err, "creating plan repo must not fail")

	addonSrv, err := addonservice.New(addonservice.Config{
		Adapter:   addonAdapter,
		Feature:   featureSrv,
		Logger:    logger,
		Publisher: publisher,
	})

	planAddonRepo := &adapter{
		db:     entClient,
		logger: logger,
	}

	t.Run("Addon", func(t *testing.T) {
		var (
			feature1  feature.Feature
			planV1    *plan.Plan
			addonV1   *addon.Addon
			planAddon *planaddon.PlanAddon
		)

		feature1, err = featureSrv.CreateFeature(ctx, featureInput)
		require.NoErrorf(t, err, "creating feature must not fail")

		planV1Input.Phases = []productcatalog.Phase{
			{
				PhaseMeta: productcatalog.PhaseMeta{
					Key:         "invalid",
					Name:        "Invalid",
					Description: lo.ToPtr("Invalid invalid"),
					Metadata:    models.Metadata{"name": "trial"},
					Duration:    &MonthPeriod,
				},
				RateCards: []productcatalog.RateCard{
					&productcatalog.FlatFeeRateCard{
						RateCardMeta: productcatalog.RateCardMeta{
							Key:                 feature1.Key,
							Name:                feature1.Name,
							Description:         lo.ToPtr("invalid RateCard 1"),
							Metadata:            models.Metadata{"name": feature1.Name},
							FeatureKey:          lo.ToPtr(feature1.Key),
							FeatureID:           lo.ToPtr(feature1.ID),
							EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.BooleanEntitlementTemplate{}),
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
			{
				PhaseMeta: productcatalog.PhaseMeta{
					Key:         "trial",
					Name:        "Trial",
					Description: lo.ToPtr("Trial phase"),
					Metadata:    models.Metadata{"name": "trial"},
					Duration:    &MonthPeriod,
				},
				RateCards: []productcatalog.RateCard{
					&productcatalog.FlatFeeRateCard{
						RateCardMeta: productcatalog.RateCardMeta{
							Key:                 feature1.Key,
							Name:                feature1.Name,
							Description:         lo.ToPtr("Trial RateCard 1"),
							Metadata:            models.Metadata{"name": feature1.Name},
							FeatureKey:          lo.ToPtr(feature1.Key),
							FeatureID:           lo.ToPtr(feature1.ID),
							EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.BooleanEntitlementTemplate{}),
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
								Commitments: productcatalog.Commitments{
									MinimumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
									MaximumAmount: nil,
								},
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
					Duration:    nil,
				},
				RateCards: []productcatalog.RateCard{
					&productcatalog.UsageBasedRateCard{
						RateCardMeta: productcatalog.RateCardMeta{
							Key:                 feature1.Key,
							Name:                feature1.Name,
							Description:         lo.ToPtr("Pro RateCard 1"),
							Metadata:            models.Metadata{"name": feature1.Name},
							FeatureKey:          lo.ToPtr(feature1.Key),
							FeatureID:           lo.ToPtr(feature1.ID),
							EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.BooleanEntitlementTemplate{}),
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
		}

		planV1, err = planSrv.CreatePlan(ctx, planV1Input)
		require.NoErrorf(t, err, "creating plan must not fail")

		planV1, err = planSrv.PublishPlan(ctx, plan.PublishPlanInput{
			NamespacedID: models.NamespacedID{
				Namespace: namespace,
				ID:        planV1.ID,
			},
			EffectivePeriod: productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(time.Now()),
				EffectiveTo:   nil,
			},
		})
		require.NoErrorf(t, err, "publishing plan must not fail")

		addonV1Input.RateCards = productcatalog.RateCards{
			&productcatalog.UsageBasedRateCard{
				RateCardMeta: productcatalog.RateCardMeta{
					Key:                 feature1.Key,
					Name:                feature1.Name,
					Description:         lo.ToPtr(feature1.Name),
					Metadata:            models.Metadata{"name": feature1.Name},
					FeatureKey:          lo.ToPtr(feature1.Key),
					FeatureID:           lo.ToPtr(feature1.ID),
					EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.BooleanEntitlementTemplate{}),
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
						Commitments: productcatalog.Commitments{
							MinimumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
							MaximumAmount: nil,
						},
					}),
				},
				BillingCadence: MonthPeriod,
			},
		}

		addonV1, err = addonSrv.CreateAddon(ctx, addonV1Input)
		require.NoErrorf(t, err, "creating add-on must not fail")

		addonV1, err = addonSrv.PublishAddon(ctx, addon.PublishAddonInput{
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

		planAddonInput := planaddon.CreatePlanAddonInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: namespace,
			},
			Annotations: map[string]interface{}{
				"openmeter.key": "openmeter.value",
			},
			PlanID:        planV1.ID,
			AddonID:       addonV1.ID,
			FromPlanPhase: planV1.Phases[1].Key,
		}

		t.Run("Create", func(t *testing.T) {
			planAddon, err = planAddonRepo.CreatePlanAddon(ctx, planAddonInput)
			require.NoErrorf(t, err, "creating new plan add-on assignment must not fail")

			require.NotNilf(t, planAddon, "plan add-on assignment must not be nil")

			planaddon.AssertPlanAddonCreateInputEqual(t, planAddonInput, *planAddon)

			t.Run("Get", func(t *testing.T) {
				t.Run("ById", func(t *testing.T) {
					getPlanAddon, err := planAddonRepo.GetPlanAddon(ctx, planaddon.GetPlanAddonInput{
						NamespacedModel: models.NamespacedModel{
							Namespace: namespace,
						},
						ID: planAddon.ID,
					})
					assert.NoErrorf(t, err, "getting plan add-on assignment by id must not fail")

					require.NotNilf(t, addonV1, "plan add-on assignment must not be nil")

					planaddon.AssertPlanAddonEqual(t, *planAddon, *getPlanAddon)
				})

				t.Run("ByKey", func(t *testing.T) {
					getPlanAddon, err := planAddonRepo.GetPlanAddon(ctx, planaddon.GetPlanAddonInput{
						NamespacedModel: models.NamespacedModel{
							Namespace: namespace,
						},
						PlanIDOrKey:  planAddon.Plan.Key,
						AddonIDOrKey: planAddon.Addon.Key,
					})
					assert.NoErrorf(t, err, "getting plan add-on assignment by plan and add-on key must not fail")

					require.NotNilf(t, addonV1, "plan add-on assignment must not be nil")

					planaddon.AssertPlanAddonEqual(t, *planAddon, *getPlanAddon)
				})
			})

			t.Run("List", func(t *testing.T) {
				t.Run("ById", func(t *testing.T) {
					listPlanAddons, err := planAddonRepo.ListPlanAddons(ctx, planaddon.ListPlanAddonsInput{
						Namespaces: []string{namespace},
						IDs:        []string{planAddon.ID},
					})
					assert.NoErrorf(t, err, "listing plan add-on assignment by id must not fail")

					require.Lenf(t, listPlanAddons.Items, 1, "plan add-on assignments must not be empty")

					planaddon.AssertPlanAddonEqual(t, *planAddon, listPlanAddons.Items[0])
				})

				t.Run("ByResourceKey", func(t *testing.T) {
					listPlanAddons, err := planAddonRepo.ListPlanAddons(ctx, planaddon.ListPlanAddonsInput{
						Namespaces: []string{namespace},
						PlanKeys:   []string{planV1.Key},
						AddonKeys:  []string{addonV1.Key},
					})
					assert.NoErrorf(t, err, "listing plan add-on assignment by plan and add-on keys must not fail")

					require.Lenf(t, listPlanAddons.Items, 1, "plan add-on assignments must not be empty")

					planaddon.AssertPlanAddonEqual(t, *planAddon, listPlanAddons.Items[0])
				})
			})

			t.Run("Update", func(t *testing.T) {
				planAddonV1Update := planaddon.UpdatePlanAddonInput{
					NamespacedModel: models.NamespacedModel{
						Namespace: namespace,
					},
					Annotations: &models.Annotations{
						"openmeter.key2": "openmeter.value2",
					},
					Metadata: &models.Metadata{
						"openmeter.key2": "openmeter.value2",
					},
					ID:            planAddon.ID,
					FromPlanPhase: &planV1.Phases[2].Key,
				}

				updatedPlanAddon, err := planAddonRepo.UpdatePlanAddon(ctx, planAddonV1Update)
				require.NoErrorf(t, err, "updating plan add-on assignment must not fail")

				require.NotNilf(t, updatedPlanAddon, "plan add-on assignment must not be nil")

				planaddon.AssertPlanAddonUpdateInputEqual(t, planAddonV1Update, *updatedPlanAddon)
			})

			t.Run("Delete", func(t *testing.T) {
				err = planAddonRepo.DeletePlanAddon(ctx, planaddon.DeletePlanAddonInput{
					NamespacedModel: models.NamespacedModel{
						Namespace: namespace,
					},
					ID: planAddon.ID,
				})
				require.NoErrorf(t, err, "deleting plan add-on assignment must not fail")

				getPlanAddon, err := planAddonRepo.GetPlanAddon(ctx, planaddon.GetPlanAddonInput{
					NamespacedModel: models.NamespacedModel{
						Namespace: namespace,
					},
					ID: planAddon.ID,
				})
				require.NoErrorf(t, err, "getting plan add-on assignment by id must not fail")

				require.NotNilf(t, getPlanAddon, "plan add-on assignment must not be nil")

				assert.NotNilf(t, getPlanAddon.DeletedAt, "plan add-on assignment must be deleted")
			})
		})
	})
}
