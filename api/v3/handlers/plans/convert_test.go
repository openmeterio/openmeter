package plans

import (
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

func newTestPlan(t *testing.T) plan.Plan {
	t.Helper()

	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	billingCadence, err := datetime.ISODurationString("P1M").Parse()
	require.NoError(t, err)

	return plan.Plan{
		NamespacedID: models.NamespacedID{
			Namespace: "test-ns",
			ID:        "01J8GFKQ0000000000000000",
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: now,
			UpdatedAt: now,
		},
		PlanMeta: productcatalog.PlanMeta{
			Key:            "pro",
			Version:        1,
			Name:           "Pro Plan",
			Currency:       currency.USD,
			BillingCadence: billingCadence,
			ProRatingConfig: productcatalog.ProRatingConfig{
				Enabled: true,
				Mode:    productcatalog.ProRatingModeProratePrices,
			},
		},
		Phases: []plan.Phase{},
	}
}

func TestFromPlan(t *testing.T) {
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("maps scalar fields correctly", func(t *testing.T) {
		p := newTestPlan(t)

		result, err := ToAPIBillingPlan(p)
		require.NoError(t, err)

		assert.Equal(t, "01J8GFKQ0000000000000000", result.Id)
		assert.Equal(t, api.ResourceKey("pro"), result.Key)
		assert.Equal(t, "Pro Plan", result.Name)
		assert.Equal(t, api.CurrencyCode("USD"), result.Currency)
		assert.Equal(t, api.ISO8601Duration("P1M"), result.BillingCadence)
		assert.Equal(t, 1, result.Version)
		assert.Equal(t, now, result.CreatedAt)
		assert.Equal(t, now, result.UpdatedAt)
		assert.Nil(t, result.DeletedAt)
		assert.Nil(t, result.Description)
	})

	t.Run("maps optional description", func(t *testing.T) {
		p := newTestPlan(t)
		p.Description = lo.ToPtr("A great plan")

		result, err := ToAPIBillingPlan(p)
		require.NoError(t, err)
		require.NotNil(t, result.Description)
		assert.Equal(t, "A great plan", *result.Description)
	})

	t.Run("maps deleted at", func(t *testing.T) {
		p := newTestPlan(t)
		deleted := now.Add(-time.Hour)
		p.DeletedAt = &deleted

		result, err := ToAPIBillingPlan(p)
		require.NoError(t, err)
		require.NotNil(t, result.DeletedAt)
		assert.Equal(t, deleted, *result.DeletedAt)
	})

	t.Run("pro rating enabled maps correctly", func(t *testing.T) {
		p := newTestPlan(t)
		p.ProRatingConfig.Enabled = false

		result, err := ToAPIBillingPlan(p)
		require.NoError(t, err)
		require.NotNil(t, result.ProRatingEnabled)
		assert.False(t, *result.ProRatingEnabled)
	})

	t.Run("effective period maps correctly", func(t *testing.T) {
		p := newTestPlan(t)
		from := now.Add(-24 * time.Hour)
		to := now.Add(30 * 24 * time.Hour)
		p.EffectiveFrom = &from
		p.EffectiveTo = &to

		result, err := ToAPIBillingPlan(p)
		require.NoError(t, err)
		require.NotNil(t, result.EffectiveFrom)
		assert.Equal(t, from, *result.EffectiveFrom)
		require.NotNil(t, result.EffectiveTo)
		assert.Equal(t, to, *result.EffectiveTo)
	})

	t.Run("empty phases produces empty slice", func(t *testing.T) {
		p := newTestPlan(t)

		result, err := ToAPIBillingPlan(p)
		require.NoError(t, err)
		assert.Empty(t, result.Phases)
	})
}

func TestFromPlanStatus(t *testing.T) {
	now := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	clock.SetTime(now)
	defer clock.ResetTime()

	tests := []struct {
		name          string
		effectiveFrom *time.Time
		effectiveTo   *time.Time
		wantStatus    api.BillingPlanStatus
	}{
		{
			name:       "draft — no effective dates",
			wantStatus: api.BillingPlanStatusDraft,
		},
		{
			name:          "active — started in past, no end",
			effectiveFrom: lo.ToPtr(now.Add(-time.Hour)),
			wantStatus:    api.BillingPlanStatusActive,
		},
		{
			name:          "scheduled — starts in future",
			effectiveFrom: lo.ToPtr(now.Add(24 * time.Hour)),
			wantStatus:    api.BillingPlanStatusScheduled,
		},
		{
			name:          "archived — both dates in past",
			effectiveFrom: lo.ToPtr(now.Add(-48 * time.Hour)),
			effectiveTo:   lo.ToPtr(now.Add(-time.Hour)),
			wantStatus:    api.BillingPlanStatusArchived,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestPlan(t)
			p.EffectiveFrom = tt.effectiveFrom
			p.EffectiveTo = tt.effectiveTo

			result, err := ToAPIBillingPlan(p)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, result.Status)
		})
	}
}

func TestFromPlanPhase(t *testing.T) {
	t.Run("maps phase fields", func(t *testing.T) {
		duration, err := datetime.ISODurationString("P3M").Parse()
		require.NoError(t, err)

		phase := plan.Phase{
			Phase: productcatalog.Phase{
				PhaseMeta: productcatalog.PhaseMeta{
					Key:         "trial",
					Name:        "Trial Phase",
					Description: lo.ToPtr("First 3 months"),
					Duration:    lo.ToPtr(duration),
				},
			},
		}

		result, err := ToAPIBillingPlanPhase(phase)
		require.NoError(t, err)

		assert.Equal(t, api.ResourceKey("trial"), result.Key)
		assert.Equal(t, "Trial Phase", result.Name)
		require.NotNil(t, result.Description)
		assert.Equal(t, "First 3 months", *result.Description)
		require.NotNil(t, result.Duration)
		assert.Equal(t, api.ISO8601Duration("P3M"), *result.Duration)
	})

	t.Run("nil duration maps to nil", func(t *testing.T) {
		phase := plan.Phase{
			Phase: productcatalog.Phase{
				PhaseMeta: productcatalog.PhaseMeta{
					Key:  "forever",
					Name: "Forever Phase",
				},
			},
		}

		result, err := ToAPIBillingPlanPhase(phase)
		require.NoError(t, err)
		assert.Nil(t, result.Duration)
	})

	t.Run("phase produces empty rate cards slice", func(t *testing.T) {
		phase := plan.Phase{
			Phase: productcatalog.Phase{
				PhaseMeta: productcatalog.PhaseMeta{Key: "p1", Name: "Phase 1"},
			},
		}

		result, err := ToAPIBillingPlanPhase(phase)
		require.NoError(t, err)
		assert.NotNil(t, result.RateCards)
		assert.Empty(t, result.RateCards)
	})
}

func TestFromValidationErrors(t *testing.T) {
	t.Run("nil when no issues", func(t *testing.T) {
		result := ToAPIProductCatalogValidationErrors(nil)
		assert.Nil(t, result)
	})

	t.Run("nil when empty issues", func(t *testing.T) {
		result := ToAPIProductCatalogValidationErrors(models.ValidationIssues{})
		assert.Nil(t, result)
	})

	t.Run("maps issues to errors", func(t *testing.T) {
		issues := models.ValidationIssues{
			models.NewValidationIssue("plan.missing_phase", "plan must have at least one phase"),
		}

		result := ToAPIProductCatalogValidationErrors(issues)
		require.NotNil(t, result)
		require.Len(t, *result, 1)
		assert.Equal(t, "plan.missing_phase", (*result)[0].Code)
		assert.Equal(t, "plan must have at least one phase", (*result)[0].Message)
	})

	t.Run("maps multiple issues", func(t *testing.T) {
		issues := models.ValidationIssues{
			models.NewValidationIssue("err.one", "first error"),
			models.NewValidationIssue("err.two", "second error"),
		}

		result := ToAPIProductCatalogValidationErrors(issues)
		require.NotNil(t, result)
		assert.Len(t, *result, 2)
	})
}

func TestFromPlanWithPhases(t *testing.T) {
	t.Run("multiple phases are all converted", func(t *testing.T) {
		p := newTestPlan(t)
		p.Phases = []plan.Phase{
			{Phase: productcatalog.Phase{PhaseMeta: productcatalog.PhaseMeta{Key: "p1", Name: "Phase 1"}}},
			{Phase: productcatalog.Phase{PhaseMeta: productcatalog.PhaseMeta{Key: "p2", Name: "Phase 2"}}},
		}

		result, err := ToAPIBillingPlan(p)
		require.NoError(t, err)
		require.Len(t, result.Phases, 2)
		assert.Equal(t, api.ResourceKey("p1"), result.Phases[0].Key)
		assert.Equal(t, api.ResourceKey("p2"), result.Phases[1].Key)
	})

	t.Run("phase order is preserved", func(t *testing.T) {
		p := newTestPlan(t)
		p.Phases = []plan.Phase{
			{Phase: productcatalog.Phase{PhaseMeta: productcatalog.PhaseMeta{Key: "first", Name: "First"}}},
			{Phase: productcatalog.Phase{PhaseMeta: productcatalog.PhaseMeta{Key: "second", Name: "Second"}}},
			{Phase: productcatalog.Phase{PhaseMeta: productcatalog.PhaseMeta{Key: "third", Name: "Third"}}},
		}

		result, err := ToAPIBillingPlan(p)
		require.NoError(t, err)
		require.Len(t, result.Phases, 3)
		assert.Equal(t, api.ResourceKey("first"), result.Phases[0].Key)
		assert.Equal(t, api.ResourceKey("second"), result.Phases[1].Key)
		assert.Equal(t, api.ResourceKey("third"), result.Phases[2].Key)
	})

	t.Run("nil phases maps to empty slice", func(t *testing.T) {
		p := newTestPlan(t)
		p.Phases = nil

		result, err := ToAPIBillingPlan(p)
		require.NoError(t, err)
		assert.NotNil(t, result.Phases)
		assert.Empty(t, result.Phases)
	})
}

func TestFromRateCard(t *testing.T) {
	t.Run("flat fee — no price, no cadence (one-time free)", func(t *testing.T) {
		rc := &productcatalog.FlatFeeRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:  "setup",
				Name: "Setup Fee",
			},
		}

		result, err := ToAPIBillingRateCard(rc)
		require.NoError(t, err)

		assert.Equal(t, "setup", result.Key)
		assert.Equal(t, "Setup Fee", result.Name)
		assert.Nil(t, result.BillingCadence)
		assert.Nil(t, result.Feature)
		assert.Nil(t, result.Discounts)
		assert.Nil(t, result.TaxConfig)

		disc, err := result.Price.Discriminator()
		require.NoError(t, err)
		assert.Equal(t, "free", disc)
	})

	t.Run("flat fee — with price and cadence", func(t *testing.T) {
		cadence, err := datetime.ISODurationString("P1M").Parse()
		require.NoError(t, err)

		price := productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      decimal.NewFromFloat(9.99),
			PaymentTerm: productcatalog.InAdvancePaymentTerm,
		})

		rc := &productcatalog.FlatFeeRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:   "base",
				Name:  "Base Fee",
				Price: price,
			},
			BillingCadence: &cadence,
		}

		result, err := ToAPIBillingRateCard(rc)
		require.NoError(t, err)

		require.NotNil(t, result.BillingCadence)
		assert.Equal(t, api.ISO8601Duration("P1M"), *result.BillingCadence)
		require.NotNil(t, result.PaymentTerm)
		assert.Equal(t, api.BillingPricePaymentTerm("in_advance"), *result.PaymentTerm)

		disc, err := result.Price.Discriminator()
		require.NoError(t, err)
		assert.Equal(t, "flat", disc)

		flat, err := result.Price.AsBillingPriceFlat()
		require.NoError(t, err)
		assert.Equal(t, api.Numeric("9.99"), flat.Amount)
	})

	t.Run("usage based — with unit price", func(t *testing.T) {
		cadence, err := datetime.ISODurationString("P1M").Parse()
		require.NoError(t, err)

		minAmt := decimal.NewFromFloat(10)
		maxAmt := decimal.NewFromFloat(100)
		price := productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: decimal.NewFromFloat(0.05),
			Commitments: productcatalog.Commitments{
				MinimumAmount: &minAmt,
				MaximumAmount: &maxAmt,
			},
		})

		rc := &productcatalog.UsageBasedRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:   "api-calls",
				Name:  "API Calls",
				Price: price,
			},
			BillingCadence: cadence,
		}

		result, err := ToAPIBillingRateCard(rc)
		require.NoError(t, err)

		require.NotNil(t, result.BillingCadence)
		assert.Equal(t, api.ISO8601Duration("P1M"), *result.BillingCadence)

		require.NotNil(t, result.Commitments)
		assert.Equal(t, lo.ToPtr(api.Numeric("10")), result.Commitments.MinimumAmount)
		assert.Equal(t, lo.ToPtr(api.Numeric("100")), result.Commitments.MaximumAmount)

		disc, err := result.Price.Discriminator()
		require.NoError(t, err)
		assert.Equal(t, "unit", disc)

		unit, err := result.Price.AsBillingPriceUnit()
		require.NoError(t, err)
		assert.Equal(t, api.Numeric("0.05"), unit.Amount)
	})

	t.Run("usage based — with feature ID", func(t *testing.T) {
		cadence, err := datetime.ISODurationString("P1M").Parse()
		require.NoError(t, err)

		featureID := "01J8FEATURE000000000000000"
		rc := &productcatalog.UsageBasedRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:       "tokens",
				Name:      "Tokens",
				FeatureID: &featureID,
			},
			BillingCadence: cadence,
		}

		result, err := ToAPIBillingRateCard(rc)
		require.NoError(t, err)
		require.NotNil(t, result.Feature)
		assert.Equal(t, featureID, result.Feature.Id)
	})

	t.Run("usage based — no feature ID, no feature reference", func(t *testing.T) {
		cadence, err := datetime.ISODurationString("P1M").Parse()
		require.NoError(t, err)

		rc := &productcatalog.UsageBasedRateCard{
			RateCardMeta:   productcatalog.RateCardMeta{Key: "rc", Name: "RC"},
			BillingCadence: cadence,
		}

		result, err := ToAPIBillingRateCard(rc)
		require.NoError(t, err)
		assert.Nil(t, result.Feature)
	})
}

func TestFromBillingPrice(t *testing.T) {
	t.Run("nil price maps to free", func(t *testing.T) {
		result, err := ToAPIBillingPrice(nil)
		require.NoError(t, err)

		disc, err := result.Discriminator()
		require.NoError(t, err)
		assert.Equal(t, "free", disc)
	})

	t.Run("flat price", func(t *testing.T) {
		p := productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount: decimal.NewFromFloat(5.00),
		})

		result, err := ToAPIBillingPrice(p)
		require.NoError(t, err)

		disc, err := result.Discriminator()
		require.NoError(t, err)
		assert.Equal(t, "flat", disc)

		flat, err := result.AsBillingPriceFlat()
		require.NoError(t, err)
		assert.Equal(t, api.Numeric("5"), flat.Amount)
	})

	t.Run("unit price", func(t *testing.T) {
		p := productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: decimal.NewFromFloat(0.001),
		})

		result, err := ToAPIBillingPrice(p)
		require.NoError(t, err)

		disc, err := result.Discriminator()
		require.NoError(t, err)
		assert.Equal(t, "unit", disc)

		unit, err := result.AsBillingPriceUnit()
		require.NoError(t, err)
		assert.Equal(t, api.Numeric("0.001"), unit.Amount)
	})

	t.Run("graduated tiered price", func(t *testing.T) {
		upTo := decimal.NewFromInt(1000)
		p := productcatalog.NewPriceFrom(productcatalog.TieredPrice{
			Mode: productcatalog.GraduatedTieredPrice,
			Tiers: []productcatalog.PriceTier{
				{
					UpToAmount: &upTo,
					UnitPrice:  &productcatalog.PriceTierUnitPrice{Amount: decimal.NewFromFloat(0.01)},
				},
				{
					UnitPrice: &productcatalog.PriceTierUnitPrice{Amount: decimal.NewFromFloat(0.005)},
				},
			},
		})

		result, err := ToAPIBillingPrice(p)
		require.NoError(t, err)

		disc, err := result.Discriminator()
		require.NoError(t, err)
		assert.Equal(t, "graduated", disc)

		graduated, err := result.AsBillingPriceGraduated()
		require.NoError(t, err)
		require.Len(t, graduated.Tiers, 2)
		assert.Equal(t, lo.ToPtr(api.Numeric("1000")), graduated.Tiers[0].UpToAmount)
		assert.Nil(t, graduated.Tiers[1].UpToAmount)
	})

	t.Run("volume tiered price", func(t *testing.T) {
		p := productcatalog.NewPriceFrom(productcatalog.TieredPrice{
			Mode:  productcatalog.VolumeTieredPrice,
			Tiers: []productcatalog.PriceTier{},
		})

		result, err := ToAPIBillingPrice(p)
		require.NoError(t, err)

		disc, err := result.Discriminator()
		require.NoError(t, err)
		assert.Equal(t, "volume", disc)
	})

	t.Run("dynamic price translates to unit price of amount 1", func(t *testing.T) {
		p := productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
			Multiplier: decimal.NewFromFloat(1.2),
		})

		result, err := ToAPIBillingPrice(p)
		require.NoError(t, err)

		disc, err := result.Discriminator()
		require.NoError(t, err)
		assert.Equal(t, "unit", disc)

		unit, err := result.AsBillingPriceUnit()
		require.NoError(t, err)
		assert.Equal(t, api.Numeric("1"), unit.Amount)
	})

	t.Run("package price translates to unit price with package amount", func(t *testing.T) {
		p := productcatalog.NewPriceFrom(productcatalog.PackagePrice{
			Amount:             decimal.NewFromFloat(0.5),
			QuantityPerPackage: decimal.NewFromInt(1000),
		})

		result, err := ToAPIBillingPrice(p)
		require.NoError(t, err)

		disc, err := result.Discriminator()
		require.NoError(t, err)
		assert.Equal(t, "unit", disc)

		unit, err := result.AsBillingPriceUnit()
		require.NoError(t, err)
		assert.Equal(t, api.Numeric("0.5"), unit.Amount)
	})
}

func TestToAPIBillingRateCardUnitConfig(t *testing.T) {
	t.Run("nil price has no unit config", func(t *testing.T) {
		result, err := ToAPIBillingRateCardUnitConfig(nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("flat price has no unit config", func(t *testing.T) {
		p := productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: decimal.NewFromFloat(5)})

		result, err := ToAPIBillingRateCardUnitConfig(p)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("unit price has no unit config", func(t *testing.T) {
		p := productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: decimal.NewFromFloat(0.05)})

		result, err := ToAPIBillingRateCardUnitConfig(p)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("dynamic price produces multiply unit config", func(t *testing.T) {
		p := productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
			Multiplier: decimal.NewFromFloat(1.2),
		})

		result, err := ToAPIBillingRateCardUnitConfig(p)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, api.BillingUnitConfigOperationMultiply, result.Operation)
		assert.Equal(t, api.Numeric("1.2"), result.ConversionFactor)
		assert.Nil(t, result.Rounding)
		assert.Nil(t, result.Precision)
		assert.Nil(t, result.DisplayUnit)
	})

	t.Run("package price produces divide unit config with ceiling rounding", func(t *testing.T) {
		p := productcatalog.NewPriceFrom(productcatalog.PackagePrice{
			Amount:             decimal.NewFromFloat(10),
			QuantityPerPackage: decimal.NewFromInt(1000),
		})

		result, err := ToAPIBillingRateCardUnitConfig(p)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, api.BillingUnitConfigOperationDivide, result.Operation)
		assert.Equal(t, api.Numeric("1000"), result.ConversionFactor)
		require.NotNil(t, result.Rounding)
		assert.Equal(t, api.BillingUnitConfigRoundingModeCeiling, *result.Rounding)
	})
}

func TestFromRateCard_DynamicAndPackagePrices(t *testing.T) {
	cadence, err := datetime.ISODurationString("P1M").Parse()
	require.NoError(t, err)

	t.Run("dynamic price renders as unit price plus multiply unit config and preserves commitments", func(t *testing.T) {
		minAmt := decimal.NewFromFloat(10)
		maxAmt := decimal.NewFromFloat(100)
		price := productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
			Multiplier: decimal.NewFromFloat(1.2),
			Commitments: productcatalog.Commitments{
				MinimumAmount: &minAmt,
				MaximumAmount: &maxAmt,
			},
		})

		rc := &productcatalog.UsageBasedRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:   "tokens",
				Name:  "Tokens",
				Price: price,
			},
			BillingCadence: cadence,
		}

		result, err := ToAPIBillingRateCard(rc)
		require.NoError(t, err)

		disc, err := result.Price.Discriminator()
		require.NoError(t, err)
		assert.Equal(t, "unit", disc)

		unit, err := result.Price.AsBillingPriceUnit()
		require.NoError(t, err)
		assert.Equal(t, api.Numeric("1"), unit.Amount)

		require.NotNil(t, result.UnitConfig)
		assert.Equal(t, api.BillingUnitConfigOperationMultiply, result.UnitConfig.Operation)
		assert.Equal(t, api.Numeric("1.2"), result.UnitConfig.ConversionFactor)
		assert.Nil(t, result.UnitConfig.Rounding)

		require.NotNil(t, result.Commitments)
		assert.Equal(t, lo.ToPtr(api.Numeric("10")), result.Commitments.MinimumAmount)
		assert.Equal(t, lo.ToPtr(api.Numeric("100")), result.Commitments.MaximumAmount)
	})

	t.Run("package price renders as unit price plus divide unit config and preserves commitments", func(t *testing.T) {
		minAmt := decimal.NewFromFloat(5)
		price := productcatalog.NewPriceFrom(productcatalog.PackagePrice{
			Amount:             decimal.NewFromFloat(0.5),
			QuantityPerPackage: decimal.NewFromInt(1000),
			Commitments: productcatalog.Commitments{
				MinimumAmount: &minAmt,
			},
		})

		rc := &productcatalog.UsageBasedRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:   "api-calls",
				Name:  "API Calls",
				Price: price,
			},
			BillingCadence: cadence,
		}

		result, err := ToAPIBillingRateCard(rc)
		require.NoError(t, err)

		disc, err := result.Price.Discriminator()
		require.NoError(t, err)
		assert.Equal(t, "unit", disc)

		unit, err := result.Price.AsBillingPriceUnit()
		require.NoError(t, err)
		assert.Equal(t, api.Numeric("0.5"), unit.Amount)

		require.NotNil(t, result.UnitConfig)
		assert.Equal(t, api.BillingUnitConfigOperationDivide, result.UnitConfig.Operation)
		assert.Equal(t, api.Numeric("1000"), result.UnitConfig.ConversionFactor)
		require.NotNil(t, result.UnitConfig.Rounding)
		assert.Equal(t, api.BillingUnitConfigRoundingModeCeiling, *result.UnitConfig.Rounding)

		require.NotNil(t, result.Commitments)
		assert.Equal(t, lo.ToPtr(api.Numeric("5")), result.Commitments.MinimumAmount)
		assert.Nil(t, result.Commitments.MaximumAmount)
	})

	t.Run("unit price has no unit config on rate card", func(t *testing.T) {
		price := productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: decimal.NewFromFloat(0.05),
		})

		rc := &productcatalog.UsageBasedRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:   "api-calls",
				Name:  "API Calls",
				Price: price,
			},
			BillingCadence: cadence,
		}

		result, err := ToAPIBillingRateCard(rc)
		require.NoError(t, err)
		assert.Nil(t, result.UnitConfig)
	})
}

func TestFromBillingDiscounts(t *testing.T) {
	t.Run("nil when no discounts", func(t *testing.T) {
		result := ToAPIBillingRateCardDiscount(productcatalog.Discounts{})
		assert.Nil(t, result)
	})

	t.Run("percentage discount", func(t *testing.T) {
		result := ToAPIBillingRateCardDiscount(productcatalog.Discounts{
			Percentage: &productcatalog.PercentageDiscount{
				Percentage: models.NewPercentage(20),
			},
		})

		require.NotNil(t, result)
		require.NotNil(t, result.Percentage)
		assert.InDelta(t, float32(20), *result.Percentage, 0.001)
		assert.Nil(t, result.Usage)
	})

	t.Run("usage discount", func(t *testing.T) {
		result := ToAPIBillingRateCardDiscount(productcatalog.Discounts{
			Usage: &productcatalog.UsageDiscount{
				Quantity: decimal.NewFromInt(100),
			},
		})

		require.NotNil(t, result)
		assert.Nil(t, result.Percentage)
		require.NotNil(t, result.Usage)
		assert.Equal(t, lo.ToPtr(api.Numeric("100")), result.Usage)
	})

	t.Run("both discounts", func(t *testing.T) {
		result := ToAPIBillingRateCardDiscount(productcatalog.Discounts{
			Percentage: &productcatalog.PercentageDiscount{
				Percentage: models.NewPercentage(10),
			},
			Usage: &productcatalog.UsageDiscount{
				Quantity: decimal.NewFromInt(50),
			},
		})

		require.NotNil(t, result)
		require.NotNil(t, result.Percentage)
		require.NotNil(t, result.Usage)
	})
}

func TestFromBillingCommitments(t *testing.T) {
	t.Run("nil when no commitments", func(t *testing.T) {
		result := ToAPIBillingSpendCommitments(productcatalog.Commitments{})
		assert.Nil(t, result)
	})

	t.Run("minimum amount only", func(t *testing.T) {
		min := decimal.NewFromFloat(10)
		result := ToAPIBillingSpendCommitments(productcatalog.Commitments{MinimumAmount: &min})

		require.NotNil(t, result)
		assert.Equal(t, lo.ToPtr(api.Numeric("10")), result.MinimumAmount)
		assert.Nil(t, result.MaximumAmount)
	})

	t.Run("maximum amount only", func(t *testing.T) {
		max := decimal.NewFromFloat(500)
		result := ToAPIBillingSpendCommitments(productcatalog.Commitments{MaximumAmount: &max})

		require.NotNil(t, result)
		assert.Nil(t, result.MinimumAmount)
		assert.Equal(t, lo.ToPtr(api.Numeric("500")), result.MaximumAmount)
	})

	t.Run("both amounts", func(t *testing.T) {
		min := decimal.NewFromFloat(10)
		max := decimal.NewFromFloat(500)
		result := ToAPIBillingSpendCommitments(productcatalog.Commitments{
			MinimumAmount: &min,
			MaximumAmount: &max,
		})

		require.NotNil(t, result)
		assert.Equal(t, lo.ToPtr(api.Numeric("10")), result.MinimumAmount)
		assert.Equal(t, lo.ToPtr(api.Numeric("500")), result.MaximumAmount)
	})
}

func TestFromBillingTaxConfig(t *testing.T) {
	t.Run("nil when tax config is nil", func(t *testing.T) {
		result := ToAPIBillingRateCardTaxConfig(nil, &taxcode.TaxCode{NamespacedID: models.NamespacedID{ID: "01TAXCODE"}})
		assert.Nil(t, result)
	})

	t.Run("nil when tax code is nil", func(t *testing.T) {
		result := ToAPIBillingRateCardTaxConfig(&productcatalog.TaxConfig{}, nil)
		assert.Nil(t, result)
	})

	t.Run("maps tax code ID", func(t *testing.T) {
		tc := &taxcode.TaxCode{NamespacedID: models.NamespacedID{ID: "01TAXCODE000000000000000000"}}
		result := ToAPIBillingRateCardTaxConfig(&productcatalog.TaxConfig{}, tc)

		require.NotNil(t, result)
		assert.Equal(t, api.ULID("01TAXCODE000000000000000000"), result.Code.Id)
		assert.Nil(t, result.Behavior)
	})

	t.Run("maps behavior", func(t *testing.T) {
		tc := &taxcode.TaxCode{NamespacedID: models.NamespacedID{ID: "01TAXCODE000000000000000000"}}
		cfg := &productcatalog.TaxConfig{
			Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		}

		result := ToAPIBillingRateCardTaxConfig(cfg, tc)
		require.NotNil(t, result)
		require.NotNil(t, result.Behavior)
		assert.Equal(t, api.BillingTaxBehavior("inclusive"), *result.Behavior)
	})
}

func TestFromPlanPhaseWithRateCards(t *testing.T) {
	t.Run("phase with flat fee rate card", func(t *testing.T) {
		rc := &productcatalog.FlatFeeRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:  "setup",
				Name: "Setup Fee",
			},
		}

		phase := plan.Phase{
			Phase: productcatalog.Phase{
				PhaseMeta: productcatalog.PhaseMeta{Key: "p1", Name: "Phase 1"},
				RateCards: productcatalog.RateCards{rc},
			},
		}

		result, err := ToAPIBillingPlanPhase(phase)
		require.NoError(t, err)
		require.Len(t, result.RateCards, 1)
		assert.Equal(t, "setup", result.RateCards[0].Key)
	})
}

func TestToUpdatePlanInput(t *testing.T) {
	t.Run("maps scalar fields correctly", func(t *testing.T) {
		body := api.UpsertPlanRequest{
			Name:        "Updated Plan",
			Description: lo.ToPtr("New description"),
			Phases:      []api.BillingPlanPhase{},
		}

		result, err := FromAPIUpsertPlanRequest("test-ns", "plan-123", body)
		require.NoError(t, err)

		assert.Equal(t, "test-ns", result.Namespace)
		assert.Equal(t, "plan-123", result.ID)
		require.NotNil(t, result.Name)
		assert.Equal(t, "Updated Plan", *result.Name)
		require.NotNil(t, result.Description)
		assert.Equal(t, "New description", *result.Description)
		require.NotNil(t, result.Phases)
		assert.Empty(t, *result.Phases)
	})

	t.Run("labels map to metadata", func(t *testing.T) {
		body := api.UpsertPlanRequest{
			Name:   "Plan",
			Labels: &api.Labels{"env": "prod"},
			Phases: []api.BillingPlanPhase{},
		}

		result, err := FromAPIUpsertPlanRequest("ns", "id", body)
		require.NoError(t, err)
		require.NotNil(t, result.Metadata)
		assert.Equal(t, "prod", (*result.Metadata)["env"])
	})

	t.Run("nil labels result in nil metadata", func(t *testing.T) {
		body := api.UpsertPlanRequest{
			Name:   "Plan",
			Phases: []api.BillingPlanPhase{},
		}

		result, err := FromAPIUpsertPlanRequest("ns", "id", body)
		require.NoError(t, err)
		assert.Nil(t, result.Metadata)
	})

	t.Run("pro rating enabled maps correctly", func(t *testing.T) {
		body := api.UpsertPlanRequest{
			Name:             "Plan",
			ProRatingEnabled: lo.ToPtr(false),
			Phases:           []api.BillingPlanPhase{},
		}

		result, err := FromAPIUpsertPlanRequest("ns", "id", body)
		require.NoError(t, err)
		require.NotNil(t, result.ProRatingConfig)
		assert.False(t, result.ProRatingConfig.Enabled)
	})

	t.Run("phases with rate cards are converted", func(t *testing.T) {
		var price api.BillingPrice
		require.NoError(t, price.FromBillingPriceUnit(api.BillingPriceUnit{Amount: "0.05", Type: "unit"}))

		bc := api.ISO8601Duration("P1M")
		body := api.UpsertPlanRequest{
			Name: "Plan",
			Phases: []api.BillingPlanPhase{
				{
					Key:  "p1",
					Name: "Phase 1",
					RateCards: []api.BillingRateCard{
						{Key: "rc1", Name: "RC", Price: price, BillingCadence: &bc},
					},
				},
			},
		}

		result, err := FromAPIUpsertPlanRequest("ns", "id", body)
		require.NoError(t, err)
		require.NotNil(t, result.Phases)
		require.Len(t, *result.Phases, 1)
		assert.Equal(t, "p1", (*result.Phases)[0].Key)
		require.Len(t, (*result.Phases)[0].RateCards, 1)
		assert.Equal(t, productcatalog.UsageBasedRateCardType, (*result.Phases)[0].RateCards[0].Type())
	})
}

func TestToCreatePlanInput(t *testing.T) {
	t.Run("maps scalar fields correctly", func(t *testing.T) {
		body := api.CreatePlanRequest{
			Key:            "pro",
			Name:           "Pro Plan",
			Currency:       "USD",
			BillingCadence: "P1M",
			Description:    lo.ToPtr("A great plan"),
		}

		result, err := FromAPICreatePlanRequest("test-ns", body)
		require.NoError(t, err)

		assert.Equal(t, "test-ns", result.Namespace)
		assert.Equal(t, "pro", result.Key)
		assert.Equal(t, "Pro Plan", result.Name)
		assert.Equal(t, "USD", result.Currency.String())
		assert.Equal(t, "P1M", result.BillingCadence.ISOString().String())
		require.NotNil(t, result.Description)
		assert.Equal(t, "A great plan", *result.Description)
	})

	t.Run("labels map to metadata", func(t *testing.T) {
		body := api.CreatePlanRequest{
			Key:            "pro",
			Name:           "Pro",
			Currency:       "USD",
			BillingCadence: "P1M",
			Labels:         &api.Labels{"env": "prod"},
		}

		result, err := FromAPICreatePlanRequest("ns", body)
		require.NoError(t, err)
		assert.Equal(t, "prod", result.Metadata["env"])
	})

	t.Run("nil labels result in nil metadata", func(t *testing.T) {
		body := api.CreatePlanRequest{
			Key:            "pro",
			Name:           "Pro",
			Currency:       "USD",
			BillingCadence: "P1M",
		}

		result, err := FromAPICreatePlanRequest("ns", body)
		require.NoError(t, err)
		assert.Nil(t, result.Metadata)
	})

	t.Run("invalid billing cadence returns error", func(t *testing.T) {
		body := api.CreatePlanRequest{
			Key:            "pro",
			Name:           "Pro",
			Currency:       "USD",
			BillingCadence: "not-a-duration",
		}

		_, err := FromAPICreatePlanRequest("ns", body)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid billing cadence")
	})

	t.Run("pro rating enabled true", func(t *testing.T) {
		body := api.CreatePlanRequest{
			Key:              "pro",
			Name:             "Pro",
			Currency:         "USD",
			BillingCadence:   "P1M",
			ProRatingEnabled: lo.ToPtr(true),
		}

		result, err := FromAPICreatePlanRequest("ns", body)
		require.NoError(t, err)
		assert.True(t, result.ProRatingConfig.Enabled)
		assert.Equal(t, productcatalog.ProRatingModeProratePrices, result.ProRatingConfig.Mode)
	})

	t.Run("pro rating enabled false", func(t *testing.T) {
		body := api.CreatePlanRequest{
			Key:              "pro",
			Name:             "Pro",
			Currency:         "USD",
			BillingCadence:   "P1M",
			ProRatingEnabled: lo.ToPtr(false),
		}

		result, err := FromAPICreatePlanRequest("ns", body)
		require.NoError(t, err)
		assert.False(t, result.ProRatingConfig.Enabled)
	})

	t.Run("pro rating nil defaults to enabled", func(t *testing.T) {
		body := api.CreatePlanRequest{
			Key:            "pro",
			Name:           "Pro",
			Currency:       "USD",
			BillingCadence: "P1M",
		}

		result, err := FromAPICreatePlanRequest("ns", body)
		require.NoError(t, err)
		assert.True(t, result.ProRatingConfig.Enabled)
	})

	t.Run("phases are converted", func(t *testing.T) {
		body := api.CreatePlanRequest{
			Key:            "pro",
			Name:           "Pro",
			Currency:       "USD",
			BillingCadence: "P1M",
			Phases: []api.BillingPlanPhase{
				{Key: "p1", Name: "Phase 1", RateCards: []api.BillingRateCard{}},
				{Key: "p2", Name: "Phase 2", RateCards: []api.BillingRateCard{}},
			},
		}

		result, err := FromAPICreatePlanRequest("ns", body)
		require.NoError(t, err)
		require.Len(t, result.Phases, 2)
		assert.Equal(t, "p1", result.Phases[0].Key)
		assert.Equal(t, "p2", result.Phases[1].Key)
	})
}

func TestToPlanPhase(t *testing.T) {
	t.Run("maps fields correctly", func(t *testing.T) {
		dur := api.ISO8601Duration("P3M")
		phase := api.BillingPlanPhase{
			Key:         "trial",
			Name:        "Trial",
			Description: lo.ToPtr("Free trial"),
			Duration:    &dur,
			Labels:      &api.Labels{"tier": "free"},
			RateCards:   []api.BillingRateCard{},
		}

		result, err := FromAPIBillingPlanPhase(phase)
		require.NoError(t, err)

		assert.Equal(t, "trial", result.Key)
		assert.Equal(t, "Trial", result.Name)
		require.NotNil(t, result.Description)
		assert.Equal(t, "Free trial", *result.Description)
		require.NotNil(t, result.Duration)
		assert.Equal(t, "P3M", result.Duration.ISOString().String())
		assert.Equal(t, "free", result.Metadata["tier"])
	})

	t.Run("nil duration maps to nil", func(t *testing.T) {
		phase := api.BillingPlanPhase{
			Key:       "main",
			Name:      "Main",
			RateCards: []api.BillingRateCard{},
		}

		result, err := FromAPIBillingPlanPhase(phase)
		require.NoError(t, err)
		assert.Nil(t, result.Duration)
	})
}

func TestToRateCard(t *testing.T) {
	t.Run("free price creates flat fee rate card with nil price", func(t *testing.T) {
		var price api.BillingPrice
		require.NoError(t, price.FromBillingPriceFree(api.BillingPriceFree{Type: "free"}))

		rc := api.BillingRateCard{
			Key:   "setup",
			Name:  "Setup",
			Price: price,
		}

		result, err := FromAPIBillingRateCard(rc)
		require.NoError(t, err)

		assert.Equal(t, productcatalog.FlatFeeRateCardType, result.Type())
		assert.Nil(t, result.AsMeta().Price)
		assert.Nil(t, result.GetBillingCadence())
	})

	t.Run("flat price creates flat fee rate card", func(t *testing.T) {
		var price api.BillingPrice
		require.NoError(t, price.FromBillingPriceFlat(api.BillingPriceFlat{Amount: "9.99", Type: "flat"}))

		bc := api.ISO8601Duration("P1M")
		pt := api.BillingPricePaymentTerm("in_advance")

		rc := api.BillingRateCard{
			Key:            "base",
			Name:           "Base Fee",
			Price:          price,
			BillingCadence: &bc,
			PaymentTerm:    &pt,
		}

		result, err := FromAPIBillingRateCard(rc)
		require.NoError(t, err)

		assert.Equal(t, productcatalog.FlatFeeRateCardType, result.Type())
		require.NotNil(t, result.AsMeta().Price)

		flat, err := result.AsMeta().Price.AsFlat()
		require.NoError(t, err)
		assert.Equal(t, "9.99", flat.Amount.String())
		assert.Equal(t, productcatalog.InAdvancePaymentTerm, flat.PaymentTerm)
		require.NotNil(t, result.GetBillingCadence())
	})

	t.Run("unit price creates usage based rate card", func(t *testing.T) {
		var price api.BillingPrice
		require.NoError(t, price.FromBillingPriceUnit(api.BillingPriceUnit{Amount: "0.05", Type: "unit"}))

		bc := api.ISO8601Duration("P1M")

		rc := api.BillingRateCard{
			Key:            "api-calls",
			Name:           "API Calls",
			Price:          price,
			BillingCadence: &bc,
		}

		result, err := FromAPIBillingRateCard(rc)
		require.NoError(t, err)

		assert.Equal(t, productcatalog.UsageBasedRateCardType, result.Type())
		require.NotNil(t, result.GetBillingCadence())
		assert.Equal(t, "P1M", result.GetBillingCadence().ISOString().String())

		unit, err := result.AsMeta().Price.AsUnit()
		require.NoError(t, err)
		assert.Equal(t, "0.05", unit.Amount.String())
	})

	t.Run("usage based without billing cadence returns error", func(t *testing.T) {
		var price api.BillingPrice
		require.NoError(t, price.FromBillingPriceUnit(api.BillingPriceUnit{Amount: "0.05", Type: "unit"}))

		rc := api.BillingRateCard{
			Key:   "api-calls",
			Name:  "API Calls",
			Price: price,
		}

		_, err := FromAPIBillingRateCard(rc)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "billing cadence is required")
	})

	t.Run("usage based with commitments", func(t *testing.T) {
		var price api.BillingPrice
		require.NoError(t, price.FromBillingPriceUnit(api.BillingPriceUnit{Amount: "0.05", Type: "unit"}))

		bc := api.ISO8601Duration("P1M")

		rc := api.BillingRateCard{
			Key:            "api-calls",
			Name:           "API Calls",
			Price:          price,
			BillingCadence: &bc,
			Commitments: &api.BillingSpendCommitments{
				MinimumAmount: lo.ToPtr(api.Numeric("10")),
				MaximumAmount: lo.ToPtr(api.Numeric("100")),
			},
		}

		result, err := FromAPIBillingRateCard(rc)
		require.NoError(t, err)

		c := result.AsMeta().Price.GetCommitments()
		require.NotNil(t, c.MinimumAmount)
		assert.Equal(t, "10", c.MinimumAmount.String())
		require.NotNil(t, c.MaximumAmount)
		assert.Equal(t, "100", c.MaximumAmount.String())
	})

	t.Run("feature ID is mapped", func(t *testing.T) {
		var price api.BillingPrice
		require.NoError(t, price.FromBillingPriceFree(api.BillingPriceFree{Type: "free"}))

		rc := api.BillingRateCard{
			Key:     "rc",
			Name:    "RC",
			Price:   price,
			Feature: &api.FeatureReferenceItem{Id: "01FEATURE00000000"},
		}

		result, err := FromAPIBillingRateCard(rc)
		require.NoError(t, err)
		require.NotNil(t, result.AsMeta().FeatureID)
		assert.Equal(t, "01FEATURE00000000", *result.AsMeta().FeatureID)
	})

	t.Run("graduated price creates usage based rate card", func(t *testing.T) {
		var price api.BillingPrice
		require.NoError(t, price.FromBillingPriceGraduated(api.BillingPriceGraduated{
			Type: "graduated",
			Tiers: []api.BillingPriceTier{
				{UpToAmount: lo.ToPtr(api.Numeric("100")), UnitPrice: &api.BillingPriceUnit{Amount: "0.10", Type: "unit"}},
				{UnitPrice: &api.BillingPriceUnit{Amount: "0.05", Type: "unit"}},
			},
		}))

		bc := api.ISO8601Duration("P1M")

		rc := api.BillingRateCard{
			Key:            "usage",
			Name:           "Usage",
			Price:          price,
			BillingCadence: &bc,
		}

		result, err := FromAPIBillingRateCard(rc)
		require.NoError(t, err)

		assert.Equal(t, productcatalog.UsageBasedRateCardType, result.Type())

		tiered, err := result.AsMeta().Price.AsTiered()
		require.NoError(t, err)
		assert.Equal(t, productcatalog.GraduatedTieredPrice, tiered.Mode)
		require.Len(t, tiered.Tiers, 2)
	})

	t.Run("volume price creates usage based rate card", func(t *testing.T) {
		var price api.BillingPrice
		require.NoError(t, price.FromBillingPriceVolume(api.BillingPriceVolume{
			Type:  "volume",
			Tiers: []api.BillingPriceTier{},
		}))

		bc := api.ISO8601Duration("P1M")

		rc := api.BillingRateCard{
			Key:            "usage",
			Name:           "Usage",
			Price:          price,
			BillingCadence: &bc,
		}

		result, err := FromAPIBillingRateCard(rc)
		require.NoError(t, err)

		assert.Equal(t, productcatalog.UsageBasedRateCardType, result.Type())

		tiered, err := result.AsMeta().Price.AsTiered()
		require.NoError(t, err)
		assert.Equal(t, productcatalog.VolumeTieredPrice, tiered.Mode)
	})
}

func TestToBillingPrice(t *testing.T) {
	t.Run("free maps to nil", func(t *testing.T) {
		var price api.BillingPrice
		require.NoError(t, price.FromBillingPriceFree(api.BillingPriceFree{Type: "free"}))

		result, err := FromAPIBillingPrice(price, nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("flat with payment term", func(t *testing.T) {
		var price api.BillingPrice
		require.NoError(t, price.FromBillingPriceFlat(api.BillingPriceFlat{Amount: "5.00", Type: "flat"}))

		pt := api.BillingPricePaymentTerm("in_arrears")
		result, err := FromAPIBillingPrice(price, &pt)
		require.NoError(t, err)
		require.NotNil(t, result)

		flat, err := result.AsFlat()
		require.NoError(t, err)
		assert.Equal(t, "5", flat.Amount.String())
		assert.Equal(t, productcatalog.InArrearsPaymentTerm, flat.PaymentTerm)
	})

	t.Run("flat with nil payment term uses default", func(t *testing.T) {
		var price api.BillingPrice
		require.NoError(t, price.FromBillingPriceFlat(api.BillingPriceFlat{Amount: "5.00", Type: "flat"}))

		result, err := FromAPIBillingPrice(price, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		flat, err := result.AsFlat()
		require.NoError(t, err)
		assert.Equal(t, productcatalog.DefaultPaymentTerm, flat.PaymentTerm)
	})
}

func TestToBillingPriceTiers(t *testing.T) {
	t.Run("maps tiers correctly", func(t *testing.T) {
		tiers := []api.BillingPriceTier{
			{
				UpToAmount: lo.ToPtr(api.Numeric("1000")),
				FlatPrice:  &api.BillingPriceFlat{Amount: "5", Type: "flat"},
				UnitPrice:  &api.BillingPriceUnit{Amount: "0.01", Type: "unit"},
			},
			{
				UnitPrice: &api.BillingPriceUnit{Amount: "0.005", Type: "unit"},
			},
		}

		result, err := FromAPIBillingPriceTiers(tiers)
		require.NoError(t, err)
		require.Len(t, result, 2)

		require.NotNil(t, result[0].UpToAmount)
		assert.Equal(t, "1000", result[0].UpToAmount.String())
		require.NotNil(t, result[0].FlatPrice)
		assert.Equal(t, "5", result[0].FlatPrice.Amount.String())
		require.NotNil(t, result[0].UnitPrice)
		assert.Equal(t, "0.01", result[0].UnitPrice.Amount.String())

		assert.Nil(t, result[1].UpToAmount)
		assert.Nil(t, result[1].FlatPrice)
		require.NotNil(t, result[1].UnitPrice)
		assert.Equal(t, "0.005", result[1].UnitPrice.Amount.String())
	})
}

func TestToBillingTaxConfig(t *testing.T) {
	t.Run("maps code ID", func(t *testing.T) {
		tc := api.BillingRateCardTaxConfig{
			Code: api.TaxCodeReferenceItem{Id: "01TAXCODE000"},
		}

		result := FromAPIBillingRateCardTaxConfig(tc)
		require.NotNil(t, result)
		require.NotNil(t, result.TaxCodeID)
		assert.Equal(t, "01TAXCODE000", *result.TaxCodeID)
		assert.Nil(t, result.Behavior)
	})

	t.Run("maps behavior", func(t *testing.T) {
		tc := api.BillingRateCardTaxConfig{
			Code:     api.TaxCodeReferenceItem{Id: "01TAXCODE000"},
			Behavior: lo.ToPtr(api.BillingTaxBehavior("inclusive")),
		}

		result := FromAPIBillingRateCardTaxConfig(tc)
		require.NotNil(t, result)
		require.NotNil(t, result.Behavior)
		assert.Equal(t, productcatalog.InclusiveTaxBehavior, *result.Behavior)
	})
}

func TestToBillingDiscounts(t *testing.T) {
	t.Run("percentage discount", func(t *testing.T) {
		pct := float32(20)
		result, err := FromAPIBillingRateCardDiscounts(api.BillingRateCardDiscounts{Percentage: &pct})
		require.NoError(t, err)

		require.NotNil(t, result.Percentage)
		assert.InDelta(t, 20, result.Percentage.Percentage.InexactFloat64(), 0.001)
		assert.Nil(t, result.Usage)
	})

	t.Run("usage discount", func(t *testing.T) {
		result, err := FromAPIBillingRateCardDiscounts(api.BillingRateCardDiscounts{Usage: lo.ToPtr(api.Numeric("100"))})
		require.NoError(t, err)

		assert.Nil(t, result.Percentage)
		require.NotNil(t, result.Usage)
		assert.Equal(t, "100", result.Usage.Quantity.String())
	})

	t.Run("both discounts", func(t *testing.T) {
		pct := float32(10)
		result, err := FromAPIBillingRateCardDiscounts(api.BillingRateCardDiscounts{
			Percentage: &pct,
			Usage:      lo.ToPtr(api.Numeric("50")),
		})
		require.NoError(t, err)

		require.NotNil(t, result.Percentage)
		require.NotNil(t, result.Usage)
	})
}

func TestFromPlanInvalidStatus(t *testing.T) {
	t.Run("invalid status returns error", func(t *testing.T) {
		now := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
		clock.SetTime(now)
		defer clock.ResetTime()

		p := newTestPlan(t)
		// StatusAt returns PlanStatusInvalid when no conditions match
		// Force it by setting effectiveTo before effectiveFrom
		past := now.Add(-48 * time.Hour)
		p.EffectiveFrom = &now // now, not in the past
		p.EffectiveTo = &past  // to before from → invalid

		_, err := ToAPIBillingPlan(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid PlanStatus")
	})
}
