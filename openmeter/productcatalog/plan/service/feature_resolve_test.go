package service_test

import (
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	pctestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// TestCreatePlanResolvesFeatureKeyFromFeatureID verifies that resolveFeatures correctly
// writes the resolved FeatureKey back into the plan phases when only FeatureID is provided.
// A previous bug iterated phases by value, discarding the mutation.
func TestCreatePlanResolvesFeatureKeyFromFeatureID(t *testing.T) {
	MonthPeriod := datetime.MustParseDuration(t, "P1M")

	ctx := t.Context()

	env := pctestutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	env.DBSchemaMigrate(t)

	namespace := pctestutils.NewTestNamespace(t)

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

	// Use the first metered feature (api_requests_total).
	m := result.Items[0]
	feat, err := env.Feature.CreateFeature(ctx, pctestutils.NewTestFeatureFromMeter(t, &m))
	require.NoError(t, err)

	t.Run("KeyMismatchIsRejected", func(t *testing.T) {
		// Rate-card Key deliberately differs from the feature's Key.
		// After resolveFeatures backfills FeatureKey from the feature, post-resolve
		// validation must fire ErrRateCardKeyFeatureKeyMismatch and reject the plan.
		rc := &productcatalog.UsageBasedRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:       "different-key",
				Name:      "Usage rate card",
				FeatureID: lo.ToPtr(feat.ID),
				Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: decimal.NewFromInt(1),
				}),
			},
			BillingCadence: MonthPeriod,
		}

		input := pctestutils.NewTestPlan(t, namespace, productcatalog.Phase{
			PhaseMeta: productcatalog.PhaseMeta{Key: "default", Name: "Default"},
			RateCards: productcatalog.RateCards{rc},
		})
		input.Key = "mismatch-plan"
		input.Name = "Mismatch Plan"

		_, err := env.Plan.CreatePlan(ctx, input)
		require.Error(t, err, "plan with rate-card key != feature key must be rejected after resolution")
		require.True(t, models.IsGenericValidationError(err), "error must be a validation error, got: %v", err)
	})

	t.Run("FeatureKeyIsBackfilledAndPersisted", func(t *testing.T) {
		// Rate-card Key == feature Key so validation passes after resolution.
		// Only FeatureID is supplied; FeatureKey must be populated by resolveFeatures.
		rc := &productcatalog.UsageBasedRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:       feat.Key,
				Name:      "Usage rate card",
				FeatureID: lo.ToPtr(feat.ID),
				// FeatureKey intentionally nil — should be resolved from FeatureID.
				Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: decimal.NewFromInt(1),
				}),
			},
			BillingCadence: MonthPeriod,
		}

		input := pctestutils.NewTestPlan(t, namespace, productcatalog.Phase{
			PhaseMeta: productcatalog.PhaseMeta{Key: "default", Name: "Default"},
			RateCards: productcatalog.RateCards{rc},
		})
		input.Key = "feature-id-only-plan"
		input.Name = "Feature ID Only Plan"

		p, err := env.Plan.CreatePlan(ctx, input)
		require.NoError(t, err)
		require.NotEmpty(t, p.Phases)
		require.NotEmpty(t, p.Phases[0].RateCards)

		gotKey := p.Phases[0].RateCards[0].AsMeta().FeatureKey
		require.NotNil(t, gotKey, "FeatureKey must be populated after resolution")
		require.Equal(t, feat.Key, *gotKey, "resolved FeatureKey must equal the feature's Key")
	})
}
