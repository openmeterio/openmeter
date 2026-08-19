package productcatalog_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestValidatePlanWithCurrencies(t *testing.T) {
	custom := currencyx.Code("CREDITS")
	usd := currencyx.Code(currency.USD)
	now := time.Date(2026, time.July, 27, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	tests := []struct {
		name           string
		settlementMode productcatalog.SettlementMode
		costBasis      []currencies.CostBasis
		expected       error
	}{
		{
			name:           "credit then invoice with matching cost basis",
			settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			costBasis: []currencies.CostBasis{{
				CostBasis: currencyx.CostBasis{FiatCode: usd},
			}},
		},
		{
			name:           "credit then invoice with missing cost basis",
			settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			costBasis:      []currencies.CostBasis{},
			expected:       productcatalog.ErrCurrencyCostBasisNotFound,
		},
		{
			name:           "credit only with missing cost basis",
			settlementMode: productcatalog.CreditOnlySettlementMode,
			costBasis:      []currencies.CostBasis{},
		},
		{
			name:           "credit then invoice with scheduled cost basis",
			settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			costBasis: []currencies.CostBasis{{
				CostBasis: currencyx.CostBasis{
					FiatCode:      usd,
					EffectiveFrom: now.Add(time.Hour),
				},
			}},
			expected: productcatalog.ErrCurrencyCostBasisNotFound,
		},
		{
			name:           "credit only with scheduled cost basis",
			settlementMode: productcatalog.CreditOnlySettlementMode,
			costBasis: []currencies.CostBasis{{
				CostBasis: currencyx.CostBasis{
					FiatCode:      usd,
					EffectiveFrom: now.Add(time.Hour),
				},
			}},
		},
		{
			name:           "credit then invoice with active and scheduled cost basis",
			settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			costBasis: []currencies.CostBasis{
				{
					CostBasis: currencyx.CostBasis{
						FiatCode:      usd,
						EffectiveFrom: now.Add(-time.Hour),
						EffectiveTo:   lo.ToPtr(now.Add(time.Hour)),
					},
				},
				{
					CostBasis: currencyx.CostBasis{
						FiatCode:      usd,
						EffectiveFrom: now.Add(time.Hour),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given:
			// - a fiat plan with one custom-currency rate card
			customCurrency := mustManagedPlanCustomCurrency(t, "currency-id", custom)
			customCurrency.CostBasis = &tt.costBasis
			plan := productcatalog.Plan{
				PlanMeta: productcatalog.PlanMeta{
					Currency:       mustPlanFiatCurrencyReference(t, usd),
					SettlementMode: tt.settlementMode,
				},
				Phases: []productcatalog.Phase{{
					PhaseMeta: productcatalog.PhaseMeta{Key: "default"},
					RateCards: productcatalog.RateCards{
						newPlanCurrencyTestRateCard("base", customCurrency.Reference()),
					},
				}},
			}

			// when:
			// - resolved currency references are validated
			err := productcatalog.ValidatePlanWithCurrencies()(plan)

			// then:
			// - only credit-then-invoice plans require an active USD cost-basis pair
			if tt.expected == nil {
				require.NoError(t, err)
				return
			}

			require.ErrorIs(t, err, tt.expected)
		})
	}
}

func TestValidatePlanWithCurrenciesRequiresResolvedReferences(t *testing.T) {
	usd := currencyx.Code(currency.USD)
	custom := currencyx.Code("CREDITS")

	tests := []struct {
		name string
		plan productcatalog.Plan
	}{
		{
			name: "plan currency",
			plan: productcatalog.Plan{
				PlanMeta: productcatalog.PlanMeta{
					Currency:       currencies.NewCurrencyReference(custom),
					SettlementMode: productcatalog.CreditOnlySettlementMode,
				},
			},
		},
		{
			name: "rate card currency",
			plan: productcatalog.Plan{
				PlanMeta: productcatalog.PlanMeta{
					Currency:       mustPlanFiatCurrencyReference(t, usd),
					SettlementMode: productcatalog.CreditOnlySettlementMode,
				},
				Phases: []productcatalog.Phase{{
					RateCards: productcatalog.RateCards{
						newPlanCurrencyTestRateCard("ratecard", currencies.NewCurrencyReference(custom)),
					},
				}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := productcatalog.ValidatePlanWithCurrencies()(tt.plan)
			require.ErrorContains(t, err, "is not resolved")
			require.False(t, models.IsGenericValidationError(err))
		})
	}
}

func TestValidatePlanWithCurrenciesReturnsSpecificOverrideErrors(t *testing.T) {
	usd := mustPlanFiatCurrencyReference(t, currencyx.Code(currency.USD))
	eur := mustPlanFiatCurrencyReference(t, currencyx.Code(currency.EUR))
	credits := mustManagedPlanCustomCurrency(t, "credits-id", "CREDITS")
	credits.CostBasis = &[]currencies.CostBasis{}
	tokens := mustManagedPlanCustomCurrency(t, "tokens-id", "TOKENS")
	tokens.CostBasis = &[]currencies.CostBasis{}

	tests := []struct {
		name      string
		reference currencies.CurrencyReference
		override  currencies.CurrencyReference
		expected  error
	}{
		{
			name:      "custom default rejects override",
			reference: credits.Reference(),
			override:  tokens.Reference(),
			expected:  productcatalog.ErrRateCardCurrencyOverrideNotAllowed,
		},
		{
			name:      "fiat default rejects redundant override",
			reference: usd,
			override:  usd,
			expected:  productcatalog.ErrRateCardCurrencyOverrideRedundant,
		},
		{
			name:      "fiat default rejects second fiat",
			reference: usd,
			override:  eur,
			expected:  productcatalog.ErrPlanMultipleFiatCurrencies,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given:
			// - a credit-only plan with resolved default and override currencies
			plan := productcatalog.Plan{
				PlanMeta: productcatalog.PlanMeta{
					Currency:       tt.reference,
					SettlementMode: productcatalog.CreditOnlySettlementMode,
				},
				Phases: []productcatalog.Phase{{
					PhaseMeta: productcatalog.PhaseMeta{Key: "default"},
					RateCards: productcatalog.RateCards{
						newPlanCurrencyTestRateCard("ratecard", tt.override),
					},
				}},
			}

			// when:
			// - runtime currency validation checks the override relationship
			err := productcatalog.ValidatePlanWithCurrencies()(plan)

			// then:
			// - the canonical product-catalog validation issue is preserved
			require.ErrorIs(t, err, tt.expected)
		})
	}
}

func TestValidatePlanWithCurrenciesUsesResolvedCurrencyReference(t *testing.T) {
	// given:
	// - two managed custom currency resources reuse the same code
	// - only the older resource has a cost-basis pair with USD
	usd := currencyx.Code(currency.USD)
	oldCredits := mustManagedPlanCustomCurrency(t, "old-credits-id", "CREDITS")
	oldCredits.CostBasis = &[]currencies.CostBasis{{
		CostBasis: currencyx.CostBasis{FiatCode: usd},
	}}
	newCredits := mustManagedPlanCustomCurrency(t, "new-credits-id", "CREDITS")
	newCredits.CostBasis = &[]currencies.CostBasis{}

	plan := productcatalog.Plan{
		PlanMeta: productcatalog.PlanMeta{Currency: mustPlanFiatCurrencyReference(t, usd)},
		Phases: []productcatalog.Phase{{
			PhaseMeta: productcatalog.PhaseMeta{Key: "default"},
			RateCards: productcatalog.RateCards{
				newPlanCurrencyTestRateCard("old", oldCredits.Reference()),
				newPlanCurrencyTestRateCard("new", newCredits.Reference()),
			},
		}},
	}

	// when:
	// - plan cost-basis validation checks both priced rate cards
	err := productcatalog.ValidatePlanWithCurrencies()(plan)

	// then:
	// - each managed identity is checked independently despite the shared code
	require.ErrorIs(t, err, productcatalog.ErrCurrencyCostBasisNotFound)
}

func TestValidatePlanCurrencyCodes(t *testing.T) {
	custom := currencyx.Code("CREDITS")

	tests := []struct {
		name         string
		planCurrency currencies.CurrencyReference
		override     *currencies.CurrencyReference
		expected     error
	}{
		{
			name:     "missing plan currency",
			expected: productcatalog.ErrCurrencyInvalid,
		},
		{
			name:         "fiat plan with inherited currency",
			planCurrency: currencies.NewCurrencyReference(currencyx.Code(currency.USD)),
		},
		{
			name:         "fiat plan with custom override",
			planCurrency: currencies.NewCurrencyReference(currencyx.Code(currency.USD)),
			override:     lo.ToPtr(currencies.NewCurrencyReference(custom)),
		},
		{
			name:         "fiat plan with redundant override",
			planCurrency: currencies.NewCurrencyReference(currencyx.Code(currency.USD)),
			override:     lo.ToPtr(currencies.NewCurrencyReference(currencyx.Code(currency.USD))),
			expected:     productcatalog.ErrRateCardCurrencyOverrideRedundant,
		},
		{
			name:         "fiat plan with second fiat",
			planCurrency: currencies.NewCurrencyReference(currencyx.Code(currency.USD)),
			override:     lo.ToPtr(currencies.NewCurrencyReference(currencyx.Code(currency.EUR))),
			expected:     productcatalog.ErrPlanMultipleFiatCurrencies,
		},
		{
			name:         "custom plan with inherited currency",
			planCurrency: currencies.NewCurrencyReference(custom),
		},
		{
			name:         "custom plan with override",
			planCurrency: currencies.NewCurrencyReference(custom),
			override:     lo.ToPtr(currencies.NewCurrencyReference("TOKENS")),
			expected:     productcatalog.ErrRateCardCurrencyOverrideNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := productcatalog.Plan{
				PlanMeta: productcatalog.PlanMeta{Currency: tt.planCurrency},
				Phases: []productcatalog.Phase{{
					PhaseMeta: productcatalog.PhaseMeta{Key: "default"},
					RateCards: productcatalog.RateCards{&productcatalog.FlatFeeRateCard{
						RateCardMeta: productcatalog.RateCardMeta{Key: "base", Currency: tt.override},
					}},
				}},
			}

			err := productcatalog.ValidatePlanCurrencyCodes()(plan)
			if tt.expected == nil {
				require.NoError(t, err)
				return
			}

			require.ErrorIs(t, err, tt.expected)
		})
	}
}

func TestPlanStatus(t *testing.T) {
	now := time.Now()

	tests := []struct {
		Name string

		Plan     productcatalog.Plan
		Expected productcatalog.PlanStatus
	}{
		{
			Name: "Draft",
			Plan: productcatalog.Plan{
				PlanMeta: productcatalog.PlanMeta{
					EffectivePeriod: productcatalog.EffectivePeriod{
						EffectiveFrom: nil,
						EffectiveTo:   nil,
					},
					BillingCadence: datetime.MustParseDuration(t, "P1M"),
					ProRatingConfig: productcatalog.ProRatingConfig{
						Enabled: true,
						Mode:    productcatalog.ProRatingModeProratePrices,
					},
				},
			},
			Expected: productcatalog.PlanStatusDraft,
		},
		{
			Name: "Archived",
			Plan: productcatalog.Plan{
				PlanMeta: productcatalog.PlanMeta{
					EffectivePeriod: productcatalog.EffectivePeriod{
						EffectiveFrom: lo.ToPtr(now.Add(-24 * time.Hour)),
						EffectiveTo:   lo.ToPtr(now.Add(-1 * time.Hour)),
					},
					BillingCadence: datetime.MustParseDuration(t, "P1M"),
					ProRatingConfig: productcatalog.ProRatingConfig{
						Enabled: true,
						Mode:    productcatalog.ProRatingModeProratePrices,
					},
				},
			},
			Expected: productcatalog.PlanStatusArchived,
		},
		{
			Name: "Active with open end",
			Plan: productcatalog.Plan{
				PlanMeta: productcatalog.PlanMeta{
					EffectivePeriod: productcatalog.EffectivePeriod{
						EffectiveFrom: lo.ToPtr(now.Add(-24 * time.Hour)),
						EffectiveTo:   nil,
					},
					BillingCadence: datetime.MustParseDuration(t, "P1M"),
					ProRatingConfig: productcatalog.ProRatingConfig{
						Enabled: true,
						Mode:    productcatalog.ProRatingModeProratePrices,
					},
				},
			},
			Expected: productcatalog.PlanStatusActive,
		},
		{
			Name: "Active with fixed end",
			Plan: productcatalog.Plan{
				PlanMeta: productcatalog.PlanMeta{
					EffectivePeriod: productcatalog.EffectivePeriod{
						EffectiveFrom: lo.ToPtr(now.Add(-24 * time.Hour)),
						EffectiveTo:   lo.ToPtr(now.Add(24 * time.Hour)),
					},
					BillingCadence: datetime.MustParseDuration(t, "P1M"),
					ProRatingConfig: productcatalog.ProRatingConfig{
						Enabled: true,
						Mode:    productcatalog.ProRatingModeProratePrices,
					},
				},
			},
			Expected: productcatalog.PlanStatusActive,
		},
		{
			Name: "Scheduled with open end",
			Plan: productcatalog.Plan{
				PlanMeta: productcatalog.PlanMeta{
					EffectivePeriod: productcatalog.EffectivePeriod{
						EffectiveFrom: lo.ToPtr(now.Add(24 * time.Hour)),
						EffectiveTo:   nil,
					},
					BillingCadence: datetime.MustParseDuration(t, "P1M"),
					ProRatingConfig: productcatalog.ProRatingConfig{
						Enabled: true,
						Mode:    productcatalog.ProRatingModeProratePrices,
					},
				},
			},
			Expected: productcatalog.PlanStatusScheduled,
		},
		{
			Name: "Scheduled with fixed period",
			Plan: productcatalog.Plan{
				PlanMeta: productcatalog.PlanMeta{
					EffectivePeriod: productcatalog.EffectivePeriod{
						EffectiveFrom: lo.ToPtr(now.Add(24 * time.Hour)),
						EffectiveTo:   lo.ToPtr(now.Add(48 * time.Hour)),
					},
					BillingCadence: datetime.MustParseDuration(t, "P1M"),
					ProRatingConfig: productcatalog.ProRatingConfig{
						Enabled: true,
						Mode:    productcatalog.ProRatingModeProratePrices,
					},
				},
			},
			Expected: productcatalog.PlanStatusScheduled,
		},
		{
			Name: "Invalid with inverse period",
			Plan: productcatalog.Plan{
				PlanMeta: productcatalog.PlanMeta{
					EffectivePeriod: productcatalog.EffectivePeriod{
						EffectiveFrom: lo.ToPtr(now.Add(24 * time.Hour)),
						EffectiveTo:   lo.ToPtr(now.Add(-24 * time.Hour)),
					},
					BillingCadence: datetime.MustParseDuration(t, "P1M"),
					ProRatingConfig: productcatalog.ProRatingConfig{
						Enabled: true,
						Mode:    productcatalog.ProRatingModeProratePrices,
					},
				},
			},
			Expected: productcatalog.PlanStatusInvalid,
		},
		{
			Name: "Archived with no start with end in the past",
			Plan: productcatalog.Plan{
				PlanMeta: productcatalog.PlanMeta{
					EffectivePeriod: productcatalog.EffectivePeriod{
						EffectiveFrom: nil,
						EffectiveTo:   lo.ToPtr(now.Add(-24 * time.Hour)),
					},
					BillingCadence: datetime.MustParseDuration(t, "P1M"),
					ProRatingConfig: productcatalog.ProRatingConfig{
						Enabled: true,
						Mode:    productcatalog.ProRatingModeProratePrices,
					},
				},
			},
			Expected: productcatalog.PlanStatusArchived,
		},
		{
			Name: "Actvive with no start with end in the future",
			Plan: productcatalog.Plan{
				PlanMeta: productcatalog.PlanMeta{
					EffectivePeriod: productcatalog.EffectivePeriod{
						EffectiveFrom: nil,
						EffectiveTo:   lo.ToPtr(now.Add(24 * time.Hour)),
					},
					BillingCadence: datetime.MustParseDuration(t, "P1M"),
					ProRatingConfig: productcatalog.ProRatingConfig{
						Enabled: true,
						Mode:    productcatalog.ProRatingModeProratePrices,
					},
				},
			},
			Expected: productcatalog.PlanStatusActive,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.Expected, test.Plan.Status())
		})
	}
}

func TestAlignmentEnforcement(t *testing.T) {
	t.Run("Should allow plan with aligning RateCards", func(t *testing.T) {
		p := productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:            "Plan 1",
				Key:             "plan-1",
				EffectivePeriod: productcatalog.EffectivePeriod{},
				Version:         1,
				Currency:        currencies.NewCurrencyReference(currencyx.Code(currency.USD)),
				BillingCadence:  datetime.MustParseDuration(t, "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:  "phase-1",
						Name: "Phase 1",
					},
					RateCards: []productcatalog.RateCard{
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "flat-fee-1",
								Name: "Flat Fee 1",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromInt(100),
									PaymentTerm: productcatalog.InAdvancePaymentTerm,
								}),
							},
							BillingCadence: lo.ToPtr(datetime.MustParseDuration(t, "P1M")),
						},
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "flat-fee-2",
								Name: "Flat Fee 2",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromInt(100),
									PaymentTerm: productcatalog.InAdvancePaymentTerm,
								}),
							},
							BillingCadence: lo.ToPtr(datetime.MustParseDuration(t, "P1M")),
						},
					},
				},
			},
		}

		err := p.Validate()
		assert.NoError(t, err)
	})

	t.Run("Should never allow plan with misaligned RateCards", func(t *testing.T) {
		p := productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:            "Plan 1",
				Key:             "plan-1",
				EffectivePeriod: productcatalog.EffectivePeriod{},
				Version:         1,
				Currency:        currencies.NewCurrencyReference(currencyx.Code(currency.USD)),
				BillingCadence:  datetime.MustParseDuration(t, "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:  "phase-1",
						Name: "Phase 1",
					},
					RateCards: []productcatalog.RateCard{
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "flat-fee-1",
								Name: "Flat Fee 1",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromInt(100),
									PaymentTerm: productcatalog.InAdvancePaymentTerm,
								}),
							},
							BillingCadence: lo.ToPtr(datetime.MustParseDuration(t, "P1M")),
						},
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "flat-fee-2",
								Name: "Flat Fee 2",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromInt(100),
									PaymentTerm: productcatalog.InAdvancePaymentTerm,
								}),
							},
							BillingCadence: lo.ToPtr(datetime.MustParseDuration(t, "P1W")),
						},
					},
				},
			},
		}

		err := p.Validate()
		assert.Error(t, err)
		assert.ErrorContains(t, err, "ratecards with prices must have compatible billing cadence")
	})

	t.Run("Should NOT allow plan with misaligned RateCards if enforced", func(t *testing.T) {
		p := productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:            "Plan 1",
				Key:             "plan-1",
				EffectivePeriod: productcatalog.EffectivePeriod{},
				Version:         1,
				Currency:        currencies.NewCurrencyReference(currencyx.Code(currency.USD)),
				BillingCadence:  datetime.MustParseDuration(t, "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:  "phase-1",
						Name: "Phase 1",
					},
					RateCards: []productcatalog.RateCard{
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "flat-fee-1",
								Name: "Flat Fee 1",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromInt(100),
									PaymentTerm: productcatalog.InAdvancePaymentTerm,
								}),
							},
							BillingCadence: lo.ToPtr(datetime.MustParseDuration(t, "P1M")),
						},
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "flat-fee-2",
								Name: "Flat Fee 2",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromInt(100),
									PaymentTerm: productcatalog.InAdvancePaymentTerm,
								}),
							},
							BillingCadence: lo.ToPtr(datetime.MustParseDuration(t, "P1W")),
						},
					},
				},
			},
		}

		err := p.Validate()
		assert.Error(t, err)
		assert.ErrorContains(t, err, "ratecards with prices must have compatible billing cadence")
	})
}

func TestPlanHasUnitConfig(t *testing.T) {
	card := func(uc *productcatalog.UnitConfig) productcatalog.RateCard {
		return &productcatalog.UsageBasedRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:        "feat-1",
				Name:       "Feature 1",
				Feature:    productcatalog.NewFeatureReference(nil, lo.ToPtr("feat-1")),
				Price:      productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
				UnitConfig: uc,
			},
		}
	}
	phase := func(cards ...productcatalog.RateCard) productcatalog.Phase {
		return productcatalog.Phase{RateCards: cards}
	}
	divide := &productcatalog.UnitConfig{Operation: productcatalog.UnitConfigOperationDivide, ConversionFactor: alpacadecimal.NewFromInt(1000)}

	t.Run("plan with no phases has none", func(t *testing.T) {
		assert.False(t, productcatalog.Plan{}.HasUnitConfig())
	})

	t.Run("no phase carries unit_config", func(t *testing.T) {
		p := productcatalog.Plan{Phases: []productcatalog.Phase{phase(card(nil)), phase(card(nil))}}
		assert.False(t, p.HasUnitConfig())
	})

	t.Run("unit_config in any later phase is detected", func(t *testing.T) {
		p := productcatalog.Plan{Phases: []productcatalog.Phase{phase(card(nil)), phase(card(nil), card(divide))}}
		assert.True(t, p.HasUnitConfig())
	})
}

func TestPlanHasCurrencyOverrides(t *testing.T) {
	card := func(currencyOverride *currencies.CurrencyReference) productcatalog.RateCard {
		return &productcatalog.FlatFeeRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:      "flat-fee",
				Name:     "Flat fee",
				Currency: currencyOverride,
			},
		}
	}
	phase := func(cards ...productcatalog.RateCard) productcatalog.Phase {
		return productcatalog.Phase{RateCards: cards}
	}
	customCurrency := currencyx.Code("TOK")

	t.Run("plan with no phases has none", func(t *testing.T) {
		assert.False(t, productcatalog.Plan{}.HasCurrencyOverrides())
	})

	t.Run("inherited currencies are not overrides", func(t *testing.T) {
		p := productcatalog.Plan{Phases: []productcatalog.Phase{phase(card(nil)), phase(card(nil))}}
		assert.False(t, p.HasCurrencyOverrides())
	})

	t.Run("override in any later phase is detected", func(t *testing.T) {
		p := productcatalog.Plan{Phases: []productcatalog.Phase{phase(card(nil)), phase(card(nil), card(lo.ToPtr(currencies.NewCurrencyReference(customCurrency))))}}
		assert.True(t, p.HasCurrencyOverrides())
	})
}

func newPlanCurrencyTestRateCard(key string, reference currencies.CurrencyReference) productcatalog.RateCard {
	return &productcatalog.FlatFeeRateCard{RateCardMeta: productcatalog.RateCardMeta{
		Key:      key,
		Name:     key,
		Currency: lo.ToPtr(reference),
		Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      alpacadecimal.NewFromInt(1),
			PaymentTerm: productcatalog.InAdvancePaymentTerm,
		}),
	}}
}

func mustManagedPlanCustomCurrency(t *testing.T, id string, code currencyx.Code) currencies.Currency {
	t.Helper()

	currency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode(code).
		WithName(code.String()).
		Build()
	require.NoError(t, err)

	return currencies.Currency{
		NamespacedID: models.NamespacedID{ID: id},
		Currency:     currency,
	}
}

func mustPlanFiatCurrencyReference(t *testing.T, code currencyx.Code) currencies.CurrencyReference {
	t.Helper()

	currency, err := currencies.NewFiatCurrency(code)
	require.NoError(t, err)

	return currency.Reference()
}
