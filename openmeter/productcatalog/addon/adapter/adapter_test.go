package adapter_test

import (
	"context"
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	pctestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var MonthPeriod = datetime.ISODurationFromDuration(30 * 24 * time.Hour)

func TestPostgresAdapter(t *testing.T) {
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
						UnitConfig: &productcatalog.UnitConfig{
							Operation:        productcatalog.UnitConfigOperationDivide,
							ConversionFactor: decimal.NewFromInt(1000),
							Rounding:         productcatalog.UnitConfigRoundingModeCeiling,
							Precision:        0,
							DisplayUnit:      lo.ToPtr("K"),
						},
					},
					BillingCadence: MonthPeriod,
				},
			}...)

			var addonV1 *addon.Addon

			addonV1, err = env.AddonRepository.CreateAddon(ctx, addonV1Input)
			require.NoErrorf(t, err, "creating new add-on must not fail")

			require.NotNilf(t, addonV1, "add-on must not be nil")

			addon.AssertAddonCreateInputEqual(t, addonV1Input, *addonV1)

			t.Run("Get", func(t *testing.T) {
				t.Run("ById", func(t *testing.T) {
					getAddonV1, err := env.AddonRepository.GetAddon(ctx, addon.GetAddonInput{
						NamespacedID: models.NamespacedID{
							Namespace: namespace,
							ID:        addonV1.ID,
						},
					})
					assert.NoErrorf(t, err, "getting add-on by id must not fail")

					require.NotNilf(t, getAddonV1, "add-on must not be nil")

					addon.AssertAddonEqual(t, *addonV1, *getAddonV1)
				})

				t.Run("ByKey", func(t *testing.T) {
					getAddonV1, err := env.AddonRepository.GetAddon(ctx, addon.GetAddonInput{
						NamespacedID: models.NamespacedID{
							Namespace: namespace,
						},
						Key:           addonV1Input.Key,
						IncludeLatest: true,
					})
					assert.NoErrorf(t, err, "getting add-on by key must not fail")

					require.NotNilf(t, getAddonV1, "add-on must not be nil")

					addon.AssertAddonEqual(t, *addonV1, *getAddonV1)
				})

				t.Run("ByKeyVersion", func(t *testing.T) {
					getAddonV1, err := env.AddonRepository.GetAddon(ctx, addon.GetAddonInput{
						NamespacedID: models.NamespacedID{
							Namespace: namespace,
						},
						Key:     addonV1Input.Key,
						Version: 1,
					})
					assert.NoErrorf(t, err, "getting plan by key and version must not fail")

					require.NotNilf(t, getAddonV1, "plan must not be nil")

					addon.AssertAddonEqual(t, *addonV1, *getAddonV1)
				})
			})

			t.Run("List", func(t *testing.T) {
				t.Run("ByIdFilter", func(t *testing.T) {
					listAddonV1, err := env.AddonRepository.ListAddons(ctx, addon.ListAddonsInput{
						Namespaces: []string{namespace},
						ID: &filter.FilterULID{
							FilterString: filter.FilterString{
								Eq: &addonV1.ID,
							},
						},
					})
					assert.NoErrorf(t, err, "listing add-on by id filter must not fail")

					require.Lenf(t, listAddonV1.Items, 1, "add-ons must not be empty")

					addon.AssertAddonEqual(t, *addonV1, listAddonV1.Items[0])
				})

				t.Run("ByKeyFilter", func(t *testing.T) {
					listAddonV1, err := env.AddonRepository.ListAddons(ctx, addon.ListAddonsInput{
						Namespaces: []string{namespace},
						Key: &filter.FilterString{
							Eq: &addonV1Input.Key,
						},
					})
					assert.NoErrorf(t, err, "getting add-on by key filter must not fail")

					require.Lenf(t, listAddonV1.Items, 1, "add-ons must not be empty")

					addon.AssertAddonEqual(t, *addonV1, listAddonV1.Items[0])
				})

				t.Run("ByNameFilter", func(t *testing.T) {
					listAddonV1, err := env.AddonRepository.ListAddons(ctx, addon.ListAddonsInput{
						Namespaces: []string{namespace},
						Name: &filter.FilterString{
							Eq: &addonV1Input.Name,
						},
					})
					assert.NoErrorf(t, err, "getting add-on by name filter must not fail")

					require.Lenf(t, listAddonV1.Items, 1, "add-ons must not be empty")

					addon.AssertAddonEqual(t, *addonV1, listAddonV1.Items[0])
				})

				t.Run("ByCurrencyFilter", func(t *testing.T) {
					currencyStr := string(addonV1Input.Currency)
					listAddonV1, err := env.AddonRepository.ListAddons(ctx, addon.ListAddonsInput{
						Namespaces: []string{namespace},
						Currency: &filter.FilterString{
							Eq: &currencyStr,
						},
					})
					assert.NoErrorf(t, err, "getting add-on by currency filter must not fail")

					require.NotEmpty(t, listAddonV1.Items, "add-ons must not be empty")
				})

				t.Run("ByKeyVersion", func(t *testing.T) {
					listAddonV1, err := env.AddonRepository.ListAddons(ctx, addon.ListAddonsInput{
						Namespaces:  []string{namespace},
						KeyVersions: map[string][]int{addonV1Input.Key: {1}},
					})
					assert.NoErrorf(t, err, "getting add-on by key and version must not fail")

					require.Lenf(t, listAddonV1.Items, 1, "add-ons must not be empty")

					addon.AssertAddonEqual(t, *addonV1, listAddonV1.Items[0])
				})
			})

			t.Run("Update", func(t *testing.T) {
				now := time.Now()

				addonV1Update := addon.UpdateAddonInput{
					NamespacedID: models.NamespacedID{
						Namespace: namespace,
						ID:        addonV1.ID,
					},
					EffectivePeriod: productcatalog.EffectivePeriod{
						EffectiveFrom: lo.ToPtr(now.UTC()),
						EffectiveTo:   lo.ToPtr(now.Add(30 * 24 * time.Hour).UTC()),
					},
					Name:        lo.ToPtr("Addon v1 Published"),
					Description: lo.ToPtr("Addon v1 Published"),
					Metadata: &models.Metadata{
						"name":        "Addon v1 Published",
						"description": "Addon v1 Published",
					},
					RateCards: &productcatalog.RateCards{
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:                 "ratecard-2",
								Name:                "RateCard 2",
								Description:         lo.ToPtr("RateCard 2"),
								Metadata:            models.Metadata{"name": "ratecard-2"},
								FeatureKey:          nil,
								FeatureID:           nil,
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
				}

				addonV1, err = env.AddonRepository.UpdateAddon(ctx, addonV1Update)
				require.NoErrorf(t, err, "updating add-on must not fail")

				require.NotNilf(t, addonV1, "add-on must not be nil")

				addon.AssertAddonUpdateInputEqual(t, addonV1Update, *addonV1)
			})

			t.Run("Delete", func(t *testing.T) {
				err = env.AddonRepository.DeleteAddon(ctx, addon.DeleteAddonInput{
					NamespacedID: models.NamespacedID{
						Namespace: addonV1.Namespace,
						ID:        addonV1.ID,
					},
				})
				require.NoErrorf(t, err, "deleting ad-on must not fail")

				getAddonV1, err := env.AddonRepository.GetAddon(ctx, addon.GetAddonInput{
					NamespacedID: models.NamespacedID{
						Namespace: namespace,
						ID:        addonV1.ID,
					},
				})
				require.NoErrorf(t, err, "getting add-on by id must not fail")

				require.NotNilf(t, getAddonV1, "add-on must not be nil")

				addon.AssertAddonEqual(t, *addonV1, *getAddonV1)
			})
		})

		t.Run("ListAddonStatusFilter", func(t *testing.T) {
			// Get new namespace ID
			namespace := pctestutils.NewTestNamespace(t)

			addonV1Input := pctestutils.NewTestAddon(t, namespace, productcatalog.RateCards{
				&productcatalog.FlatFeeRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "ratecard",
						Name: "ratecard",
					},
				},
			}...)

			inputs := []struct {
				Version         int
				EffectivePeriod productcatalog.EffectivePeriod
			}{
				{
					Version: 1,
					EffectivePeriod: productcatalog.EffectivePeriod{
						EffectiveFrom: lo.ToPtr(testutils.GetRFC3339Time(t, "2025-03-15T00:00:00Z")),
						EffectiveTo:   lo.ToPtr(testutils.GetRFC3339Time(t, "2025-03-15T12:00:00Z")),
					},
				},
				{
					Version: 2,
					EffectivePeriod: productcatalog.EffectivePeriod{
						EffectiveFrom: lo.ToPtr(testutils.GetRFC3339Time(t, "2025-03-15T12:00:00Z")),
					},
				},
				{
					Version:         3,
					EffectivePeriod: productcatalog.EffectivePeriod{},
				},
			}

			for _, in := range inputs {
				addonV1Input.Addon.AddonMeta.Version = in.Version

				addonVersion, err := env.AddonRepository.CreateAddon(ctx, addonV1Input)
				require.NoErrorf(t, err, "creating new add-on must not fail")

				_, err = env.AddonRepository.UpdateAddon(ctx, addon.UpdateAddonInput{
					NamespacedID: models.NamespacedID{
						Namespace: namespace,
						ID:        addonVersion.ID,
					},
					EffectivePeriod: in.EffectivePeriod,
				})
				require.NoErrorf(t, err, "updating new add-on must not fail")
			}

			tests := []struct {
				name          string
				at            time.Time
				filter        []productcatalog.AddonStatus
				expectVersion []int
			}{
				{
					name: "Active",
					at:   testutils.GetRFC3339Time(t, "2025-03-16T00:00:00Z"),
					filter: []productcatalog.AddonStatus{
						productcatalog.AddonStatusActive,
					},
					expectVersion: []int{2},
				},
				{
					name: "Draft",
					at:   testutils.GetRFC3339Time(t, "2025-03-16T00:00:00Z"),
					filter: []productcatalog.AddonStatus{
						productcatalog.AddonStatusDraft,
					},
					expectVersion: []int{3},
				},
				{
					name: "Archived",
					at:   testutils.GetRFC3339Time(t, "2025-03-16T00:00:00Z"),
					filter: []productcatalog.AddonStatus{
						productcatalog.AddonStatusArchived,
					},
					expectVersion: []int{1},
				},
				{
					name: "All",
					at:   testutils.GetRFC3339Time(t, "2025-03-16T00:00:00Z"),
					filter: []productcatalog.AddonStatus{
						productcatalog.AddonStatusActive,
						productcatalog.AddonStatusDraft,
						productcatalog.AddonStatusArchived,
					},
					expectVersion: []int{1, 2, 3},
				},
				{
					name: "Scheduled",
					at:   testutils.GetRFC3339Time(t, "2025-03-15T01:00:00Z"),
					filter: []productcatalog.AddonStatus{
						productcatalog.AddonStatusInvalid,
					},
					expectVersion: []int{},
				},
			}

			defer clock.ResetTime()

			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					clock.SetTime(test.at)

					list, err := env.AddonRepository.ListAddons(ctx, addon.ListAddonsInput{
						Namespaces: []string{namespace},
						Status:     test.filter,
					})
					require.NoError(t, err, "listing add-ons must not fail")

					versions := lo.Map(list.Items, func(item addon.Addon, _ int) int {
						return item.Version
					})

					require.ElementsMatch(t, test.expectVersion, versions)
				})
			}
		})
	})
}

func TestListAddonsExcludeUnitConfig(t *testing.T) {
	ctx := context.Background()

	env := pctestutils.NewTestEnv(t)
	t.Cleanup(func() { env.Close(t) })

	namespace := pctestutils.NewTestNamespace(t)

	require.NoError(t, env.Meter.ReplaceMeters(ctx, pctestutils.NewTestMeters(t, namespace)),
		"replacing meters must not fail")
	meters, err := env.Meter.ListMeters(ctx, meter.ListMetersParams{
		Page:      pagination.Page{PageSize: 1000, PageNumber: 1},
		Namespace: namespace,
	})
	require.NoError(t, err, "listing meters must not fail")
	require.NotEmpty(t, meters.Items, "list of meters must not be empty")

	feat, err := env.Feature.CreateFeature(ctx, pctestutils.NewTestFeatureFromMeter(t, &meters.Items[0]))
	require.NoError(t, err, "creating feature must not fail")

	// Plain add-on: flat rate card, no unit_config → v1-representable.
	plainInput := pctestutils.NewTestAddon(t, namespace, &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{Key: "flat", Name: "Flat"},
	})
	plainInput.Addon.Key = "plain"
	_, err = env.AddonRepository.CreateAddon(ctx, plainInput)
	require.NoError(t, err, "creating plain add-on must not fail")

	// unit_config add-on: usage-based rate card carrying a unit_config → not v1-representable.
	ucInput := pctestutils.NewTestAddon(t, namespace, &productcatalog.UsageBasedRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:        feat.Key,
			Name:       "UC RateCard",
			FeatureKey: lo.ToPtr(feat.Key),
			FeatureID:  lo.ToPtr(feat.ID),
			Price:      productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: decimal.NewFromInt(1)}),
			UnitConfig: &productcatalog.UnitConfig{
				Operation:        productcatalog.UnitConfigOperationDivide,
				ConversionFactor: decimal.NewFromInt(1000),
			},
		},
		BillingCadence: MonthPeriod,
	})
	ucInput.Addon.Key = "with-uc"
	_, err = env.AddonRepository.CreateAddon(ctx, ucInput)
	require.NoError(t, err, "creating unit_config add-on must not fail")

	t.Run("included when ExcludeUnitConfig is false", func(t *testing.T) {
		list, err := env.AddonRepository.ListAddons(ctx, addon.ListAddonsInput{
			Namespaces: []string{namespace},
		})
		require.NoError(t, err, "listing add-ons must not fail")

		keys := lo.Map(list.Items, func(a addon.Addon, _ int) string { return a.Key })
		require.ElementsMatch(t, []string{"plain", "with-uc"}, keys)
		require.Equal(t, 2, list.TotalCount, "TotalCount must count both add-ons")
	})

	t.Run("excluded when ExcludeUnitConfig is true, TotalCount stays consistent", func(t *testing.T) {
		list, err := env.AddonRepository.ListAddons(ctx, addon.ListAddonsInput{
			Namespaces:        []string{namespace},
			ExcludeUnitConfig: true,
		})
		require.NoError(t, err, "listing add-ons must not fail")

		keys := lo.Map(list.Items, func(a addon.Addon, _ int) string { return a.Key })
		require.ElementsMatch(t, []string{"plain"}, keys)
		require.Equal(t, 1, list.TotalCount, "TotalCount must exclude the unit_config add-on, not just the page slice")
	})
}

// TestFromPlanRateCardRowMapsUnitConfig guards the cross-package mapper used when an
// add-on is loaded with expanded linked plans
// (FromAddonRow → FromPlanAddonRow → FromPlanRow → FromPlanPhaseRow → FromPlanRateCardRow).
// This mapper is separate from the own-type add-on rate-card mapper, so a RateCardMeta field
// added to one is not automatically carried by the other; UnitConfig dropping here would
// surface a stored config as nil and rate raw usage instead of converted units.
func TestFromPlanRateCardRowMapsUnitConfig(t *testing.T) {
	unitConfig := &productcatalog.UnitConfig{
		Operation:        productcatalog.UnitConfigOperationDivide,
		ConversionFactor: decimal.NewFromInt(1000),
		Rounding:         productcatalog.UnitConfigRoundingModeCeiling,
		Precision:        0,
		DisplayUnit:      lo.ToPtr("K"),
	}

	rc, err := adapter.FromPlanRateCardRow(entdb.PlanRateCard{
		Key:        "rc",
		Name:       "RC",
		Type:       productcatalog.UsageBasedRateCardType,
		Price:      productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: decimal.NewFromInt(1)}),
		UnitConfig: unitConfig,
	})
	require.NoError(t, err, "mapping plan rate card row must not fail")

	require.Equal(t, unitConfig, rc.AsMeta().UnitConfig,
		"UnitConfig must survive the add-on adapter's linked plan mapper")
}
