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
				models.NewValidationIssue(ErrPlanAddonCurrencyMismatch.Code(), ErrPlanAddonCurrencyMismatch.Message(), commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest)).
					WithField(
						models.NewFieldSelector("addon"),
						models.NewFieldSelector("currency"),
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
