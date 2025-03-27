package productcatalog

import (
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestFlatFeeRateCard(t *testing.T) {
	t.Run("Validate", func(t *testing.T) {
		tests := []struct {
			Name          string
			RateCard      FlatFeeRateCard
			ExpectedError bool
		}{
			{
				Name: "valid",
				RateCard: FlatFeeRateCard{
					RateCardMeta: RateCardMeta{
						Key:         "feat-1",
						Name:        "Flat 1",
						Description: lo.ToPtr("Flat 1"),
						Metadata: map[string]string{
							"name": "Flat 1",
						},
						Feature: &feature.Feature{
							Namespace:           "namespace-1",
							ID:                  "01JBP3SGZ20Y7VRVC351TDFXYZ",
							Name:                "Feature 1",
							Key:                 "feat-1",
							MeterSlug:           lo.ToPtr("meter-1"),
							MeterGroupByFilters: nil,
							Metadata: map[string]string{
								"name": "Feature 1",
							},
							ArchivedAt: &time.Time{},
							CreatedAt:  time.Time{},
							UpdatedAt:  time.Time{},
						},
						EntitlementTemplate: NewEntitlementTemplateFrom(
							StaticEntitlementTemplate{
								Metadata: map[string]string{
									"name": "static-1",
								},
								Config: []byte(`"test"`),
							}),
						TaxConfig: &TaxConfig{
							Stripe: &StripeTaxConfig{
								Code: "txcd_99999999",
							},
						},
						Price: NewPriceFrom(FlatPrice{
							Amount:      decimal.NewFromInt(1000),
							PaymentTerm: InArrearsPaymentTerm,
						}),
					},
					BillingCadence: lo.ToPtr(isodate.MustParse(t, "P1M")),
				},
				ExpectedError: false,
			},
			{
				Name: "invalid",
				RateCard: FlatFeeRateCard{
					RateCardMeta: RateCardMeta{
						Key:         "feat-2",
						Name:        "Flat 2",
						Description: lo.ToPtr("Flat 2"),
						Metadata: map[string]string{
							"name": "Flat 2",
						},
						Feature: &feature.Feature{
							Namespace:           "namespace-2",
							ID:                  "01JBP3SGZ2YTM6DVH2W318TPNH",
							Name:                "Feature 2",
							Key:                 "feat-2",
							MeterSlug:           lo.ToPtr("meter-2"),
							MeterGroupByFilters: nil,
							Metadata: map[string]string{
								"name": "Feature 2",
							},
							ArchivedAt: &time.Time{},
							CreatedAt:  time.Time{},
							UpdatedAt:  time.Time{},
						},
						EntitlementTemplate: NewEntitlementTemplateFrom(
							StaticEntitlementTemplate{
								Metadata: map[string]string{
									"name": "static-1",
								},
								Config: []byte("invalid JSON"),
							}),
						TaxConfig: &TaxConfig{
							Stripe: &StripeTaxConfig{
								Code: "invalid_code",
							},
						},
						Price: NewPriceFrom(
							FlatPrice{
								Amount:      decimal.NewFromInt(-1000),
								PaymentTerm: PaymentTermType("invalid"),
							}),
					},
					BillingCadence: lo.ToPtr(isodate.MustParse(t, "P0M")),
				},
				ExpectedError: true,
			},
			{
				Name: "valid percentage discount",
				RateCard: FlatFeeRateCard{
					RateCardMeta: RateCardMeta{
						Key:         "feat-1",
						Name:        "Flat 1",
						Description: lo.ToPtr("Flat 1"),
						Price: NewPriceFrom(FlatPrice{
							Amount:      decimal.NewFromInt(1000),
							PaymentTerm: InArrearsPaymentTerm,
						}),
						Discounts: Discounts{
							NewDiscountFrom(PercentageDiscount{
								Percentage: models.NewPercentage(10),
							}),
						},
					},
					BillingCadence: lo.ToPtr(isodate.MustParse(t, "P1M")),
				},
				ExpectedError: false,
			},
			{
				Name: "invalid usage discount",
				RateCard: FlatFeeRateCard{
					RateCardMeta: RateCardMeta{
						Key:         "feat-1",
						Name:        "Flat 1",
						Description: lo.ToPtr("Flat 1"),
						Price: NewPriceFrom(FlatPrice{
							Amount:      decimal.NewFromInt(1000),
							PaymentTerm: InArrearsPaymentTerm,
						}),
						Discounts: Discounts{
							NewDiscountFrom(UsageDiscount{
								Quantity: decimal.NewFromInt(100),
							}),
						},
					},
					BillingCadence: lo.ToPtr(isodate.MustParse(t, "P1M")),
				},
				ExpectedError: true,
			},
		}

		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				err := test.RateCard.Validate()

				if test.ExpectedError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}

func TestUsageBasedRateCard(t *testing.T) {
	feat1 := &feature.Feature{
		Namespace:           "namespace-1",
		ID:                  "01JBP3SGZ20Y7VRVC351TDFXYZ",
		Name:                "Feature 1",
		Key:                 "feat-1",
		MeterSlug:           lo.ToPtr("meter-1"),
		MeterGroupByFilters: nil,
		Metadata: map[string]string{
			"name": "Feature 1",
		},
		ArchivedAt: &time.Time{},
		CreatedAt:  time.Time{},
		UpdatedAt:  time.Time{},
	}

	t.Run("Validate", func(t *testing.T) {
		tests := []struct {
			Name          string
			RateCard      UsageBasedRateCard
			ExpectedError bool
		}{
			{
				Name: "valid",
				RateCard: UsageBasedRateCard{
					RateCardMeta: RateCardMeta{
						Key:         "feat-1",
						Name:        "Usage 1",
						Description: lo.ToPtr("Usage 1"),
						Metadata: map[string]string{
							"name": "usage-1",
						},
						Feature: &feature.Feature{
							Namespace:           "namespace-1",
							ID:                  "01JBP3SGZ20Y7VRVC351TDFXYZ",
							Name:                "Feature 1",
							Key:                 "feat-1",
							MeterSlug:           lo.ToPtr("meter-1"),
							MeterGroupByFilters: nil,
							Metadata: map[string]string{
								"name": "Feature 1",
							},
							ArchivedAt: &time.Time{},
							CreatedAt:  time.Time{},
							UpdatedAt:  time.Time{},
						},
						EntitlementTemplate: NewEntitlementTemplateFrom(
							MeteredEntitlementTemplate{
								Metadata: map[string]string{
									"name": "Entitlement 1",
								},
								IsSoftLimit:             true,
								IssueAfterReset:         lo.ToPtr(500.0),
								IssueAfterResetPriority: lo.ToPtr[uint8](1),
								PreserveOverageAtReset:  nil,
								UsagePeriod:             isodate.MustParse(t, "P1M"),
							}),
						TaxConfig: &TaxConfig{
							Stripe: &StripeTaxConfig{
								Code: "txcd_99999999",
							},
						},
						Price: NewPriceFrom(
							UnitPrice{
								Amount: decimal.NewFromInt(1000),
								Commitments: Commitments{
									MinimumAmount: lo.ToPtr(decimal.NewFromInt(500)),
									MaximumAmount: lo.ToPtr(decimal.NewFromInt(1500)),
								},
							}),
					},
					BillingCadence: isodate.MustParse(t, "P1M"),
				},
				ExpectedError: false,
			},
			{
				Name: "invalid",
				RateCard: UsageBasedRateCard{
					RateCardMeta: RateCardMeta{
						Key:         "feat-2",
						Name:        "Usage 2",
						Description: lo.ToPtr("Usage 2"),
						Metadata: map[string]string{
							"name": "usage-2",
						},
						Feature: &feature.Feature{
							Namespace:           "namespace-2",
							ID:                  "01JBWYR0G2PYB9DVADKQXF8E0P",
							Name:                "Feature 2",
							Key:                 "feat-2",
							MeterSlug:           lo.ToPtr("meter-2"),
							MeterGroupByFilters: nil,
							Metadata: map[string]string{
								"name": "Feature 2",
							},
							ArchivedAt: &time.Time{},
							CreatedAt:  time.Time{},
							UpdatedAt:  time.Time{},
						},
						EntitlementTemplate: NewEntitlementTemplateFrom(
							MeteredEntitlementTemplate{
								Metadata: map[string]string{
									"name": "Entitlement 1",
								},
								IsSoftLimit:             true,
								IssueAfterReset:         lo.ToPtr(500.0),
								IssueAfterResetPriority: lo.ToPtr[uint8](1),
								PreserveOverageAtReset:  nil,
								UsagePeriod:             isodate.MustParse(t, "P1M"),
							}),
						TaxConfig: &TaxConfig{
							Stripe: &StripeTaxConfig{
								Code: "invalid_code",
							},
						},
						Price: NewPriceFrom(
							UnitPrice{
								Amount: decimal.NewFromInt(-1000),
								Commitments: Commitments{
									MinimumAmount: lo.ToPtr(decimal.NewFromInt(1500)),
									MaximumAmount: lo.ToPtr(decimal.NewFromInt(500)),
								},
							}),
					},
					BillingCadence: isodate.MustParse(t, "P0M"),
				},
				ExpectedError: true,
			},
			{
				Name: "valid, mixed discounts",
				RateCard: UsageBasedRateCard{
					RateCardMeta: RateCardMeta{
						Key:     "feat-1",
						Name:    "Usage 1",
						Feature: feat1,
						Price: NewPriceFrom(
							UnitPrice{
								Amount: decimal.NewFromInt(1000),
								Commitments: Commitments{
									MinimumAmount: lo.ToPtr(decimal.NewFromInt(500)),
									MaximumAmount: lo.ToPtr(decimal.NewFromInt(1500)),
								},
							}),
						Discounts: Discounts{
							NewDiscountFrom(PercentageDiscount{
								Percentage: models.NewPercentage(10),
							}),
							NewDiscountFrom(UsageDiscount{
								Quantity: decimal.NewFromInt(100),
							}),
						},
					},
					BillingCadence: isodate.MustParse(t, "P1M"),
				},
				ExpectedError: false,
			},
			{
				Name: "invalid, usage discount for flat price",
				RateCard: UsageBasedRateCard{
					RateCardMeta: RateCardMeta{
						Key:     "feat-1",
						Name:    "Usage 1",
						Feature: feat1,
						Price: NewPriceFrom(
							FlatPrice{
								Amount: decimal.NewFromInt(1000),
							}),
						Discounts: Discounts{
							NewDiscountFrom(PercentageDiscount{
								Percentage: models.NewPercentage(10),
							}),
							NewDiscountFrom(UsageDiscount{
								Quantity: decimal.NewFromInt(100),
							}),
						},
					},
					BillingCadence: isodate.MustParse(t, "P1M"),
				},
				ExpectedError: true,
			},
			{
				Name: "invalid, usage discount for dynamic price",
				RateCard: UsageBasedRateCard{
					RateCardMeta: RateCardMeta{
						Key:     "feat-1",
						Name:    "Usage 1",
						Feature: feat1,
						Price: NewPriceFrom(
							DynamicPrice{
								MarkupRate: decimal.NewFromInt(1),
							}),
						Discounts: Discounts{
							NewDiscountFrom(PercentageDiscount{
								Percentage: models.NewPercentage(10),
							}),
							NewDiscountFrom(UsageDiscount{
								Quantity: decimal.NewFromInt(100),
							}),
						},
					},
					BillingCadence: isodate.MustParse(t, "P1M"),
				},
				ExpectedError: true,
			},
			{
				Name: "invalid, usage discount without price",
				RateCard: UsageBasedRateCard{
					RateCardMeta: RateCardMeta{
						Key:     "feat-1",
						Name:    "Usage 1",
						Feature: feat1,
						Discounts: Discounts{
							NewDiscountFrom(PercentageDiscount{
								Percentage: models.NewPercentage(10),
							}),
							NewDiscountFrom(UsageDiscount{
								Quantity: decimal.NewFromInt(100),
							}),
						},
					},
					BillingCadence: isodate.MustParse(t, "P1M"),
				},
				ExpectedError: true,
			},
		}

		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				err := test.RateCard.Validate()

				if test.ExpectedError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}

func TestRateCardsEqual(t *testing.T) {
	feat1 := &feature.Feature{
		Namespace:           "namespace-1",
		ID:                  "01JBP3SGZ20Y7VRVC351TDFXYZ",
		Name:                "Feature 1",
		Key:                 "feat-1",
		MeterSlug:           lo.ToPtr("meter-1"),
		MeterGroupByFilters: nil,
		Metadata: map[string]string{
			"name": "Feature 1",
		},
		ArchivedAt: &time.Time{},
		CreatedAt:  time.Time{},
		UpdatedAt:  time.Time{},
	}

	t.Run("Equal", func(t *testing.T) {
		tests := []struct {
			Name          string
			Left          RateCards
			Right         RateCards
			ExpectedEqual bool
		}{
			{
				Name: "True",
				Left: []RateCard{
					&UsageBasedRateCard{
						RateCardMeta: RateCardMeta{
							Key:         "feat-1",
							Name:        "Usage 1",
							Description: lo.ToPtr("Usage 1"),
							Metadata: map[string]string{
								"name": "usage-1",
							},
							Feature: &feature.Feature{
								Namespace:           "namespace-1",
								ID:                  "01JBP3SGZ20Y7VRVC351TDFXYZ",
								Name:                "Feature 1",
								Key:                 "feat-1",
								MeterSlug:           lo.ToPtr("meter-1"),
								MeterGroupByFilters: nil,
								Metadata: map[string]string{
									"name": "Feature 1",
								},
								ArchivedAt: &time.Time{},
								CreatedAt:  time.Time{},
								UpdatedAt:  time.Time{},
							},
							EntitlementTemplate: NewEntitlementTemplateFrom(
								MeteredEntitlementTemplate{
									Metadata: map[string]string{
										"name": "Entitlement 1",
									},
									IsSoftLimit:             true,
									IssueAfterReset:         lo.ToPtr(500.0),
									IssueAfterResetPriority: lo.ToPtr[uint8](1),
									PreserveOverageAtReset:  nil,
									UsagePeriod:             isodate.MustParse(t, "P1M"),
								}),
							TaxConfig: &TaxConfig{
								Stripe: &StripeTaxConfig{
									Code: "txcd_99999999",
								},
							},
							Price: NewPriceFrom(
								UnitPrice{
									Amount: decimal.NewFromInt(1000),
									Commitments: Commitments{
										MinimumAmount: lo.ToPtr(decimal.NewFromInt(500)),
										MaximumAmount: lo.ToPtr(decimal.NewFromInt(1500)),
									},
								}),
							Discounts: Discounts{
								NewDiscountFrom(PercentageDiscount{
									Percentage: models.NewPercentage(10),
								}),
								NewDiscountFrom(UsageDiscount{
									Quantity: decimal.NewFromInt(100),
								}),
							},
						},
						BillingCadence: isodate.MustParse(t, "P1M"),
					},
				},
				Right: []RateCard{
					&UsageBasedRateCard{
						RateCardMeta: RateCardMeta{
							Key:         "feat-1",
							Name:        "Usage 1",
							Description: lo.ToPtr("Usage 1"),
							Metadata: map[string]string{
								"name": "usage-1",
							},
							Feature: &feature.Feature{
								Namespace:           "namespace-1",
								ID:                  "01JBP3SGZ20Y7VRVC351TDFXYZ",
								Name:                "Feature 1",
								Key:                 "feat-1",
								MeterSlug:           lo.ToPtr("meter-1"),
								MeterGroupByFilters: nil,
								Metadata: map[string]string{
									"name": "Feature 1",
								},
								ArchivedAt: &time.Time{},
								CreatedAt:  time.Time{},
								UpdatedAt:  time.Time{},
							},
							EntitlementTemplate: NewEntitlementTemplateFrom(
								MeteredEntitlementTemplate{
									Metadata: map[string]string{
										"name": "Entitlement 1",
									},
									IsSoftLimit:             true,
									IssueAfterReset:         lo.ToPtr(500.0),
									IssueAfterResetPriority: lo.ToPtr[uint8](1),
									PreserveOverageAtReset:  nil,
									UsagePeriod:             isodate.MustParse(t, "P1M"),
								}),
							TaxConfig: &TaxConfig{
								Stripe: &StripeTaxConfig{
									Code: "txcd_99999999",
								},
							},
							Price: NewPriceFrom(
								UnitPrice{
									Amount: decimal.NewFromInt(1000),
									Commitments: Commitments{
										MinimumAmount: lo.ToPtr(decimal.NewFromInt(500)),
										MaximumAmount: lo.ToPtr(decimal.NewFromInt(1500)),
									},
								}),
							Discounts: Discounts{
								NewDiscountFrom(PercentageDiscount{
									Percentage: models.NewPercentage(10),
								}),
								NewDiscountFrom(UsageDiscount{
									Quantity: decimal.NewFromInt(100),
								}),
							},
						},
						BillingCadence: isodate.MustParse(t, "P1M"),
					},
				},
				ExpectedEqual: true,
			},
			{
				Name: "False",
				Left: []RateCard{
					&UsageBasedRateCard{
						RateCardMeta: RateCardMeta{
							Key:         "feat-1",
							Name:        "Usage 1",
							Description: lo.ToPtr("Usage 1"),
							Metadata: map[string]string{
								"name": "usage-1",
							},
							Feature: &feature.Feature{
								Namespace:           "namespace-1",
								ID:                  "01JBP3SGZ20Y7VRVC351TDFXYZ",
								Name:                "Feature 1",
								Key:                 "feat-1",
								MeterSlug:           lo.ToPtr("meter-1"),
								MeterGroupByFilters: nil,
								Metadata: map[string]string{
									"name": "Feature 1",
								},
								ArchivedAt: &time.Time{},
								CreatedAt:  time.Time{},
								UpdatedAt:  time.Time{},
							},
							EntitlementTemplate: NewEntitlementTemplateFrom(
								MeteredEntitlementTemplate{
									Metadata: map[string]string{
										"name": "Entitlement 1",
									},
									IsSoftLimit:             true,
									IssueAfterReset:         lo.ToPtr(500.0),
									IssueAfterResetPriority: lo.ToPtr[uint8](1),
									PreserveOverageAtReset:  nil,
									UsagePeriod:             isodate.MustParse(t, "P1M"),
								}),
							TaxConfig: &TaxConfig{
								Stripe: &StripeTaxConfig{
									Code: "txcd_99999999",
								},
							},
							Price: NewPriceFrom(
								UnitPrice{
									Amount: decimal.NewFromInt(1000),
									Commitments: Commitments{
										MinimumAmount: lo.ToPtr(decimal.NewFromInt(500)),
										MaximumAmount: lo.ToPtr(decimal.NewFromInt(1500)),
									},
								}),
						},
						BillingCadence: isodate.MustParse(t, "P1M"),
					},
				},
				Right: []RateCard{
					&FlatFeeRateCard{
						RateCardMeta: RateCardMeta{
							Key:         "feat-1",
							Name:        "Flat 1",
							Description: lo.ToPtr("Flat 1"),
							Metadata: map[string]string{
								"name": "Flat 1",
							},
							Feature: &feature.Feature{
								Namespace:           "namespace-1",
								ID:                  "01JBP3SGZ20Y7VRVC351TDFXYZ",
								Name:                "Feature 1",
								Key:                 "feat-1",
								MeterSlug:           lo.ToPtr("meter-1"),
								MeterGroupByFilters: nil,
								Metadata: map[string]string{
									"name": "Feature 1",
								},
								ArchivedAt: &time.Time{},
								CreatedAt:  time.Time{},
								UpdatedAt:  time.Time{},
							},
							EntitlementTemplate: NewEntitlementTemplateFrom(
								StaticEntitlementTemplate{
									Metadata: map[string]string{
										"name": "static-1",
									},
									Config: []byte(`"test"`),
								}),
							TaxConfig: &TaxConfig{
								Stripe: &StripeTaxConfig{
									Code: "txcd_99999999",
								},
							},
							Price: NewPriceFrom(FlatPrice{
								Amount:      decimal.NewFromInt(1000),
								PaymentTerm: InArrearsPaymentTerm,
							}),
						},
						BillingCadence: lo.ToPtr(isodate.MustParse(t, "P1M")),
					},
				},
				ExpectedEqual: false,
			},
			{
				// Usage and percentage discounts are applied at different stages, so we don't care about the
				// ordering of discounts.
				Name: "Discount ordering is not important (true)",
				Left: []RateCard{
					&UsageBasedRateCard{
						RateCardMeta: RateCardMeta{
							Key:         "feat-1",
							Name:        "Usage 1",
							Description: lo.ToPtr("Usage 1"),
							Metadata: map[string]string{
								"name": "usage-1",
							},
							Feature: feat1,
							Price: NewPriceFrom(
								UnitPrice{
									Amount: decimal.NewFromInt(1000),
									Commitments: Commitments{
										MinimumAmount: lo.ToPtr(decimal.NewFromInt(500)),
										MaximumAmount: lo.ToPtr(decimal.NewFromInt(1500)),
									},
								}),
							Discounts: Discounts{
								NewDiscountFrom(PercentageDiscount{
									Percentage: models.NewPercentage(10),
								}),
								NewDiscountFrom(UsageDiscount{
									Quantity: decimal.NewFromInt(100),
								}),
							},
						},
						BillingCadence: isodate.MustParse(t, "P1M"),
					},
				},
				Right: []RateCard{
					&UsageBasedRateCard{
						RateCardMeta: RateCardMeta{
							Key:         "feat-1",
							Name:        "Usage 1",
							Description: lo.ToPtr("Usage 1"),
							Metadata: map[string]string{
								"name": "usage-1",
							},
							Feature: feat1,
							Price: NewPriceFrom(
								UnitPrice{
									Amount: decimal.NewFromInt(1000),
									Commitments: Commitments{
										MinimumAmount: lo.ToPtr(decimal.NewFromInt(500)),
										MaximumAmount: lo.ToPtr(decimal.NewFromInt(1500)),
									},
								}),
							Discounts: Discounts{
								NewDiscountFrom(UsageDiscount{
									Quantity: decimal.NewFromInt(100),
								}),
								NewDiscountFrom(PercentageDiscount{
									Percentage: models.NewPercentage(10),
								}),
							},
						},
						BillingCadence: isodate.MustParse(t, "P1M"),
					},
				},
				ExpectedEqual: true,
			},
		}

		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				match := test.Left.Equal(test.Right)

				if test.ExpectedEqual {
					assert.True(t, match)
				} else {
					assert.False(t, match)
				}
			})
		}
	})
}
