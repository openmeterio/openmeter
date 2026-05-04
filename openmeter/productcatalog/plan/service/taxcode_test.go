package service_test

import (
	"context"
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/ent/db/planratecard"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	pctestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func newTestFlatRateCard(feat feature.Feature, tc *productcatalog.TaxConfig, billingCadence *datetime.ISODuration) productcatalog.RateCard {
	return &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:        feat.Key,
			Name:       feat.Name,
			FeatureKey: lo.ToPtr(feat.Key),
			FeatureID:  lo.ToPtr(feat.ID),
			TaxConfig:  tc,
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      decimal.NewFromInt(100),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
		},
		BillingCadence: billingCadence,
	}
}

func newTestPlanInput(t *testing.T, namespace string, rc productcatalog.RateCard) plan.CreatePlanInput {
	t.Helper()
	return pctestutils.NewTestPlan(t, namespace, productcatalog.Phase{
		PhaseMeta: productcatalog.PhaseMeta{
			Key:  "default",
			Name: "Default",
		},
		RateCards: productcatalog.RateCards{rc},
	})
}

func getFirstRCTaxConfig(t *testing.T, p *plan.Plan) *productcatalog.TaxConfig {
	t.Helper()
	require.NotEmpty(t, p.Phases)
	require.NotEmpty(t, p.Phases[0].RateCards)
	return p.Phases[0].RateCards[0].AsMeta().TaxConfig
}

func findTaxCodeByStripeCode(t *testing.T, ctx context.Context, svc taxcode.Service, namespace string, stripeCode string) (taxcode.TaxCode, error) {
	t.Helper()
	return svc.GetTaxCodeByAppMapping(ctx, taxcode.GetTaxCodeByAppMappingInput{
		Namespace: namespace,
		AppType:   app.AppTypeStripe,
		TaxCode:   stripeCode,
	})
}

// assertPlanRCDBCols queries the PlanRateCard row directly from the database and asserts the
// dedicated tax_code_id and tax_behavior columns match the expected values.
func assertPlanRCDBCols(t *testing.T, ctx context.Context, env *pctestutils.TestEnv, phaseID string, rcKey string, wantTaxCodeID *string, wantBehavior *productcatalog.TaxBehavior) {
	t.Helper()
	row, err := env.Client.PlanRateCard.Query().
		Where(
			planratecard.PhaseID(phaseID),
			planratecard.Key(rcKey),
			planratecard.DeletedAtIsNil(),
		).
		Only(ctx)
	require.NoError(t, err, "direct DB read of PlanRateCard must succeed")
	assert.Equal(t, wantTaxCodeID, row.TaxCodeID, "tax_code_id column mismatch")
	assert.Equal(t, wantBehavior, row.TaxBehavior, "tax_behavior column mismatch")
}

func TestPlanTaxCodeDualWrite(t *testing.T) {
	MonthPeriod := datetime.MustParseDuration(t, "P1M")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := pctestutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	env.DBSchemaMigrate(t)

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
			input := newTestPlanInput(t, namespace, newTestFlatRateCard(features[0], nil, &MonthPeriod))
			input.Key = "no-tax-config"
			input.Name = "No Tax Config"

			p, err := env.Plan.CreatePlan(ctx, input)
			require.NoError(t, err)

			tc := getFirstRCTaxConfig(t, p)
			assert.Nil(t, tc, "TaxConfig should be nil")

			phaseID := p.Phases[0].PhaseManagedFields.NamespacedID.ID
			assertPlanRCDBCols(t, ctx, env, phaseID, features[0].Key, nil, nil)
		})

		t.Run("StripeCodeOnly", func(t *testing.T) {
			input := newTestPlanInput(t, namespace, newTestFlatRateCard(features[0], &productcatalog.TaxConfig{
				Stripe: &productcatalog.StripeTaxConfig{
					Code: "txcd_10000000",
				},
			}, &MonthPeriod))
			input.Key = "stripe-only"
			input.Name = "Stripe Only"

			p, err := env.Plan.CreatePlan(ctx, input)
			require.NoError(t, err)

			tc := getFirstRCTaxConfig(t, p)
			require.NotNil(t, tc)

			// Stripe code preserved
			require.NotNil(t, tc.Stripe)
			assert.Equal(t, "txcd_10000000", tc.Stripe.Code)

			// TaxCodeID should be set
			require.NotNil(t, tc.TaxCodeID, "TaxCodeID must be populated after resolution")

			// Verify TaxCode entity exists
			tcEntity, err := findTaxCodeByStripeCode(t, ctx, env.TaxCode, namespace, "txcd_10000000")
			require.NoError(t, err)
			assert.Equal(t, *tc.TaxCodeID, tcEntity.ID)
			assert.Equal(t, namespace, tcEntity.Namespace)

			phaseID := p.Phases[0].PhaseManagedFields.NamespacedID.ID
			assertPlanRCDBCols(t, ctx, env, phaseID, features[0].Key, lo.ToPtr(tcEntity.ID), nil)
		})

		t.Run("StripeCodeAndBehavior", func(t *testing.T) {
			input := newTestPlanInput(t, namespace, newTestFlatRateCard(features[0], &productcatalog.TaxConfig{
				Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
				Stripe: &productcatalog.StripeTaxConfig{
					Code: "txcd_20000000",
				},
			}, &MonthPeriod))
			input.Key = "stripe-and-behavior"
			input.Name = "Stripe and Behavior"

			p, err := env.Plan.CreatePlan(ctx, input)
			require.NoError(t, err)

			tc := getFirstRCTaxConfig(t, p)
			require.NotNil(t, tc)

			require.NotNil(t, tc.Behavior)
			assert.Equal(t, productcatalog.ExclusiveTaxBehavior, *tc.Behavior)

			require.NotNil(t, tc.Stripe)
			assert.Equal(t, "txcd_20000000", tc.Stripe.Code)

			require.NotNil(t, tc.TaxCodeID)

			tcEntity, err := findTaxCodeByStripeCode(t, ctx, env.TaxCode, namespace, "txcd_20000000")
			require.NoError(t, err)
			assert.Equal(t, *tc.TaxCodeID, tcEntity.ID)

			phaseID := p.Phases[0].PhaseManagedFields.NamespacedID.ID
			assertPlanRCDBCols(t, ctx, env, phaseID, features[0].Key, lo.ToPtr(tcEntity.ID), lo.ToPtr(productcatalog.ExclusiveTaxBehavior))
		})

		t.Run("BehaviorOnlyNoStripe", func(t *testing.T) {
			input := newTestPlanInput(t, namespace, newTestFlatRateCard(features[0], &productcatalog.TaxConfig{
				Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior),
			}, &MonthPeriod))
			input.Key = "behavior-only"
			input.Name = "Behavior Only"

			p, err := env.Plan.CreatePlan(ctx, input)
			require.NoError(t, err)

			tc := getFirstRCTaxConfig(t, p)
			require.NotNil(t, tc)

			require.NotNil(t, tc.Behavior)
			assert.Equal(t, productcatalog.InclusiveTaxBehavior, *tc.Behavior)

			assert.Nil(t, tc.Stripe, "Stripe should be nil when not provided")
			assert.Nil(t, tc.TaxCodeID, "TaxCodeID should be nil when no Stripe code")

			phaseID := p.Phases[0].PhaseManagedFields.NamespacedID.ID
			assertPlanRCDBCols(t, ctx, env, phaseID, features[0].Key, nil, lo.ToPtr(productcatalog.InclusiveTaxBehavior))
		})

		t.Run("ReuseExistingTaxCode", func(t *testing.T) {
			// Create first plan with txcd_30000000
			input1 := newTestPlanInput(t, namespace, newTestFlatRateCard(features[0], &productcatalog.TaxConfig{
				Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_30000000"},
			}, &MonthPeriod))
			input1.Key = "reuse-1"
			input1.Name = "Reuse 1"

			p1, err := env.Plan.CreatePlan(ctx, input1)
			require.NoError(t, err)

			tc1 := getFirstRCTaxConfig(t, p1)
			require.NotNil(t, tc1)
			require.NotNil(t, tc1.TaxCodeID)

			// Create second plan with same stripe code
			input2 := newTestPlanInput(t, namespace, newTestFlatRateCard(features[0], &productcatalog.TaxConfig{
				Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_30000000"},
			}, &MonthPeriod))
			input2.Key = "reuse-2"
			input2.Name = "Reuse 2"

			p2, err := env.Plan.CreatePlan(ctx, input2)
			require.NoError(t, err)

			tc2 := getFirstRCTaxConfig(t, p2)
			require.NotNil(t, tc2)
			require.NotNil(t, tc2.TaxCodeID)

			// Both plans should reference the same TaxCode entity
			assert.Equal(t, *tc1.TaxCodeID, *tc2.TaxCodeID, "both plans must reference the same TaxCode entity")

			assertPlanRCDBCols(t, ctx, env, p1.Phases[0].PhaseManagedFields.NamespacedID.ID, features[0].Key, lo.ToPtr(*tc1.TaxCodeID), nil)
			assertPlanRCDBCols(t, ctx, env, p2.Phases[0].PhaseManagedFields.NamespacedID.ID, features[0].Key, lo.ToPtr(*tc2.TaxCodeID), nil)
		})

		t.Run("MultipleDifferentStripeCodes", func(t *testing.T) {
			rc1 := &productcatalog.FlatFeeRateCard{
				RateCardMeta: productcatalog.RateCardMeta{
					Key:  "rc-a",
					Name: "RC A",
					TaxConfig: &productcatalog.TaxConfig{
						Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_40000000"},
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
						Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_50000000"},
					},
					Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      decimal.NewFromInt(200),
						PaymentTerm: productcatalog.InArrearsPaymentTerm,
					}),
				},
				BillingCadence: &MonthPeriod,
			}

			input := pctestutils.NewTestPlan(t, namespace, productcatalog.Phase{
				PhaseMeta: productcatalog.PhaseMeta{
					Key:  "default",
					Name: "Default",
				},
				RateCards: productcatalog.RateCards{rc1, rc2},
			})
			input.Key = "multi-stripe"
			input.Name = "Multi Stripe"

			p, err := env.Plan.CreatePlan(ctx, input)
			require.NoError(t, err)

			require.Len(t, p.Phases[0].RateCards, 2)

			rcMap := make(map[string]*productcatalog.TaxConfig)
			for _, rc := range p.Phases[0].RateCards {
				rcMap[rc.AsMeta().Key] = rc.AsMeta().TaxConfig
			}

			tcA := rcMap["rc-a"]
			tcB := rcMap["rc-b"]

			require.NotNil(t, tcA)
			require.NotNil(t, tcA.TaxCodeID)
			require.NotNil(t, tcB)
			require.NotNil(t, tcB.TaxCodeID)

			assert.NotEqual(t, *tcA.TaxCodeID, *tcB.TaxCodeID, "different stripe codes must create different TaxCode entities")

			tcEntityA, err := findTaxCodeByStripeCode(t, ctx, env.TaxCode, namespace, "txcd_40000000")
			require.NoError(t, err)
			tcEntityB, err := findTaxCodeByStripeCode(t, ctx, env.TaxCode, namespace, "txcd_50000000")
			require.NoError(t, err)
			phaseID := p.Phases[0].PhaseManagedFields.NamespacedID.ID
			assertPlanRCDBCols(t, ctx, env, phaseID, "rc-a", lo.ToPtr(tcEntityA.ID), nil)
			assertPlanRCDBCols(t, ctx, env, phaseID, "rc-b", lo.ToPtr(tcEntityB.ID), nil)
		})

		t.Run("TaxCodeIdOnly", func(t *testing.T) {
			// Pre-create a TaxCode entity with a Stripe mapping.
			tcEntity, err := env.TaxCode.GetOrCreateByAppMapping(ctx, taxcode.GetOrCreateByAppMappingInput{
				Namespace: namespace,
				AppType:   app.AppTypeStripe,
				TaxCode:   "txcd_60000001",
			})
			require.NoError(t, err)

			input := newTestPlanInput(t, namespace, newTestFlatRateCard(features[0], &productcatalog.TaxConfig{
				TaxCodeID: lo.ToPtr(tcEntity.ID),
			}, &MonthPeriod))
			input.Key = "taxcodeid-only"
			input.Name = "TaxCodeId Only"

			p, err := env.Plan.CreatePlan(ctx, input)
			require.NoError(t, err)

			tc := getFirstRCTaxConfig(t, p)
			require.NotNil(t, tc)
			require.NotNil(t, tc.TaxCodeID)
			assert.Equal(t, tcEntity.ID, *tc.TaxCodeID)

			// Stripe code should be backfilled from the TaxCode entity's app mapping.
			require.NotNil(t, tc.Stripe, "Stripe must be backfilled from TaxCode app mapping")
			assert.Equal(t, "txcd_60000001", tc.Stripe.Code)

			phaseID := p.Phases[0].PhaseManagedFields.NamespacedID.ID
			assertPlanRCDBCols(t, ctx, env, phaseID, features[0].Key, lo.ToPtr(tcEntity.ID), nil)
		})

		t.Run("TaxCodeIdNotFound", func(t *testing.T) {
			input := newTestPlanInput(t, namespace, newTestFlatRateCard(features[0], &productcatalog.TaxConfig{
				TaxCodeID: lo.ToPtr("01JNON_EXISTENT_TAX_CODE_ID"),
			}, &MonthPeriod))
			input.Key = "taxcodeid-not-found"
			input.Name = "TaxCodeId Not Found"

			_, err := env.Plan.CreatePlan(ctx, input)
			require.Error(t, err)
			assert.True(t, models.IsGenericValidationError(err), "expected validation error for unknown taxCodeId, got: %v", err)
		})

		t.Run("MultiplePhasesSameStripeCode", func(t *testing.T) {
			TwoMonthPeriod := datetime.MustParseDuration(t, "P2M")

			input := pctestutils.NewTestPlan(t, namespace,
				productcatalog.Phase{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:      "phase-a",
						Name:     "Phase A",
						Duration: &TwoMonthPeriod,
					},
					RateCards: productcatalog.RateCards{
						newTestFlatRateCard(features[0], &productcatalog.TaxConfig{
							Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_60000000"},
						}, &MonthPeriod),
					},
				},
				productcatalog.Phase{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:  "phase-b",
						Name: "Phase B",
					},
					RateCards: productcatalog.RateCards{
						newTestFlatRateCard(features[0], &productcatalog.TaxConfig{
							Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_60000000"},
						}, &MonthPeriod),
					},
				},
			)
			input.Key = "multi-phase-same-code"
			input.Name = "Multi Phase Same Code"

			p, err := env.Plan.CreatePlan(ctx, input)
			require.NoError(t, err)

			require.Len(t, p.Phases, 2)

			require.NotEmpty(t, p.Phases[0].RateCards)
			require.NotEmpty(t, p.Phases[1].RateCards)
			tcA := p.Phases[0].RateCards[0].AsMeta().TaxConfig
			tcB := p.Phases[1].RateCards[0].AsMeta().TaxConfig

			require.NotNil(t, tcA)
			require.NotNil(t, tcA.TaxCodeID)
			require.NotNil(t, tcB)
			require.NotNil(t, tcB.TaxCodeID)

			assert.Equal(t, *tcA.TaxCodeID, *tcB.TaxCodeID, "same stripe code across phases must reuse the same TaxCode entity")

			tcEntity60, err := findTaxCodeByStripeCode(t, ctx, env.TaxCode, namespace, "txcd_60000000")
			require.NoError(t, err)
			assertPlanRCDBCols(t, ctx, env, p.Phases[0].PhaseManagedFields.NamespacedID.ID, features[0].Key, lo.ToPtr(tcEntity60.ID), nil)
			assertPlanRCDBCols(t, ctx, env, p.Phases[1].PhaseManagedFields.NamespacedID.ID, features[0].Key, lo.ToPtr(tcEntity60.ID), nil)
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("AddTaxConfig", func(t *testing.T) {
			// Create plan without TaxConfig
			input := newTestPlanInput(t, namespace, newTestFlatRateCard(features[0], nil, &MonthPeriod))
			input.Key = "update-add-tax"
			input.Name = "Update Add Tax"

			p, err := env.Plan.CreatePlan(ctx, input)
			require.NoError(t, err)

			tc := getFirstRCTaxConfig(t, p)
			assert.Nil(t, tc)

			// Update to add TaxConfig with Stripe code
			updatedPhases := []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:  "default",
						Name: "Default",
					},
					RateCards: productcatalog.RateCards{
						newTestFlatRateCard(features[0], &productcatalog.TaxConfig{
							Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_70000000"},
						}, &MonthPeriod),
					},
				},
			}

			updated, err := env.Plan.UpdatePlan(ctx, plan.UpdatePlanInput{
				NamespacedID: p.NamespacedID,
				Phases:       &updatedPhases,
			})
			require.NoError(t, err)

			tc = getFirstRCTaxConfig(t, updated)
			require.NotNil(t, tc)
			require.NotNil(t, tc.Stripe)
			assert.Equal(t, "txcd_70000000", tc.Stripe.Code)
			require.NotNil(t, tc.TaxCodeID, "TaxCodeID must be populated after update")

			tcEntity, err := findTaxCodeByStripeCode(t, ctx, env.TaxCode, namespace, "txcd_70000000")
			require.NoError(t, err)
			assert.Equal(t, *tc.TaxCodeID, tcEntity.ID)

			assertPlanRCDBCols(t, ctx, env, updated.Phases[0].PhaseManagedFields.NamespacedID.ID, features[0].Key, lo.ToPtr(tcEntity.ID), nil)
		})

		t.Run("ChangeStripeCode", func(t *testing.T) {
			input := newTestPlanInput(t, namespace, newTestFlatRateCard(features[0], &productcatalog.TaxConfig{
				Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_80000000"},
			}, &MonthPeriod))
			input.Key = "update-change-code"
			input.Name = "Update Change Code"

			p, err := env.Plan.CreatePlan(ctx, input)
			require.NoError(t, err)

			oldTC := getFirstRCTaxConfig(t, p)
			require.NotNil(t, oldTC)
			require.NotNil(t, oldTC.TaxCodeID)
			oldTaxCodeID := *oldTC.TaxCodeID

			// Update to different stripe code
			updatedPhases := []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:  "default",
						Name: "Default",
					},
					RateCards: productcatalog.RateCards{
						newTestFlatRateCard(features[0], &productcatalog.TaxConfig{
							Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_90000000"},
						}, &MonthPeriod),
					},
				},
			}

			updated, err := env.Plan.UpdatePlan(ctx, plan.UpdatePlanInput{
				NamespacedID: p.NamespacedID,
				Phases:       &updatedPhases,
			})
			require.NoError(t, err)

			newTC := getFirstRCTaxConfig(t, updated)
			require.NotNil(t, newTC)
			require.NotNil(t, newTC.Stripe)
			assert.Equal(t, "txcd_90000000", newTC.Stripe.Code)
			require.NotNil(t, newTC.TaxCodeID)

			assert.NotEqual(t, oldTaxCodeID, *newTC.TaxCodeID, "new stripe code must create a new TaxCode entity")

			// Old TaxCode entity should still exist
			_, err = findTaxCodeByStripeCode(t, ctx, env.TaxCode, namespace, "txcd_80000000")
			assert.NoError(t, err, "old TaxCode entity should still exist")

			newTCEntity, err := findTaxCodeByStripeCode(t, ctx, env.TaxCode, namespace, "txcd_90000000")
			require.NoError(t, err)
			assertPlanRCDBCols(t, ctx, env, updated.Phases[0].PhaseManagedFields.NamespacedID.ID, features[0].Key, lo.ToPtr(newTCEntity.ID), nil)
		})

		t.Run("UpdateWithTaxCodeId", func(t *testing.T) {
			// Pre-create a plan without TaxConfig.
			input := newTestPlanInput(t, namespace, newTestFlatRateCard(features[0], nil, &MonthPeriod))
			input.Key = "update-taxcodeid"
			input.Name = "Update TaxCodeId"

			p, err := env.Plan.CreatePlan(ctx, input)
			require.NoError(t, err)

			tcEntity, err := env.TaxCode.GetOrCreateByAppMapping(ctx, taxcode.GetOrCreateByAppMappingInput{
				Namespace: namespace,
				AppType:   app.AppTypeStripe,
				TaxCode:   "txcd_60000002",
			})
			require.NoError(t, err)

			updatedPhases := []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{Key: "default", Name: "Default"},
					RateCards: productcatalog.RateCards{
						newTestFlatRateCard(features[0], &productcatalog.TaxConfig{
							TaxCodeID: lo.ToPtr(tcEntity.ID),
						}, &MonthPeriod),
					},
				},
			}

			updated, err := env.Plan.UpdatePlan(ctx, plan.UpdatePlanInput{
				NamespacedID: p.NamespacedID,
				Phases:       &updatedPhases,
			})
			require.NoError(t, err)

			tc := getFirstRCTaxConfig(t, updated)
			require.NotNil(t, tc)
			require.NotNil(t, tc.TaxCodeID)
			assert.Equal(t, tcEntity.ID, *tc.TaxCodeID)
			require.NotNil(t, tc.Stripe, "Stripe must be backfilled from TaxCode app mapping")
			assert.Equal(t, "txcd_60000002", tc.Stripe.Code)

			phaseID := updated.Phases[0].PhaseManagedFields.NamespacedID.ID
			assertPlanRCDBCols(t, ctx, env, phaseID, features[0].Key, lo.ToPtr(tcEntity.ID), nil)
		})

		t.Run("RemoveTaxConfig", func(t *testing.T) {
			input := newTestPlanInput(t, namespace, newTestFlatRateCard(features[0], &productcatalog.TaxConfig{
				Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_11000000"},
			}, &MonthPeriod))
			input.Key = "update-remove-tax"
			input.Name = "Update Remove Tax"

			p, err := env.Plan.CreatePlan(ctx, input)
			require.NoError(t, err)

			tc := getFirstRCTaxConfig(t, p)
			require.NotNil(t, tc)
			require.NotNil(t, tc.TaxCodeID)

			// Update to remove TaxConfig
			updatedPhases := []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:  "default",
						Name: "Default",
					},
					RateCards: productcatalog.RateCards{
						newTestFlatRateCard(features[0], nil, &MonthPeriod),
					},
				},
			}

			updated, err := env.Plan.UpdatePlan(ctx, plan.UpdatePlanInput{
				NamespacedID: p.NamespacedID,
				Phases:       &updatedPhases,
			})
			require.NoError(t, err)

			tc = getFirstRCTaxConfig(t, updated)
			assert.Nil(t, tc, "TaxConfig should be nil after removal")

			// TaxCode entity should still exist (orphaned, not deleted)
			_, err = findTaxCodeByStripeCode(t, ctx, env.TaxCode, namespace, "txcd_11000000")
			assert.NoError(t, err, "TaxCode entity should not be deleted")

			assertPlanRCDBCols(t, ctx, env, updated.Phases[0].PhaseManagedFields.NamespacedID.ID, features[0].Key, nil, nil)
		})

		t.Run("MetadataOnlyUpdate", func(t *testing.T) {
			input := newTestPlanInput(t, namespace, newTestFlatRateCard(features[0], &productcatalog.TaxConfig{
				Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_12000000"},
			}, &MonthPeriod))
			input.Key = "update-metadata-only"
			input.Name = "Update Metadata Only"

			p, err := env.Plan.CreatePlan(ctx, input)
			require.NoError(t, err)

			tc := getFirstRCTaxConfig(t, p)
			require.NotNil(t, tc)
			require.NotNil(t, tc.TaxCodeID)
			originalTaxCodeID := *tc.TaxCodeID

			// Update only plan name, no phases
			updated, err := env.Plan.UpdatePlan(ctx, plan.UpdatePlanInput{
				NamespacedID: p.NamespacedID,
				Name:         lo.ToPtr("Updated Name"),
			})
			require.NoError(t, err)

			tc = getFirstRCTaxConfig(t, updated)
			require.NotNil(t, tc)
			require.NotNil(t, tc.TaxCodeID)
			assert.Equal(t, originalTaxCodeID, *tc.TaxCodeID, "TaxCodeID should be unchanged on metadata-only update")

			assertPlanRCDBCols(t, ctx, env, updated.Phases[0].PhaseManagedFields.NamespacedID.ID, features[0].Key, lo.ToPtr(originalTaxCodeID), nil)
		})
	})

	t.Run("ReadBackVerification", func(t *testing.T) {
		t.Run("BackfillFromNewColumns", func(t *testing.T) {
			// Create plan with full TaxConfig (Stripe + Behavior)
			input := newTestPlanInput(t, namespace, newTestFlatRateCard(features[0], &productcatalog.TaxConfig{
				Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
				Stripe:   &productcatalog.StripeTaxConfig{Code: "txcd_13000000"},
			}, &MonthPeriod))
			input.Key = "readback-verify"
			input.Name = "Readback Verify"

			p, err := env.Plan.CreatePlan(ctx, input)
			require.NoError(t, err)

			// Re-read the plan
			fetched, err := env.Plan.GetPlan(ctx, plan.GetPlanInput{
				NamespacedID: p.NamespacedID,
			})
			require.NoError(t, err)

			tc := getFirstRCTaxConfig(t, fetched)
			require.NotNil(t, tc)

			// Behavior should be present (from JSONB or backfilled from tax_behavior column)
			require.NotNil(t, tc.Behavior)
			assert.Equal(t, productcatalog.ExclusiveTaxBehavior, *tc.Behavior)

			// Stripe should be present (from JSONB or backfilled from TaxCode entity)
			require.NotNil(t, tc.Stripe)
			assert.Equal(t, "txcd_13000000", tc.Stripe.Code)

			tcEntity13, err := findTaxCodeByStripeCode(t, ctx, env.TaxCode, namespace, "txcd_13000000")
			require.NoError(t, err)
			assertPlanRCDBCols(t, ctx, env, fetched.Phases[0].PhaseManagedFields.NamespacedID.ID, features[0].Key, lo.ToPtr(tcEntity13.ID), lo.ToPtr(productcatalog.ExclusiveTaxBehavior))
		})
	})
}

func TestPlanTaxCodeBackfill(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := pctestutils.NewTestEnv(t)
	t.Cleanup(func() { env.Close(t) })
	env.DBSchemaMigrate(t)

	namespace := pctestutils.NewTestNamespace(t)
	MonthPeriod := datetime.MustParseDuration(t, "P1M")

	// Setup meters and features
	err := env.Meter.ReplaceMeters(ctx, pctestutils.NewTestMeters(t, namespace))
	require.NoError(t, err)

	result, err := env.Meter.ListMeters(ctx, meter.ListMetersParams{
		Page:      pagination.Page{PageSize: 1000, PageNumber: 1},
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

	t.Run("BackfillFromDedicatedColumns", func(t *testing.T) {
		// Create a plan via service to get a phase ID
		input := newTestPlanInput(t, namespace, newTestFlatRateCard(features[0], nil, &MonthPeriod))
		input.Key = "backfill-plan-test"
		input.Name = "Backfill Plan Test"

		p, err := env.Plan.CreatePlan(ctx, input)
		require.NoError(t, err)
		require.NotEmpty(t, p.Phases)

		phaseID := p.Phases[0].PhaseManagedFields.NamespacedID.ID

		// Create TaxCode entity directly (bypassing service)
		tcEntity, err := env.Client.TaxCode.Create().
			SetNamespace(namespace).
			SetKey("stripe_txcd_99000001").
			SetName("txcd_99000001").
			SetMetadata(map[string]string{}).
			SetAppMappings(&taxcode.TaxCodeAppMappings{
				{AppType: app.AppTypeStripe, TaxCode: "txcd_99000001"},
			}).
			Save(ctx)
		require.NoError(t, err)

		// Insert a PlanRateCard row directly — no tax_config JSONB, only dedicated columns
		behavior := productcatalog.ExclusiveTaxBehavior
		_, err = env.Client.PlanRateCard.Create().
			SetPhaseID(phaseID).
			SetNamespace(namespace).
			SetKey("backfill-rc").
			SetType(productcatalog.FlatFeeRateCardType).
			SetName("Backfill RC").
			SetMetadata(map[string]string{}).
			SetEntitlementTemplate(nil).
			SetDiscounts(nil).
			SetTaxCodeID(tcEntity.ID).
			SetTaxBehavior(behavior).
			Save(ctx)
		require.NoError(t, err)

		// Read via service — adapter must backfill TaxConfig from dedicated columns
		fetched, err := env.Plan.GetPlan(ctx, plan.GetPlanInput{
			NamespacedID: p.NamespacedID,
		})
		require.NoError(t, err)

		var backfillRC productcatalog.RateCard
		for _, rc := range fetched.Phases[0].RateCards {
			if rc.AsMeta().Key == "backfill-rc" {
				backfillRC = rc
				break
			}
		}
		require.NotNil(t, backfillRC, "backfill rate card must be present in plan")

		tc := backfillRC.AsMeta().TaxConfig
		require.NotNil(t, tc, "TaxConfig must be backfilled from dedicated columns")
		require.NotNil(t, tc.Stripe, "Stripe code must be backfilled from TaxCode entity")
		assert.Equal(t, "txcd_99000001", tc.Stripe.Code)
		require.NotNil(t, tc.Behavior, "Behavior must be backfilled from tax_behavior column")
		assert.Equal(t, productcatalog.ExclusiveTaxBehavior, *tc.Behavior)
		require.NotNil(t, tc.TaxCodeID, "TaxCodeID must be backfilled from TaxCode entity")
		assert.Equal(t, tcEntity.ID, *tc.TaxCodeID)
	})

	t.Run("BackfillTaxCodeOnly", func(t *testing.T) {
		input := newTestPlanInput(t, namespace, newTestFlatRateCard(features[0], nil, &MonthPeriod))
		input.Key = "backfill-taxcode-only"
		input.Name = "Backfill TaxCode Only"

		p, err := env.Plan.CreatePlan(ctx, input)
		require.NoError(t, err)

		phaseID := p.Phases[0].PhaseManagedFields.NamespacedID.ID

		tcEntity, err := env.Client.TaxCode.Create().
			SetNamespace(namespace).
			SetKey("stripe_txcd_99000010").
			SetName("txcd_99000010").
			SetMetadata(map[string]string{}).
			SetAppMappings(&taxcode.TaxCodeAppMappings{
				{AppType: app.AppTypeStripe, TaxCode: "txcd_99000010"},
			}).
			Save(ctx)
		require.NoError(t, err)

		// Only tax_code_id, no tax_behavior
		_, err = env.Client.PlanRateCard.Create().
			SetPhaseID(phaseID).
			SetNamespace(namespace).
			SetKey("backfill-tc-only").
			SetType(productcatalog.FlatFeeRateCardType).
			SetName("Backfill TC Only").
			SetMetadata(map[string]string{}).
			SetEntitlementTemplate(nil).
			SetDiscounts(nil).
			SetTaxCodeID(tcEntity.ID).
			Save(ctx)
		require.NoError(t, err)

		fetched, err := env.Plan.GetPlan(ctx, plan.GetPlanInput{NamespacedID: p.NamespacedID})
		require.NoError(t, err)

		var rc productcatalog.RateCard
		for _, r := range fetched.Phases[0].RateCards {
			if r.AsMeta().Key == "backfill-tc-only" {
				rc = r
				break
			}
		}
		require.NotNil(t, rc)

		tc := rc.AsMeta().TaxConfig
		require.NotNil(t, tc, "TaxConfig must be backfilled from TaxCode entity alone")
		require.NotNil(t, tc.Stripe)
		assert.Equal(t, "txcd_99000010", tc.Stripe.Code)
		assert.Nil(t, tc.Behavior, "Behavior must be nil when tax_behavior column is not set")
		require.NotNil(t, tc.TaxCodeID, "TaxCodeID must be backfilled from TaxCode entity")
		assert.Equal(t, tcEntity.ID, *tc.TaxCodeID)
	})

	t.Run("BackfillBehaviorOnly", func(t *testing.T) {
		input := newTestPlanInput(t, namespace, newTestFlatRateCard(features[0], nil, &MonthPeriod))
		input.Key = "backfill-behavior-only"
		input.Name = "Backfill Behavior Only"

		p, err := env.Plan.CreatePlan(ctx, input)
		require.NoError(t, err)

		phaseID := p.Phases[0].PhaseManagedFields.NamespacedID.ID

		// Only tax_behavior, no tax_code_id
		behavior := productcatalog.InclusiveTaxBehavior
		_, err = env.Client.PlanRateCard.Create().
			SetPhaseID(phaseID).
			SetNamespace(namespace).
			SetKey("backfill-beh-only").
			SetType(productcatalog.FlatFeeRateCardType).
			SetName("Backfill Behavior Only").
			SetMetadata(map[string]string{}).
			SetEntitlementTemplate(nil).
			SetDiscounts(nil).
			SetTaxBehavior(behavior).
			Save(ctx)
		require.NoError(t, err)

		fetched, err := env.Plan.GetPlan(ctx, plan.GetPlanInput{NamespacedID: p.NamespacedID})
		require.NoError(t, err)

		var rc productcatalog.RateCard
		for _, r := range fetched.Phases[0].RateCards {
			if r.AsMeta().Key == "backfill-beh-only" {
				rc = r
				break
			}
		}
		require.NotNil(t, rc)

		tc := rc.AsMeta().TaxConfig
		require.NotNil(t, tc, "TaxConfig must be backfilled from tax_behavior column alone")
		require.NotNil(t, tc.Behavior)
		assert.Equal(t, productcatalog.InclusiveTaxBehavior, *tc.Behavior)
		assert.Nil(t, tc.Stripe, "Stripe must be nil when no TaxCode entity is linked")
	})
}

func TestPlanWithAddonTaxCode(t *testing.T) {
	MonthPeriod := datetime.MustParseDuration(t, "P1M")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := pctestutils.NewTestEnv(t)
	t.Cleanup(func() { env.Close(t) })
	env.DBSchemaMigrate(t)

	namespace := pctestutils.NewTestNamespace(t)

	// Setup meters and features
	err := env.Meter.ReplaceMeters(ctx, pctestutils.NewTestMeters(t, namespace))
	require.NoError(t, err)

	result, err := env.Meter.ListMeters(ctx, meter.ListMetersParams{
		Page:      pagination.Page{PageSize: 1000, PageNumber: 1},
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

	t.Run("EmbeddedAddonTaxCodeInPlanResponse", func(t *testing.T) {
		// Create an addon with a Stripe tax code
		addonInput := pctestutils.NewTestAddon(t, namespace, newTestFlatRateCard(features[0], &productcatalog.TaxConfig{
			Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
			Stripe:   &productcatalog.StripeTaxConfig{Code: "txcd_99000003"},
		}, &MonthPeriod))
		addonInput.Key = "tax-addon-for-plan"
		addonInput.Name = "Tax Addon For Plan"

		a, err := env.Addon.CreateAddon(ctx, addonInput)
		require.NoError(t, err)

		// Publish the addon (must be active to attach to a plan)
		publishAt := time.Now().Truncate(time.Microsecond)
		a, err = env.Addon.PublishAddon(ctx, addon.PublishAddonInput{
			NamespacedID:    a.NamespacedID,
			EffectivePeriod: productcatalog.EffectivePeriod{EffectiveFrom: &publishAt},
		})
		require.NoError(t, err)

		// Create a plan
		planInput := newTestPlanInput(t, namespace, newTestFlatRateCard(features[0], nil, &MonthPeriod))
		planInput.Key = "plan-with-tax-addon"
		planInput.Name = "Plan With Tax Addon"

		p, err := env.Plan.CreatePlan(ctx, planInput)
		require.NoError(t, err)
		require.NotEmpty(t, p.Phases)

		// Attach addon to plan
		_, err = env.PlanAddon.CreatePlanAddon(ctx, planaddon.CreatePlanAddonInput{
			NamespacedModel: models.NamespacedModel{Namespace: namespace},
			PlanID:          p.ID,
			AddonID:         a.ID,
			FromPlanPhase:   p.Phases[0].Key,
		})
		require.NoError(t, err)

		// Read plan with addons expanded
		fetched, err := env.Plan.GetPlan(ctx, plan.GetPlanInput{
			NamespacedID: p.NamespacedID,
			Expand:       plan.ExpandFields{PlanAddons: true},
		})
		require.NoError(t, err)
		require.NotNil(t, fetched.Addons, "Addons must be expanded")
		require.Len(t, *fetched.Addons, 1)

		addonInPlan := (*fetched.Addons)[0]
		require.NotEmpty(t, addonInPlan.RateCards, "addon rate cards must be present in plan response")

		var found bool
		for _, rc := range addonInPlan.RateCards {
			if rc.AsMeta().Key == features[0].Key {
				tc := rc.AsMeta().TaxConfig
				require.NotNil(t, tc, "TaxConfig must be present on embedded addon rate card")

				require.NotNil(t, tc.Behavior)
				assert.Equal(t, productcatalog.ExclusiveTaxBehavior, *tc.Behavior)

				require.NotNil(t, tc.Stripe)
				assert.Equal(t, "txcd_99000003", tc.Stripe.Code)

				require.NotNil(t, tc.TaxCodeID, "TaxCodeID must be set on embedded addon rate card")

				found = true
				break
			}
		}
		require.True(t, found, "addon rate card with tax config must be found in plan response")
	})
}
