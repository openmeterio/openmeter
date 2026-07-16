package productcatalog

import (
	"net/http"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestPlanAddon_ValidationErrors(t *testing.T) {
	var (
		trialPeriod      = datetime.MustParseDuration(t, "P14D")
		oneMonthPeriod   = datetime.MustParseDuration(t, "P1M")
		threeMonthPeriod = datetime.MustParseDuration(t, "P3M")
	)

	tests := []struct {
		name           string
		planAddon      PlanAddon
		expectedIssues models.ValidationIssues
	}{
		{
			name: "valid",
			planAddon: PlanAddon{
				PlanAddonMeta: PlanAddonMeta{
					PlanAddonConfig: PlanAddonConfig{
						FromPlanPhase: "pro",
						MaxQuantity:   lo.ToPtr(5),
					},
				},
				Plan: Plan{
					PlanMeta: PlanMeta{
						EffectivePeriod: EffectivePeriod{
							EffectiveFrom: lo.ToPtr(clock.Now().Add(-24 * time.Hour)),
						},
						Key:            "pro",
						Version:        1,
						Name:           "Pro",
						Currency:       currency.USD,
						BillingCadence: datetime.MustParseDuration(t, "P1M"),
						ProRatingConfig: ProRatingConfig{
							Enabled: true,
							Mode:    ProRatingModeProratePrices,
						},
					},
					Phases: []Phase{
						{
							PhaseMeta: PhaseMeta{
								Key:      "trial",
								Name:     "Trial",
								Duration: &trialPeriod,
							},
							RateCards: RateCards{
								&UsageBasedRateCard{
									RateCardMeta: RateCardMeta{
										Key:        "storage_capacity",
										Name:       "Storage Capacity",
										FeatureKey: lo.ToPtr("storage_capacity"),
										EntitlementTemplate: NewEntitlementTemplateFrom(MeteredEntitlementTemplate{
											IsSoftLimit:             false,
											IssueAfterReset:         lo.ToPtr(5000.0),
											IssueAfterResetPriority: lo.ToPtr[uint8](1),
											PreserveOverageAtReset:  lo.ToPtr(false),
											UsagePeriod:             oneMonthPeriod,
										}),
										TaxConfig: &TaxConfig{
											Behavior: lo.ToPtr(InclusiveTaxBehavior),
											Stripe: &StripeTaxConfig{
												Code: "txcd_10000000",
											},
										},
										Price: nil,
									},
									BillingCadence: oneMonthPeriod,
								},
								&FlatFeeRateCard{
									RateCardMeta: RateCardMeta{
										Key:  "base_fee",
										Name: "Base fee",
										TaxConfig: &TaxConfig{
											Behavior: lo.ToPtr(InclusiveTaxBehavior),
											Stripe: &StripeTaxConfig{
												Code: "txcd_10000001",
											},
										},
										Price: nil,
									},
									BillingCadence: &oneMonthPeriod,
								},
							},
						},
						{
							PhaseMeta: PhaseMeta{
								Key:  "pro",
								Name: "Pro",
							},
							RateCards: RateCards{
								&UsageBasedRateCard{
									RateCardMeta: RateCardMeta{
										Key:        "storage_capacity",
										Name:       "Storage Capacity",
										FeatureKey: lo.ToPtr("storage_capacity"),
										EntitlementTemplate: NewEntitlementTemplateFrom(MeteredEntitlementTemplate{
											IsSoftLimit:             false,
											IssueAfterReset:         lo.ToPtr(10000.0),
											IssueAfterResetPriority: lo.ToPtr[uint8](1),
											PreserveOverageAtReset:  lo.ToPtr(false),
											UsagePeriod:             oneMonthPeriod,
										}),
										TaxConfig: &TaxConfig{
											Behavior: lo.ToPtr(InclusiveTaxBehavior),
											Stripe: &StripeTaxConfig{
												Code: "txcd_10000000",
											},
										},
										Price: NewPriceFrom(FlatPrice{
											Amount:      alpacadecimal.NewFromInt(100),
											PaymentTerm: InArrearsPaymentTerm,
										}),
									},
									BillingCadence: oneMonthPeriod,
								},
								&FlatFeeRateCard{
									RateCardMeta: RateCardMeta{
										Key:  "base_fee",
										Name: "Base fee",
										TaxConfig: &TaxConfig{
											Behavior: lo.ToPtr(InclusiveTaxBehavior),
											Stripe: &StripeTaxConfig{
												Code: "txcd_10000001",
											},
										},
										Price: NewPriceFrom(FlatPrice{
											Amount: alpacadecimal.NewFromInt(250),
										}),
									},
									BillingCadence: &oneMonthPeriod,
								},
							},
						},
					},
				},
				Addon: Addon{
					AddonMeta: AddonMeta{
						EffectivePeriod: EffectivePeriod{
							EffectiveFrom: lo.ToPtr(clock.Now()),
						},
						Key:          "storage",
						Version:      1,
						Name:         "Storage",
						Currency:     currency.USD,
						InstanceType: AddonInstanceTypeMultiple,
					},
					RateCards: RateCards{
						&UsageBasedRateCard{
							RateCardMeta: RateCardMeta{
								Key:        "storage_capacity",
								Name:       "Storage Capacity",
								FeatureKey: lo.ToPtr("storage_capacity"),
								FeatureID:  nil,
								EntitlementTemplate: NewEntitlementTemplateFrom(MeteredEntitlementTemplate{
									IsSoftLimit:             true,
									IssueAfterReset:         lo.ToPtr(10000.0),
									IssueAfterResetPriority: lo.ToPtr[uint8](1),
									PreserveOverageAtReset:  lo.ToPtr(false),
									UsagePeriod:             oneMonthPeriod,
								}),
								TaxConfig: &TaxConfig{
									Behavior: lo.ToPtr(InclusiveTaxBehavior),
									Stripe: &StripeTaxConfig{
										Code: "txcd_10000000",
									},
								},
								Price: NewPriceFrom(FlatPrice{
									Amount:      alpacadecimal.NewFromInt(99),
									PaymentTerm: InArrearsPaymentTerm,
								}),
							},
							BillingCadence: oneMonthPeriod,
						},
					},
				},
			},
			expectedIssues: nil,
		},
		{
			name: "invalid",
			planAddon: PlanAddon{
				PlanAddonMeta: PlanAddonMeta{
					PlanAddonConfig: PlanAddonConfig{
						FromPlanPhase: "prox",
						MaxQuantity:   lo.ToPtr(5),
					},
				},
				Plan: Plan{
					PlanMeta: PlanMeta{
						EffectivePeriod: EffectivePeriod{
							EffectiveFrom: lo.ToPtr(clock.Now().Add(-24 * time.Hour)),
							EffectiveTo:   lo.ToPtr(clock.Now().Add(-2 * time.Hour)),
						},
						Key:            "pro",
						Version:        2,
						Name:           "Pro",
						Currency:       currency.USD,
						BillingCadence: datetime.MustParseDuration(t, "P1M"),
						ProRatingConfig: ProRatingConfig{
							Enabled: true,
							Mode:    ProRatingModeProratePrices,
						},
					},
					Phases: []Phase{
						{
							PhaseMeta: PhaseMeta{
								Key:      "trial",
								Name:     "Trial",
								Duration: &trialPeriod,
							},
							RateCards: RateCards{
								&UsageBasedRateCard{
									RateCardMeta: RateCardMeta{
										Key:        "storage_capacity",
										Name:       "Storage Capacity",
										FeatureKey: lo.ToPtr("storage_capacity"),
										EntitlementTemplate: NewEntitlementTemplateFrom(MeteredEntitlementTemplate{
											IsSoftLimit:             false,
											IssueAfterReset:         lo.ToPtr(5000.0),
											IssueAfterResetPriority: lo.ToPtr[uint8](1),
											PreserveOverageAtReset:  lo.ToPtr(false),
											UsagePeriod:             oneMonthPeriod,
										}),
										TaxConfig: &TaxConfig{
											Behavior: lo.ToPtr(InclusiveTaxBehavior),
											Stripe: &StripeTaxConfig{
												Code: "txcd_10000000",
											},
										},
										Price: nil,
									},
									BillingCadence: oneMonthPeriod,
								},
								&FlatFeeRateCard{
									RateCardMeta: RateCardMeta{
										Key:  "base_fee",
										Name: "Base fee",
										TaxConfig: &TaxConfig{
											Behavior: lo.ToPtr(InclusiveTaxBehavior),
											Stripe: &StripeTaxConfig{
												Code: "txcd_10000001",
											},
										},
										Price: nil,
									},
									BillingCadence: &oneMonthPeriod,
								},
							},
						},
						{
							PhaseMeta: PhaseMeta{
								Key:  "pro",
								Name: "Pro",
							},
							RateCards: RateCards{
								&UsageBasedRateCard{
									RateCardMeta: RateCardMeta{
										Key:        "storage_capacity",
										Name:       "Storage Capacity",
										FeatureKey: lo.ToPtr("storage_capacity"),
										EntitlementTemplate: NewEntitlementTemplateFrom(MeteredEntitlementTemplate{
											IsSoftLimit:             false,
											IssueAfterReset:         lo.ToPtr(10000.0),
											IssueAfterResetPriority: lo.ToPtr[uint8](1),
											PreserveOverageAtReset:  lo.ToPtr(false),
											UsagePeriod:             oneMonthPeriod,
										}),
										TaxConfig: &TaxConfig{
											Behavior: lo.ToPtr(InclusiveTaxBehavior),
											Stripe: &StripeTaxConfig{
												Code: "txcd_10000000",
											},
										},
										Price: NewPriceFrom(UnitPrice{
											Amount: alpacadecimal.NewFromInt(100),
										}),
									},
									BillingCadence: oneMonthPeriod,
								},
								&FlatFeeRateCard{
									RateCardMeta: RateCardMeta{
										Key:  "base_fee",
										Name: "Base fee",
										TaxConfig: &TaxConfig{
											Behavior: lo.ToPtr(InclusiveTaxBehavior),
											Stripe: &StripeTaxConfig{
												Code: "txcd_10000001",
											},
										},
										Price: NewPriceFrom(FlatPrice{
											Amount: alpacadecimal.NewFromInt(250),
										}),
									},
									BillingCadence: &oneMonthPeriod,
								},
							},
						},
					},
				},
				Addon: Addon{
					AddonMeta: AddonMeta{
						EffectivePeriod: EffectivePeriod{
							EffectiveFrom: lo.ToPtr(clock.Now().Add(-24 * time.Hour)),
							EffectiveTo:   lo.ToPtr(clock.Now().Add(-1 * time.Hour)),
						},
						Key:          "storage",
						Version:      1,
						Name:         "Storage",
						Currency:     currency.AUD,
						InstanceType: AddonInstanceTypeSingle,
					},
					RateCards: RateCards{
						&UsageBasedRateCard{
							RateCardMeta: RateCardMeta{
								Key:        "storage_capacity",
								Name:       "Storage Capacity",
								FeatureKey: lo.ToPtr("storage_capacity"),
								FeatureID:  nil,
								EntitlementTemplate: NewEntitlementTemplateFrom(MeteredEntitlementTemplate{
									IsSoftLimit:             true,
									IssueAfterReset:         lo.ToPtr(10000.0),
									IssueAfterResetPriority: lo.ToPtr[uint8](1),
									PreserveOverageAtReset:  lo.ToPtr(false),
									UsagePeriod:             oneMonthPeriod,
								}),
								TaxConfig: &TaxConfig{
									Behavior: lo.ToPtr(InclusiveTaxBehavior),
									Stripe: &StripeTaxConfig{
										Code: "txcd_10000000",
									},
								},
								Price: NewPriceFrom(FlatPrice{
									Amount:      alpacadecimal.NewFromInt(99),
									PaymentTerm: InArrearsPaymentTerm,
								}),
							},
							BillingCadence: oneMonthPeriod,
						},
					},
				},
			},
			expectedIssues: models.ValidationIssues{
				models.NewValidationIssue(ErrPlanAddonIncompatibleStatus.Code(), ErrPlanAddonIncompatibleStatus.Message(), commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest)).
					WithField(
						models.NewFieldSelector("plan"),
						models.NewFieldSelector("status"),
					).
					WithSeverity(models.ErrorSeverityWarning),
				models.NewValidationIssue(ErrPlanAddonIncompatibleStatus.Code(), ErrPlanAddonIncompatibleStatus.Message(), commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest)).
					WithField(
						models.NewFieldSelector("addon"),
						models.NewFieldSelector("status"),
					).
					WithSeverity(models.ErrorSeverityWarning),
				models.NewValidationIssue(ErrPlanAddonMaxQuantityMustNotBeSet.Code(), ErrPlanAddonMaxQuantityMustNotBeSet.Message(), commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest)).
					WithField(
						models.NewFieldSelector("maxQuantity"),
					).
					WithSeverity(models.ErrorSeverityWarning),
				models.NewValidationIssue(ErrPlanAddonUnknownPlanPhaseKey.Code(), ErrPlanAddonUnknownPlanPhaseKey.Message()).
					WithField(
						models.NewFieldSelector("fromPlanPhase"),
					).
					WithSeverity(models.ErrorSeverityWarning),
			},
		},
		{
			name: "incompatible",
			planAddon: PlanAddon{
				PlanAddonMeta: PlanAddonMeta{
					PlanAddonConfig: PlanAddonConfig{
						FromPlanPhase: "pro",
					},
				},
				Plan: Plan{
					PlanMeta: PlanMeta{
						EffectivePeriod: EffectivePeriod{
							EffectiveFrom: lo.ToPtr(clock.Now().Add(-24 * time.Hour)),
						},
						Key:            "pro",
						Version:        2,
						Name:           "Pro",
						Currency:       currency.USD,
						BillingCadence: datetime.MustParseDuration(t, "P1M"),
						ProRatingConfig: ProRatingConfig{
							Enabled: true,
							Mode:    ProRatingModeProratePrices,
						},
					},
					Phases: []Phase{
						{
							PhaseMeta: PhaseMeta{
								Key:      "trial",
								Name:     "Trial",
								Duration: &trialPeriod,
							},
							RateCards: RateCards{
								&UsageBasedRateCard{
									RateCardMeta: RateCardMeta{
										Key:        "storage_capacity",
										Name:       "Storage Capacity",
										FeatureKey: lo.ToPtr("storage_capacity"),
										EntitlementTemplate: NewEntitlementTemplateFrom(MeteredEntitlementTemplate{
											IsSoftLimit:             false,
											IssueAfterReset:         lo.ToPtr(5000.0),
											IssueAfterResetPriority: lo.ToPtr[uint8](1),
											PreserveOverageAtReset:  lo.ToPtr(false),
											UsagePeriod:             oneMonthPeriod,
										}),
										TaxConfig: &TaxConfig{
											Behavior: lo.ToPtr(InclusiveTaxBehavior),
											Stripe: &StripeTaxConfig{
												Code: "txcd_10000000",
											},
										},
										Price: nil,
									},
									BillingCadence: oneMonthPeriod,
								},
								&FlatFeeRateCard{
									RateCardMeta: RateCardMeta{
										Key:  "base_fee",
										Name: "Base fee",
										TaxConfig: &TaxConfig{
											Behavior: lo.ToPtr(InclusiveTaxBehavior),
											Stripe: &StripeTaxConfig{
												Code: "txcd_10000001",
											},
										},
										Price: nil,
									},
									BillingCadence: &oneMonthPeriod,
								},
							},
						},
						{
							PhaseMeta: PhaseMeta{
								Key:  "pro",
								Name: "Pro",
							},
							RateCards: RateCards{
								&UsageBasedRateCard{
									RateCardMeta: RateCardMeta{
										Key:        "storage_capacity",
										Name:       "Storage Capacity",
										FeatureKey: lo.ToPtr("storage_capacity"),
										EntitlementTemplate: NewEntitlementTemplateFrom(MeteredEntitlementTemplate{
											IsSoftLimit:             false,
											IssueAfterReset:         lo.ToPtr(10000.0),
											IssueAfterResetPriority: lo.ToPtr[uint8](1),
											PreserveOverageAtReset:  lo.ToPtr(false),
											UsagePeriod:             oneMonthPeriod,
										}),
										TaxConfig: &TaxConfig{
											Behavior: lo.ToPtr(InclusiveTaxBehavior),
											Stripe: &StripeTaxConfig{
												Code: "txcd_10000000",
											},
										},
										Price: NewPriceFrom(UnitPrice{
											Amount: alpacadecimal.NewFromInt(100),
										}),
									},
									BillingCadence: oneMonthPeriod,
								},
								&FlatFeeRateCard{
									RateCardMeta: RateCardMeta{
										Key:  "base_fee",
										Name: "Base fee",
										TaxConfig: &TaxConfig{
											Behavior: lo.ToPtr(InclusiveTaxBehavior),
											Stripe: &StripeTaxConfig{
												Code: "txcd_10000001",
											},
										},
										Price: NewPriceFrom(FlatPrice{
											Amount: alpacadecimal.NewFromInt(250),
										}),
									},
									BillingCadence: &oneMonthPeriod,
								},
							},
						},
					},
				},
				Addon: Addon{
					AddonMeta: AddonMeta{
						EffectivePeriod: EffectivePeriod{
							EffectiveFrom: lo.ToPtr(clock.Now().Add(-24 * time.Hour)),
						},
						Key:          "storage",
						Version:      1,
						Name:         "Storage",
						Currency:     currency.USD,
						InstanceType: AddonInstanceTypeSingle,
					},
					RateCards: RateCards{
						&UsageBasedRateCard{
							RateCardMeta: RateCardMeta{
								Key:        "storage_capacity",
								Name:       "Storage Capacity",
								FeatureKey: lo.ToPtr("storage_capacity_x"),
								FeatureID:  nil,
								EntitlementTemplate: NewEntitlementTemplateFrom(StaticEntitlementTemplate{
									Config: []byte(`{"storage_capacity": 1000}`),
								}),
								TaxConfig: &TaxConfig{
									Behavior: lo.ToPtr(InclusiveTaxBehavior),
									Stripe: &StripeTaxConfig{
										Code: "txcd_10000000",
									},
								},
								Price: NewPriceFrom(FlatPrice{
									Amount:      alpacadecimal.NewFromInt(99),
									PaymentTerm: InArrearsPaymentTerm,
								}),
							},
							BillingCadence: threeMonthPeriod,
						},
					},
				},
			},
			expectedIssues: models.ValidationIssues{
				models.NewValidationIssue(ErrRateCardPriceTypeMismatch.Code(), ErrRateCardPriceTypeMismatch.Message()).
					WithField(
						models.NewFieldSelector("addon"),
						models.NewFieldSelector("ratecards").WithExpression(
							models.NewFieldAttrValue("key", "storage_capacity"),
						),
						models.NewFieldSelector("price"),
					).
					WithSeverity(models.ErrorSeverityWarning),
				models.NewValidationIssue(ErrRateCardOnlyFlatPriceAllowed.Code(), ErrRateCardOnlyFlatPriceAllowed.Message()).
					WithField(
						models.NewFieldSelector("addon"),
						models.NewFieldSelector("ratecards").WithExpression(
							models.NewFieldAttrValue("key", "storage_capacity"),
						),
						models.NewFieldSelector("price"),
					).
					WithSeverity(models.ErrorSeverityWarning),
				models.NewValidationIssue(ErrRateCardFeatureKeyMismatch.Code(), ErrRateCardFeatureKeyMismatch.Message(), commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest)).
					WithField(
						models.NewFieldSelector("addon"),
						models.NewFieldSelector("ratecards").WithExpression(
							models.NewFieldAttrValue("key", "storage_capacity"),
						),
						models.NewFieldSelector("featureKey"),
					).
					WithSeverity(models.ErrorSeverityWarning),
				models.NewValidationIssue(ErrRateCardBillingCadenceMismatch.Code(), ErrRateCardBillingCadenceMismatch.Message(), commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest)).
					WithField(
						models.NewFieldSelector("addon"),
						models.NewFieldSelector("ratecards").WithExpression(
							models.NewFieldAttrValue("key", "storage_capacity"),
						),
						models.NewFieldSelector("billingCadence"),
					).
					WithSeverity(models.ErrorSeverityWarning),
				models.NewValidationIssue(ErrRateCardEntitlementTemplateTypeMismatch.Code(), ErrRateCardEntitlementTemplateTypeMismatch.Message(), commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest)).
					WithField(
						models.NewFieldSelector("addon"),
						models.NewFieldSelector("ratecards").WithExpression(
							models.NewFieldAttrValue("key", "storage_capacity"),
						),
						models.NewFieldSelector("entitlementTemplate"),
						models.NewFieldSelector("type"),
					).
					WithSeverity(models.ErrorSeverityWarning),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			issues, err := test.planAddon.ValidationErrors()
			assert.NoErrorf(t, err, "expected no error")

			models.RequireValidationIssuesMatch(t, test.expectedIssues, issues)
		})
	}
}

func TestValidatePlanPhaseAndAddonRateCardsAreCompatibleUnitConfig(t *testing.T) {
	cadence := datetime.MustParseDuration(t, "P1M")
	usageCard := func(uc *UnitConfig) RateCard {
		return &UsageBasedRateCard{
			RateCardMeta: RateCardMeta{
				Key:        "feat-1",
				Name:       "rate card",
				UnitConfig: uc,
			},
			BillingCadence: cadence,
		}
	}
	divide := func(factor int64) *UnitConfig {
		return &UnitConfig{Operation: UnitConfigOperationDivide, ConversionFactor: alpacadecimal.NewFromInt(factor)}
	}

	t.Run("rejects an addon whose unit_config diverges from the plan phase rate card", func(t *testing.T) {
		phase := Phase{RateCards: RateCards{usageCard(divide(1000))}}
		addonRateCards := RateCards{usageCard(divide(500))}

		err := ValidatePlanPhaseAndAddonRateCardsAreCompatible(addonRateCards)(phase)
		assert.ErrorIs(t, err, ErrAddonRateCardUnitConfigMismatch)
	})

	t.Run("accepts a matching unit_config", func(t *testing.T) {
		phase := Phase{RateCards: RateCards{usageCard(divide(1000))}}
		addonRateCards := RateCards{usageCard(divide(1000))}

		err := ValidatePlanPhaseAndAddonRateCardsAreCompatible(addonRateCards)(phase)
		assert.NoError(t, err)
	})
}

func TestPlanAddonValidateRateCardCurrencies(t *testing.T) {
	customCurrency := currency.Code("CREDITS")
	otherCustomCurrency := currency.Code("POINTS")
	activeFrom := clock.Now().Add(-time.Hour)
	month := datetime.MustParseDuration(t, "P1M")

	newRateCard := func(key string, price bool, override *currency.Code) RateCard {
		meta := RateCardMeta{
			Key:      key,
			Name:     key,
			Currency: override,
		}
		if price {
			meta.Price = NewPriceFrom(FlatPrice{Amount: alpacadecimal.NewFromInt(10)})
		}

		return &FlatFeeRateCard{RateCardMeta: meta}
	}

	tests := []struct {
		name          string
		planCurrency  currency.Code
		planRateCards RateCards
		addon         Addon
		expectedError error
	}{
		{
			name:          "matching effective custom currency",
			planCurrency:  currency.USD,
			planRateCards: RateCards{newRateCard("fee", true, &customCurrency)},
			addon: Addon{
				AddonMeta: AddonMeta{Currency: customCurrency},
				RateCards: RateCards{newRateCard("fee", true, nil)},
			},
		},
		{
			name:         "matches overlapping rate cards by key",
			planCurrency: currency.USD,
			planRateCards: RateCards{
				newRateCard("fiat-first", true, nil),
				newRateCard("custom-target", true, &customCurrency),
			},
			addon: Addon{
				AddonMeta: AddonMeta{Currency: customCurrency},
				RateCards: RateCards{
					newRateCard("custom-target", true, nil),
					newRateCard("custom-new", true, nil),
				},
			},
		},
		{
			name:          "cannot change existing rate card currency",
			planCurrency:  currency.USD,
			planRateCards: RateCards{newRateCard("fee", true, nil)},
			addon: Addon{
				AddonMeta: AddonMeta{Currency: customCurrency},
				RateCards: RateCards{newRateCard("fee", true, nil)},
			},
			expectedError: ErrPlanAddonCurrencyMismatch,
		},
		{
			name:         "new custom priced rate card under fiat plan",
			planCurrency: currency.USD,
			addon: Addon{
				AddonMeta: AddonMeta{Currency: customCurrency},
				RateCards: RateCards{newRateCard("fee", true, nil)},
			},
		},
		{
			name:         "new second fiat is rejected",
			planCurrency: currency.USD,
			addon: Addon{
				AddonMeta: AddonMeta{Currency: currency.EUR},
				RateCards: RateCards{newRateCard("fee", true, nil)},
			},
			expectedError: ErrPlanMultipleFiatCurrencies,
		},
		{
			name:          "different fiat defaults are rejected without priced rate cards",
			planCurrency:  currency.USD,
			planRateCards: nil,
			addon: Addon{
				AddonMeta: AddonMeta{Currency: currency.EUR},
			},
			expectedError: ErrPlanMultipleFiatCurrencies,
		},
		{
			name:         "custom plan rejects another custom currency",
			planCurrency: customCurrency,
			addon: Addon{
				AddonMeta: AddonMeta{Currency: otherCustomCurrency},
				RateCards: RateCards{newRateCard("fee", true, nil)},
			},
			expectedError: ErrRateCardCurrencyOverrideNotAllowed,
		},
		{
			name:          "unpriced add-on rate card has no effective currency",
			planCurrency:  currency.USD,
			planRateCards: RateCards{newRateCard("fee", true, nil)},
			addon: Addon{
				AddonMeta: AddonMeta{Currency: customCurrency},
				RateCards: RateCards{newRateCard("fee", false, nil)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given:
			// - an otherwise-valid plan add-on assignment whose rate cards may overlap by key
			// when:
			// - the complete assignment validation runs
			// then:
			// - priced overlays preserve currency and new prices preserve the plan's single-fiat invariant
			addon := tt.addon
			addon.EffectiveFrom = lo.ToPtr(activeFrom)
			addon.InstanceType = AddonInstanceTypeSingle

			planAddon := PlanAddon{
				PlanAddonMeta: PlanAddonMeta{
					PlanAddonConfig: PlanAddonConfig{FromPlanPhase: "default"},
				},
				Plan: Plan{
					PlanMeta: PlanMeta{
						EffectivePeriod: EffectivePeriod{EffectiveFrom: lo.ToPtr(activeFrom)},
						Currency:        tt.planCurrency,
						BillingCadence:  month,
					},
					Phases: []Phase{
						{
							PhaseMeta: PhaseMeta{Key: "default", Name: "Default"},
							RateCards: tt.planRateCards,
						},
					},
				},
				Addon: addon,
			}

			err := planAddon.Validate()

			if tt.expectedError == nil {
				assert.NoError(t, err)
				return
			}

			assert.ErrorIs(t, err, tt.expectedError)
		})
	}
}
