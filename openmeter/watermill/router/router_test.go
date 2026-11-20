package router

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/config"
)

type mockHandler struct {
	mock.Mock
}

const (
	WaitForContextHeader = "x-wait-for-context"
)

func (m *mockHandler) Handle(msg *message.Message) error {
	args := m.Called(msg)
	err := args.Error(0)

	slog.Info("handling message (initial handler)", "result", err)

	if msg.Metadata.Get(WaitForContextHeader) != "" {
		start := time.Now()
		<-msg.Context().Done()
		slog.Info("context done", "held_for", time.Since(start))
	}

	return err
}

func (m *mockHandler) HandleDLQ(msg *message.Message) error {
	slog.Info("message arrived at DLQ (final)")

	args := m.Called(msg)

	return args.Error(0)
}

type DoneCondition struct {
	done chan struct{}
}

func NewDoneSignal() *DoneCondition {
	return &DoneCondition{
		done: make(chan struct{}, 1),
	}
}

func (d *DoneCondition) Done(mock.Arguments) {
	close(d.done)
}

func (d *DoneCondition) Wait(timeout time.Duration) error {
	t := time.NewTimer(timeout)
	defer t.Stop()

	select {
	case <-d.done:
		return nil
	case <-t.C:
		return errors.New("timeout")
	}
}

func TestDefaultRouter(t *testing.T) {
	tcs := []struct {
		Name       string
		Config     Options
		SetupMocks func(*mockHandler, *DoneCondition)
	}{
		{
			Name: "DLQ enabled, no retry queue, happy path",
			Config: Options{
				Config: config.ConsumerConfiguration{
					Retry: config.RetryConfiguration{
						InitialInterval: 100 * time.Millisecond,
						MaxInterval:     100 * time.Millisecond,
					},
				},
			},
			SetupMocks: func(mh *mockHandler, done *DoneCondition) {
				mh.On("Handle", mock.Anything).Return(nil).Run(done.Done).Once()
			},
		},
		{
			Name: "DLQ enabled, no retry queue, failed message",
			// After the first failure the message is sent to the DLQ
			Config: Options{
				Config: config.ConsumerConfiguration{
					Retry: config.RetryConfiguration{
						InitialInterval: 100 * time.Millisecond,
						MaxInterval:     100 * time.Millisecond,
					},
				},
			},
			SetupMocks: func(mh *mockHandler, done *DoneCondition) {
				mh.On("Handle", mock.Anything).Return(assert.AnError).Once()
				mh.On("HandleDLQ", mock.Anything).Return(nil).Run(done.Done).Once()
			},
		},
		{
			Name: "DLQ enabled, retry queue enabled, happy path",
			// Message gets processed without any additional steps
			Config: Options{
				Config: config.ConsumerConfiguration{
					Retry: config.RetryConfiguration{
						InitialInterval: 100 * time.Millisecond,
						MaxInterval:     100 * time.Millisecond,
					},
				},
			},
			SetupMocks: func(mh *mockHandler, done *DoneCondition) {
				mh.On("Handle", mock.Anything).Return(nil).Run(done.Done).Once()
			},
		},
		{
			Name: "DLQ enabled, retry queue enabled, failed message",
			// Message gets processed without any additional steps
			Config: Options{
				Config: config.ConsumerConfiguration{
					Retry: config.RetryConfiguration{
						InitialInterval: 10 * time.Millisecond,
						MaxInterval:     10 * time.Millisecond,
						MaxRetries:      5,
					},
				},
			},
			SetupMocks: func(mh *mockHandler, done *DoneCondition) {
				// Flow: 1st failure -> retries -> DLQ
				mh.On("Handle", mock.Anything).Return(assert.AnError).Times(5)

				mh.On("HandleDLQ", mock.Anything).Return(nil).Run(done.Done).Once()
			},
		},
		{
			Name: "Timeout handling",
			// No retry queue, no DLQ, just timeout => retry every time the timeout passes
			Config: Options{
				Config: config.ConsumerConfiguration{
					ProcessingTimeout: 50 * time.Millisecond,
					Retry: config.RetryConfiguration{
						InitialInterval: 10 * time.Millisecond,
						MaxInterval:     10 * time.Millisecond,
						MaxRetries:      5,
					},
				},
			},
			SetupMocks: func(mh *mockHandler, done *DoneCondition) {
				mh.On("Handle", mock.Anything).Return(assert.AnError).Run(func(args mock.Arguments) {
					msg := args.Get(0).(*message.Message)
					// Let's instruct the handler to simulate a timeout
					msg.Metadata.Set(WaitForContextHeader, "true")
				}).Times(2)

				mh.On("Handle", mock.Anything).Return(nil).Run(done.Done).Once()
			},
		},
		{
			Name: "Timeout handling => DLQ",
			// No retry queue, no DLQ, just timeout => retry every time the timeout passes
			Config: Options{
				Config: config.ConsumerConfiguration{
					ProcessingTimeout: 100 * time.Millisecond,
					Retry: config.RetryConfiguration{
						InitialInterval: 10 * time.Millisecond,
						MaxInterval:     10 * time.Millisecond,
						MaxRetries:      3,
					},
				},
			},
			SetupMocks: func(mh *mockHandler, done *DoneCondition) {
				mh.On("Handle", mock.Anything).Return(assert.AnError).Run(func(args mock.Arguments) {
					msg := args.Get(0).(*message.Message)
					// Let's instruct the handler to simulate a timeout
					msg.Metadata.Set(WaitForContextHeader, "true")
				}).Times(4)

				mh.On("HandleDLQ", mock.Anything).Return(nil).Run(done.Done).Once()
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			inMemoryPubSub := gochannel.NewGoChannel(
				gochannel.Config{
					OutputChannelBuffer: 10,
				},
				watermill.NewSlogLogger(slog.Default()),
			)
			defer func() {
				assert.NoError(t, inMemoryPubSub.Close())
			}()

			handler := mockHandler{}

			options := tc.Config
			options.Subscriber = inMemoryPubSub
			options.Publisher = inMemoryPubSub
			options.Logger = slog.Default()
			options.MetricMeter = sdkmetric.NewMeterProvider().Meter("router_test")
			options.Tracer = sdktrace.NewTracerProvider().Tracer("router_test")

			options.Config.DLQ.Topic = "test-dlq"
			options.Config.ConsumerGroupName = "test-group"

			router, err := NewDefaultRouter(options)

			assert.NoError(t, err)
			assert.NotNil(t, router)

			const topicName = "testTopic"

			router.AddConsumerHandler(
				"test",
				topicName,
				inMemoryPubSub,
				handler.Handle,
			)

			router.AddConsumerHandler(
				"test-dlq",
				options.Config.DLQ.Topic,
				inMemoryPubSub,
				handler.HandleDLQ,
			)

			go func() {
				assert.NoError(t, router.Run(context.Background()))
			}()

			<-router.Running()
			defer func() {
				if router.IsRunning() && !router.IsClosed() {
					assert.NoError(t, router.Close())
				}
			}()

			done := NewDoneSignal()
			tc.SetupMocks(&handler, done)

			msg := message.NewMessage(watermill.NewUUID(), []byte("testPayload"))

			assert.NoError(t, inMemoryPubSub.Publish(topicName, msg))

			if !assert.NoError(t, done.Wait(120*time.Second)) {
				assert.FailNow(t, "timeout during test execution")
			}

			assert.NoError(t, router.Close())
		})
	}
}

func TestDefaultRouterTraceContext(t *testing.T) {
	previousPropagator := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { otel.SetTextMapPropagator(previousPropagator) })

	publisherSpanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01},
		SpanID:     trace.SpanID{0x02},
		TraceFlags: trace.FlagsSampled,
	})

	tcs := []struct {
		Name      string
		Publisher trace.SpanContext
	}{
		{Name: "message with a trace context continues the publisher's trace", Publisher: publisherSpanContext},
		{Name: "message without a trace context starts a new trace"},
	}

	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			inMemoryPubSub := gochannel.NewGoChannel(
				gochannel.Config{OutputChannelBuffer: 10},
				watermill.NewSlogLogger(slog.Default()),
			)
			t.Cleanup(func() { require.NoError(t, inMemoryPubSub.Close()) })

			recorder := tracetest.NewSpanRecorder()

			router, err := NewDefaultRouter(Options{
				Subscriber:  inMemoryPubSub,
				Publisher:   inMemoryPubSub,
				Logger:      slog.Default(),
				MetricMeter: sdkmetric.NewMeterProvider().Meter("router_test"),
				Tracer:      sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)).Tracer("router_test"),
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

			const topicName = "testTopic"

			handlerSpanContexts := make(chan trace.SpanContext, 1)

			router.AddConsumerHandler("test", topicName, inMemoryPubSub, func(msg *message.Message) error {
				handlerSpanContexts <- trace.SpanContextFromContext(msg.Context())

				return nil
			})

			go func() {
				_ = router.Run(t.Context())
			}()

			<-router.Running()
			t.Cleanup(func() { _ = router.Close() })

			msg := message.NewMessage(watermill.NewUUID(), []byte("testPayload"))
			otel.GetTextMapPropagator().Inject(
				trace.ContextWithSpanContext(t.Context(), tc.Publisher),
				propagation.MapCarrier(msg.Metadata),
			)

			require.NoError(t, inMemoryPubSub.Publish(topicName, msg))

			var handlerSpanContext trace.SpanContext
			select {
			case handlerSpanContext = <-handlerSpanContexts:
			case <-time.After(10 * time.Second):
				require.FailNow(t, "timed out waiting for the handler")
			}

			var routerSpan sdktrace.ReadOnlySpan
			require.Eventually(t, func() bool {
				for _, span := range recorder.Ended() {
					if span.Name() == "watermill.router.full_message_processing" {
						routerSpan = span

						return true
					}
				}

				return false
			}, 10*time.Second, 10*time.Millisecond)

			require.Equal(t, routerSpan.SpanContext().TraceID(), handlerSpanContext.TraceID())

			if tc.Publisher.IsValid() {
				require.Equal(t, tc.Publisher.TraceID(), routerSpan.SpanContext().TraceID())
				require.Equal(t, tc.Publisher.SpanID(), routerSpan.Parent().SpanID())
				require.True(t, routerSpan.Parent().IsRemote())
			} else {
				require.False(t, routerSpan.Parent().IsValid())
			}
		})
	}
}
