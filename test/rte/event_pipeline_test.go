package rte_test

import (
	"context"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/stretchr/testify/require"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
)

type eventPipeline struct {
	transport   message.Publisher
	deadLetters <-chan *message.Message
	service     rte.Service
	publisher   eventbus.Publisher
	messages    <-chan *message.Message
}

func (s *RTEIntegrationSuite) newEventPipeline() eventPipeline {
	t := s.T()
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	logger := testutils.NewDiscardLogger(t)
	transport := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 32}, watermill.NewSlogLogger(logger))
	t.Cleanup(func() { require.NoError(t, transport.Close()) })
	events := config.EventsConfiguration{
		SystemEvents:        config.EventSubsystemConfiguration{Topic: "rte-system-events"},
		IngestEvents:        config.EventSubsystemConfiguration{Topic: "ingest"},
		BalanceWorkerEvents: config.EventSubsystemConfiguration{Topic: "balance"},
		Outbox:              config.OutboxConfiguration{DrainLimit: 100, DrainTimeout: time.Second, DrainConcurrency: 1, RetryInterval: time.Hour, MaxAttempts: 3},
	}
	bus, closeBus, err := common.NewEventBusPublisher(ctx, transport, s.DBClient, events, logger)
	require.NoError(t, err)
	t.Cleanup(closeBus)
	service, err := common.NewRTEService(s.DBClient, s.CustomerBalanceSvc, s.Entitlements, logger, config.CreditsConfiguration{Enabled: true}, bus)
	require.NoError(t, err)
	observed, err := transport.Subscribe(ctx, events.SystemEvents.Topic)
	require.NoError(t, err)
	deadLetters, err := transport.Subscribe(ctx, "rte-dlq")
	require.NoError(t, err)
	worker, err := common.NewRTEWorker(service, events, router.Options{
		Subscriber:  transport,
		Publisher:   transport,
		Logger:      logger,
		MetricMeter: metricnoop.NewMeterProvider().Meter("test"),
		Tracer:      noop.NewTracerProvider().Tracer("test"),
		Config: config.ConsumerConfiguration{
			ConsumerGroupName: "rte-test",
			Retry:             config.RetryConfiguration{MaxRetries: 2, InitialInterval: time.Millisecond, MaxInterval: time.Millisecond},
			DLQ:               config.DLQConfiguration{Enabled: true, Topic: "rte-dlq"},
		},
	}, bus)
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() {
		done <- worker.Run(ctx)
		close(done)
	}()
	t.Cleanup(func() {
		require.NoError(t, worker.Close())
		require.NoError(t, <-done)
	})
	select {
	case <-worker.Running():
	case err := <-done:
		t.Fatalf("worker stopped before subscription: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("worker did not subscribe")
	}

	return eventPipeline{service: service, publisher: bus, messages: observed, transport: transport, deadLetters: deadLetters}
}

func (p eventPipeline) calculate(t *testing.T, key rte.Key) rte.CalculationUpdatedEvent {
	t.Helper()
	request, err := rte.NewCalculationRequestedEvent(key)
	require.NoError(t, err)
	require.NoError(t, p.publisher.Publish(t.Context(), request))
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	for {
		select {
		case msg, ok := <-p.messages:
			require.True(t, ok, "event subscription closed")
			msg.Ack()
			var event rte.CalculationUpdatedEvent
			if p.publisher.Marshaler().NameFromMessage(msg) != event.EventName() {
				continue
			}

			require.NoError(t, p.publisher.Marshaler().Unmarshal(msg, &event))
			require.Equal(t, request.Kind, event.Kind)
			require.JSONEq(t, string(request.Key), string(event.Key))

			return event
		case <-timer.C:
			t.Fatal("calculation update was not delivered")
		}
	}
}

func (s *RTEIntegrationSuite) TestInvalidCalculationRequestReachesDLQ() {
	// Given the production router receiving a request for an unsupported calculation.
	pipeline := s.newEventPipeline()
	ctx := s.T().Context()
	creditsBefore, err := s.DBClient.RTECreditBalance.Query().Count(ctx)
	s.Require().NoError(err)
	meteredBefore, err := s.DBClient.RTEMeteredEntitlementBalance.Query().Count(ctx)
	s.Require().NoError(err)
	request := message.NewMessage("invalid-kind", []byte(`{"specversion":"1.0","id":"invalid-kind","source":"/test/rte","type":"io.openmeter.rte.v1.calculationRequested","datacontenttype":"application/json","data":{"kind":"unknown","key":{}}}`))
	request.Metadata.Set("ce_type", rte.CalculationRequestedEvent{}.EventName())

	// When decoding fails, the shared retry and poison-queue middleware handles it.
	s.Require().NoError(pipeline.transport.Publish("rte-system-events", request))
	select {
	case dead, ok := <-pipeline.deadLetters:
		s.Require().True(ok)
		dead.Ack()
		s.Contains(dead.Metadata.Get(middleware.ReasonForPoisonedKey), "unsupported calculation kind")
	case <-time.After(10 * time.Second):
		s.FailNow("invalid request did not reach the DLQ")
	}

	// Then no calculator state or update event was written.
	credits, err := s.DBClient.RTECreditBalance.Query().Count(ctx)
	s.Require().NoError(err)
	s.Equal(creditsBefore, credits)
	metered, err := s.DBClient.RTEMeteredEntitlementBalance.Query().Count(ctx)
	s.Require().NoError(err)
	s.Equal(meteredBefore, metered)
	updates, err := s.DBClient.EventOutbox.Query().Count(ctx)
	s.Require().NoError(err)
	s.Zero(updates)
}
