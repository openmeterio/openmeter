package rte_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	eventmodels "github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/ledger/historical"
	historicaladapter "github.com/openmeterio/openmeter/openmeter/ledger/historical/adapter"
	"github.com/openmeterio/openmeter/openmeter/ledger/routingrules"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/meter"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	progressadapter "github.com/openmeterio/openmeter/openmeter/progressmanager/adapter"
	"github.com/openmeterio/openmeter/openmeter/rte"
	rteadapter "github.com/openmeterio/openmeter/openmeter/rte/adapter"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	rtemetered "github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	ingestevents "github.com/openmeterio/openmeter/openmeter/sink/flushhandler/ingestnotification/events"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	streamingclickhouse "github.com/openmeterio/openmeter/openmeter/streaming/clickhouse"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	creditstest "github.com/openmeterio/openmeter/test/credits"
)

// The child is the production executable, with real Kafka and ClickHouse. Only
// service-backed fixture creation and stored-state reads run in the test process.
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
	creditStore, err := rteadapter.NewCreditBalanceStore(rteadapter.Config{Client: s.DBClient, Logger: logger})
	s.Require().NoError(err)
	meteredStore, err := rteadapter.NewMeteredEntitlementBalanceStore(rteadapter.Config{Client: s.DBClient, Logger: logger})
	s.Require().NoError(err)
	engine, err := common.NewRTEService(s.DBClient, s.CustomerBalanceSvc, s.Entitlements,
		logger, config.CreditsConfiguration{Enabled: true}, eventbus.NewMock(t), noop.NewMeterProvider().Meter("test"))
	s.Require().NoError(err)
	demand, err := common.NewRTEDemandService(s.DBClient, engine, logger, noop.NewMeterProvider().Meter("test"), config.RTEConfiguration{CalculationTimeout: 30 * time.Second})
	s.Require().NoError(err)

	// Given active credit and metered demand backed by real customer, grant,
	// charge and ledger services. Event times are in completed minute windows.
	ns := s.GetUniqueNamespace("rte-live")
	const subjectKey = "rte-live-customer"
	cust := s.CreateLedgerBackedCustomer(ns, subjectKey)
	feature := s.SetupApiRequestsTotalFeature(ctx, ns)
	defer feature.Cleanup()
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(t, ns)
	app := s.InstallSandboxApp(t, ns)
	s.ProvisionBillingProfile(ctx, ns, app.GetID())
	start := time.Now().UTC().Truncate(time.Minute).Add(-5 * time.Minute)
	period := timeutil.ClosedPeriod{From: start, To: start.Add(24 * time.Hour)}
	_, err = s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.NewCreateChargeIntents(s.CreateMockChargeIntent(creditstest.CreateMockChargeIntentInput{
			Customer: cust.GetID(), Currency: creditstest.USD, ServicePeriod: period,
			Price:      productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
			FeatureKey: feature.Feature.Key, Name: "live usage", ManagedBy: billing.SubscriptionManagedLine,
			SettlementMode: productcatalog.CreditOnlySettlementMode, UniqueReferenceID: "live-usage",
		})),
	})
	s.Require().NoError(err)
	measure := &entitlement.MeasureUsageFromInput{}
	s.Require().NoError(measure.FromTime(start))
	ent, err := s.Entitlements.Entitlement.CreateCustomerEntitlement(ctx, entitlement.CreateCustomerEntitlementInput{
		CustomerID: cust.GetID(),
		Entitlement: entitlement.CreateEntitlementInputs{
			FeatureID: &feature.Feature.ID, EntitlementType: entitlement.EntitlementTypeMetered,
			MeasureUsageFrom: measure, IssueAfterReset: lo.ToPtr(float64(100)),
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Anchor: start, Interval: timeutil.RecurrencePeriodMonth,
			})),
		},
	})
	s.Require().NoError(err)
	currency := currencies.NewCurrencyReference(creditstest.USD)
	creditKey := creditbalance.Key{CustomerID: cust.GetID(), Currency: currency}
	meteredKey := rtemetered.Key{EntitlementID: models.NamespacedID{Namespace: ns, ID: ent.ID}}
	creditTarget, err := rte.NewCalculationTarget(creditKey)
	s.Require().NoError(err)
	meteredTarget, err := rte.NewCalculationTarget(meteredKey)
	s.Require().NoError(err)

	postingLedger := s.Ledger
	fund := func(amount int64) {
		at := time.Now().UTC().Add(-time.Second)
		inputs, err := transactions.ResolveTransactions(ctx, transactions.ResolverDependencies{
			AccountService: s.LedgerResolver, AccountCatalog: s.LedgerAccountService, BalanceQuerier: s.BalanceQuerier,
		}, transactions.ResolutionScope{CustomerID: cust.GetID(), Namespace: ns},
			transactions.IssueCustomerReceivableTemplate{At: at, Amount: alpacadecimal.NewFromInt(amount), Currency: currency},
			transactions.AuthorizeCustomerReceivablePaymentTemplate{At: at, Amount: alpacadecimal.NewFromInt(amount), Currency: currency},
			transactions.SettleCustomerReceivableFromPaymentTemplate{At: at, Amount: alpacadecimal.NewFromInt(amount), Currency: currency},
		)
		s.Require().NoError(err)
		_, err = postingLedger.CommitGroup(ctx, transactions.GroupInputs(ns, nil, inputs...))
		s.Require().NoError(err)
	}
	fund(100)
	postingLedger = historical.NewLedger(historicaladapter.NewRepo(s.DBClient), s.LedgerAccountService,
		s.LedgerAccountService, routingrules.DefaultValidator, historical.WithEventPublisher(live.bus))
	s.Require().NoError(demand.Activate(ctx, rte.ActivateDemandInput{Target: creditTarget}))
	s.Require().NoError(demand.Activate(ctx, rte.ActivateDemandInput{Target: meteredTarget}))

	var creditState rte.State[creditbalance.Balance]
	var meteredState rte.State[rtemetered.Balance]
	s.Require().Eventually(func() bool {
		var creditErr, meteredErr error
		creditState, creditErr = creditStore.Get(ctx, creditKey)
		meteredState, meteredErr = meteredStore.Get(ctx, meteredKey)

		return creditErr == nil && meteredErr == nil && creditState.Value.Live.InexactFloat64() == 100 && meteredState.Value.Balance == 100
	}, 20*time.Second, 50*time.Millisecond, "child must calculate both activated keys; log: %s", live.logPath)

	s.Run("credit push cached pull and targeted ledger refresh", func() {
		baselineMetered := meteredState.CalculatedAt
		baselineCredit := creditState.CalculatedAt
		before := live.metrics(t)
		request, err := rte.NewCalculationRequestedEvent(creditKey)
		s.Require().NoError(err)
		request.Freshness = &rte.FreshnessPolicy{TrackForSeconds: 60, MaxAgeSeconds: 30}
		s.Require().NoError(live.bus.Publish(ctx, request))
		s.Require().Eventually(func() bool {
			return metricCount(live.metrics(t), "rte_demand_operations_total", "operation=\"pull\"", "outcome=\"cache_hit\"") >
				metricCount(before, "rte_demand_operations_total", "operation=\"pull\"", "outcome=\"cache_hit\"")
		}, 10*time.Second, 50*time.Millisecond)
		stored, err := creditStore.Get(ctx, creditKey)
		s.Require().NoError(err)
		s.Equal(baselineCredit, stored.CalculatedAt)

		// When credit is funded, the committed ledger event changes this credit
		// balance, while metered demand stays untouched. Rating inputs are equal.
		fund(25)
		s.Require().Eventually(func() bool {
			var err error
			creditState, err = creditStore.Get(ctx, creditKey)

			return err == nil && creditState.Value.Live.InexactFloat64() == 125
		}, 15*time.Second, 50*time.Millisecond)
		s.True(creditState.CalculatedAt.After(baselineCredit))
		storedMetered, err := meteredStore.Get(ctx, meteredKey)
		s.Require().NoError(err)
		s.Equal(baselineMetered, storedMetered.CalculatedAt)
		s.Greater(metricCount(live.metrics(t), "rte_rating_lookups_total", "outcome=\"hit\""), float64(0))
	})

	s.Run("metered late usage refresh and duplicate suppression", func() {
		defer func() {
			if !s.T().Failed() {
				return
			}
			lastCredit, creditErr := creditStore.Get(ctx, creditKey)
			lastMetered, meteredErr := meteredStore.Get(ctx, meteredKey)
			s.T().Logf("last credit: booked=%s live=%s err=%v; metered: balance=%g usage=%g err=%v",
				lastCredit.Value.Booked, lastCredit.Value.Live, creditErr, lastMetered.Value.Balance, lastMetered.Value.UsageInPeriod, meteredErr)
			for _, line := range strings.Split(live.metrics(t), "\n") {
				for _, prefix := range []string{"rte_upstream_events_total{", "rte_demand_operations_total{", "rte_calculations_total{", "rte_rating_lookups_total{"} {
					if strings.HasPrefix(line, prefix) {
						s.T().Log(line)
					}
				}
			}
		}()
		// When a newly stored event refers to an earlier minute, both the
		// monetary live exposure and metered grant balance must be recalculated.
		eventTime := start.Add(time.Minute)
		storedAt := time.Now().UTC()
		raw := streaming.RawEvent{
			Namespace: ns, ID: ulid.Make().String(), Type: "test", Source: "rte-live-test",
			Subject: subjectKey, CustomerID: &cust.ID, Time: eventTime, IngestedAt: storedAt, StoredAt: storedAt, Data: `{"value":25}`,
		}
		s.Require().NoError(live.streaming.BatchInsert(ctx, []streaming.RawEvent{raw}))
		meterReader, err := meteradapter.New(meteradapter.Config{Client: s.DBClient, Logger: logger})
		s.Require().NoError(err)
		definition, err := meterReader.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{Namespace: ns, IDOrSlug: *feature.Feature.MeterID})
		s.Require().NoError(err)
		rows, err := live.streaming.QueryMeter(ctx, ns, definition, streaming.QueryParams{
			FilterCustomer: []streaming.Customer{cust}, From: &period.From, To: &period.To,
		})
		s.Require().NoError(err)
		var storedUsage float64
		for _, row := range rows {
			storedUsage += row.Value
		}
		s.Require().Equal(float64(25), storedUsage, "persisted production meter must see inserted usage")
		s.T().Logf("inserted usage=%g meter=%s aggregation=%s eventType=%s topic=%s",
			storedUsage, definition.Key, definition.Aggregation, definition.EventType, live.conf.Events.IngestEvents.Topic)
		notification := ingestevents.EventBatchedIngest{
			Namespace: eventmodels.NamespaceID{ID: ns}, SubjectKey: subjectKey, MeterSlugs: []string{"api-requests-total"}, StoredAt: storedAt,
			RawEvents: []serializer.CloudEventsKafkaPayload{{Id: raw.ID, Type: raw.Type, Source: raw.Source, Subject: raw.Subject, Time: eventTime.Unix(), Data: raw.Data}},
		}
		msg, err := live.bus.Marshaler().Marshal(notification)
		s.Require().NoError(err)
		s.Require().NoError(live.publisher.Publish(live.conf.Events.IngestEvents.Topic, msg))
		s.Require().Eventually(func() bool {
			var creditErr, meteredErr error
			creditState, creditErr = creditStore.Get(ctx, creditKey)
			meteredState, meteredErr = meteredStore.Get(ctx, meteredKey)

			return creditErr == nil && meteredErr == nil && creditState.Value.Live.InexactFloat64() == 100 &&
				meteredState.Value.Balance == 75 && meteredState.Value.UsageInPeriod == 25
		}, 20*time.Second, 50*time.Millisecond, "late usage must refresh exact balances; log: %s", live.logPath)
		s.Equal(float64(125), creditState.Value.Booked.InexactFloat64())
		s.Equal(float64(100), meteredState.Value.TotalAvailableGrantAmount)

		// Then redelivering the same CloudEvent ID is consumed as a duplicate,
		// without new state or calculations. Waiting for its metric proves that
		// the worker observed it, rather than merely being slow.
		before := live.metrics(t)
		s.Require().NoError(live.publisher.Publish(live.conf.Events.IngestEvents.Topic, msg.Copy()))
		s.Require().Eventually(func() bool {
			return metricCount(live.metrics(t), "rte_demand_operations_total", "operation=\"invalidate\"", "outcome=\"duplicate\"") >
				metricCount(before, "rte_demand_operations_total", "operation=\"invalidate\"", "outcome=\"duplicate\"")
		}, 10*time.Second, 50*time.Millisecond)
		currentCredit, err := creditStore.Get(ctx, creditKey)
		s.Require().NoError(err)
		currentMetered, err := meteredStore.Get(ctx, meteredKey)
		s.Require().NoError(err)
		s.Equal(creditState, currentCredit)
		s.Equal(meteredState, currentMetered)
		after := live.metrics(t)
		s.Equal(metricCount(before, "rte_calculations_total"), metricCount(after, "rte_calculations_total"))
		s.T().Logf("child calculations after late usage: credit=%g metered=%g; rating hits=%g misses=%g",
			metricCount(after, "rte_calculations_total", "kind=\"credit_balance\""),
			metricCount(after, "rte_calculations_total", "kind=\"metered_entitlement\""),
			metricCount(after, "rte_rating_lookups_total", "outcome=\"hit\""),
			metricCount(after, "rte_rating_lookups_total", "outcome=\"miss\""))
	})
}

var liveProcessHTTPClient = &http.Client{Timeout: 2 * time.Second}

type liveProcess struct {
	bus       eventbus.Publisher
	publisher message.Publisher
	streaming streaming.Connector
	conf      config.Configuration
	baseURL   string
	logPath   string
	stop      func()
}

func (s *RTELiveProcessSuite) startLiveProcess() *liveProcess {
	t := s.T()
	ctx := t.Context()
	logger := testutils.NewDiscardLogger(t)
	v := viper.NewWithOptions(viper.WithDecodeHook(config.DecodeHook()))
	config.SetViperDefaults(v, pflag.NewFlagSet("rte-live", pflag.ContinueOnError))
	v.SetConfigFile(os.Getenv("RTE_E2E_CONFIG"))
	require.NoError(t, v.ReadInConfig())
	for _, field := range []string{"host", "user", "password", "database"} {
		v.Set("postgres."+field, "")
	}

	v.Set("postgres.port", 0)
	v.Set("postgres.url", s.TestDB.URL)
	v.Set("postgres.autoMigrate", "false")
	suffix := strings.ToLower(ulid.Make().String())
	for _, field := range []string{"systemEvents", "ingestEvents", "balanceWorkerEvents"} {
		v.Set("events."+field+".topic", "rte_live_"+suffix+"_"+strings.ToLower(field))
		v.Set("events."+field+".autoProvision.enabled", true)
		v.Set("events."+field+".autoProvision.partitions", 1)
	}

	v.Set("rte.consumerGroupName", "rte_live_"+suffix)
	v.Set("rte.dlq.topic", "rte_live_"+suffix+"_dlq")
	v.Set("rte.dlq.autoProvision.enabled", true)
	v.Set("rte.dlq.autoProvision.partitions", 1)
	v.Set("rte.pollInterval", "100ms")
	v.Set("telemetry.metrics.exporters.prometheus.enabled", true)
	v.Set("telemetry.log.exporters.file.enabled", false)
	v.Set("telemetry.log.exporters.stdout.enabled", true)
	v.Set("aggregation.asyncInsert", false)
	v.Set("aggregation.asyncInsertWait", false)
	v.Set("meters", nil)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())
	v.Set("telemetry.address", address)
	var conf config.Configuration
	require.NoError(t, v.Unmarshal(&conf))
	for _, broker := range strings.Split(conf.Ingest.Kafka.Broker, ",") {
		requireLoopbackAddress(t, broker)
	}

	requireLoopbackAddress(t, conf.Aggregation.ClickHouse.Address)
	chConfig := conf.Aggregation.ClickHouse
	chConfig.Database = "default"
	ch, err := clickhouse.Open(chConfig.GetClientOptions())
	require.NoError(t, err)
	database := "rte_live_" + suffix
	require.NoError(t, ch.Exec(ctx, "CREATE DATABASE "+database))
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = ch.Exec(cleanupCtx, "DROP DATABASE IF EXISTS "+database+" SYNC")
		_ = ch.Close()
	})
	v.Set("aggregation.clickhouse.database", database)
	require.NoError(t, v.Unmarshal(&conf))
	require.NoError(t, conf.Validate())
	connector, err := streamingclickhouse.New(ctx, streamingclickhouse.Config{
		ClickHouse: ch, Database: database, EventsTableName: conf.Aggregation.EventsTableName,
		Logger: logger, ProgressManager: progressadapter.NewNoop(),
	})
	require.NoError(t, err)
	artifactDir, err := os.MkdirTemp("", "openmeter-rte-live-")
	require.NoError(t, err)
	t.Cleanup(func() {
		if !t.Failed() {
			_ = os.RemoveAll(artifactDir)
		}
	})
	configPath := filepath.Join(artifactDir, "rte.yaml")
	require.NoError(t, v.WriteConfigAs(configPath))
	logPath := filepath.Join(artifactDir, "rte-process.log")
	log, err := os.Create(logPath)
	require.NoError(t, err)
	command := exec.CommandContext(ctx, os.Getenv("RTE_E2E_BINARY"), "--config", configPath)
	command.Stdout, command.Stderr = log, log
	for _, variable := range os.Environ() {
		name, _, _ := strings.Cut(variable, "=")
		overridden := false
		for _, prefix := range []string{"POSTGRES_", "TELEMETRY_", "EVENTS_", "RTE_", "AGGREGATION_"} {
			overridden = overridden || strings.HasPrefix(name, prefix)
		}

		if !overridden {
			command.Env = append(command.Env, variable)
		}
	}

	require.NoError(t, command.Start())
	exited := make(chan struct{})
	var exitErr error
	go func() {
		exitErr = command.Wait()
		close(exited)
	}()
	stop := func() {
		_ = command.Process.Signal(os.Interrupt)
		select {
		case <-exited:
		case <-time.After(5 * time.Second):
			_ = command.Process.Kill()
			<-exited
		}

		_ = log.Close()
	}
	t.Cleanup(stop)
	live := &liveProcess{conf: conf, streaming: connector, baseURL: "http://" + address, logPath: logPath, stop: stop}
	require.Eventually(t, func() bool {
		select {
		case <-exited:
			t.Fatalf("RTE process exited: %v; log: %s", exitErr, logPath)
		default:
		}

		response, err := liveProcessHTTPClient.Get(live.baseURL + "/healthz/subscriptions")
		if err != nil {
			return false
		}
		defer response.Body.Close()

		return response.StatusCode == http.StatusOK
	}, 40*time.Second, 100*time.Millisecond, "RTE must subscribe to real Kafka; log: %s", logPath)
	publisher, err := watermillkafka.NewPublisher(ctx, watermillkafka.PublisherOptions{
		Broker: watermillkafka.BrokerOptions{
			KafkaConfig: conf.Ingest.Kafka.KafkaConfiguration,
			ClientID:    "rte-live-test-" + suffix, Logger: logger, MetricMeter: noop.NewMeterProvider().Meter("test"),
		},
		TopicProvisioner: common.NewNoopKafkaTopicProvisioner(),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = publisher.Close() })
	bus, cleanup, err := common.NewEventBusPublisher(ctx, publisher, s.DBClient, conf.Events, logger)
	require.NoError(t, err)
	t.Cleanup(cleanup)
	live.bus, live.publisher = bus, publisher
	admin, err := common.NewKafkaAdminClient(conf.Ingest.Kafka.KafkaConfiguration)
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_, _ = admin.DeleteTopics(cleanupCtx, []string{
			conf.Events.SystemEvents.Topic, conf.Events.IngestEvents.Topic,
			conf.Events.BalanceWorkerEvents.Topic, conf.RTE.DLQ.Topic,
		})
		admin.Close()
	})
	t.Logf("RTE child ready; process log: %s", logPath)

	return live
}

func (p *liveProcess) metrics(t *testing.T) string {
	t.Helper()
	response, err := liveProcessHTTPClient.Get(p.baseURL + "/metrics")
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	return string(body)
}

func metricCount(body, name string, labels ...string) float64 {
	var total float64
	for _, line := range strings.Split(body, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || (fields[0] != name && !strings.HasPrefix(fields[0], name+"{")) {
			continue
		}

		match := true
		for _, label := range labels {
			match = match && strings.Contains(fields[0], label)
		}

		if match {
			value, _ := strconv.ParseFloat(fields[1], 64)
			total += value
		}
	}

	return total
}

func requireLoopbackAddress(t *testing.T, address string) {
	t.Helper()
	host, _, err := net.SplitHostPort(address)
	require.NoError(t, err)
	requireLoopbackHost(t, host)
}

func requireLoopbackHost(t *testing.T, host string) {
	t.Helper()
	ip := net.ParseIP(host)
	require.True(t, host == "localhost" || ip != nil && ip.IsLoopback(), "live process tests require loopback services")
}
