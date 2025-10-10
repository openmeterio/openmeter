package service_test

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
	pctestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var MonthPeriod = datetime.ISODurationFromDuration(30 * 24 * time.Hour)

func TestAddonService(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := pctestutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	env.DBSchemaMigrate(t)

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
					},
					BillingCadence: MonthPeriod,
				},
			}...)

			var addonV1 *addon.Addon

			addonV1, err = env.Addon.CreateAddon(ctx, addonV1Input)
			require.NoErrorf(t, err, "creating add-on must not fail")
			require.NotNil(t, addonV1, "add-on must not be empty")

			addon.AssertAddonCreateInputEqual(t, addonV1Input, *addonV1)

			assert.Equalf(t, productcatalog.AddonStatusDraft, addonV1.Status(),
				"add-on status mismatch: expected=%s, actual=%s", productcatalog.AddonStatusDraft, addonV1.Status())

			t.Run("Get", func(t *testing.T) {
				getAddon, err := env.Addon.GetAddon(ctx, addon.GetAddonInput{
					NamespacedID: models.NamespacedID{
						Namespace: addonV1Input.Namespace,
					},
					Key:           addonV1Input.Key,
					IncludeLatest: true,
				})
				require.NoErrorf(t, err, "getting draft add-on must not fail")
				require.NotNil(t, getAddon, "draft add-on must not be empty")

				assert.Equalf(t, addonV1.ID, getAddon.ID,
					"Plan ID mismatch: %s = %s", addonV1.ID, getAddon.ID)

				assert.Equalf(t, addonV1.Key, getAddon.Key,
					"Plan Key mismatch: %s = %s", addonV1.Key, getAddon.Key)

				assert.Equalf(t, addonV1.Version, getAddon.Version,
					"Plan Version mismatch: %d = %d", addonV1.Version, getAddon.Version)

				assert.Equalf(t, productcatalog.AddonStatusDraft, getAddon.Status(),
					"Plan Status mismatch: expected=%s, actual=%s", productcatalog.AddonStatusDraft, getAddon.Status())
			})

			t.Run("Update", func(t *testing.T) {
				updateInput := addon.UpdateAddonInput{
					NamespacedID: addonV1.NamespacedID,
					RateCards: &productcatalog.RateCards{
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
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:                 features[1].Key,
								Name:                features[1].Name,
								Description:         lo.ToPtr(features[1].Name),
								Metadata:            models.Metadata{"name": features[1].Name},
								FeatureKey:          lo.ToPtr(features[1].Key),
								FeatureID:           lo.ToPtr(features[1].ID),
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
				}

				updateInput.IgnoreNonCriticalIssues = true

				updatedAddon, err := env.Addon.UpdateAddon(ctx, updateInput)
				require.NoErrorf(t, err, "updating draft add-on must not fail")
				require.NotNil(t, updatedAddon, "updated draft add-on must not be empty")

				addon.AssertAddonUpdateInputEqual(t, updateInput, *updatedAddon)
			})

			var publishedAddonV1 *addon.Addon

			t.Run("Publish", func(t *testing.T) {
				publishAt := time.Now().Truncate(time.Microsecond)

				publishInput := addon.PublishAddonInput{
					NamespacedID: addonV1.NamespacedID,
					EffectivePeriod: productcatalog.EffectivePeriod{
						EffectiveFrom: &publishAt,
						EffectiveTo:   nil,
					},
				}

				publishedAddonV1, err = env.Addon.PublishAddon(ctx, publishInput)
				require.NoErrorf(t, err, "publishing draft add-on must not fail")
				require.NotNil(t, publishedAddonV1, "published add-on must not be empty")
				require.NotNil(t, publishedAddonV1.EffectiveFrom, "EffectiveFrom for published add-on must not be empty")

				assert.Equalf(t, publishAt, *publishedAddonV1.EffectiveFrom,
					"EffectiveFrom for published add-on mismatch: expected=%s, actual=%s", publishAt, *publishedAddonV1.EffectiveFrom)

				assert.Equalf(t, productcatalog.AddonStatusActive, publishedAddonV1.Status(),
					"add-on Status mismatch: expected=%s, actual=%s", productcatalog.AddonStatusActive, publishedAddonV1.Status())

				t.Run("Update", func(t *testing.T) {
					updateInput := addon.UpdateAddonInput{
						NamespacedID: addonV1.NamespacedID,
						Name:         lo.ToPtr("Invalid Update"),
					}

					_, err = env.Addon.UpdateAddon(ctx, updateInput)
					require.Errorf(t, err, "updating active add-on must fail")
				})
			})

			var addonV2 *addon.Addon

			t.Run("V2", func(t *testing.T) {
				addonV2, err = env.Addon.CreateAddon(ctx, addonV1Input)
				require.NoErrorf(t, err, "creating a new draft add-on from active must not fail")
				require.NotNil(t, addonV2, "new draft add-on must not be empty")

				assert.Equalf(t, publishedAddonV1.Version+1, addonV2.Version,
					"new draft add-on must have higher version number")

				assert.Equalf(t, productcatalog.AddonStatusDraft, addonV2.Status(),
					"add-on Status mismatch: expected=%s, actual=%s", productcatalog.AddonStatusDraft, addonV2.Status())

				t.Run("PublishUnaligned", func(t *testing.T) {
					updateInput := addon.UpdateAddonInput{
						NamespacedID: addonV2.NamespacedID,
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
					}

					_, err := env.Addon.UpdateAddon(ctx, updateInput)
					require.NoError(t, err)

					// Get the updated add-on
					_, err = env.Addon.GetAddon(ctx, addon.GetAddonInput{
						NamespacedID: addonV2.NamespacedID,
					})
					require.NoError(t, err)

					// Let's try to publish the add-on
					publishAt := time.Now().Truncate(time.Microsecond)

					publishInput := addon.PublishAddonInput{
						NamespacedID: addonV2.NamespacedID,
						EffectivePeriod: productcatalog.EffectivePeriod{
							EffectiveFrom: &publishAt,
							EffectiveTo:   nil,
						},
					}

					_, err = env.Addon.PublishAddon(ctx, publishInput)
					require.Error(t, err, "publishing draft add-on with alignment issues must fail")

					// Let's update the plan to fix the alignment issue
					_, err = env.Addon.UpdateAddon(ctx, addon.UpdateAddonInput{
						NamespacedID: addonV2.NamespacedID,
						RateCards:    lo.ToPtr(publishedAddonV1.RateCards.AsProductCatalogRateCards()),
					})
					require.NoError(t, err)
				})

				t.Run("Publish", func(t *testing.T) {
					publishAt := time.Now().Truncate(time.Microsecond)

					publishInput := addon.PublishAddonInput{
						NamespacedID: addonV2.NamespacedID,
						EffectivePeriod: productcatalog.EffectivePeriod{
							EffectiveFrom: &publishAt,
							EffectiveTo:   nil,
						},
					}

					publishedAddonV2, err := env.Addon.PublishAddon(ctx, publishInput)
					require.NoErrorf(t, err, "publishing draft add-on must not fail")
					require.NotNil(t, publishedAddonV2, "published add-on must not be empty")
					require.NotNil(t, publishedAddonV2.EffectiveFrom, "EffectiveFrom for published add-on must not be empty")

					assert.Equalf(t, publishAt, *publishedAddonV2.EffectiveFrom,
						"EffectiveFrom for published add-on mismatch: expected=%s, actual=%s", publishAt, *publishedAddonV2.EffectiveFrom)

					assert.Equalf(t, productcatalog.AddonStatusActive, publishedAddonV2.Status(),
						"add-on Status mismatch: expected=%s, actual=%s", productcatalog.AddonStatusActive, publishedAddonV2.Status())

					getAddonV1, err := env.Addon.GetAddon(ctx, addon.GetAddonInput{
						NamespacedID: publishedAddonV1.NamespacedID,
					})
					require.NoErrorf(t, err, "getting previous add-on version must not fail")
					require.NotNil(t, getAddonV1, "previous add version must not be empty")

					assert.Equalf(t, productcatalog.AddonStatusArchived, getAddonV1.Status(),
						"add Status mismatch: expected=%s, actual=%s", productcatalog.AddonStatusArchived, getAddonV1.Status())

					t.Run("Archive", func(t *testing.T) {
						archiveAt := time.Now().Truncate(time.Microsecond)

						archiveInput := addon.ArchiveAddonInput{
							NamespacedID: addonV2.NamespacedID,
							EffectiveTo:  archiveAt,
						}

						archivedAddonV2, err := env.Addon.ArchiveAddon(ctx, archiveInput)
						require.NoErrorf(t, err, "archiving add-on must not fail")
						require.NotNil(t, archivedAddonV2, "archived add-on must not be empty")
						require.NotNil(t, archivedAddonV2.EffectiveTo, "EffectiveFrom for archived add-on must not be empty")

						assert.Equalf(t, archiveAt, *archivedAddonV2.EffectiveTo,
							"EffectiveTo for published add-on mismatch: expected=%s, actual=%s", archiveAt, *archivedAddonV2.EffectiveTo)

						assert.Equalf(t, productcatalog.AddonStatusArchived, archivedAddonV2.Status(),
							"Status mismatch for archived add-on: expected=%s, actual=%s", productcatalog.AddonStatusArchived, archivedAddonV2.Status())
					})
				})

				t.Run("Delete", func(t *testing.T) {
					deleteInput := addon.DeleteAddonInput{
						NamespacedID: addonV2.NamespacedID,
					}

					err = env.Addon.DeleteAddon(ctx, deleteInput)
					require.NoErrorf(t, err, "deleting add-on must not fail")

					deletedAddonV2, err := env.Addon.GetAddon(ctx, addon.GetAddonInput{
						NamespacedID: addonV2.NamespacedID,
					})
					require.NoErrorf(t, err, "getting deleted add-on version must not fail")
					require.NotNil(t, deletedAddonV2, "deleted add-on version must not be empty")

					assert.NotNilf(t, deletedAddonV2.DeletedAt, "deletedAt must not be empty")

					err = env.Addon.DeleteAddon(ctx, deleteInput)
					require.NoErrorf(t, err, "deleting add-on must not fail")

					deletedAddonV2Next, err := env.Addon.GetAddon(ctx, addon.GetAddonInput{
						NamespacedID: addonV2.NamespacedID,
					})
					require.NoErrorf(t, err, "getting deleted add-on version must not fail")
					require.NotNil(t, deletedAddonV2Next, "deleted add-on version must not be empty")

					assert.Truef(t, deletedAddonV2.DeletedAt.Equal(*deletedAddonV2Next.DeletedAt), "deletedAt field must not be updated")
				})
			})
		})
	})
}
