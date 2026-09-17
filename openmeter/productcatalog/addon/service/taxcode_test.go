package service_test

import (
	"context"
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/ent/db/addonratecard"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	pctestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func newTestAddonFlatRateCard(feat feature.Feature, tc *productcatalog.TaxConfig) productcatalog.RateCard {
	return &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:       feat.Key,
			Name:      feat.Name,
			Feature:   productcatalog.NewFeatureReference(lo.ToPtr(feat.ID), lo.ToPtr(feat.Key)),
			TaxConfig: tc,
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      decimal.NewFromInt(100),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
		},
		BillingCadence: &MonthPeriod,
	}
}

func newTestAddonInput(t *testing.T, namespace string, rcs ...productcatalog.RateCard) addon.CreateAddonInput {
	t.Helper()
	return pctestutils.NewTestAddon(t, namespace, rcs...)
}

func getFirstAddonRCTaxConfig(t *testing.T, a *addon.Addon) *productcatalog.TaxConfig {
	t.Helper()
	require.NotEmpty(t, a.RateCards)
	return a.RateCards[0].AsMeta().TaxConfig
}

func findAddonTaxCodeByStripeCode(t *testing.T, ctx context.Context, svc taxcode.Service, namespace string, stripeCode string) (taxcode.TaxCode, error) {
	t.Helper()
	return svc.GetTaxCodeByAppMapping(ctx, taxcode.GetTaxCodeByAppMappingInput{
		Namespace: namespace,
		AppType:   app.AppTypeStripe,
		TaxCode:   stripeCode,
	})
}

// assertAddonRCDBCols queries the AddonRateCard row directly from the database and asserts the
// dedicated tax_code_id and tax_behavior columns match the expected values.
func assertAddonRCDBCols(t *testing.T, ctx context.Context, env *pctestutils.TestEnv, addonID string, rcKey string, wantTaxCodeID *string, wantBehavior *productcatalog.TaxBehavior) {
	t.Helper()
	row, err := env.Client.AddonRateCard.Query().
		Where(
			addonratecard.AddonID(addonID),
			addonratecard.Key(rcKey),
			addonratecard.DeletedAtIsNil(),
		).
		Only(ctx)
	require.NoError(t, err, "direct DB read of AddonRateCard must succeed")
	assert.Equal(t, wantTaxCodeID, row.TaxCodeID, "tax_code_id column mismatch")
	assert.Equal(t, wantBehavior, row.TaxBehavior, "tax_behavior column mismatch")
}

func TestAddonTaxCodeDualWrite(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := pctestutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	namespace := pctestutils.NewTestNamespace(t)

	// Setup meters and features
	err := env.Meter.ReplaceMeters(ctx, pctestutils.NewTestMeters(t, namespace))
	require.NoError(t, err)

	result, err := env.Meter.ListMeters(ctx, meter.ListMetersParams{
		Page: pagination.Page{
			PageSize:   1000,
			PageNumber: 1,
		},
		Namespace: namespace,
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.Items)

	features := make([]feature.Feature, 0, len(result.Items))
	for _, m := range result.Items {
		feat, err := env.Feature.CreateFeature(ctx, pctestutils.NewTestFeatureFromMeter(t, &m))
		require.NoError(t, err)
		features = append(features, feat)
	}

	t.Run("Create", func(t *testing.T) {
		t.Run("NoTaxConfig", func(t *testing.T) {
			input := newTestAddonInput(t, namespace, newTestAddonFlatRateCard(features[0], nil))
			input.Key = "addon-no-tax-config"
			input.Name = "No Tax Config"

			a, err := env.Addon.CreateAddon(ctx, input)
			require.NoError(t, err)

			tc := getFirstAddonRCTaxConfig(t, a)
			assert.Nil(t, tc, "TaxConfig should be nil")

			assertAddonRCDBCols(t, ctx, env, a.ID, features[0].Key, nil, nil)
		})

		t.Run("StripeCodeOnly", func(t *testing.T) {
			input := newTestAddonInput(t, namespace, newTestAddonFlatRateCard(features[0], &productcatalog.TaxConfig{
				Stripe: &productcatalog.StripeTaxConfig{
					Code: "txcd_10000001",
				},
			}))
			input.Key = "addon-stripe-only"
			input.Name = "Stripe Only"

			a, err := env.Addon.CreateAddon(ctx, input)
			require.NoError(t, err)

			tc := getFirstAddonRCTaxConfig(t, a)
			require.NotNil(t, tc)

			// Stripe code preserved
			require.NotNil(t, tc.Stripe)
			assert.Equal(t, "txcd_10000001", tc.Stripe.Code)

			// TaxCodeID should be set
			require.NotNil(t, tc.TaxCodeID, "TaxCodeID must be populated after resolution")

			// Verify TaxCode entity exists
			tcEntity, err := findAddonTaxCodeByStripeCode(t, ctx, env.TaxCode, namespace, "txcd_10000001")
			require.NoError(t, err)
			assert.Equal(t, *tc.TaxCodeID, tcEntity.ID)
			assert.Equal(t, namespace, tcEntity.Namespace)

			assertAddonRCDBCols(t, ctx, env, a.ID, features[0].Key, lo.ToPtr(tcEntity.ID), nil)
		})

		t.Run("StripeCodeAndBehavior", func(t *testing.T) {
			input := newTestAddonInput(t, namespace, newTestAddonFlatRateCard(features[0], &productcatalog.TaxConfig{
				Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
				Stripe: &productcatalog.StripeTaxConfig{
					Code: "txcd_20000001",
				},
			}))
			input.Key = "addon-stripe-and-behavior"
			input.Name = "Stripe and Behavior"

			a, err := env.Addon.CreateAddon(ctx, input)
			require.NoError(t, err)

			tc := getFirstAddonRCTaxConfig(t, a)
			require.NotNil(t, tc)

			require.NotNil(t, tc.Behavior)
			assert.Equal(t, productcatalog.ExclusiveTaxBehavior, *tc.Behavior)

			require.NotNil(t, tc.Stripe)
			assert.Equal(t, "txcd_20000001", tc.Stripe.Code)

			require.NotNil(t, tc.TaxCodeID)

			tcEntity, err := findAddonTaxCodeByStripeCode(t, ctx, env.TaxCode, namespace, "txcd_20000001")
			require.NoError(t, err)
			assert.Equal(t, *tc.TaxCodeID, tcEntity.ID)

			assertAddonRCDBCols(t, ctx, env, a.ID, features[0].Key, lo.ToPtr(tcEntity.ID), lo.ToPtr(productcatalog.ExclusiveTaxBehavior))
		})

		t.Run("BehaviorOnlyNoStripe", func(t *testing.T) {
			input := newTestAddonInput(t, namespace, newTestAddonFlatRateCard(features[0], &productcatalog.TaxConfig{
				Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior),
			}))
			input.Key = "addon-behavior-only"
			input.Name = "Behavior Only"

			a, err := env.Addon.CreateAddon(ctx, input)
			require.NoError(t, err)

			tc := getFirstAddonRCTaxConfig(t, a)
			require.NotNil(t, tc)

			require.NotNil(t, tc.Behavior)
			assert.Equal(t, productcatalog.InclusiveTaxBehavior, *tc.Behavior)

			assert.Nil(t, tc.Stripe, "Stripe should be nil when not provided")
			assert.Nil(t, tc.TaxCodeID, "TaxCodeID should be nil when no Stripe code")

			assertAddonRCDBCols(t, ctx, env, a.ID, features[0].Key, nil, lo.ToPtr(productcatalog.InclusiveTaxBehavior))
		})

		t.Run("ReuseExistingTaxCode", func(t *testing.T) {
			// Create first addon with txcd_30000001
			input1 := newTestAddonInput(t, namespace, newTestAddonFlatRateCard(features[0], &productcatalog.TaxConfig{
				Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_30000001"},
			}))
			input1.Key = "addon-reuse-1"
			input1.Name = "Reuse 1"

			a1, err := env.Addon.CreateAddon(ctx, input1)
			require.NoError(t, err)

			tc1 := getFirstAddonRCTaxConfig(t, a1)
			require.NotNil(t, tc1)
			require.NotNil(t, tc1.TaxCodeID)

			// Create second addon with same stripe code
			input2 := newTestAddonInput(t, namespace, newTestAddonFlatRateCard(features[0], &productcatalog.TaxConfig{
				Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_30000001"},
			}))
			input2.Key = "addon-reuse-2"
			input2.Name = "Reuse 2"

			a2, err := env.Addon.CreateAddon(ctx, input2)
			require.NoError(t, err)

			tc2 := getFirstAddonRCTaxConfig(t, a2)
			require.NotNil(t, tc2)
			require.NotNil(t, tc2.TaxCodeID)

			// Both addons should reference the same TaxCode entity
			assert.Equal(t, *tc1.TaxCodeID, *tc2.TaxCodeID, "both addons must reference the same TaxCode entity")

			assertAddonRCDBCols(t, ctx, env, a1.ID, features[0].Key, lo.ToPtr(*tc1.TaxCodeID), nil)
			assertAddonRCDBCols(t, ctx, env, a2.ID, features[0].Key, lo.ToPtr(*tc2.TaxCodeID), nil)
		})

		t.Run("MultipleDifferentStripeCodes", func(t *testing.T) {
			rc1 := &productcatalog.FlatFeeRateCard{
				RateCardMeta: productcatalog.RateCardMeta{
					Key:  "rc-a",
					Name: "RC A",
					TaxConfig: &productcatalog.TaxConfig{
						Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_40000001"},
					},
					Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      decimal.NewFromInt(100),
						PaymentTerm: productcatalog.InArrearsPaymentTerm,
					}),
				},
				BillingCadence: &MonthPeriod,
			}

			rc2 := &productcatalog.FlatFeeRateCard{
				RateCardMeta: productcatalog.RateCardMeta{
					Key:  "rc-b",
					Name: "RC B",
					TaxConfig: &productcatalog.TaxConfig{
						Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_50000001"},
					},
					Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      decimal.NewFromInt(200),
						PaymentTerm: productcatalog.InArrearsPaymentTerm,
					}),
				},
				BillingCadence: &MonthPeriod,
			}

			input := pctestutils.NewTestAddon(t, namespace, rc1, rc2)
			input.Key = "addon-multi-stripe"
			input.Name = "Multi Stripe"

			a, err := env.Addon.CreateAddon(ctx, input)
			require.NoError(t, err)

			require.Len(t, a.RateCards, 2)

			rcMap := make(map[string]*productcatalog.TaxConfig)
			for _, rc := range a.RateCards {
				rcMap[rc.AsMeta().Key] = rc.AsMeta().TaxConfig
			}

			tcA := rcMap["rc-a"]
			tcB := rcMap["rc-b"]

			require.NotNil(t, tcA)
			require.NotNil(t, tcA.TaxCodeID)
			require.NotNil(t, tcB)
			require.NotNil(t, tcB.TaxCodeID)

			assert.NotEqual(t, *tcA.TaxCodeID, *tcB.TaxCodeID, "different stripe codes must create different TaxCode entities")

			tcEntityA, err := findAddonTaxCodeByStripeCode(t, ctx, env.TaxCode, namespace, "txcd_40000001")
			require.NoError(t, err)
			tcEntityB, err := findAddonTaxCodeByStripeCode(t, ctx, env.TaxCode, namespace, "txcd_50000001")
			require.NoError(t, err)
			assertAddonRCDBCols(t, ctx, env, a.ID, "rc-a", lo.ToPtr(tcEntityA.ID), nil)
			assertAddonRCDBCols(t, ctx, env, a.ID, "rc-b", lo.ToPtr(tcEntityB.ID), nil)
		})

		t.Run("TaxCodeIdOnly", func(t *testing.T) {
			// Pre-create a TaxCode entity with a Stripe mapping.
			tcEntity, err := env.TaxCode.GetOrCreateByAppMapping(ctx, taxcode.GetOrCreateByAppMappingInput{
				Namespace: namespace,
				AppType:   app.AppTypeStripe,
				TaxCode:   "txcd_60000003",
			})
			require.NoError(t, err)

			input := newTestAddonInput(t, namespace, newTestAddonFlatRateCard(features[0], &productcatalog.TaxConfig{
				TaxCodeID: lo.ToPtr(tcEntity.ID),
			}))
			input.Key = "addon-taxcodeid-only"
			input.Name = "TaxCodeId Only"

			a, err := env.Addon.CreateAddon(ctx, input)
			require.NoError(t, err)

			tc := getFirstAddonRCTaxConfig(t, a)
			require.NotNil(t, tc)
			require.NotNil(t, tc.TaxCodeID)
			assert.Equal(t, tcEntity.ID, *tc.TaxCodeID)

			// Stripe code should be backfilled from the TaxCode entity's app mapping.
			require.NotNil(t, tc.Stripe, "Stripe must be backfilled from TaxCode app mapping")
			assert.Equal(t, "txcd_60000003", tc.Stripe.Code)

			assertAddonRCDBCols(t, ctx, env, a.ID, features[0].Key, lo.ToPtr(tcEntity.ID), nil)
		})

		t.Run("TaxCodeIdNotFound", func(t *testing.T) {
			input := newTestAddonInput(t, namespace, newTestAddonFlatRateCard(features[0], &productcatalog.TaxConfig{
				TaxCodeID: lo.ToPtr("01JNON_EXISTENT_TAX_CODE_ID"),
			}))
			input.Key = "addon-taxcodeid-not-found"
			input.Name = "TaxCodeId Not Found"

			_, err := env.Addon.CreateAddon(ctx, input)
			require.Error(t, err)
			assert.True(t, models.IsGenericValidationError(err), "expected validation error for unknown taxCodeId, got: %v", err)
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("AddTaxConfig", func(t *testing.T) {
			// Create addon without TaxConfig
			input := newTestAddonInput(t, namespace, newTestAddonFlatRateCard(features[0], nil))
			input.Key = "addon-update-add-tax"
			input.Name = "Update Add Tax"

			a, err := env.Addon.CreateAddon(ctx, input)
			require.NoError(t, err)

			tc := getFirstAddonRCTaxConfig(t, a)
			assert.Nil(t, tc)

			// Update to add TaxConfig with Stripe code
			updatedRateCards := productcatalog.RateCards{
				newTestAddonFlatRateCard(features[0], &productcatalog.TaxConfig{
					Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_70000001"},
				}),
			}

			updated, err := env.Addon.UpdateAddon(ctx, addon.UpdateAddonInput{
				NamespacedID: a.NamespacedID,
				RateCards:    &updatedRateCards,
			})
			require.NoError(t, err)

			tc = getFirstAddonRCTaxConfig(t, updated)
			require.NotNil(t, tc)
			require.NotNil(t, tc.Stripe)
			assert.Equal(t, "txcd_70000001", tc.Stripe.Code)
			require.NotNil(t, tc.TaxCodeID, "TaxCodeID must be populated after update")

			tcEntity, err := findAddonTaxCodeByStripeCode(t, ctx, env.TaxCode, namespace, "txcd_70000001")
			require.NoError(t, err)
			assert.Equal(t, *tc.TaxCodeID, tcEntity.ID)

			assertAddonRCDBCols(t, ctx, env, updated.ID, features[0].Key, lo.ToPtr(tcEntity.ID), nil)
		})

		t.Run("ChangeStripeCode", func(t *testing.T) {
			input := newTestAddonInput(t, namespace, newTestAddonFlatRateCard(features[0], &productcatalog.TaxConfig{
				Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_80000001"},
			}))
			input.Key = "addon-update-change-code"
			input.Name = "Update Change Code"

			a, err := env.Addon.CreateAddon(ctx, input)
			require.NoError(t, err)

			oldTC := getFirstAddonRCTaxConfig(t, a)
			require.NotNil(t, oldTC)
			require.NotNil(t, oldTC.TaxCodeID)
			oldTaxCodeID := *oldTC.TaxCodeID

			// Update to different stripe code
			updatedRateCards := productcatalog.RateCards{
				newTestAddonFlatRateCard(features[0], &productcatalog.TaxConfig{
					Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_90000001"},
				}),
			}

			updated, err := env.Addon.UpdateAddon(ctx, addon.UpdateAddonInput{
				NamespacedID: a.NamespacedID,
				RateCards:    &updatedRateCards,
			})
			require.NoError(t, err)

			newTC := getFirstAddonRCTaxConfig(t, updated)
			require.NotNil(t, newTC)
			require.NotNil(t, newTC.Stripe)
			assert.Equal(t, "txcd_90000001", newTC.Stripe.Code)
			require.NotNil(t, newTC.TaxCodeID)

			assert.NotEqual(t, oldTaxCodeID, *newTC.TaxCodeID, "new stripe code must create a new TaxCode entity")

			// Old TaxCode entity should still exist
			_, err = findAddonTaxCodeByStripeCode(t, ctx, env.TaxCode, namespace, "txcd_80000001")
			assert.NoError(t, err, "old TaxCode entity should still exist")

			newTCEntity, err := findAddonTaxCodeByStripeCode(t, ctx, env.TaxCode, namespace, "txcd_90000001")
			require.NoError(t, err)
			assertAddonRCDBCols(t, ctx, env, updated.ID, features[0].Key, lo.ToPtr(newTCEntity.ID), nil)
		})

		t.Run("UpdateWithTaxCodeId", func(t *testing.T) {
			input := newTestAddonInput(t, namespace, newTestAddonFlatRateCard(features[0], nil))
			input.Key = "addon-update-taxcodeid"
			input.Name = "Update TaxCodeId"

			a, err := env.Addon.CreateAddon(ctx, input)
			require.NoError(t, err)

			tcEntity, err := env.TaxCode.GetOrCreateByAppMapping(ctx, taxcode.GetOrCreateByAppMappingInput{
				Namespace: namespace,
				AppType:   app.AppTypeStripe,
				TaxCode:   "txcd_60000004",
			})
			require.NoError(t, err)

			updatedRateCards := productcatalog.RateCards{
				newTestAddonFlatRateCard(features[0], &productcatalog.TaxConfig{
					TaxCodeID: lo.ToPtr(tcEntity.ID),
				}),
			}

			updated, err := env.Addon.UpdateAddon(ctx, addon.UpdateAddonInput{
				NamespacedID: a.NamespacedID,
				RateCards:    &updatedRateCards,
			})
			require.NoError(t, err)

			tc := getFirstAddonRCTaxConfig(t, updated)
			require.NotNil(t, tc)
			require.NotNil(t, tc.TaxCodeID)
			assert.Equal(t, tcEntity.ID, *tc.TaxCodeID)
			require.NotNil(t, tc.Stripe, "Stripe must be backfilled from TaxCode app mapping")
			assert.Equal(t, "txcd_60000004", tc.Stripe.Code)

			assertAddonRCDBCols(t, ctx, env, updated.ID, features[0].Key, lo.ToPtr(tcEntity.ID), nil)
		})

		t.Run("RemoveTaxConfig", func(t *testing.T) {
			input := newTestAddonInput(t, namespace, newTestAddonFlatRateCard(features[0], &productcatalog.TaxConfig{
				Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_11000001"},
			}))
			input.Key = "addon-update-remove-tax"
			input.Name = "Update Remove Tax"

			a, err := env.Addon.CreateAddon(ctx, input)
			require.NoError(t, err)

			tc := getFirstAddonRCTaxConfig(t, a)
			require.NotNil(t, tc)
			require.NotNil(t, tc.TaxCodeID)

			// Update to remove TaxConfig
			updatedRateCards := productcatalog.RateCards{
				newTestAddonFlatRateCard(features[0], nil),
			}

			updated, err := env.Addon.UpdateAddon(ctx, addon.UpdateAddonInput{
				NamespacedID: a.NamespacedID,
				RateCards:    &updatedRateCards,
			})
			require.NoError(t, err)

			tc = getFirstAddonRCTaxConfig(t, updated)
			assert.Nil(t, tc, "TaxConfig should be nil after removal")

			// TaxCode entity should still exist (orphaned, not deleted)
			_, err = findAddonTaxCodeByStripeCode(t, ctx, env.TaxCode, namespace, "txcd_11000001")
			assert.NoError(t, err, "TaxCode entity should not be deleted")

			assertAddonRCDBCols(t, ctx, env, updated.ID, features[0].Key, nil, nil)
		})

		t.Run("MetadataOnlyUpdate", func(t *testing.T) {
			input := newTestAddonInput(t, namespace, newTestAddonFlatRateCard(features[0], &productcatalog.TaxConfig{
				Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_12000001"},
			}))
			input.Key = "addon-update-metadata-only"
			input.Name = "Update Metadata Only"

			a, err := env.Addon.CreateAddon(ctx, input)
			require.NoError(t, err)

			tc := getFirstAddonRCTaxConfig(t, a)
			require.NotNil(t, tc)
			require.NotNil(t, tc.TaxCodeID)
			originalTaxCodeID := *tc.TaxCodeID

			// Update only addon name, no ratecards
			updated, err := env.Addon.UpdateAddon(ctx, addon.UpdateAddonInput{
				NamespacedID: a.NamespacedID,
				Name:         lo.ToPtr("Updated Name"),
			})
			require.NoError(t, err)

			tc = getFirstAddonRCTaxConfig(t, updated)
			require.NotNil(t, tc)
			require.NotNil(t, tc.TaxCodeID)
			assert.Equal(t, originalTaxCodeID, *tc.TaxCodeID, "TaxCodeID should be unchanged on metadata-only update")

			assertAddonRCDBCols(t, ctx, env, updated.ID, features[0].Key, lo.ToPtr(originalTaxCodeID), nil)
		})
	})

	t.Run("ReadBackVerification", func(t *testing.T) {
		t.Run("BackfillFromNewColumns", func(t *testing.T) {
			// Create addon with full TaxConfig (Stripe + Behavior)
			input := newTestAddonInput(t, namespace, newTestAddonFlatRateCard(features[0], &productcatalog.TaxConfig{
				Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
				Stripe:   &productcatalog.StripeTaxConfig{Code: "txcd_13000001"},
			}))
			input.Key = "addon-readback-verify"
			input.Name = "Readback Verify"

			a, err := env.Addon.CreateAddon(ctx, input)
			require.NoError(t, err)

			// Re-read the addon
			fetched, err := env.Addon.GetAddon(ctx, addon.GetAddonInput{
				NamespacedID: a.NamespacedID,
			})
			require.NoError(t, err)

			tc := getFirstAddonRCTaxConfig(t, fetched)
			require.NotNil(t, tc)

			// Behavior should be present
			require.NotNil(t, tc.Behavior)
			assert.Equal(t, productcatalog.ExclusiveTaxBehavior, *tc.Behavior)

			// Stripe should be present
			require.NotNil(t, tc.Stripe)
			assert.Equal(t, "txcd_13000001", tc.Stripe.Code)

			tcEntity13, err := findAddonTaxCodeByStripeCode(t, ctx, env.TaxCode, namespace, "txcd_13000001")
			require.NoError(t, err)
			assertAddonRCDBCols(t, ctx, env, fetched.ID, features[0].Key, lo.ToPtr(tcEntity13.ID), lo.ToPtr(productcatalog.ExclusiveTaxBehavior))
		})
	})
}
