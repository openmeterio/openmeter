package rte_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/openmeter/ledger/historical"
	historicaladapter "github.com/openmeterio/openmeter/openmeter/ledger/historical/adapter"
	"github.com/openmeterio/openmeter/openmeter/ledger/routingrules"
	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	rtemetered "github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	"github.com/openmeterio/openmeter/openmeter/rte/shadow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	creditstest "github.com/openmeterio/openmeter/test/credits"
)

// The test drives the production worker through its client and the real source event topics.
type RTELiveProcessSuite struct {
	creditstest.BaseSuite
}

func TestRTELiveProcess(t *testing.T) {
	if os.Getenv("RTE_E2E_BINARY") == "" || os.Getenv("RTE_E2E_CONFIG") == "" {
		t.Skip("set RTE_E2E_BINARY and RTE_E2E_CONFIG to run the local process test")
	}

	requireLoopbackHost(t, os.Getenv("POSTGRES_HOST"))
	suite.Run(t, new(RTELiveProcessSuite))
}

func (s *RTELiveProcessSuite) TestDemandAndSourceChangesAcrossProcessBoundary() {
	t := s.T()
	ctx := t.Context()
	live := s.startLiveProcess()
	defer live.stop()

	logger := testutils.NewDiscardLogger(t)
	client, err := common.NewRTEClient(live.conf.RTE, live.conf.Events, s.DBClient, live.publisher, logger)
	require.NoError(t, err)

	creditClient := client.CreditBalances()
	meteredClient := client.MeteredEntitlements()

	// Given a funded customer whose credit charge and metered entitlement share usage.
	fixture, cleanupBalances := s.setupLiveBalances()
	defer cleanupBalances()

	creditKey, meteredKey := fixture.creditKey, fixture.meteredKey

	_, err = creditClient.Get(ctx, creditKey)
	require.True(t, models.IsGenericNotFoundError(err))

	s.fundLiveCredits(ctx, s.Ledger, creditKey, 100)

	// Publish subsequent ledger writes so the worker can discover the balance change.
	recorder, closeRecorder, err := common.NewRTEShadowRecorder(ctx, live.conf.RTE, live.conf.Events,
		live.publisher, noop.NewMeterProvider().Meter("test"), logger)
	require.NoError(t, err)

	defer closeRecorder()

	postingLedger := historical.NewLedger(historicaladapter.NewRepo(s.DBClient), s.LedgerAccountService,
		s.LedgerAccountService, routingrules.DefaultValidator, historical.WithSourcePublisher(common.NewRTESourcePublisher(recorder, logger)))

	s.Run("product reads create demand and emit paired timings", func() {
		t := s.T()
		ctx := t.Context()

		facade, err := common.NewCustomerBalanceFacadeWithShadow(s.CustomerBalanceSvc, live.conf.Credits, recorder)
		require.NoError(t, err)

		request := customerbalance.GetBalancesInput{CustomerID: fixture.customer.GetID()}

		// When two product reads arrive, the first creates demand and the second
		// reuses it. Both fetch stored state.
		for n := 1; n <= 2; n++ {
			balances, err := facade.GetBalances(ctx, request)
			require.NoError(t, err)
			require.Len(t, balances, 1)
			require.Equal(t, float64(100), balances[0].Balance.Live().InexactFloat64())

			require.Eventually(t, func() bool {
				body := live.metrics(t)
				for _, variant := range []string{"direct", "rte"} {
					if metricCount(body, "rte_comparison_duration_milliseconds_count", `kind="credit_balance"`, `operation="read"`, `variant="`+variant+`"`) != float64(n) {
						return false
					}
				}

				return true
			}, 15*time.Second, 100*time.Millisecond)
		}

		// Then the worker has calculated once and reused the stored balance once.
		body := live.metrics(t)
		require.Equal(t, float64(1), metricCount(body, "rte_demand_ensure_fresh_duration_seconds_count", `kind="credit_balance"`, `outcome="calculated"`))
		require.Equal(t, float64(1), metricCount(body, "rte_demand_ensure_fresh_duration_seconds_count", `kind="credit_balance"`, `outcome="cache_hit"`))
		require.Equal(t, float64(2), metricCount(body, "rte_state_read_duration_seconds_count", `kind="credit_balance"`, `outcome="hit"`))
	})

	s.Run("metered reads and worker calculations emit distinct timings", func() {
		t := s.T()
		ctx := t.Context()

		// Given a metered calculator with comparison recording enabled.
		source := shadow.ObserveMeteredValues(s.Entitlements.MeteredEntitlement, recorder)

		// When the API and balance worker each invoke the calculator.
		for _, operation := range []entitlement.CurrentValueOperation{entitlement.CurrentValueOperationRead, entitlement.CurrentValueOperationCalculate} {
			_, err := source.GetValue(operation.WithContext(ctx), fixture.entitlement, time.Now())
			require.NoError(t, err)
		}

		// Then each operation has its own direct and RTE duration sample.
		require.Eventually(t, func() bool {
			body := live.metrics(t)
			for _, op := range []string{"read", "calculate"} {
				for _, variant := range []string{"direct", "rte"} {
					if metricCount(body, "rte_comparison_duration_milliseconds_count", `kind="metered_entitlement"`, `operation="`+op+`"`, `variant="`+variant+`"`) != 1 {
						return false
					}
				}
			}

			return true
		}, 15*time.Second, 100*time.Millisecond)
	})

	// Activate both keys so subsequent input changes are calculated in the worker.
	require.NoError(t, creditClient.Activate(ctx, creditKey))
	require.NoError(t, meteredClient.Activate(ctx, meteredKey))

	var creditState rte.State[creditbalance.Balance]
	var meteredState rte.State[rtemetered.Balance]
	require.Eventually(t, func() bool {
		var creditErr, meteredErr error
		creditState, creditErr = creditClient.Get(ctx, creditKey)
		meteredState, meteredErr = meteredClient.Get(ctx, meteredKey)

		return creditErr == nil && meteredErr == nil &&
			creditState.Value.Live.InexactFloat64() == 100 && meteredState.Value.Balance == 100
	}, 20*time.Second, 50*time.Millisecond, "child must calculate both activated keys; log: %s", live.logPath)

	s.Run("credit push cached pull and targeted ledger refresh", func() {
		t := s.T()
		ctx := t.Context()

		// Given clean, active balances for both calculations.
		baselineMetered := meteredState.CalculatedAt
		baselineCredit := creditState.CalculatedAt

		// When a queued freshness check finds a clean key, it reuses stored state.
		before := live.metrics(t)
		require.NoError(t, creditClient.RequestRefresh(ctx, creditKey, rte.FreshnessPolicy{TrackForSeconds: 60, MaxAgeSeconds: 30}))
		require.Eventually(t, func() bool {
			return metricCount(live.metrics(t), "rte_demand_operations_total", `operation="pull"`, `outcome="cache_hit"`) >
				metricCount(before, "rte_demand_operations_total", `operation="pull"`, `outcome="cache_hit"`)
		}, 10*time.Second, 50*time.Millisecond)

		stored, err := creditClient.Get(ctx, creditKey)
		require.NoError(t, err)
		require.Equal(t, baselineCredit, stored.CalculatedAt)

		// When credit is funded, the committed ledger event changes this credit
		// balance, while metered demand stays untouched. Rating inputs are equal.
		s.fundLiveCredits(ctx, postingLedger, creditKey, 25)
		require.Eventually(t, func() bool {
			var err error
			creditState, err = creditClient.Get(ctx, creditKey)

			return err == nil && creditState.Value.Live.InexactFloat64() == 125
		}, 15*time.Second, 50*time.Millisecond)

		// Then only credit changed, and rating reused its unchanged usage inputs.
		require.True(t, creditState.CalculatedAt.After(baselineCredit))

		storedMetered, err := meteredClient.Get(ctx, meteredKey)
		require.NoError(t, err)
		require.Equal(t, baselineMetered, storedMetered.CalculatedAt)
		require.Greater(t, metricCount(live.metrics(t), "rte_rating_lookups_total", "outcome=\"hit\""), float64(0))
	})

	s.Run("metered late usage refresh and duplicate suppression", func() {
		t := s.T()
		ctx := t.Context()

		defer live.logBalanceFailure(t, client, fixture)

		// Given both keys are active and credit includes the extra 25-unit funding.
		// When a newly stored event refers to an earlier minute, both the
		// monetary live exposure and metered grant balance must be recalculated.
		msg := s.storeLateUsage(live, fixture)

		require.NoError(t, live.publisher.Publish(live.conf.Events.IngestEvents.Topic, msg))
		require.Eventually(t, func() bool {
			var creditErr, meteredErr error
			creditState, creditErr = creditClient.Get(ctx, creditKey)
			meteredState, meteredErr = meteredClient.Get(ctx, meteredKey)

			return creditErr == nil && meteredErr == nil && creditState.Value.Live.InexactFloat64() == 100 &&
				meteredState.Value.Balance == 75 && meteredState.Value.UsageInPeriod == 25
		}, 20*time.Second, 50*time.Millisecond, "late usage must refresh exact balances; log: %s", live.logPath)
		require.Equal(t, float64(125), creditState.Value.Booked.InexactFloat64())
		require.Equal(t, float64(100), meteredState.Value.TotalAvailableGrantAmount)

		// Then redelivering the same CloudEvent ID is consumed as a duplicate,
		// without new state or calculations. Waiting for its metric proves that
		// the worker observed it, rather than merely being slow.
		before := live.metrics(t)
		require.NoError(t, live.publisher.Publish(live.conf.Events.IngestEvents.Topic, msg.Copy()))
		require.Eventually(t, func() bool {
			return metricCount(live.metrics(t), "rte_demand_operations_total", "operation=\"invalidate\"", "outcome=\"duplicate\"") >
				metricCount(before, "rte_demand_operations_total", "operation=\"invalidate\"", "outcome=\"duplicate\"")
		}, 10*time.Second, 50*time.Millisecond)

		currentCredit, err := creditClient.Get(ctx, creditKey)
		require.NoError(t, err)

		currentMetered, err := meteredClient.Get(ctx, meteredKey)
		require.NoError(t, err)
		require.Equal(t, creditState, currentCredit)
		require.Equal(t, meteredState, currentMetered)

		after := live.metrics(t)
		require.Equal(t, metricCount(before, "rte_calculations_total"), metricCount(after, "rte_calculations_total"))
		t.Logf("child calculations after late usage: credit=%g metered=%g; rating hits=%g misses=%g",
			metricCount(after, "rte_calculations_total", "kind=\"credit_balance\""),
			metricCount(after, "rte_calculations_total", "kind=\"metered_entitlement\""),
			metricCount(after, "rte_rating_lookups_total", "outcome=\"hit\""),
			metricCount(after, "rte_rating_lookups_total", "outcome=\"miss\""))
	})
}
