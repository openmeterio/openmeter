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
	"github.com/oklog/ulid/v2"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
	progressadapter "github.com/openmeterio/openmeter/openmeter/progressmanager/adapter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	streamingclickhouse "github.com/openmeterio/openmeter/openmeter/streaming/clickhouse"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

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
	t.Helper()

	ctx := t.Context()
	logger := testutils.NewDiscardLogger(t)

	// Build an isolated worker configuration around the suite database.
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

	// Use unique topics and a consumer group so this worker only sees its fixture.
	suffix := strings.ToLower(ulid.Make().String())
	for _, field := range []string{"systemEvents", "ingestEvents", "balanceWorkerEvents"} {
		v.Set("events."+field+".topic", "rte_live_"+suffix+"_"+strings.ToLower(field))
		v.Set("events."+field+".autoProvision.enabled", true)
		v.Set("events."+field+".autoProvision.partitions", 1)
	}

	v.Set("rte.enabled", true)
	v.Set("rte.shadowSamplePercent", 100)
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

	// Reserve a local telemetry address for readiness and metric assertions.
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

	// The child and test share an isolated ClickHouse database.
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
		ClickHouse:      ch,
		Database:        database,
		EventsTableName: conf.Aggregation.EventsTableName,
		Logger:          logger,
		ProgressManager: progressadapter.NewNoop(),
	})
	require.NoError(t, err)

	// Retain the generated config and process log only when the test fails.
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

	// Launch the production executable with this fixture configuration.
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

	// A ready subscription means source events can reach the child.
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

	// Publish real source facts and calculation requests to the child.
	publisher, err := watermillkafka.NewPublisher(ctx, watermillkafka.PublisherOptions{
		Broker: watermillkafka.BrokerOptions{
			KafkaConfig: conf.Ingest.Kafka.KafkaConfiguration,
			ClientID:    "rte-live-test-" + suffix,
			Logger:      logger,
			MetricMeter: noop.NewMeterProvider().Meter("test"),
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
