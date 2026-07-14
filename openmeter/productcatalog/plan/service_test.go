package plan

import (
	"slices"
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

// TestUpdatePlanInputValidateWithPlanRejectsUnitConfig covers the v1 write-loss guard: a v1 update
// replaces rate cards from a body that has no unit_config field, so the guard must reject based on
// the STORED plan (the argument), before the incoming fields are merged, or the conversion would be
// silently dropped. The flag is opt-in so the v3 path (flag off) is unaffected.
func TestUpdatePlanInputValidateWithPlanRejectsUnitConfig(t *testing.T) {
	card := func(uc *productcatalog.UnitConfig) productcatalog.RateCard {
		return &productcatalog.UsageBasedRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:        "feat-1",
				Name:       "Feature 1",
				FeatureKey: lo.ToPtr("feat-1"),
				Price:      productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: decimal.NewFromInt(1)}),
				UnitConfig: uc,
			},
		}
	}
	planWith := func(cards ...productcatalog.RateCard) productcatalog.Plan {
		return productcatalog.Plan{Phases: []productcatalog.Phase{{RateCards: cards}}}
	}
	divide := &productcatalog.UnitConfig{Operation: productcatalog.UnitConfigOperationDivide, ConversionFactor: decimal.NewFromInt(1000)}

	rejectedForUnitConfig := func(err error) bool {
		if err == nil {
			return false
		}
		issues, convErr := models.AsValidationIssues(err)
		if convErr != nil {
			return false
		}
		return slices.ContainsFunc(issues, func(i models.ValidationIssue) bool {
			return i.Code() == productcatalog.ErrCodeUnitConfigNotRepresentable
		})
	}

	t.Run("flag on + stored plan has unit_config is rejected even though the update body carries none", func(t *testing.T) {
		// The input intentionally carries no phases (a v1 body cannot express unit_config); the
		// rejection must come from the stored plan, proving the check runs before the merge.
		in := UpdatePlanInput{RejectUnitConfig: true}

		err := in.ValidateWithPlan(planWith(card(divide)))
		require.Error(t, err)
		assert.True(t, rejectedForUnitConfig(err), "expected unit_config rejection, got %v", err)
	})

	t.Run("flag off does not reject a stored unit_config plan", func(t *testing.T) {
		in := UpdatePlanInput{}

		assert.False(t, rejectedForUnitConfig(in.ValidateWithPlan(planWith(card(divide)))))
	})

	t.Run("flag on does not reject a plan without unit_config", func(t *testing.T) {
		in := UpdatePlanInput{RejectUnitConfig: true}

		assert.False(t, rejectedForUnitConfig(in.ValidateWithPlan(planWith(card(nil)))))
	})
}
