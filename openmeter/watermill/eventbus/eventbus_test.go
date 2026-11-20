package eventbus_test

import (
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
)

type tracedEvent struct {
	Value string `json:"value"`
}

func (tracedEvent) EventName() string {
	return "io.openmeter.test.v1.traced"
}

func (tracedEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{Source: "//openmeter.io/test"}
}

func (tracedEvent) Validate() error {
	return nil
}

func TestConsumerContinuesPublisherTrace(t *testing.T) {
	previousPropagator := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { otel.SetTextMapPropagator(previousPropagator) })

	const systemEventsTopic = "test-system-events"

	logger := testutils.NewDiscardLogger(t)
	tracerProvider := sdktrace.NewTracerProvider()

	// Given an event bus and a default router that share an in-memory broker, which does not
	// hand the publisher's context to the subscriber.
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 10}, watermill.NewSlogLogger(logger))
	t.Cleanup(func() { require.NoError(t, pubSub.Close()) })

	bus, err := eventbus.New(eventbus.Options{
		Publisher: pubSub,
		TopicMapping: eventbus.TopicMapping{
			SystemEventsTopic: systemEventsTopic, IngestEventsTopic: "test-ingest-events", BalanceWorkerEventsTopic: "test-balance-worker-events",
		},
		Logger: logger,
	})
	require.NoError(t, err)

	consumerRouter, err := router.NewDefaultRouter(router.Options{
		Subscriber:  pubSub,
		Publisher:   pubSub,
		Logger:      logger,
		MetricMeter: sdkmetric.NewMeterProvider().Meter("eventbus_test"),
		Tracer:      tracerProvider.Tracer("consumer"),
		Config: config.ConsumerConfiguration{
			ConsumerGroupName: "test-group",
			DLQ:               config.DLQConfiguration{Topic: "test-dlq"},
			Retry: config.RetryConfiguration{
				InitialInterval: 100 * time.Millisecond,
				MaxInterval:     100 * time.Millisecond,
			},
		},
	})
	require.NoError(t, err)

	type handled struct {
		spanContext trace.SpanContext
		event       tracedEvent
		err         error
	}

	handledCh := make(chan handled, 1)

	consumerRouter.AddConsumerHandler("test", systemEventsTopic, pubSub, func(msg *message.Message) error {
		var event tracedEvent

		err := bus.Marshaler().Unmarshal(msg, &event)
		handledCh <- handled{spanContext: trace.SpanContextFromContext(msg.Context()), event: event, err: err}

		return nil
	})

	go func() {
		_ = consumerRouter.Run(t.Context())
	}()

	<-consumerRouter.Running()
	t.Cleanup(func() { _ = consumerRouter.Close() })

	// When an event is published inside a span.
	ctx, publisherSpan := tracerProvider.Tracer("publisher").Start(t.Context(), "request")
	require.NoError(t, bus.Publish(ctx, tracedEvent{Value: "value"}))
	publisherSpan.End()

	// Then the handler runs in the publisher's trace and still decodes the event.
	select {
	case got := <-handledCh:
		require.NoError(t, got.err)
		require.Equal(t, tracedEvent{Value: "value"}, got.event)
		require.Equal(t, publisherSpan.SpanContext().TraceID(), got.spanContext.TraceID())
	case <-time.After(10 * time.Second):
		require.FailNow(t, "timed out waiting for the handler")
	}
}
