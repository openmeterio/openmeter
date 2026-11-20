package rating

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	billingratingservice "github.com/openmeterio/openmeter/openmeter/billing/rating/service"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestTotalsReuseRequeriesUsageAndReratesChangedInputs(t *testing.T) {
	t.Parallel()

	fixture := newGetDetailedRatingForUsageFixture(t, billingrating.GenerateDetailedLinesResult{})
	query := &countingTotalsStreaming{Connector: fixture.config.StreamingConnector}
	rater := &countingTotalsRater{Service: billingratingservice.New(billingratingservice.Config{UnitConfigEnabled: true})}
	fixture.config.StreamingConnector = query
	fixture.config.RatingService = rater
	svc, err := New(fixture.config)
	require.NoError(t, err)
	reuse, err := NewTotalsReuse(2)
	require.NoError(t, err)
	var hits, misses int
	ctx := WithTotalsReuse(t.Context(), reuse, func(_ context.Context, hit bool) {
		if hit {
			hits++
		} else {
			misses++
		}
	})
	input := GetTotalsForUsageInput{
		Charge: fixture.input.Charge, Customer: fixture.input.Customer,
		FeatureMeter: fixture.input.FeatureMeter, StoredAtLT: fixture.input.StoredAtLT,
	}

	// Given a previous full rating, an unchanged quantity only reuses rating.
	initial, err := svc.GetTotalsForUsage(ctx, input)
	require.NoError(t, err)
	require.Equal(t, float64(36), initial.Totals.Total.InexactFloat64())
	input.StoredAtLT = input.StoredAtLT.Add(time.Hour)
	unchanged, err := svc.GetTotalsForUsage(ctx, input)
	require.NoError(t, err)
	require.Equal(t, initial, unchanged)
	require.Equal(t, int64(2), query.calls.Load())
	require.Equal(t, int64(1), rater.calls.Load())
	require.Equal(t, 1, hits)
	require.Equal(t, 1, misses)

	// When late usage arrives inside the original period, requerying sees it.
	source := query.Connector.(*streamingtestutils.MockStreamingConnector)
	source.AddSimpleEvent("meter-1", 4, input.Charge.Intent.GetEffectiveServicePeriod().From.Add(time.Hour), streamingtestutils.WithStoredAt(input.StoredAtLT.Add(-time.Minute)))
	late, err := svc.GetTotalsForUsage(ctx, input)
	require.NoError(t, err)
	require.Equal(t, float64(48), late.Totals.Total.InexactFloat64())
	require.Equal(t, int64(2), rater.calls.Load())

	// Corrections may reduce cumulative usage too; no monotonicity is assumed.
	source.AddSimpleEvent("meter-1", -2, input.Charge.Intent.GetEffectiveServicePeriod().From.Add(2*time.Hour), streamingtestutils.WithStoredAt(input.StoredAtLT.Add(-time.Second)))
	corrected, err := svc.GetTotalsForUsage(ctx, input)
	require.NoError(t, err)
	require.Equal(t, float64(42), corrected.Totals.Total.InexactFloat64())

	// Same charge identity and quantity, different effective price.
	require.NoError(t, input.Charge.Intent.MutateEffective(func(fields *usagebased.IntentMutableFields) error {
		fields.Price = *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(5)})
		return nil
	}))
	repriced, err := svc.GetTotalsForUsage(ctx, input)
	require.NoError(t, err)
	require.Equal(t, float64(70), repriced.Totals.Total.InexactFloat64())
	require.Equal(t, int64(4), rater.calls.Load())

	// Then calls without opt-in still rate normally.
	_, err = svc.GetTotalsForUsage(t.Context(), input)
	require.NoError(t, err)
	require.Equal(t, int64(5), rater.calls.Load())
	require.Equal(t, int64(6), query.calls.Load())
	require.Equal(t, 1, hits)
	require.Equal(t, 4, misses)

	// End-of-period commitment policy changes rating at the same quantity.
	minimum := alpacadecimal.NewFromInt(100)
	require.NoError(t, input.Charge.Intent.MutateEffective(func(fields *usagebased.IntentMutableFields) error {
		fields.Price = *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount:      alpacadecimal.NewFromInt(5),
			Commitments: productcatalog.Commitments{MinimumAmount: &minimum},
		})

		return nil
	}))
	committed, err := svc.GetTotalsForUsage(ctx, input)
	require.NoError(t, err)
	require.Equal(t, float64(100), committed.Totals.Total.InexactFloat64())
	input.IgnoreMinimumCommitment = true
	uncommitted, err := svc.GetTotalsForUsage(ctx, input)
	require.NoError(t, err)
	require.Equal(t, float64(70), uncommitted.Totals.Total.InexactFloat64())

	// Unit conversion and discounts also require fresh rating, even though the
	// raw meter quantity has not changed.
	require.NoError(t, input.Charge.Intent.MutateEffective(func(fields *usagebased.IntentMutableFields) error {
		fields.UnitConfig = &productcatalog.UnitConfig{
			Operation:        productcatalog.UnitConfigOperationDivide,
			ConversionFactor: alpacadecimal.NewFromInt(2),
		}

		return nil
	}))
	converted, err := svc.GetTotalsForUsage(ctx, input)
	require.NoError(t, err)
	require.Equal(t, float64(35), converted.Totals.Total.InexactFloat64())
	require.NoError(t, input.Charge.Intent.MutateEffective(func(fields *usagebased.IntentMutableFields) error {
		fields.Discounts.Percentage = &billing.PercentageDiscount{
			PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(50)},
			CorrelationID:      "half-price",
		}

		return nil
	}))
	discounted, err := svc.GetTotalsForUsage(ctx, input)
	require.NoError(t, err)
	require.Equal(t, float64(17.5), discounted.Totals.Total.InexactFloat64())
	require.Equal(t, int64(9), rater.calls.Load())
}

func TestTotalsReuseComparesAllRatingInputsExactly(t *testing.T) {
	t.Parallel()
	fixture := newGetDetailedRatingForUsageFixture(t, billingrating.GenerateDetailedLinesResult{})
	intent := usagebased.RateableIntent{
		Intent:        fixture.input.Charge.Intent.GetEffectiveIntent(),
		ServicePeriod: fixture.input.Charge.Intent.GetEffectiveServicePeriod(),
		MeterValue:    alpacadecimal.RequireFromString("9.99999999999999999"),
	}
	original, ok := totalsReuseInputs(intent, false)
	require.True(t, ok)
	require.Contains(t, original, "9.99999999999999999")
	otherQuantity := intent
	otherQuantity.MeterValue = alpacadecimal.NewFromInt(10)
	other, ok := totalsReuseInputs(otherQuantity, false)
	require.True(t, ok)
	require.NotEqual(t, original, other)

	changedPeriod := intent
	changedPeriod.ServicePeriod.To = changedPeriod.ServicePeriod.To.Add(time.Hour)
	other, ok = totalsReuseInputs(changedPeriod, false)
	require.True(t, ok)
	require.NotEqual(t, original, other)
	other, ok = totalsReuseInputs(intent, true)
	require.True(t, ok)
	require.NotEqual(t, original, other)

	// Currency identity alone does not capture rating precision.
	currency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).WithCode("credits").WithName("Credits").WithPrecision(2).Build()
	require.NoError(t, err)
	intent.Intent.Intent.Currency.Currency = currency
	precision2, ok := totalsReuseInputs(intent, false)
	require.True(t, ok)
	currency, err = currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).WithCode("credits").WithName("Credits").WithPrecision(4).Build()
	require.NoError(t, err)
	intent.Intent.Intent.Currency.Currency = currency
	precision4, ok := totalsReuseInputs(intent, false)
	require.True(t, ok)
	require.NotEqual(t, precision2, precision4)

	intent.Name = strings.Repeat("x", maxTotalsReuseInputBytes)
	_, ok = totalsReuseInputs(intent, false)
	require.False(t, ok)
}

func TestTotalsReuseBoundsEntriesAndSeparatesServices(t *testing.T) {
	t.Parallel()
	fixture := newGetDetailedRatingForUsageFixture(t, billingrating.GenerateDetailedLinesResult{})
	rater := &countingTotalsRater{Service: billingratingservice.New(billingratingservice.Config{})}
	fixture.config.RatingService = rater
	svc, err := New(fixture.config)
	require.NoError(t, err)
	reuse, err := NewTotalsReuse(1)
	require.NoError(t, err)
	ctx := WithTotalsReuse(t.Context(), reuse, nil)
	input := GetTotalsForUsageInput{
		Charge: fixture.input.Charge, Customer: fixture.input.Customer,
		FeatureMeter: fixture.input.FeatureMeter, StoredAtLT: fixture.input.StoredAtLT,
	}

	// Given capacity one, another charge evicts the first one.
	for _, id := range []string{"first", "second", "first"} {
		input.Charge.ID = id
		_, err := svc.GetTotalsForUsage(ctx, input)
		require.NoError(t, err)
		require.Equal(t, 1, reuse.entries.Len())
	}

	require.Equal(t, int64(3), rater.calls.Load())

	// Different service instances may have different rating configuration.
	otherService, err := New(fixture.config)
	require.NoError(t, err)
	_, err = otherService.GetTotalsForUsage(ctx, input)
	require.NoError(t, err)
	require.Equal(t, int64(4), rater.calls.Load())
}

func TestTotalsReuseDoesNotRememberFailuresOrWarnings(t *testing.T) {
	t.Parallel()
	for _, issue := range []error{
		errors.New("rating unavailable"),
		billing.ValidationIssue{Code: "rating_warning", Message: "rating warning", Severity: billing.ValidationIssueSeverityWarning},
	} {
		t.Run(issue.Error(), func(t *testing.T) {
			fixture := newGetDetailedRatingForUsageFixture(t, billingrating.GenerateDetailedLinesResult{})
			rater := &countingTotalsRater{Service: billingratingservice.New(billingratingservice.Config{}), err: issue}
			fixture.config.RatingService = rater
			svc, err := New(fixture.config)
			require.NoError(t, err)
			reuse, err := NewTotalsReuse(1)
			require.NoError(t, err)
			ctx := WithTotalsReuse(t.Context(), reuse, nil)
			input := GetTotalsForUsageInput{
				Charge: fixture.input.Charge, Customer: fixture.input.Customer,
				FeatureMeter: fixture.input.FeatureMeter, StoredAtLT: fixture.input.StoredAtLT,
			}
			for range 2 {
				_, err = svc.GetTotalsForUsage(ctx, input)
				require.Error(t, err)
			}

			require.Equal(t, int64(2), rater.calls.Load())
			require.Zero(t, reuse.entries.Len())
		})
	}
}

func TestTotalsReuseConcurrentCalculations(t *testing.T) {
	t.Parallel()
	fixture := newGetDetailedRatingForUsageFixture(t, billingrating.GenerateDetailedLinesResult{})
	rater := &countingTotalsRater{Service: billingratingservice.New(billingratingservice.Config{})}
	fixture.config.RatingService = rater
	svc, err := New(fixture.config)
	require.NoError(t, err)
	reuse, err := NewTotalsReuse(1)
	require.NoError(t, err)
	ctx := WithTotalsReuse(t.Context(), reuse, nil)
	input := GetTotalsForUsageInput{
		Charge: fixture.input.Charge, Customer: fixture.input.Customer,
		FeatureMeter: fixture.input.FeatureMeter, StoredAtLT: fixture.input.StoredAtLT,
	}
	initial, err := svc.GetTotalsForUsage(ctx, input)
	require.NoError(t, err)
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			out, err := svc.GetTotalsForUsage(ctx, input)
			require.NoError(t, err)
			require.Equal(t, initial, out)
		})
	}

	wg.Wait()
	require.Equal(t, int64(1), rater.calls.Load())
}

type countingTotalsRater struct {
	billingrating.Service
	calls atomic.Int64
	err   error
}

func (r *countingTotalsRater) GenerateDetailedLines(in billingrating.StandardLineAccessor, opts ...billingrating.GenerateDetailedLinesOption) (billingrating.GenerateDetailedLinesResult, error) {
	r.calls.Add(1)
	if r.err != nil {
		return billingrating.GenerateDetailedLinesResult{}, r.err
	}

	return r.Service.GenerateDetailedLines(in, opts...)
}

type countingTotalsStreaming struct {
	streaming.Connector
	calls atomic.Int64
}

func (s *countingTotalsStreaming) QueryMeter(ctx context.Context, namespace string, m meter.Meter, params streaming.QueryParams) ([]meter.MeterQueryRow, error) {
	s.calls.Add(1)
	return s.Connector.QueryMeter(ctx, namespace, m, params)
}

// The benchmark controls the quantity returned by each fresh meter read. This
// keeps query latency and a growing mock query history out of the comparison.
type benchmarkMeter struct {
	streaming.Connector
	quantity float64
}

func (m *benchmarkMeter) QueryMeter(context.Context, string, meter.Meter, streaming.QueryParams) ([]meter.MeterQueryRow, error) {
	return []meter.MeterQueryRow{{Value: m.quantity}}, nil
}

func BenchmarkGetTotalsForUsageReuse(b *testing.B) {
	type workload struct {
		name     string
		price    *productcatalog.Price
		quantity float64
	}

	workloads := []workload{
		{"unit", productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(3)}), 12},
	}
	for _, mode := range []productcatalog.TieredPriceMode{productcatalog.GraduatedTieredPrice, productcatalog.VolumeTieredPrice} {
		for _, size := range []int{2, 5, 10} {
			tiers := make([]productcatalog.PriceTier, size)
			for i := range tiers {
				tiers[i].UnitPrice = &productcatalog.PriceTierUnitPrice{Amount: alpacadecimal.NewFromInt(int64(size - i))}
				if i < size-1 {
					upTo := alpacadecimal.NewFromInt(int64(10 * (i + 1)))
					tiers[i].UpToAmount = &upTo
				}
			}

			price := productcatalog.NewPriceFrom(productcatalog.TieredPrice{Mode: mode, Tiers: tiers})
			name := fmt.Sprintf("%s_%d", mode, size)
			workloads = append(workloads, workload{name + "/first_tier", price, 3}, workload{name + "/last_tier", price, float64(10 * size)})
		}
	}

	for _, w := range workloads {
		for _, hitPercent := range []int{0, 50, 75, 100} {
			for _, enabled := range []bool{false, true} {
				path := "full_rating"
				if enabled {
					path = "reuse_rating"
				}

				b.Run(fmt.Sprintf("%s/hits_%d/%s", w.name, hitPercent, path), func(b *testing.B) {
					benchmarkRatingSequence(b, w.price, w.quantity, hitPercent, enabled)
				})
			}
		}
	}
}

func benchmarkRatingSequence(b *testing.B, price *productcatalog.Price, quantity float64, hitPercent int, enabled bool) {
	period := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	stream := &benchmarkMeter{quantity: quantity}
	svc, err := New(Config{
		StreamingConnector:   stream,
		RatingService:        billingratingservice.New(billingratingservice.Config{}),
		DetailedLinesFetcher: passthroughDetailedLinesFetcher,
	})
	require.NoError(b, err)
	charge := newDetailedRatingTestCharge(b, period, nil)
	require.NoError(b, charge.Intent.MutateEffective(func(fields *usagebased.IntentMutableFields) error {
		fields.Price = *price.Clone()
		return nil
	}))
	input := GetTotalsForUsageInput{
		Charge: charge, Customer: newDetailedRatingTestCustomer(),
		FeatureMeter: newDetailedRatingTestFeatureMeter(), StoredAtLT: period.To,
	}
	ctx := b.Context()
	hits := 0
	if enabled {
		reuse, err := NewTotalsReuse(256)
		require.NoError(b, err)
		ctx = WithTotalsReuse(ctx, reuse, func(_ context.Context, hit bool) {
			if hit {
				hits++
			}
		})
	}

	_, err = svc.GetTotalsForUsage(ctx, input)
	require.NoError(b, err)
	var iterations, expectedHits int
	b.ReportAllocs()
	for b.Loop() {
		// Change the quantity on 0, 1, 2 or 4 reads per group of four.
		if iterations%4 < hitPercent/25 {
			expectedHits++
		} else if stream.quantity == quantity {
			stream.quantity = quantity + 1
		} else {
			stream.quantity = quantity
		}

		if _, err := svc.GetTotalsForUsage(ctx, input); err != nil {
			b.Fatal(err)
		}
		iterations++
	}

	if enabled {
		require.Equal(b, expectedHits, hits)
	}
}
