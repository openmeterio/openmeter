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
						FeatureKey: lo.ToPtr("feat-1"),
						FeatureID:  lo.ToPtr("01JBP3SGZ20Y7VRVC351TDFXYZ"),
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
						FeatureKey: lo.ToPtr("feat-2"),
						FeatureID:  lo.ToPtr("01JBP3SGZ2YTM6DVH2W318TPNH"),
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
							Percentage: &PercentageDiscount{
								Percentage: models.NewPercentage(10),
							},
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
							Usage: &UsageDiscount{
								Quantity: decimal.NewFromInt(100),
							},
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
						FeatureKey: lo.ToPtr("feat-1"),
						FeatureID:  lo.ToPtr("01JBP3SGZ20Y7VRVC351TDFXYZ"),
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
						FeatureKey: lo.ToPtr("feat-2"),
						FeatureID:  lo.ToPtr("01JBWYR0G2PYB9DVADKQXF8E0P"),
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
						Key:        "feat-1",
						Name:       "Usage 1",
						FeatureKey: lo.ToPtr(feat1.Key),
						FeatureID:  lo.ToPtr(feat1.ID),
						Price: NewPriceFrom(
							UnitPrice{
								Amount: decimal.NewFromInt(1000),
								Commitments: Commitments{
									MinimumAmount: lo.ToPtr(decimal.NewFromInt(500)),
									MaximumAmount: lo.ToPtr(decimal.NewFromInt(1500)),
								},
							}),
						Discounts: Discounts{
							Percentage: &PercentageDiscount{
								Percentage: models.NewPercentage(10),
							},
							Usage: &UsageDiscount{
								Quantity: decimal.NewFromInt(100),
							},
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
						Key:        "feat-1",
						Name:       "Usage 1",
						FeatureKey: lo.ToPtr(feat1.Key),
						FeatureID:  lo.ToPtr(feat1.ID),
						Price: NewPriceFrom(
							FlatPrice{
								Amount: decimal.NewFromInt(1000),
							}),
						Discounts: Discounts{
							Percentage: &PercentageDiscount{
								Percentage: models.NewPercentage(10),
							},
							Usage: &UsageDiscount{
								Quantity: decimal.NewFromInt(100),
							},
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
						Key:        "feat-1",
						Name:       "Usage 1",
						FeatureKey: lo.ToPtr(feat1.Key),
						FeatureID:  lo.ToPtr(feat1.ID),
						Discounts: Discounts{
							Percentage: &PercentageDiscount{
								Percentage: models.NewPercentage(10),
							},
							Usage: &UsageDiscount{
								Quantity: decimal.NewFromInt(100),
							},
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
							FeatureKey: lo.ToPtr("feat-1"),
							FeatureID:  lo.ToPtr("01JBP3SGZ20Y7VRVC351TDFXYZ"),
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
								Percentage: &PercentageDiscount{
									Percentage: models.NewPercentage(10),
								},
								Usage: &UsageDiscount{
									Quantity: decimal.NewFromInt(100),
								},
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
							FeatureKey: lo.ToPtr("feat-1"),
							FeatureID:  lo.ToPtr("01JBP3SGZ20Y7VRVC351TDFXYZ"),
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
								Percentage: &PercentageDiscount{
									Percentage: models.NewPercentage(10),
								},
								Usage: &UsageDiscount{
									Quantity: decimal.NewFromInt(100),
								},
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
							FeatureKey: lo.ToPtr("feat-1"),
							FeatureID:  lo.ToPtr("01JBP3SGZ20Y7VRVC351TDFXYZ"),
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
							FeatureKey: lo.ToPtr("feat-1"),
							FeatureID:  lo.ToPtr("01JBP3SGZ20Y7VRVC351TDFXYZ"),
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
							FeatureKey: lo.ToPtr(feat1.Key),
							FeatureID:  lo.ToPtr(feat1.ID),
							Price: NewPriceFrom(
								UnitPrice{
									Amount: decimal.NewFromInt(1000),
									Commitments: Commitments{
										MinimumAmount: lo.ToPtr(decimal.NewFromInt(500)),
										MaximumAmount: lo.ToPtr(decimal.NewFromInt(1500)),
									},
								}),
							Discounts: Discounts{
								Percentage: &PercentageDiscount{
									Percentage: models.NewPercentage(10),
								},
								Usage: &UsageDiscount{
									Quantity: decimal.NewFromInt(100),
								},
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
							FeatureKey: lo.ToPtr(feat1.Key),
							FeatureID:  lo.ToPtr(feat1.ID),
							Price: NewPriceFrom(
								UnitPrice{
									Amount: decimal.NewFromInt(1000),
									Commitments: Commitments{
										MinimumAmount: lo.ToPtr(decimal.NewFromInt(500)),
										MaximumAmount: lo.ToPtr(decimal.NewFromInt(1500)),
									},
								}),
							Discounts: Discounts{
								Usage: &UsageDiscount{
									Quantity: decimal.NewFromInt(100),
								},
								Percentage: &PercentageDiscount{
									Percentage: models.NewPercentage(10),
								},
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

func TestRateCards_BillingCadenceAligned(t *testing.T) {
	p1m := isodate.MustParse(t, "P1M")
	p3m := isodate.MustParse(t, "P3M")
	p1y := isodate.MustParse(t, "P1Y")

	// Helper for creating price
	price := func() *Price {
		return NewPriceFrom(FlatPrice{
			Amount:      decimal.NewFromInt(1000),
			PaymentTerm: InAdvancePaymentTerm,
		})
	}

	tests := []struct {
		name      string
		rateCards RateCards
		want      bool
	}{
		{
			name:      "Empty rate cards",
			rateCards: RateCards{},
			want:      true,
		},
		{
			name: "Single rate card",
			rateCards: RateCards{
				&FlatFeeRateCard{
					RateCardMeta: RateCardMeta{
						Price: price(),
					},
					BillingCadence: lo.ToPtr(p1m),
				},
			},
			want: true,
		},
		{
			name: "Multiple rate cards with same billing cadence",
			rateCards: RateCards{
				&FlatFeeRateCard{
					RateCardMeta: RateCardMeta{
						Price: price(),
					},
					BillingCadence: lo.ToPtr(p1m),
				},
				&FlatFeeRateCard{
					RateCardMeta: RateCardMeta{
						Price: price(),
					},
					BillingCadence: lo.ToPtr(p1m),
				},
				&UsageBasedRateCard{
					RateCardMeta: RateCardMeta{
						Price: price(),
					},
					BillingCadence: p1m,
				},
			},
			want: true,
		},
		{
			name: "Multiple rate cards with different billing cadences",
			rateCards: RateCards{
				&FlatFeeRateCard{
					RateCardMeta: RateCardMeta{
						Price: price(),
					},
					BillingCadence: lo.ToPtr(p1m),
				},
				&FlatFeeRateCard{
					RateCardMeta: RateCardMeta{
						Price: price(),
					},
					BillingCadence: lo.ToPtr(p3m),
				},
			},
			want: false,
		},
		{
			name: "Multiple rate cards with some nil billing cadences",
			rateCards: RateCards{
				&FlatFeeRateCard{
					RateCardMeta: RateCardMeta{
						Price: price(),
					},
					BillingCadence: lo.ToPtr(p1m),
				},
				&FlatFeeRateCard{
					RateCardMeta: RateCardMeta{
						Price: price(),
					},
					BillingCadence: nil,
				},
			},
			want: true,
		},
		{
			name: "Multiple rate cards with all nil billing cadences",
			rateCards: RateCards{
				&FlatFeeRateCard{
					RateCardMeta: RateCardMeta{
						Price: price(),
					},
					BillingCadence: nil,
				},
				&FlatFeeRateCard{
					RateCardMeta: RateCardMeta{
						Price: price(),
					},
					BillingCadence: nil,
				},
			},
			want: true,
		},
		{
			name: "Mix of different rate card types with same billing cadence",
			rateCards: RateCards{
				&FlatFeeRateCard{
					RateCardMeta: RateCardMeta{
						Price: price(),
					},
					BillingCadence: lo.ToPtr(p1y),
				},
				&UsageBasedRateCard{
					RateCardMeta: RateCardMeta{
						Price: price(),
					},
					BillingCadence: p1y,
				},
			},
			want: true,
		},
		{
			name: "Rate cards with no price are ignored",
			rateCards: RateCards{
				&FlatFeeRateCard{
					RateCardMeta: RateCardMeta{
						Price: price(), // This one has a price
					},
					BillingCadence: lo.ToPtr(p1m),
				},
				&FlatFeeRateCard{
					// No price set
					BillingCadence: lo.ToPtr(p3m),
				},
			},
			want: true, // Only the first rate card with price is considered
		},
		{
			name: "All rate cards with no price",
			rateCards: RateCards{
				&FlatFeeRateCard{
					// No price set
					BillingCadence: lo.ToPtr(p1m),
				},
				&FlatFeeRateCard{
					// No price set
					BillingCadence: lo.ToPtr(p3m),
				},
			},
			want: true, // No rate cards with price, so alignment check passes
		},
		{
			name: "Multiple rate cards with price, but different cadences",
			rateCards: RateCards{
				&FlatFeeRateCard{
					RateCardMeta: RateCardMeta{
						Price: price(),
					},
					BillingCadence: lo.ToPtr(p1m),
				},
				&FlatFeeRateCard{
					// No price, should be ignored
					BillingCadence: lo.ToPtr(p3m),
				},
				&FlatFeeRateCard{
					RateCardMeta: RateCardMeta{
						Price: price(),
					},
					BillingCadence: lo.ToPtr(p1y), // Different from first card
				},
			},
			want: false, // Two cards with price but different cadences
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rateCards.SingleBillingCadence()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRateCardsCompatible(t *testing.T) {
	tests := []struct {
		name    string
		rCard   RateCard
		vCard   RateCard
		wantErr bool
	}{
		{
			name: "Compatible",
			rCard: &UsageBasedRateCard{
				RateCardMeta: RateCardMeta{
					Key: "feature1",
					Price: NewPriceFrom(FlatPrice{
						Amount: decimal.NewFromInt(1000),
					}),
					FeatureKey: lo.ToPtr("feature1"),
					FeatureID:  lo.ToPtr("id1"),
					EntitlementTemplate: NewEntitlementTemplateFrom(
						MeteredEntitlementTemplate{
							Metadata: map[string]string{
								"name": "metered-1",
							},
							IsSoftLimit:             false,
							IssueAfterReset:         lo.ToPtr(500.0),
							IssueAfterResetPriority: lo.ToPtr[uint8](1),
							PreserveOverageAtReset:  lo.ToPtr(false),
							UsagePeriod:             isodate.MustParse(t, "P1M"),
						}),
				},
				BillingCadence: isodate.MustParse(t, "P1M"),
			},
			vCard: &UsageBasedRateCard{
				RateCardMeta: RateCardMeta{
					Key: "feature1",
					Price: NewPriceFrom(FlatPrice{
						Amount: decimal.NewFromInt(1000),
					}),
					FeatureKey: lo.ToPtr("feature1"),
					FeatureID:  lo.ToPtr("id1"),
					EntitlementTemplate: NewEntitlementTemplateFrom(
						MeteredEntitlementTemplate{
							Metadata: map[string]string{
								"name": "metered-1",
							},
							IsSoftLimit:             true,
							IssueAfterReset:         lo.ToPtr(1000.0),
							IssueAfterResetPriority: lo.ToPtr[uint8](3),
							PreserveOverageAtReset:  lo.ToPtr(true),
							UsagePeriod:             isodate.MustParse(t, "P1M"),
						}),
				},
				BillingCadence: isodate.MustParse(t, "P1M"),
			},
			wantErr: false,
		},
		{
			name: "Incompatible Price Type",
			rCard: &UsageBasedRateCard{
				RateCardMeta: RateCardMeta{
					Key: "feature1",
					Price: NewPriceFrom(FlatPrice{
						Amount: decimal.NewFromInt(1000),
					}),
					FeatureKey: lo.ToPtr("feature1"),
					FeatureID:  lo.ToPtr("id1"),
					EntitlementTemplate: NewEntitlementTemplateFrom(
						MeteredEntitlementTemplate{
							Metadata: map[string]string{
								"name": "metered-1",
							},
							IsSoftLimit:             false,
							IssueAfterReset:         lo.ToPtr(500.0),
							IssueAfterResetPriority: lo.ToPtr[uint8](1),
							PreserveOverageAtReset:  lo.ToPtr(false),
							UsagePeriod:             isodate.MustParse(t, "P1M"),
						}),
				},
				BillingCadence: isodate.MustParse(t, "P1M"),
			},
			vCard: &UsageBasedRateCard{
				RateCardMeta: RateCardMeta{
					Key: "feature1",
					Price: NewPriceFrom(UnitPrice{
						Commitments: Commitments{
							MinimumAmount: lo.ToPtr(decimal.NewFromInt(1000)),
						},
						Amount: decimal.NewFromInt(10),
					}),
					FeatureKey: lo.ToPtr("feature1"),
					FeatureID:  lo.ToPtr("id1"),
					EntitlementTemplate: NewEntitlementTemplateFrom(
						MeteredEntitlementTemplate{
							Metadata: map[string]string{
								"name": "metered-1",
							},
							IsSoftLimit:             true,
							IssueAfterReset:         lo.ToPtr(1000.0),
							IssueAfterResetPriority: lo.ToPtr[uint8](3),
							PreserveOverageAtReset:  lo.ToPtr(true),
							UsagePeriod:             isodate.MustParse(t, "P1M"),
						}),
				},
				BillingCadence: isodate.MustParse(t, "P1M"),
			},
			wantErr: true,
		},
		{
			name: "Mismatched Feature Key",
			rCard: &FlatFeeRateCard{
				RateCardMeta: RateCardMeta{
					Key: "feature1",
					Price: NewPriceFrom(FlatPrice{
						Amount: decimal.NewFromInt(1000),
					}),
					FeatureKey: lo.ToPtr("feature1"),
					FeatureID:  lo.ToPtr("id1"),
					EntitlementTemplate: NewEntitlementTemplateFrom(
						MeteredEntitlementTemplate{
							Metadata: map[string]string{
								"name": "metered-1",
							},
							IsSoftLimit:             true,
							IssueAfterReset:         lo.ToPtr(1000.0),
							IssueAfterResetPriority: lo.ToPtr[uint8](3),
							PreserveOverageAtReset:  lo.ToPtr(true),
							UsagePeriod:             isodate.MustParse(t, "P1M"),
						}),
				},
				BillingCadence: lo.ToPtr(isodate.MustParse(t, "P1M")),
			},
			vCard: &UsageBasedRateCard{
				RateCardMeta: RateCardMeta{
					Key: "feature1",
					Price: NewPriceFrom(FlatPrice{
						Amount: decimal.NewFromInt(1000),
					}),
					FeatureKey: lo.ToPtr("feature2"),
					FeatureID:  lo.ToPtr("id1"),
					EntitlementTemplate: NewEntitlementTemplateFrom(
						MeteredEntitlementTemplate{
							Metadata: map[string]string{
								"name": "metered-1",
							},
							IsSoftLimit:             true,
							IssueAfterReset:         lo.ToPtr(1000.0),
							IssueAfterResetPriority: lo.ToPtr[uint8](3),
							PreserveOverageAtReset:  lo.ToPtr(true),
							UsagePeriod:             isodate.MustParse(t, "P1M"),
						}),
				},
				BillingCadence: isodate.MustParse(t, "P1M"),
			},
			wantErr: true,
		},
		{
			name: "Mismatched Feature ID",
			rCard: &FlatFeeRateCard{
				RateCardMeta: RateCardMeta{
					Key: "feature1",
					Price: NewPriceFrom(FlatPrice{
						Amount: decimal.NewFromInt(1000),
					}),
					FeatureKey: lo.ToPtr("feature1"),
					FeatureID:  lo.ToPtr("id1"),
					EntitlementTemplate: NewEntitlementTemplateFrom(
						MeteredEntitlementTemplate{
							Metadata: map[string]string{
								"name": "metered-1",
							},
							IsSoftLimit:             true,
							IssueAfterReset:         lo.ToPtr(1000.0),
							IssueAfterResetPriority: lo.ToPtr[uint8](3),
							PreserveOverageAtReset:  lo.ToPtr(true),
							UsagePeriod:             isodate.MustParse(t, "P1M"),
						}),
				},
				BillingCadence: lo.ToPtr(isodate.MustParse(t, "P1M")),
			},
			vCard: &UsageBasedRateCard{
				RateCardMeta: RateCardMeta{
					Key: "feature1",
					Price: NewPriceFrom(FlatPrice{
						Amount: decimal.NewFromInt(1000),
					}),
					FeatureKey: lo.ToPtr("feature1"),
					FeatureID:  lo.ToPtr("id2"),
					EntitlementTemplate: NewEntitlementTemplateFrom(
						MeteredEntitlementTemplate{
							Metadata: map[string]string{
								"name": "metered-1",
							},
							IsSoftLimit:             true,
							IssueAfterReset:         lo.ToPtr(1000.0),
							IssueAfterResetPriority: lo.ToPtr[uint8](3),
							PreserveOverageAtReset:  lo.ToPtr(true),
							UsagePeriod:             isodate.MustParse(t, "P1M"),
						}),
				},
				BillingCadence: isodate.MustParse(t, "P1M"),
			},
			wantErr: true,
		},
		{
			name: "Incompatible Billing Cadence",
			rCard: &FlatFeeRateCard{
				RateCardMeta: RateCardMeta{
					Key: "feature1",
					Price: NewPriceFrom(FlatPrice{
						Amount: decimal.NewFromInt(1000),
					}),
					FeatureKey: lo.ToPtr("feature1"),
					FeatureID:  lo.ToPtr("id1"),
					EntitlementTemplate: NewEntitlementTemplateFrom(
						MeteredEntitlementTemplate{
							Metadata: map[string]string{
								"name": "metered-1",
							},
							IsSoftLimit:             true,
							IssueAfterReset:         lo.ToPtr(1000.0),
							IssueAfterResetPriority: lo.ToPtr[uint8](3),
							PreserveOverageAtReset:  lo.ToPtr(true),
							UsagePeriod:             isodate.MustParse(t, "P1M"),
						}),
				},
				BillingCadence: lo.ToPtr(isodate.MustParse(t, "P3M")),
			},
			vCard: &UsageBasedRateCard{
				RateCardMeta: RateCardMeta{
					Key: "feature1",
					Price: NewPriceFrom(FlatPrice{
						Amount: decimal.NewFromInt(1000),
					}),
					FeatureKey: lo.ToPtr("feature1"),
					FeatureID:  lo.ToPtr("id1"),
					EntitlementTemplate: NewEntitlementTemplateFrom(
						MeteredEntitlementTemplate{
							Metadata: map[string]string{
								"name": "metered-1",
							},
							IsSoftLimit:             true,
							IssueAfterReset:         lo.ToPtr(1000.0),
							IssueAfterResetPriority: lo.ToPtr[uint8](3),
							PreserveOverageAtReset:  lo.ToPtr(true),
							UsagePeriod:             isodate.MustParse(t, "P1M"),
						}),
				},
				BillingCadence: isodate.MustParse(t, "P1M"),
			},
			wantErr: true,
		},
		{
			name: "Incompatible Entitlement Template Type",
			rCard: &FlatFeeRateCard{
				RateCardMeta: RateCardMeta{
					Key: "feature1",
					Price: NewPriceFrom(FlatPrice{
						Amount: decimal.NewFromInt(1000),
					}),
					FeatureKey: lo.ToPtr("feature1"),
					FeatureID:  lo.ToPtr("id1"),
					EntitlementTemplate: NewEntitlementTemplateFrom(
						MeteredEntitlementTemplate{
							Metadata: map[string]string{
								"name": "metered-1",
							},
							IsSoftLimit:             true,
							IssueAfterReset:         lo.ToPtr(1000.0),
							IssueAfterResetPriority: lo.ToPtr[uint8](3),
							PreserveOverageAtReset:  lo.ToPtr(true),
							UsagePeriod:             isodate.MustParse(t, "P1M"),
						}),
				},
				BillingCadence: lo.ToPtr(isodate.MustParse(t, "P1M")),
			},
			vCard: &UsageBasedRateCard{
				RateCardMeta: RateCardMeta{
					Key: "feature1",
					Price: NewPriceFrom(FlatPrice{
						Amount: decimal.NewFromInt(1000),
					}),
					FeatureKey: lo.ToPtr("feature1"),
					FeatureID:  lo.ToPtr("id1"),
					EntitlementTemplate: NewEntitlementTemplateFrom(
						BooleanEntitlementTemplate{
							Metadata: map[string]string{
								"name": "metered-1",
							},
						}),
				},
				BillingCadence: isodate.MustParse(t, "P1M"),
			},
			wantErr: true,
		},
		{
			name: "Incompatible Usage Period",
			rCard: &FlatFeeRateCard{
				RateCardMeta: RateCardMeta{
					Key: "feature1",
					Price: NewPriceFrom(FlatPrice{
						Amount: decimal.NewFromInt(1000),
					}),
					FeatureKey: lo.ToPtr("feature1"),
					FeatureID:  lo.ToPtr("id1"),
					EntitlementTemplate: NewEntitlementTemplateFrom(
						MeteredEntitlementTemplate{
							Metadata: map[string]string{
								"name": "metered-1",
							},
							IsSoftLimit:             true,
							IssueAfterReset:         lo.ToPtr(1000.0),
							IssueAfterResetPriority: lo.ToPtr[uint8](3),
							PreserveOverageAtReset:  lo.ToPtr(true),
							UsagePeriod:             isodate.MustParse(t, "P1M"),
						}),
				},
				BillingCadence: lo.ToPtr(isodate.MustParse(t, "P1M")),
			},
			vCard: &UsageBasedRateCard{
				RateCardMeta: RateCardMeta{
					Key: "feature1",
					Price: NewPriceFrom(FlatPrice{
						Amount: decimal.NewFromInt(1000),
					}),
					FeatureKey: lo.ToPtr("feature1"),
					FeatureID:  lo.ToPtr("id1"),
					EntitlementTemplate: NewEntitlementTemplateFrom(
						MeteredEntitlementTemplate{
							Metadata: map[string]string{
								"name": "metered-1",
							},
							IsSoftLimit:             true,
							IssueAfterReset:         lo.ToPtr(1000.0),
							IssueAfterResetPriority: lo.ToPtr[uint8](3),
							PreserveOverageAtReset:  lo.ToPtr(true),
							UsagePeriod:             isodate.MustParse(t, "P3M"),
						}),
				},
				BillingCadence: isodate.MustParse(t, "P1M"),
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := NewRateCardWithOverlay(test.rCard, test.vCard).Validate()
			if test.wantErr {
				assert.Error(t, err)
				t.Logf("Expected error: %v", err)

				var expectedErr *models.GenericValidationError
				assert.ErrorAs(t, err, &expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
