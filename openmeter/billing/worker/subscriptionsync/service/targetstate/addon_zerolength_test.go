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

// An add-on applied in the same wall-clock second as the item start splits the
// item into two versions. Truncating the service period to one second turns the
// first version into a zero-length flat-fee line, which is billed at the full
// amount, and the second version is billed for the month. One nanosecond later
// the stub is a real one-second period, prorates to nothing, and is dropped.
func TestAddonSplitInTheSameSecondBillsTheBaseFeeOnce(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	fee := alpacadecimal.NewFromInt(100)

	sameSecond := chargedTotal(t, start, start.Add(999999999*time.Nanosecond))
	nextSecond := chargedTotal(t, start, start.Add(time.Second))

	t.Logf("add-on at S+999999999ns total %s", sameSecond)
	t.Logf("add-on at S+1s total %s", nextSecond)

	require.True(t, sameSecond.Equal(nextSecond),
		"totals %s and %s differ by %s; a sub-second split must not add another base fee",
		sameSecond, nextSecond, sameSecond.Sub(nextSecond))
	require.True(t, sameSecond.Equal(fee), "January total %s, want one base fee of %s", sameSecond, fee)
}

func chargedTotal(t *testing.T, start, split time.Time) alpacadecimal.Decimal {
	t.Helper()

	subs := monthlyFlatFeeSplit(t, start, split)
	it, err := NewPhaseIterator(slog.Default(), noop.NewTracerProvider().Tracer("addon"), subs, "p1")
	require.NoError(t, err)

	out, err := it.Generate(context.Background(), time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	currency := lo.Must(currencies.NewFiatCurrency(currencyx.Code("USD")))
	total := alpacadecimal.Zero
	for _, line := range out {
		si := StateItem{
			SubscriptionItemWithPeriods:  line,
			Currency:                     currency,
			Subscription:                 subs.Subscription,
			SubscriptionEndProrationMode: billing.SubscriptionEndProrationModeBillActualPeriod,
		}
		gl, err := si.GetExpectedLine()
		require.NoError(t, err)
		if gl == nil {
			t.Logf("dropped %s service [%s .. %s]", line.UniqueID, line.ServicePeriod.From.Format(time.RFC3339Nano), line.ServicePeriod.To.Format(time.RFC3339Nano))
			continue
		}
		flat, err := gl.Price.AsFlat()
		require.NoError(t, err)
		t.Logf("billed %s service [%s .. %s] $%s", line.UniqueID, line.ServicePeriod.From.Format(time.RFC3339Nano), line.ServicePeriod.To.Format(time.RFC3339Nano), flat.Amount)
		total = total.Add(flat.Amount)
	}
	return total
}

func monthlyFlatFeeSplit(t *testing.T, start, split time.Time) subscription.SubscriptionView {
	t.Helper()

	cadence := datetime.MustParseDuration(t, "P1M")
	rc := &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:  "base",
			Name: "Base fee",
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromInt(100),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
		},
		BillingCadence: &cadence,
	}

	version := func(from time.Time, to *time.Time) subscription.SubscriptionItemView {
		spec := subscription.SubscriptionItemSpec{
			CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
				CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
					ItemKey:  "base",
					PhaseKey: "p1",
					RateCard: rc,
				},
			},
		}
		view := subscription.SubscriptionItemView{Spec: spec}
		view.SubscriptionItem.RateCard = rc
		view.SubscriptionItem.ActiveFrom = from
		view.SubscriptionItem.ActiveTo = to
		return view
	}

	first := version(start, &split)
	second := version(split, nil)

	phaseSpec := subscription.SubscriptionPhaseSpec{
		CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
			PhaseKey:   "p1",
			StartAfter: datetime.ISODuration{},
		},
		ItemsByKey: map[string][]*subscription.SubscriptionItemSpec{
			"base": {&first.Spec, &second.Spec},
		},
	}
	phaseView := subscription.SubscriptionPhaseView{
		Spec:       phaseSpec,
		ItemsByKey: map[string][]subscription.SubscriptionItemView{"base": {first, second}},
	}
	phaseView.SubscriptionPhase.Key = "p1"
	phaseView.SubscriptionPhase.ID = "phaseID"
	phaseView.SubscriptionPhase.ActiveFrom = start

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
				BillingAnchor: start,
			},
			Phases: map[string]*subscription.SubscriptionPhaseSpec{"p1": &phaseSpec},
		},
		Phases: []subscription.SubscriptionPhaseView{phaseView},
	}
	subs.Subscription.ID = "subID"
	subs.Subscription.ActiveFrom = start
	subs.Subscription.BillingAnchor = start
	subs.Subscription.BillingCadence = cadence
	subs.Subscription.InvoiceCurrency = currencyx.Code("USD")
	subs.Subscription.ProRatingConfig = productcatalog.ProRatingConfig{
		Enabled: true,
		Mode:    productcatalog.ProRatingModeProratePrices,
	}
	return subs
}
