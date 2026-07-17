package service_test

import (
	"context"
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	pctestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var MonthPeriod = datetime.NewISODuration(0, 1, 0, 0, 0, 0, 0)

func TestPlanAddonService(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := pctestutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	t.Run("Addon", func(t *testing.T) {
		t.Run("Create", func(t *testing.T) {
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

			planV1Input := pctestutils.NewTestPlan(t, namespace, pctestutils.WithPlanPhases([]productcatalog.Phase{
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
								Key:                 features[0].Key,
								Name:                features[0].Name,
								Description:         lo.ToPtr("invalid RateCard 1"),
								Metadata:            models.Metadata{"name": features[0].Name},
								FeatureKey:          lo.ToPtr(features[0].Key),
								FeatureID:           lo.ToPtr(features[0].ID),
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
								Key:                 features[0].Key,
								Name:                features[0].Name,
								Description:         lo.ToPtr("Trial RateCard 1"),
								Metadata:            models.Metadata{"name": features[0].Name},
								FeatureKey:          lo.ToPtr(features[0].Key),
								FeatureID:           lo.ToPtr(features[0].ID),
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
								Key:                 features[0].Key,
								Name:                features[0].Name,
								Description:         lo.ToPtr("Pro RateCard 1"),
								Metadata:            models.Metadata{"name": features[0].Name},
								FeatureKey:          lo.ToPtr(features[0].Key),
								FeatureID:           lo.ToPtr(features[0].ID),
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
			}...))

			var planV1 *plan.Plan

			planV1, err = env.Plan.CreatePlan(ctx, planV1Input)
			require.NoErrorf(t, err, "creating plan must not fail")

			addonV1Input := pctestutils.NewTestAddon(t, namespace, productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:                 features[0].Key,
						Name:                features[0].Name,
						Description:         lo.ToPtr(features[0].Name),
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
			}...)

			var addonV1 *addon.Addon

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

			var planAddon *planaddon.PlanAddon

			planAddon, err = env.PlanAddonRepository.CreatePlanAddon(ctx, planAddonInput)
			require.NoErrorf(t, err, "creating new plan add-on assignment must not fail")

			require.NotNilf(t, planAddon, "plan add-on assignment must not be nil")

			planaddon.AssertPlanAddonCreateInputEqual(t, planAddonInput, *planAddon)

			t.Run("Get", func(t *testing.T) {
				t.Run("ById", func(t *testing.T) {
					getPlanAddon, err := env.PlanAddon.GetPlanAddon(ctx, planaddon.GetPlanAddonInput{
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
					getPlanAddon, err := env.PlanAddon.GetPlanAddon(ctx, planaddon.GetPlanAddonInput{
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
					listPlanAddons, err := env.PlanAddon.ListPlanAddons(ctx, planaddon.ListPlanAddonsInput{
						Namespaces: []string{namespace},
						IDs:        []string{planAddon.ID},
					})
					assert.NoErrorf(t, err, "listing plan add-on assignment by id must not fail")

					require.Lenf(t, listPlanAddons.Items, 1, "plan add-on assignments must not be empty")

					planaddon.AssertPlanAddonEqual(t, *planAddon, listPlanAddons.Items[0])
				})

				t.Run("ByResourceKey", func(t *testing.T) {
					listPlanAddons, err := env.PlanAddon.ListPlanAddons(ctx, planaddon.ListPlanAddonsInput{
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

					updatedPlanAddon, err := env.PlanAddon.UpdatePlanAddon(ctx, planAddonV1Update)
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

					updatedPlanAddon, err := env.PlanAddon.UpdatePlanAddon(ctx, planAddonV1Update)
					require.NoErrorf(t, err, "updating plan add-on assignment must not fail")

					require.NotNilf(t, updatedPlanAddon, "plan add-on assignment must not be nil")

					planaddon.AssertPlanAddonUpdateInputEqual(t, planAddonV1Update, *updatedPlanAddon)
				})
			})

			t.Run("Delete", func(t *testing.T) {
				err = env.PlanAddon.DeletePlanAddon(ctx, planaddon.DeletePlanAddonInput{
					NamespacedModel: models.NamespacedModel{
						Namespace: namespace,
					},
					ID: planAddon.ID,
				})
				require.NoErrorf(t, err, "deleting plan add-on assignment must not fail")

				getPlanAddon, err := env.PlanAddon.GetPlanAddon(ctx, planaddon.GetPlanAddonInput{
					NamespacedModel: models.NamespacedModel{
						Namespace: namespace,
					},
					ID: planAddon.ID,
				})
				require.NoErrorf(t, err, "getting plan add-on assignment by id must not fail")

				require.NotNilf(t, getPlanAddon, "plan add-on assignment must not be nil")

				assert.NotNilf(t, getPlanAddon.DeletedAt, "plan add-on assignment must be deleted")
			})

			t.Run("Invalid", func(t *testing.T) {
				t.Run("PublishedPlan", func(t *testing.T) {
					_, err = env.Plan.PublishPlan(ctx, plan.PublishPlanInput{
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

					planAddon, err = env.PlanAddon.CreatePlanAddon(ctx, planAddonInput)
					require.Errorf(t, err, "creating new plan add-on assignment must fail if plan is published/active")
				})
			})
		})
	})
}

func TestPlanAddonCustomCurrencyIntegration(t *testing.T) {
	// given:
	// - a USD plan whose priced rate card uses a managed custom currency
	// - an active USD add-on whose rate card overrides to that custom currency
	// when:
	// - the add-on is assigned through the plan add-on service
	// then:
	// - managed lookup, cost-basis lookup, effective-currency compatibility, and DB mapping all preserve the custom currency
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	env := pctestutils.NewTestEnv(t)
	t.Cleanup(func() { env.Close(t) })

	namespace := pctestutils.NewTestNamespace(t)
	managedCurrency, err := env.Currency.CreateCurrency(t.Context(), currencies.CreateCurrencyInput{
		Namespace: namespace,
		Code:      "CREDITS",
		Name:      "Credits",
		Symbol:    "cr",
	})
	require.NoError(t, err)

	_, err = env.Currency.CreateCostBasis(t.Context(), currencies.CreateCostBasisInput{
		Namespace:  namespace,
		CurrencyID: managedCurrency.ID,
		FiatCode:   currency.USD.String(),
		Rate:       decimal.NewFromInt(1),
	})
	require.NoError(t, err)

	customCurrency := currencyx.Code(managedCurrency.Code)
	month := datetime.MustParseDuration(t, "P1M")
	price := func() *productcatalog.Price {
		return productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      decimal.NewFromInt(25),
			PaymentTerm: productcatalog.InAdvancePaymentTerm,
		})
	}

	planInput := pctestutils.NewTestPlan(t, namespace, pctestutils.WithPlanPhases(productcatalog.Phase{
		PhaseMeta: productcatalog.PhaseMeta{Key: "default", Name: "Default"},
		RateCards: productcatalog.RateCards{
			&productcatalog.FlatFeeRateCard{
				RateCardMeta: productcatalog.RateCardMeta{
					Key:      "credits",
					Name:     "Credits",
					Currency: customCurrency,
					Price:    price(),
				},
				BillingCadence: &month,
			},
		},
	}))
	createdPlan, err := env.Plan.CreatePlan(t.Context(), planInput)
	require.NoError(t, err)

	addonInput := pctestutils.NewTestAddon(t, namespace, &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:      "credits",
			Name:     "Credits",
			Currency: customCurrency,
			Price:    price(),
		},
		BillingCadence: &month,
	})
	addonInput.Key = "custom-currency-addon"

	createdAddon, err := env.Addon.CreateAddon(t.Context(), addonInput)
	require.NoError(t, err)
	createdAddon, err = env.Addon.PublishAddon(t.Context(), addon.PublishAddonInput{
		NamespacedID: createdAddon.NamespacedID,
		EffectivePeriod: productcatalog.EffectivePeriod{
			EffectiveFrom: lo.ToPtr(now.Add(-time.Second)),
		},
	})
	require.NoError(t, err)

	assigned, err := env.PlanAddon.CreatePlanAddon(t.Context(), planaddon.CreatePlanAddonInput{
		NamespacedModel: models.NamespacedModel{Namespace: namespace},
		PlanID:          createdPlan.ID,
		AddonID:         createdAddon.ID,
		FromPlanPhase:   "default",
	})
	require.NoError(t, err)
	require.Equal(t, currencyx.Code(currency.USD), assigned.Addon.Currency.GetCode())
	require.Len(t, assigned.Addon.RateCards, 1)
	require.Equal(t, customCurrency, assigned.Addon.RateCards[0].AsMeta().Currency.GetCode())
	require.Equal(t, customCurrency, assigned.Addon.RateCards[0].AsMeta().EffectiveCurrency(assigned.Addon.Currency).GetCode())

	t.Run("missing plan fiat cost basis", func(t *testing.T) {
		// given:
		// - another managed custom currency without a USD cost basis
		// when:
		// - an add-on introduces a newly priced rate card in that currency
		// then:
		// - assignment fails before the plan add-on row is written
		currencyWithoutBasis, err := env.Currency.CreateCurrency(t.Context(), currencies.CreateCurrencyInput{
			Namespace: namespace,
			Code:      "POINTS",
			Name:      "Points",
			Symbol:    "pt",
		})
		require.NoError(t, err)

		points := currencyx.Code(currencyWithoutBasis.Code)
		input := pctestutils.NewTestAddon(t, namespace, &productcatalog.FlatFeeRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:   "points",
				Name:  "Points",
				Price: price(),
			},
			BillingCadence: &month,
		})
		input.Key = "missing-cost-basis-addon"
		input.Currency = points

		created, err := env.Addon.CreateAddon(t.Context(), input)
		require.NoError(t, err)
		created, err = env.Addon.PublishAddon(t.Context(), addon.PublishAddonInput{
			NamespacedID: created.NamespacedID,
			EffectivePeriod: productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now.Add(-time.Second)),
			},
		})
		require.NoError(t, err)

		_, err = env.PlanAddon.CreatePlanAddon(t.Context(), planaddon.CreatePlanAddonInput{
			NamespacedModel: models.NamespacedModel{Namespace: namespace},
			PlanID:          createdPlan.ID,
			AddonID:         created.ID,
			FromPlanPhase:   "default",
		})
		require.ErrorIs(t, err, productcatalog.ErrCurrencyCostBasisNotFound)
	})
}
