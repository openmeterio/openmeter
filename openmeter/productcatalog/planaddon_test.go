package productcatalog

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestPlanAddon_ValidationErrors(t *testing.T) {
	var (
		trialPeriod      = isodate.MustParse(t, "P14D")
		oneMonthPeriod   = isodate.MustParse(t, "P1M")
		threeMonthPeriod = isodate.MustParse(t, "P3M")
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
						Alignment: Alignment{
							BillablesMustAlign: true,
						},
						Key:      "pro",
						Version:  1,
						Name:     "Pro",
						Currency: currency.USD,
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
						Alignment: Alignment{
							BillablesMustAlign: true,
						},
						Key:      "pro",
						Version:  2,
						Name:     "Pro",
						Currency: currency.USD,
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
				{
					Code:     "plan_addon_incompatible_status",
					Message:  "plan status is incompatible with the addon status",
					Path:     "/plans/pro/versions/2/status",
					Severity: "critical",
				},
				{
					Code:     "plan_addon_incompatible_status",
					Message:  "plan status is incompatible with the addon status",
					Path:     "/addons/storage/versions/1/status",
					Severity: "critical",
				},
				{
					Code:     "plan_addon_max_quantity_must_not_be_set",
					Message:  "maximum quantity must not be set for add-on with single instance type",
					Path:     "/addons/storage/versions/1/maxQuantity",
					Severity: "critical",
				},
				{
					Code:     "plan_addon_currency_mismatch",
					Message:  "currency of the plan and addon must match",
					Path:     "/addons/storage/versions/1/currency",
					Severity: "critical",
				},
				{
					Code:     "plan_addon_unknown_plan_phase_key",
					Message:  "add-on must define valid/existing plan phase key from which the add-on is available for purchase",
					Path:     "/addons/storage/versions/1/fromPlanPhase",
					Severity: "critical",
				},
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
						Alignment: Alignment{
							BillablesMustAlign: true,
						},
						Key:      "pro",
						Version:  2,
						Name:     "Pro",
						Currency: currency.USD,
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
				{
					Code:     "rate_card_price_type_mismatch",
					Message:  "price type must match",
					Path:     "/plans/pro/versions/2/phases/pro/ratecards/storage_capacity/price/type",
					Severity: "critical",
				},
				{
					Code:     "rate_card_only_flat_price_allowed",
					Message:  "only flat price is allowed",
					Path:     "/plans/pro/versions/2/phases/pro/ratecards/storage_capacity/price/type",
					Severity: "critical",
				},
				{
					Code:     "rate_card_feature_key_mismatch",
					Message:  "feature key must match",
					Path:     "/plans/pro/versions/2/phases/pro/ratecards/storage_capacity/featureKey",
					Severity: "critical",
				},
				{
					Code:     "rate_card_billing_cadence_mismatch",
					Message:  "billing cadence must match",
					Path:     "/plans/pro/versions/2/phases/pro/ratecards/storage_capacity/billingCadence",
					Severity: "critical",
				},
				{
					Code:     "rate_card_entitlement_template_type_mismatch",
					Message:  "entitlement template type must match",
					Path:     "/plans/pro/versions/2/phases/pro/ratecards/storage_capacity/entitlementTemplate/type",
					Severity: "critical",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			issues, err := test.planAddon.ValidationErrors()
			assert.NoErrorf(t, err, "expected no error")

			assert.ElementsMatchf(t, test.expectedIssues, issues, "expected issues %v, got %v", test.expectedIssues, issues)
		})
	}
}
