package rte_test

import (
	"fmt"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/pkg/clock"
	creditstest "github.com/openmeterio/openmeter/test/credits"
)

func (s *RTEIntegrationSuite) TestCreditCheckpointPerformance() {
	t := s.T()
	if os.Getenv("RTE_BENCHMARK_CREDITS") != "1" {
		t.Skip("set RTE_BENCHMARK_CREDITS=1 to measure credit refresh timings")
	}

	pipeline := s.newEventPipeline()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { require.NoError(t, provider.Shutdown(t.Context())) })
	meter := provider.Meter("credit-checkpoint-performance")
	replicas := [2]creditTestReplica{
		s.newCreditTestReplica(pipeline.publisher, meter, false),
		s.newCreditTestReplica(pipeline.publisher, meter, true),
	}

	for _, grants := range []int{1, 32} {
		for _, workload := range []struct {
			name           string
			addFunding     bool
			expectedMethod string
		}{
			{name: "unchanged_journal", expectedMethod: "checkpoint_hit"},
			{name: "new_funding", addFunding: true, expectedMethod: "checkpoint_advanced"},
		} {
			s.Run(fmt.Sprintf("grants_%d/%s", grants, workload.name), func() {
				t := s.T()
				at := time.Date(2026, time.January, 5, 12, 0, 0, 0, time.UTC)
				clock.FreezeTime(at)
				defer clock.UnFreeze()
				accounts := s.fundBenchmarkAccounts(grants, at)
				before := creditCalculationMethods(t, reader)

				// Alternate execution order within each pair and swap accounts between blocks.
				// Discard the first four pairs of each block, including its first cold calculation.
				const warmup, measured = 4, 20
				var samples [2][]time.Duration
				for block := range 2 {
					for pair := range warmup + measured {
						for turn := range 2 {
							variant := (pair + turn) % 2
							account := &accounts[(block+variant)%2]
							at = at.Add(time.Second)
							elapsed := s.timeCreditRefresh(pipeline, replicas[variant], account, at, workload.addFunding)
							if pair >= warmup {
								samples[variant] = append(samples[variant], elapsed)
							}
						}
					}
				}

				after := creditCalculationMethods(t, reader)
				const perVariant = 2 * (warmup + measured)
				require.EqualValues(t, perVariant, after["full"]-before["full"])
				require.EqualValues(t, 2, after["checkpoint_miss"]-before["checkpoint_miss"])
				require.EqualValues(t, perVariant-2, after[workload.expectedMethod]-before[workload.expectedMethod])
				logCreditTimings(t, "full", samples[0])
				logCreditTimings(t, workload.expectedMethod, samples[1])
			})
		}
	}
}

type creditBenchmarkAccount struct {
	key      creditbalance.Key
	target   rte.CalculationTarget
	expected int64
}

func (s *RTEIntegrationSuite) fundBenchmarkAccounts(grants int, at time.Time) [2]creditBenchmarkAccount {
	t := s.T()
	t.Helper()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("rte-credit-performance")
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(t, ns)
	app := s.InstallSandboxApp(t, ns)
	s.ProvisionBillingProfile(ctx, ns, app.GetID())

	var accounts [2]creditBenchmarkAccount
	for i := range accounts {
		cust := s.CreateLedgerBackedCustomer(ns, fmt.Sprintf("subject-%d", i))
		key := creditbalance.Key{CustomerID: cust.GetID(), Currency: currencies.NewCurrencyReference(creditstest.USD)}
		target, err := rte.NewCalculationTarget(key)
		require.NoError(t, err)

		for j := range grants {
			priority := j + 1
			s.CreatePromotionalCreditFunding(ctx, creditstest.CreatePromotionalCreditFundingInput{
				Namespace: ns, Customer: cust.GetID(), At: at, Priority: &priority,
				Amount: alpacadecimal.NewFromInt(100), CostBasis: alpacadecimal.Zero,
			})
		}

		accounts[i] = creditBenchmarkAccount{key: key, target: target, expected: int64(100 * grants)}
	}

	return accounts
}

func (s *RTEIntegrationSuite) timeCreditRefresh(pipeline eventPipeline, replica creditTestReplica, account *creditBenchmarkAccount, at time.Time, addFunding bool) time.Duration {
	t := s.T()
	t.Helper()
	ctx := t.Context()
	clock.FreezeTime(at)
	defer clock.UnFreeze()

	if addFunding {
		s.postFundedCredits(ctx, account.key, at, 1)
		account.expected++
	}

	// Advancing time beyond MaxAge forces a calculation for both variants.
	pull := rte.EnsureFreshInput{Target: account.target, PullTTL: time.Hour, MaxAge: time.Millisecond}
	start := time.Now()
	require.NoError(t, replica.demand.EnsureFresh(ctx, pull))
	state, err := replica.engine.CreditBalances().Get(ctx, account.key)
	elapsed := time.Since(start)
	require.NoError(t, err)

	want := float64(account.expected)
	require.Equal(t, want, state.Value.Live.InexactFloat64())
	require.Equal(t, want, state.Value.Spendable.InexactFloat64())
	require.Equal(t, want, state.Value.Booked.InexactFloat64())
	require.Zero(t, state.Value.Pending.InexactFloat64())
	pipeline.awaitUpdate(t, account.target)

	return elapsed
}

func logCreditTimings(t *testing.T, method string, samples []time.Duration) {
	t.Helper()
	slices.Sort(samples)
	n := len(samples)
	median := (samples[n/2-1] + samples[n/2]) / 2
	const precision = 10 * time.Microsecond
	t.Logf("%s (%d samples)  p25=%v  p50=%v  p75=%v  p95=%v", method, n,
		samples[n/4].Round(precision), median.Round(precision),
		samples[3*n/4].Round(precision), samples[(95*n-1)/100].Round(precision))
}
