package productcatalog_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/testutils"
)

func TestPlanStatus(t *testing.T) {
	now := time.Now()

	tests := []struct {
		Name string

		Effective productcatalog.EffectivePeriod
		Expected  productcatalog.PlanStatus
	}{
		{
			Name: "Draft",
			Effective: productcatalog.EffectivePeriod{
				EffectiveFrom: nil,
				EffectiveTo:   nil,
			},
			Expected: productcatalog.DraftStatus,
		},
		{
			Name: "Archived",
			Effective: productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now.Add(-24 * time.Hour)),
				EffectiveTo:   lo.ToPtr(now.Add(-1 * time.Hour)),
			},
			Expected: productcatalog.ArchivedStatus,
		},
		{
			Name: "Active with open end",
			Effective: productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now.Add(-24 * time.Hour)),
				EffectiveTo:   nil,
			},
			Expected: productcatalog.ActiveStatus,
		},
		{
			Name: "Active with fixed end",
			Effective: productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now.Add(-24 * time.Hour)),
				EffectiveTo:   lo.ToPtr(now.Add(24 * time.Hour)),
			},
			Expected: productcatalog.ActiveStatus,
		},
		{
			Name: "Scheduled with open end",
			Effective: productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now.Add(24 * time.Hour)),
				EffectiveTo:   nil,
			},
			Expected: productcatalog.ScheduledStatus,
		},
		{
			Name: "Scheduled with fixed period",
			Effective: productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now.Add(24 * time.Hour)),
				EffectiveTo:   lo.ToPtr(now.Add(48 * time.Hour)),
			},
			Expected: productcatalog.ScheduledStatus,
		},
		{
			Name: "Invalid with inverse period",
			Effective: productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now.Add(24 * time.Hour)),
				EffectiveTo:   lo.ToPtr(now.Add(-24 * time.Hour)),
			},
			Expected: productcatalog.InvalidStatus,
		},
		{
			Name: "Invalid with no start with end in the past",
			Effective: productcatalog.EffectivePeriod{
				EffectiveFrom: nil,
				EffectiveTo:   lo.ToPtr(now.Add(-24 * time.Hour)),
			},
			Expected: productcatalog.ArchivedStatus,
		},
		{
			Name: "Invalid with no start with end in the future",
			Effective: productcatalog.EffectivePeriod{
				EffectiveFrom: nil,
				EffectiveTo:   lo.ToPtr(now.Add(24 * time.Hour)),
			},
			Expected: productcatalog.ActiveStatus,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.Expected, test.Effective.Status())
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
				Alignment: productcatalog.Alignment{
					BillablesMustAlign: true,
				},
				Version:  1,
				Currency: "USD",
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

		err := p.ValidForCreatingSubscriptions()
		assert.NoError(t, err)
	})

	t.Run("Should allow plan with misaligned RateCards if not enforced", func(t *testing.T) {
		p := productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:            "Plan 1",
				Key:             "plan-1",
				EffectivePeriod: productcatalog.EffectivePeriod{},
				Alignment: productcatalog.Alignment{
					BillablesMustAlign: false,
				},
				Version:  1,
				Currency: "USD",
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

		err := p.ValidForCreatingSubscriptions()
		assert.NoError(t, err)
	})

	t.Run("Should NOT allow plan with misaligned RateCards if enforced", func(t *testing.T) {
		p := productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:            "Plan 1",
				Key:             "plan-1",
				EffectivePeriod: productcatalog.EffectivePeriod{},
				Alignment: productcatalog.Alignment{
					BillablesMustAlign: true,
				},
				Version:  1,
				Currency: "USD",
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

		err := p.ValidForCreatingSubscriptions()
		assert.Error(t, err)
		assert.ErrorContains(t, err, "must have the same billing cadence")
	})
}
