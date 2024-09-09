package config

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	notificationwebhook "github.com/openmeterio/openmeter/openmeter/notification/webhook"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/redis"
)

func TestComplete(t *testing.T) {
	v, flags := viper.New(), pflag.NewFlagSet("OpenMeter", pflag.ExitOnError)

	SetViperDefaults(v, flags)

	v.SetConfigFile("testdata/complete.yaml")

	err := v.ReadInConfig()
	if err != nil && !errors.As(err, &viper.ConfigFileNotFoundError{}) {
		t.Fatal(err)
	}

	var actual Configuration
	err = v.Unmarshal(&actual, viper.DecodeHook(DecodeHook()))
	if err != nil {
		t.Fatal(err)
	}

	err = actual.Validate()
	if err != nil {
		t.Fatal(err)
	}

	expected := Configuration{
		Postgres: PostgresConfig{
			AutoMigrate: AutoMigrateEnt,
		},
		Address:     "127.0.0.1:8888",
		Environment: "local",
		Telemetry: TelemetryConfig{
			Address: "127.0.0.1:10000",
			Trace: TraceTelemetryConfig{
				Sampler: "always",
				Exporters: ExportersTraceTelemetryConfig{
					OTLP: OTLPExportersTraceTelemetryConfig{
						Enabled: true,
						OTLPExporterTelemetryConfig: OTLPExporterTelemetryConfig{
							Address: "127.0.0.1:4317",
						},
					},
				},
			},
			Metrics: MetricsTelemetryConfig{
				Exporters: ExportersMetricsTelemetryConfig{
					Prometheus: PrometheusExportersMetricsTelemetryConfig{
						Enabled: true,
					},
					OTLP: OTLPExportersMetricsTelemetryConfig{
						Enabled: true,
						OTLPExporterTelemetryConfig: OTLPExporterTelemetryConfig{
							Address: "127.0.0.1:4317",
						},
					},
				},
			},
			Log: LogTelemetryConfiguration{
				Format: "json",
				Level:  slog.LevelInfo,
			},
		},
		Namespace: NamespaceConfiguration{
			Default:           "default",
			DisableManagement: false,
		},
		Ingest: IngestConfiguration{
			Kafka: KafkaIngestConfiguration{
				KafkaConfiguration: KafkaConfiguration{
					Broker:           "127.0.0.1:9092",
					SecurityProtocol: "SASL_SSL",
					SaslMechanisms:   "PLAIN",
					SaslUsername:     "user",
					SaslPassword:     "pass",

					BrokerAddressFamily:          pkgkafka.BrokerAddressFamilyAny,
					TopicMetadataRefreshInterval: pkgkafka.TimeDurationMilliSeconds(time.Minute),
					StatsInterval:                pkgkafka.TimeDurationMilliSeconds(5 * time.Second),
					SocketKeepAliveEnabled:       true,
					DebugContexts: pkgkafka.DebugContexts{
						"broker",
						"topic",
						"consumer",
					},
				},
				Partitions:          1,
				EventsTopicTemplate: "om_%s_events",
			},
		},
		Aggregation: AggregationConfiguration{
			ClickHouse: ClickHouseAggregationConfiguration{
				Address:         "127.0.0.1:9440",
				TLS:             true,
				Username:        "default",
				Password:        "default",
				Database:        "openmeter",
				DialTimeout:     10 * time.Second,
				MaxOpenConns:    5,
				MaxIdleConns:    5,
				ConnMaxLifetime: 10 * time.Minute,
				BlockBufferSize: 10,
			},
		},
		Sink: SinkConfiguration{
			GroupId:          "openmeter-sink-worker",
			MinCommitCount:   500,
			MaxCommitWait:    30 * time.Second,
			NamespaceRefetch: 15 * time.Second,
			Dedupe: DedupeConfiguration{
				Enabled: true,
				DedupeDriverConfiguration: DedupeDriverRedisConfiguration{
					Config: redis.Config{
						Address:  "127.0.0.1:6379",
						Database: 0,
						Username: "default",
						Password: "pass",

						TLS: struct {
							Enabled            bool
							InsecureSkipVerify bool
						}{
							Enabled: true,
						},
					},
					Expiration: 768 * time.Hour,
				},
			},
			IngestNotifications: IngestNotificationsConfiguration{
				MaxEventsInBatch: 500,
			},
			Kafka: KafkaConfig{
				CommonConfigParams: pkgkafka.CommonConfigParams{
					Brokers:                      "127.0.0.1:9092",
					SecurityProtocol:             "SASL_SSL",
					SaslMechanisms:               "PLAIN",
					SaslUsername:                 "user",
					SaslPassword:                 "pass",
					ClientID:                     "kafka-client-1",
					StatsInterval:                pkgkafka.TimeDurationMilliSeconds(5 * time.Second),
					BrokerAddressFamily:          pkgkafka.BrokerAddressFamilyAny,
					TopicMetadataRefreshInterval: pkgkafka.TimeDurationMilliSeconds(time.Minute),
					SocketKeepAliveEnabled:       true,
					DebugContexts: pkgkafka.DebugContexts{
						"broker",
						"topic",
						"consumer",
					},
				},
				ConsumerConfigParams: pkgkafka.ConsumerConfigParams{
					ConsumerGroupID:         "consumer-group",
					ConsumerGroupInstanceID: "consumer-group-1",
					SessionTimeout:          pkgkafka.TimeDurationMilliSeconds(5 * time.Minute),
					HeartbeatInterval:       pkgkafka.TimeDurationMilliSeconds(3 * time.Second),
					EnableAutoCommit:        true,
					EnableAutoOffsetStore:   false,
					AutoOffsetReset:         "error",
				},
			},
		},
		Dedupe: DedupeConfiguration{
			Enabled: true,
			DedupeDriverConfiguration: DedupeDriverRedisConfiguration{
				Config: redis.Config{
					Address:  "127.0.0.1:6379",
					Database: 0,
					Username: "default",
					Password: "pass",

					TLS: struct {
						Enabled            bool
						InsecureSkipVerify bool
					}{
						Enabled: true,
					},
				},
				Expiration: 768 * time.Hour,
			},
		},
		Portal: PortalConfiguration{
			Enabled: false,
			CORS: PortalCORSConfiguration{
				Enabled: true,
			},
			TokenExpiration: time.Hour,
		},
		Meters: []*models.Meter{
			{
				Namespace:     "default",
				Slug:          "m1",
				Description:   "",
				Aggregation:   "SUM",
				EventType:     "api-calls",
				ValueProperty: "$.duration_ms",
				GroupBy: map[string]string{
					"method": "$.method",
					"path":   "$.path",
				},
				WindowSize: models.WindowSizeMinute,
			},
		},
		Events: EventsConfiguration{
			SystemEvents: EventSubsystemConfiguration{
				Enabled: true,
				Topic:   "om_sys.api_events",
				AutoProvision: AutoProvisionConfiguration{
					Enabled:    true,
					Partitions: 4,
				},
			},
			IngestEvents: EventSubsystemConfiguration{
				Enabled: true,
				Topic:   "om_sys.ingest_events",
				AutoProvision: AutoProvisionConfiguration{
					Enabled:    true,
					Partitions: 8,
				},
			},
		},
		BalanceWorker: BalanceWorkerConfiguration{
			ConsumerConfiguration: ConsumerConfiguration{
				ProcessingTimeout: 30 * time.Second,
				Retry: RetryConfiguration{
					InitialInterval: 10 * time.Millisecond,
					MaxInterval:     time.Second,
					MaxElapsedTime:  time.Minute,
				},
				DLQ: DLQConfiguration{
					Enabled: true,
					Topic:   "om_sys.balance_worker_dlq",
					AutoProvision: DLQAutoProvisionConfiguration{
						Enabled:    true,
						Partitions: 1,
						Retention:  90 * 24 * time.Hour,
					},
				},
				ConsumerGroupName: "om_balance_worker",
			},
		},
		Notification: NotificationConfiguration{
			Enabled: true,
			Consumer: ConsumerConfiguration{
				ProcessingTimeout: 30 * time.Second,
				Retry: RetryConfiguration{
					InitialInterval: 10 * time.Millisecond,
					MaxInterval:     time.Second,
					MaxElapsedTime:  time.Minute,
				},
				DLQ: DLQConfiguration{
					Enabled: true,
					Topic:   "om_sys.notification_service_dlq",
					AutoProvision: DLQAutoProvisionConfiguration{
						Enabled:    true,
						Partitions: 1,
						Retention:  90 * 24 * time.Hour,
					},
				},
				ConsumerGroupName: "om_notification_service",
			},
		},
		Svix: notificationwebhook.SvixConfig{
			APIKey:    "test-svix-token",
			ServerURL: "http://127.0.0.1:8071",
			Debug:     true,
		},
	}

	assert.Equal(t, expected, actual)
}
