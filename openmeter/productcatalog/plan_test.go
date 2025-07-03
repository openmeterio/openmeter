package productcatalog_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/datetime"
)

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
					BillingCadence: datetime.MustParse(t, "P1M"),
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
					BillingCadence: datetime.MustParse(t, "P1M"),
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
					BillingCadence: datetime.MustParse(t, "P1M"),
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
					BillingCadence: datetime.MustParse(t, "P1M"),
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
					BillingCadence: datetime.MustParse(t, "P1M"),
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
					BillingCadence: datetime.MustParse(t, "P1M"),
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
					BillingCadence: datetime.MustParse(t, "P1M"),
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
					BillingCadence: datetime.MustParse(t, "P1M"),
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
					BillingCadence: datetime.MustParse(t, "P1M"),
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
				Currency:        "USD",
				BillingCadence:  datetime.MustParse(t, "P1M"),
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
							BillingCadence: lo.ToPtr(testutils.GetISODuration(t, "P1M")),
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
							BillingCadence: lo.ToPtr(testutils.GetISODuration(t, "P1M")),
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
				Currency:        "USD",
				BillingCadence:  datetime.MustParse(t, "P1M"),
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
							BillingCadence: lo.ToPtr(testutils.GetISODuration(t, "P1M")),
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
							BillingCadence: lo.ToPtr(testutils.GetISODuration(t, "P1W")),
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
				Currency:        "USD",
				BillingCadence:  datetime.MustParse(t, "P1M"),
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
							BillingCadence: lo.ToPtr(testutils.GetISODuration(t, "P1M")),
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
							BillingCadence: lo.ToPtr(testutils.GetISODuration(t, "P1W")),
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
