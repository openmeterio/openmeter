package consumer

import (
	"log/slog"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
)

var _ = gochannel.NewGoChannel

type stubNotificationService struct {
	notification.Service
}

func TestNewRejectsMissingDependencies(t *testing.T) {
	taxCodeResolver := struct{ productcatalog.TaxCodeResolver }{}

	tests := []struct {
		name  string
		given func(o *Options)
		error string
	}{
		{
			name: "missing system events topic",
			given: func(o *Options) {
				o.SystemEventsTopic = ""
			},
			error: "system events topic is required",
		},
		{
			name: "missing notification service",
			given: func(o *Options) {
				o.Notification = nil
			},
			error: "notification service is required",
		},
		{
			name: "missing tax code resolver",
			given: func(o *Options) {
				o.TaxCodeResolver = nil
			},
			error: "tax code resolver is required",
		},
		{
			name: "missing logger",
			given: func(o *Options) {
				o.Logger = nil
			},
			error: "logger is required",
		},
		{
			name: "valid options",
			given: func(o *Options) {
				inMemoryPubSub := gochannel.NewGoChannel(gochannel.Config{}, watermill.NewSlogLogger(slog.New(slog.DiscardHandler)))

				o.SystemEventsTopic = "om.system.events"
				o.Router = router.Options{
					Subscriber:  inMemoryPubSub,
					Publisher:   inMemoryPubSub,
					Logger:      slog.New(slog.DiscardHandler),
					MetricMeter: sdkmetric.NewMeterProvider().Meter("consumer_test"),
					Tracer:      sdktrace.NewTracerProvider().Tracer("consumer_test"),
					Config: config.ConsumerConfiguration{
						ConsumerGroupName: "test-group",
						Retry: config.RetryConfiguration{
							MaxRetries:      10,
							InitialInterval: 10 * time.Millisecond,
							MaxInterval:     time.Second,
							MaxElapsedTime:  time.Minute,
						},
						DLQ: config.DLQConfiguration{
							Enabled: true,
							Topic:   "test-dlq",
						},
					},
				}
				o.Marshaler = marshaler.New(watermillkafka.AddPartitionKeyFromSubject)
				o.Notification = &stubNotificationService{}
				o.TaxCodeResolver = taxCodeResolver
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opts := Options{
				Logger: slog.New(slog.DiscardHandler),
			}
			test.given(&opts)

			c, err := New(opts)
			if test.error != "" {
				require.ErrorContains(t, err, test.error)
				require.Nil(t, c)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, c)
		})
	}
}
