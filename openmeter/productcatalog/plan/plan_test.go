package plan

import (
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestPlanHasUnitConfig(t *testing.T) {
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
	phase := func(cards ...productcatalog.RateCard) Phase {
		return Phase{Phase: productcatalog.Phase{RateCards: cards}}
	}
	divide := &productcatalog.UnitConfig{Operation: productcatalog.UnitConfigOperationDivide, ConversionFactor: decimal.NewFromInt(1000)}

	t.Run("plan with no phases has none", func(t *testing.T) {
		assert.False(t, Plan{}.HasUnitConfig())
	})

	t.Run("no phase carries unit_config", func(t *testing.T) {
		p := Plan{Phases: []Phase{phase(card(nil)), phase(card(nil))}}
		assert.False(t, p.HasUnitConfig())
	})

	t.Run("a later phase carrying unit_config is detected", func(t *testing.T) {
		p := Plan{Phases: []Phase{phase(card(nil)), phase(card(nil), card(divide))}}
		assert.True(t, p.HasUnitConfig())
	})
}
