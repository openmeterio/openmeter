package targetstate

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
)

// A one-time flat fee has no billing cadence, so its full service period is the
// item's own active window. When the phase ends later, the phase end bounds that
// window but must not replace the item end. GetCadence in subscriptionspec.go
// already takes the earlier of the two; GetFullServicePeriodAt has to do the same,
// because the gathering line prorates price.Amount by
// ServicePeriod.Duration() / FullServicePeriod.Duration().
func TestOneTimeFlatFeeProratesAgainstTheItemPeriod(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	itemEnd := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	lateStart := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	phaseEnd := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)

	for _, tc := range []struct {
		name     string
		itemFrom time.Time
		itemTo   time.Time
		phaseTo  *time.Time
		term     productcatalog.PaymentTermType
	}{
		// Same 31 days of the same 365-day phase. A late start bills the whole fee.
		// An early end used to bill $8.49, because the phase end replaced the item end.
		{name: "in advance, 31 days at the start of a 365-day phase", itemFrom: start, itemTo: itemEnd, phaseTo: &phaseEnd, term: productcatalog.InAdvancePaymentTerm},
		{name: "in advance, 31 days at the end of a 365-day phase", itemFrom: lateStart, itemTo: phaseEnd, phaseTo: &phaseEnd, term: productcatalog.InAdvancePaymentTerm},
		{name: "in arrears, 31 days at the start of a 365-day phase", itemFrom: start, itemTo: itemEnd, phaseTo: &phaseEnd, term: productcatalog.InArrearsPaymentTerm},
		{name: "in arrears, 31 days at the end of a 365-day phase", itemFrom: lateStart, itemTo: phaseEnd, phaseTo: &phaseEnd, term: productcatalog.InArrearsPaymentTerm},
		{name: "in advance, phase open ended", itemFrom: start, itemTo: itemEnd, phaseTo: nil, term: productcatalog.InAdvancePaymentTerm},
	} {
		t.Run(tc.name, func(t *testing.T) {
			subs := oneTimeFlatFeeSubscription(t, tc.itemFrom, tc.itemTo, tc.phaseTo, tc.term)

			it, err := NewPhaseIterator(slog.Default(), noop.NewTracerProvider().Tracer("onetime"), subs, "p1")
			require.NoError(t, err)

			out, err := it.Generate(context.Background(), time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC))
			require.NoError(t, err)
			require.Len(t, out, 1)

			line := out[0]
			require.True(t, line.FullServicePeriod.From.Equal(tc.itemFrom), "full period from %s", line.FullServicePeriod.From)
			require.True(t, line.FullServicePeriod.To.Equal(tc.itemTo), "full period to %s, want item end %s", line.FullServicePeriod.To, tc.itemTo)
			require.Equal(t, "1", line.PeriodPercentage().String())

			si := StateItem{
				SubscriptionItemWithPeriods:  line,
				Currency:                     lo.Must(currencies.NewFiatCurrency(currencyx.Code("USD"))),
				Subscription:                 subs.Subscription,
				SubscriptionEndProrationMode: billing.SubscriptionEndProrationModeBillActualPeriod,
			}

			gl, err := si.GetExpectedLine()
			require.NoError(t, err)
			require.NotNil(t, gl)

			flat, err := gl.Price.AsFlat()
			require.NoError(t, err)
			require.Equal(t, "100", flat.Amount.String(), "a $100 one-time fee served for its whole item window")
		})
	}
}

func oneTimeFlatFeeSubscription(t *testing.T, start, itemEnd time.Time, phaseEnd *time.Time, term productcatalog.PaymentTermType) subscription.SubscriptionView {
	t.Helper()

	rc := &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:  "setup-fee",
			Name: "Setup fee",
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromInt(100),
				PaymentTerm: term,
			}),
		},
		BillingCadence: nil,
	}

	spec := subscription.SubscriptionItemSpec{
		CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
			CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
				ItemKey:  "setup-fee",
				PhaseKey: "p1",
				RateCard: rc,
			},
		},
	}

	itemView := subscription.SubscriptionItemView{Spec: spec}
	itemView.SubscriptionItem.RateCard = rc
	itemView.SubscriptionItem.ActiveFrom = start
	itemView.SubscriptionItem.ActiveTo = lo.ToPtr(itemEnd)

	phaseSpec := subscription.SubscriptionPhaseSpec{
		CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
			PhaseKey:   "p1",
			StartAfter: datetime.ISODuration{},
		},
		ItemsByKey: map[string][]*subscription.SubscriptionItemSpec{"setup-fee": {&spec}},
	}

	phaseView := subscription.SubscriptionPhaseView{
		Spec:       phaseSpec,
		ItemsByKey: map[string][]subscription.SubscriptionItemView{"setup-fee": {itemView}},
	}
	phaseView.SubscriptionPhase.Key = "p1"
	phaseView.SubscriptionPhase.ID = "phaseID"
	phaseView.SubscriptionPhase.ActiveFrom = start

	cadence := datetime.MustParseDuration(t, "P1M")

	subs := subscription.SubscriptionView{
		Spec: subscription.SubscriptionSpec{
			CreateSubscriptionPlanInput: subscription.CreateSubscriptionPlanInput{
				BillingCadence: cadence,
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			CreateSubscriptionCustomerInput: subscription.CreateSubscriptionCustomerInput{
				ActiveFrom:    start,
				ActiveTo:      phaseEnd,
				BillingAnchor: start,
			},
			Phases: map[string]*subscription.SubscriptionPhaseSpec{"p1": &phaseSpec},
		},
		Phases: []subscription.SubscriptionPhaseView{phaseView},
	}
	subs.Subscription.ID = "subID"
	subs.Subscription.ActiveFrom = start
	subs.Subscription.ActiveTo = phaseEnd
	subs.Subscription.BillingAnchor = start
	subs.Subscription.BillingCadence = cadence
	subs.Subscription.InvoiceCurrency = currencyx.Code("USD")
	subs.Subscription.ProRatingConfig = productcatalog.ProRatingConfig{
		Enabled: true,
		Mode:    productcatalog.ProRatingModeProratePrices,
	}

	return subs
}
