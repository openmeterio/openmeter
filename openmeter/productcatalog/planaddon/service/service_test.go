package service

import (
	"context"
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
	planaddonadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon/adapter"
	planaddontestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon/testutils"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestPlanAddonService(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup test environment
	env := planaddontestutils.NewTestEnv(t)
	defer env.Close(t)

	// Run database migrations
	env.DBSchemaMigrate(t)

	// Get new namespace ID
	namespace := planaddontestutils.NewTestNamespace(t)

	// Setup meter repository
	err := env.Meter.ReplaceMeters(ctx, planaddontestutils.NewTestMeters(t, namespace))
	require.NoError(t, err, "replacing meters must not fail")

	result, err := env.Meter.ListMeters(ctx, meter.ListMetersParams{
		Namespace: namespace,
		Page: pagination.Page{
			PageSize:   1000,
			PageNumber: 1,
		},
	})
	require.NoErrorf(t, err, "listing meters must not fail")

	meters := result.Items
	require.NotEmptyf(t, meters, "list of Meters must not be empty")

	// Set Feature for each Meter
	features := make([]feature.Feature, 0, len(meters))
	for _, m := range meters {
		input := planaddontestutils.NewTestFeatureFromMeter(t, &m)

		feat, err := env.Feature.CreateFeature(ctx, input)
		require.NoErrorf(t, err, "creating feature must not fail")
		require.NotNil(t, feat, "feature must not be empty")

		features = append(features, feat)
	}

	planAddonAdapter, err := planaddonadapter.New(planaddonadapter.Config{
		Client: env.Client,
		Logger: env.Logger,
	})
	require.NoError(t, err, "creating plan add-on adapter must not fail")

	planAddonService, err := New(Config{
		Adapter:   planAddonAdapter,
		Plan:      env.Plan,
		Addon:     env.Addon,
		Logger:    env.Logger,
		Publisher: env.Publisher,
	})
	require.NoError(t, err, "creating plan add-on service must not fail")

	planV1Input := planaddontestutils.NewTestPlan(t, namespace)

	addonV1Input := planaddontestutils.NewTestAddon(t, namespace)

	t.Run("Addon", func(t *testing.T) {
		var (
			feature1  feature.Feature
			planV1    *plan.Plan
			addonV1   *addon.Addon
			planAddon *planaddon.PlanAddon
		)

		feature1 = features[0]

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

		planV1, err = env.Plan.CreatePlan(ctx, planV1Input)
		require.NoErrorf(t, err, "creating plan must not fail")

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

		addonV1, err = env.Addon.CreateAddon(ctx, addonV1Input)
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
			planAddon, err = planAddonService.CreatePlanAddon(ctx, planAddonInput)
			require.NoErrorf(t, err, "creating new plan add-on assignment must not fail")

			require.NotNilf(t, planAddon, "plan add-on assignment must not be nil")

			planaddon.AssertPlanAddonCreateInputEqual(t, planAddonInput, *planAddon)

			t.Run("Get", func(t *testing.T) {
				t.Run("ById", func(t *testing.T) {
					getPlanAddon, err := planAddonService.GetPlanAddon(ctx, planaddon.GetPlanAddonInput{
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
					getPlanAddon, err := planAddonService.GetPlanAddon(ctx, planaddon.GetPlanAddonInput{
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
					listPlanAddons, err := planAddonService.ListPlanAddons(ctx, planaddon.ListPlanAddonsInput{
						Namespaces: []string{namespace},
						IDs:        []string{planAddon.ID},
					})
					assert.NoErrorf(t, err, "listing plan add-on assignment by id must not fail")

					require.Lenf(t, listPlanAddons.Items, 1, "plan add-on assignments must not be empty")

					planaddon.AssertPlanAddonEqual(t, *planAddon, listPlanAddons.Items[0])
				})

				t.Run("ByResourceKey", func(t *testing.T) {
					listPlanAddons, err := planAddonService.ListPlanAddons(ctx, planaddon.ListPlanAddonsInput{
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
				t.Run("ById", func(t *testing.T) {
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

					updatedPlanAddon, err := planAddonService.UpdatePlanAddon(ctx, planAddonV1Update)
					require.NoErrorf(t, err, "updating plan add-on assignment must not fail")

					require.NotNilf(t, updatedPlanAddon, "plan add-on assignment must not be nil")

					planaddon.AssertPlanAddonUpdateInputEqual(t, planAddonV1Update, *updatedPlanAddon)
				})

				t.Run("ByResourceKey", func(t *testing.T) {
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
						PlanID:        planAddon.Plan.ID,
						AddonID:       planAddon.Addon.ID,
						FromPlanPhase: &planV1.Phases[1].Key,
					}

					updatedPlanAddon, err := planAddonService.UpdatePlanAddon(ctx, planAddonV1Update)
					require.NoErrorf(t, err, "updating plan add-on assignment must not fail")

					require.NotNilf(t, updatedPlanAddon, "plan add-on assignment must not be nil")

					planaddon.AssertPlanAddonUpdateInputEqual(t, planAddonV1Update, *updatedPlanAddon)
				})
			})

			t.Run("Delete", func(t *testing.T) {
				err = planAddonService.DeletePlanAddon(ctx, planaddon.DeletePlanAddonInput{
					NamespacedModel: models.NamespacedModel{
						Namespace: namespace,
					},
					ID: planAddon.ID,
				})
				require.NoErrorf(t, err, "deleting plan add-on assignment must not fail")

				getPlanAddon, err := planAddonService.GetPlanAddon(ctx, planaddon.GetPlanAddonInput{
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

		t.Run("Invalid", func(t *testing.T) {
			t.Run("Plan", func(t *testing.T) {
				planV1, err = env.Plan.PublishPlan(ctx, plan.PublishPlanInput{
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

				planAddon, err = planAddonService.CreatePlanAddon(ctx, planAddonInput)
				require.Errorf(t, err, "creating new plan add-on assignment must fail if plan is published/active")
			})
		})
	})
}

var MonthPeriod = isodate.FromDuration(30 * 24 * time.Hour)
