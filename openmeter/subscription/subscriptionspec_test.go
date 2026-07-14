package subscription_test

import (
	"errors"
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestGetFullServicePeriodAtInputValidate(t *testing.T) {
	clock.FreezeTime(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
	t.Cleanup(clock.UnFreeze)

	tests := []struct {
		name string
		inp  subscription.GetFullServicePeriodAtInput
		want error
	}{
		{
			name: "missing at",
			inp:  subscription.GetFullServicePeriodAtInput{},
			want: errors.New("at is zero"),
		},
		{
			name: "missing aligned billing anchor",
			inp: subscription.GetFullServicePeriodAtInput{
				At: clock.Now(),
			},
			want: errors.New("aligned billing anchor is zero"),
		},
		{
			name: "at outside of subscription period",
			inp: subscription.GetFullServicePeriodAtInput{
				At:                   clock.Now().Add(time.Hour),
				AlignedBillingAnchor: clock.Now(),
				SubscriptionCadence: models.CadencedModel{
					ActiveFrom: clock.Now().Add(-time.Hour),
					ActiveTo:   lo.ToPtr(clock.Now()),
				},
			},
			want: errors.New("subscription is not active at 2020-01-01 01:00:00 +0000 UTC: [2019-12-31 23:00:00 +0000 UTC, 2020-01-01 00:00:00 +0000 UTC]"),
		},
		{
			name: "at outside of phase cadence",
			inp: subscription.GetFullServicePeriodAtInput{
				At:                   clock.Now().Add(time.Minute),
				AlignedBillingAnchor: clock.Now(),
				SubscriptionCadence: models.CadencedModel{
					ActiveFrom: clock.Now().Add(-time.Hour),
					ActiveTo:   lo.ToPtr(clock.Now().Add(time.Hour)),
				},
				PhaseCadence: models.CadencedModel{
					ActiveFrom: clock.Now().Add(-time.Hour),
					ActiveTo:   lo.ToPtr(clock.Now()),
				},
			},
			want: errors.New("phase is not active at 2020-01-01 00:01:00 +0000 UTC: [2019-12-31 23:00:00 +0000 UTC, 2020-01-01 00:00:00 +0000 UTC]"),
		},
		{
			name: "at outside of item cadence",
			inp: subscription.GetFullServicePeriodAtInput{
				At:                   clock.Now().Add(time.Minute),
				AlignedBillingAnchor: clock.Now(),
				SubscriptionCadence: models.CadencedModel{
					ActiveFrom: clock.Now().Add(-time.Hour),
					ActiveTo:   lo.ToPtr(clock.Now().Add(time.Hour)),
				},
				PhaseCadence: models.CadencedModel{
					ActiveFrom: clock.Now().Add(-time.Hour),
					ActiveTo:   lo.ToPtr(clock.Now().Add(time.Hour)),
				},
				ItemCadence: models.CadencedModel{
					ActiveFrom: clock.Now().Add(-time.Hour),
					ActiveTo:   lo.ToPtr(clock.Now()),
				},
			},
			want: errors.New("item is not active at 2020-01-01 00:01:00 +0000 UTC: [2019-12-31 23:00:00 +0000 UTC, 2020-01-01 00:00:00 +0000 UTC]"),
		},
		{
			name: "for a zero length item during a phase",
			inp: subscription.GetFullServicePeriodAtInput{
				At:                   clock.Now(),
				AlignedBillingAnchor: clock.Now(),
				SubscriptionCadence: models.CadencedModel{
					ActiveFrom: clock.Now().Add(-time.Hour),
					ActiveTo:   lo.ToPtr(clock.Now().Add(time.Hour)),
				},
				PhaseCadence: models.CadencedModel{
					ActiveFrom: clock.Now().Add(-time.Hour),
					ActiveTo:   lo.ToPtr(clock.Now().Add(time.Hour)),
				},
				ItemCadence: models.CadencedModel{
					ActiveFrom: clock.Now(),
					ActiveTo:   lo.ToPtr(clock.Now()),
				},
			},
			want: nil,
		},
		{
			name: "at end of subscription for a zero length last item",
			inp: subscription.GetFullServicePeriodAtInput{
				At:                   clock.Now(),
				AlignedBillingAnchor: clock.Now(),
				SubscriptionCadence: models.CadencedModel{
					ActiveFrom: clock.Now().Add(-time.Hour),
					ActiveTo:   lo.ToPtr(clock.Now()),
				},
				PhaseCadence: models.CadencedModel{
					ActiveFrom: clock.Now().Add(-time.Hour),
					ActiveTo:   lo.ToPtr(clock.Now()),
				},
				ItemCadence: models.CadencedModel{
					ActiveFrom: clock.Now(),
					ActiveTo:   lo.ToPtr(clock.Now()),
				},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.want == nil {
				require.NoError(t, tt.inp.Validate())
			} else {
				require.ErrorContains(t, tt.inp.Validate(), tt.want.Error())
			}
		})
	}
}

func TestSubscriptionSpecHasUnitConfig(t *testing.T) {
	card := func(uc *productcatalog.UnitConfig) productcatalog.RateCard {
		return &productcatalog.UsageBasedRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:        "feat-1",
				Name:       "Feature 1",
				FeatureKey: lo.ToPtr("feat-1"),
				Price:      productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: decimal.NewFromInt(1)}),
				UnitConfig: uc,
			},
		}
	}
	item := func(uc *productcatalog.UnitConfig) *subscription.SubscriptionItemSpec {
		return &subscription.SubscriptionItemSpec{
			CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
				CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
					PhaseKey: "phase-1",
					ItemKey:  "item-1",
					RateCard: card(uc),
				},
			},
		}
	}
	specWith := func(items ...*subscription.SubscriptionItemSpec) subscription.SubscriptionSpec {
		return subscription.SubscriptionSpec{
			Phases: map[string]*subscription.SubscriptionPhaseSpec{
				"phase-1": {
					ItemsByKey: map[string][]*subscription.SubscriptionItemSpec{"item-1": items},
				},
			},
		}
	}
	divide := &productcatalog.UnitConfig{Operation: productcatalog.UnitConfigOperationDivide, ConversionFactor: decimal.NewFromInt(1000)}

	t.Run("empty spec has none", func(t *testing.T) {
		s := subscription.SubscriptionSpec{}
		assert.False(t, s.HasUnitConfig())
	})

	t.Run("no item carries unit_config", func(t *testing.T) {
		s := specWith(item(nil))
		assert.False(t, s.HasUnitConfig())
	})

	t.Run("an item carrying unit_config is detected", func(t *testing.T) {
		s := specWith(item(nil), item(divide))
		assert.True(t, s.HasUnitConfig())
	})
}
