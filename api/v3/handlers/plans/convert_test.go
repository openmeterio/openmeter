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

		result, err := FromPlan(p)
		require.NoError(t, err)

		assert.Equal(t, "01J8GFKQ0000000000000000", result.Id)
		assert.Equal(t, api.ResourceKey("pro"), result.Key)
		assert.Equal(t, "Pro Plan", result.Name)
		assert.Equal(t, api.CurrencyCode("USD"), result.Currency)
		assert.Equal(t, api.ISO8601Duration("P1M"), result.BillingCadence)
		assert.Equal(t, 1, result.Version)
		require.NotNil(t, result.CreatedAt)
		assert.Equal(t, now, *result.CreatedAt)
		require.NotNil(t, result.UpdatedAt)
		assert.Equal(t, now, *result.UpdatedAt)
		assert.Nil(t, result.DeletedAt)
		assert.Nil(t, result.Description)
	})

	t.Run("maps optional description", func(t *testing.T) {
		p := newTestPlan(t)
		p.Description = lo.ToPtr("A great plan")

		result, err := FromPlan(p)
		require.NoError(t, err)
		require.NotNil(t, result.Description)
		assert.Equal(t, "A great plan", *result.Description)
	})

	t.Run("maps deleted at", func(t *testing.T) {
		p := newTestPlan(t)
		deleted := now.Add(-time.Hour)
		p.DeletedAt = &deleted

		result, err := FromPlan(p)
		require.NoError(t, err)
		require.NotNil(t, result.DeletedAt)
		assert.Equal(t, deleted, *result.DeletedAt)
	})

	t.Run("pro rating enabled maps correctly", func(t *testing.T) {
		p := newTestPlan(t)
		p.ProRatingConfig.Enabled = false

		result, err := FromPlan(p)
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

		result, err := FromPlan(p)
		require.NoError(t, err)
		require.NotNil(t, result.EffectiveFrom)
		assert.Equal(t, from, *result.EffectiveFrom)
		require.NotNil(t, result.EffectiveTo)
		assert.Equal(t, to, *result.EffectiveTo)
	})

	t.Run("empty phases produces empty slice", func(t *testing.T) {
		p := newTestPlan(t)

		result, err := FromPlan(p)
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

			result, err := FromPlan(p)
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

		result, err := fromPlanPhase(phase)
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

		result, err := fromPlanPhase(phase)
		require.NoError(t, err)
		assert.Nil(t, result.Duration)
	})

	t.Run("phase produces empty rate cards slice", func(t *testing.T) {
		phase := plan.Phase{
			Phase: productcatalog.Phase{
				PhaseMeta: productcatalog.PhaseMeta{Key: "p1", Name: "Phase 1"},
			},
		}

		result, err := fromPlanPhase(phase)
		require.NoError(t, err)
		assert.NotNil(t, result.RateCards)
		assert.Empty(t, result.RateCards)
	})
}

func TestFromValidationErrors(t *testing.T) {
	t.Run("nil when no issues", func(t *testing.T) {
		result := fromValidationErrors(nil)
		assert.Nil(t, result)
	})

	t.Run("nil when empty issues", func(t *testing.T) {
		result := fromValidationErrors(models.ValidationIssues{})
		assert.Nil(t, result)
	})

	t.Run("maps issues to errors", func(t *testing.T) {
		issues := models.ValidationIssues{
			models.NewValidationIssue("plan.missing_phase", "plan must have at least one phase"),
		}

		result := fromValidationErrors(issues)
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

		result := fromValidationErrors(issues)
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

		result, err := FromPlan(p)
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

		result, err := FromPlan(p)
		require.NoError(t, err)
		require.Len(t, result.Phases, 3)
		assert.Equal(t, api.ResourceKey("first"), result.Phases[0].Key)
		assert.Equal(t, api.ResourceKey("second"), result.Phases[1].Key)
		assert.Equal(t, api.ResourceKey("third"), result.Phases[2].Key)
	})

	t.Run("nil phases maps to empty slice", func(t *testing.T) {
		p := newTestPlan(t)
		p.Phases = nil

		result, err := FromPlan(p)
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

		result, err := fromRateCard(rc)
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

		result, err := fromRateCard(rc)
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

		result, err := fromRateCard(rc)
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

		result, err := fromRateCard(rc)
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

		result, err := fromRateCard(rc)
		require.NoError(t, err)
		assert.Nil(t, result.Feature)
	})
}

func TestFromBillingPrice(t *testing.T) {
	t.Run("nil price maps to free", func(t *testing.T) {
		result, err := fromBillingPrice(nil)
		require.NoError(t, err)

		disc, err := result.Discriminator()
		require.NoError(t, err)
		assert.Equal(t, "free", disc)
	})

	t.Run("flat price", func(t *testing.T) {
		p := productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount: decimal.NewFromFloat(5.00),
		})

		result, err := fromBillingPrice(p)
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

		result, err := fromBillingPrice(p)
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

		result, err := fromBillingPrice(p)
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

		result, err := fromBillingPrice(p)
		require.NoError(t, err)

		disc, err := result.Discriminator()
		require.NoError(t, err)
		assert.Equal(t, "volume", disc)
	})

	t.Run("dynamic price returns conflict error", func(t *testing.T) {
		p := productcatalog.NewPriceFrom(productcatalog.DynamicPrice{})

		_, err := fromBillingPrice(p)
		require.Error(t, err)
		assert.True(t, models.IsGenericConflictError(err))
		assert.Contains(t, err.Error(), "dynamic price is not supported in v3 API")
	})

	t.Run("package price returns conflict error", func(t *testing.T) {
		p := productcatalog.NewPriceFrom(productcatalog.PackagePrice{
			Amount:             decimal.NewFromFloat(10),
			QuantityPerPackage: decimal.NewFromInt(100),
		})

		_, err := fromBillingPrice(p)
		require.Error(t, err)
		assert.True(t, models.IsGenericConflictError(err))
		assert.Contains(t, err.Error(), "package price is not supported in v3 API")
	})
}

func TestFromBillingDiscounts(t *testing.T) {
	t.Run("nil when no discounts", func(t *testing.T) {
		result := fromBillingDiscounts(productcatalog.Discounts{})
		assert.Nil(t, result)
	})

	t.Run("percentage discount", func(t *testing.T) {
		result := fromBillingDiscounts(productcatalog.Discounts{
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
		result := fromBillingDiscounts(productcatalog.Discounts{
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
		result := fromBillingDiscounts(productcatalog.Discounts{
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
		result := fromBillingCommitments(productcatalog.Commitments{})
		assert.Nil(t, result)
	})

	t.Run("minimum amount only", func(t *testing.T) {
		min := decimal.NewFromFloat(10)
		result := fromBillingCommitments(productcatalog.Commitments{MinimumAmount: &min})

		require.NotNil(t, result)
		assert.Equal(t, lo.ToPtr(api.Numeric("10")), result.MinimumAmount)
		assert.Nil(t, result.MaximumAmount)
	})

	t.Run("maximum amount only", func(t *testing.T) {
		max := decimal.NewFromFloat(500)
		result := fromBillingCommitments(productcatalog.Commitments{MaximumAmount: &max})

		require.NotNil(t, result)
		assert.Nil(t, result.MinimumAmount)
		assert.Equal(t, lo.ToPtr(api.Numeric("500")), result.MaximumAmount)
	})

	t.Run("both amounts", func(t *testing.T) {
		min := decimal.NewFromFloat(10)
		max := decimal.NewFromFloat(500)
		result := fromBillingCommitments(productcatalog.Commitments{
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
		result := fromBillingTaxConfig(nil, &taxcode.TaxCode{NamespacedID: models.NamespacedID{ID: "01TAXCODE"}})
		assert.Nil(t, result)
	})

	t.Run("nil when tax code is nil", func(t *testing.T) {
		result := fromBillingTaxConfig(&productcatalog.TaxConfig{}, nil)
		assert.Nil(t, result)
	})

	t.Run("maps tax code ID", func(t *testing.T) {
		tc := &taxcode.TaxCode{NamespacedID: models.NamespacedID{ID: "01TAXCODE000000000000000000"}}
		result := fromBillingTaxConfig(&productcatalog.TaxConfig{}, tc)

		require.NotNil(t, result)
		assert.Equal(t, api.ULID("01TAXCODE000000000000000000"), result.Code.Id)
		assert.Nil(t, result.Behavior)
	})

	t.Run("maps behavior", func(t *testing.T) {
		tc := &taxcode.TaxCode{NamespacedID: models.NamespacedID{ID: "01TAXCODE000000000000000000"}}
		cfg := &productcatalog.TaxConfig{
			Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		}

		result := fromBillingTaxConfig(cfg, tc)
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

		result, err := fromPlanPhase(phase)
		require.NoError(t, err)
		require.Len(t, result.RateCards, 1)
		assert.Equal(t, "setup", result.RateCards[0].Key)
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

		_, err := FromPlan(p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid PlanStatus")
	})
}
