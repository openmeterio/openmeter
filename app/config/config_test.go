package config

import (
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/dedupe/redisdedupe"
	"github.com/openmeterio/openmeter/openmeter/meter"
	notificationwebhook "github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook/svix"
	"github.com/openmeterio/openmeter/pkg/datetime"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/redis"
)

func TestComplete(t *testing.T) {
	v, flags := viper.New(), pflag.NewFlagSet("OpenMeter", pflag.ExitOnError)

	// Messes with vscode defaults
	val, set := os.LookupEnv("POSTGRES_HOST")
	os.Unsetenv("POSTGRES_HOST")
	defer func() {
		if set {
			os.Setenv("POSTGRES_HOST", val)
		}
	}()

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
		Address: "127.0.0.1:8888",
		Apps: AppsConfiguration{
			BaseURL: "https://example.com",
		},
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
			Log: LogTelemetryConfig{
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
				TopicProvisioner: TopicProvisionerConfig{
					Enabled:   true,
					CacheSize: 200,
					CacheTTL:  15 * time.Minute,
					ProtectedTopics: []string{
						"protected-topic-1",
						"protected-topic-2",
					},
				},
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
				Retry: ClickhouseQueryRetryConfig{
					Enabled:           true,
					MaxTries:          3,
					RetryWaitDuration: 20 * time.Millisecond,
				},
			},
			EventsTableName: "om_events",
			AsyncInsert:     false,
			AsyncInsertWait: false,
		},
		Entitlements: EntitlementsConfiguration{
			GracePeriod: datetime.ISODurationString("P1D"),
		},
		Billing: BillingConfiguration{
			AdvancementStrategy:          billing.ForegroundAdvancementStrategy,
			MaxParallelQuantitySnapshots: 4,
			Worker: BillingWorkerConfiguration{
				ConsumerConfiguration: ConsumerConfiguration{
					ProcessingTimeout: 30 * time.Second,
					Retry: RetryConfiguration{
						MaxRetries:      10,
						InitialInterval: 10 * time.Millisecond,
						MaxInterval:     time.Second,
						MaxElapsedTime:  time.Minute,
					},
					DLQ: DLQConfiguration{
						Enabled: true,
						Topic:   "om_sys.billing_worker_dlq",
						AutoProvision: DLQAutoProvisionConfiguration{
							Enabled:    true,
							Partitions: 1,
							Retention:  90 * 24 * time.Hour,
						},
					},
					ConsumerGroupName: "om_billing_worker",
				},
			},
		},
		Sink: SinkConfiguration{
			GroupId:                 "openmeter-sink-worker",
			MinCommitCount:          500,
			MaxCommitWait:           30 * time.Second,
			MaxPollTimeout:          100 * time.Millisecond,
			NamespaceRefetch:        15 * time.Second,
			FlushSuccessTimeout:     5 * time.Second,
			DrainTimeout:            10 * time.Second,
			NamespaceRefetchTimeout: 9 * time.Second,
			NamespaceTopicRegexp:    "^om_test_([A-Za-z0-9]+(?:_[A-Za-z0-9]+)*)_events$",
			MeterRefetchInterval:    15 * time.Second,
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
					Mode:       redisdedupe.DedupeModeRawKey,
				},
			},
			IngestNotifications: IngestNotificationsConfiguration{
				MaxEventsInBatch: 50,
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
					PartitionAssignmentStrategy: pkgkafka.PartitionAssignmentStrategies{
						pkgkafka.PartitionAssignmentStrategyRange,
						pkgkafka.PartitionAssignmentStrategyRoundRobin,
					},
				},
			},
			Storage: StorageConfiguration{
				AsyncInsert:     false,
				AsyncInsertWait: false,
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
				Mode:       redisdedupe.DedupeModeRawKey,
			},
		},
		Portal: PortalConfiguration{
			Enabled: false,
			CORS: PortalCORSConfiguration{
				Enabled: true,
			},
			TokenExpiration: time.Hour,
		},
		Meters: []*meter.Meter{
			{
				ManagedResource: models.ManagedResource{
					ID: "m1",
					NamespacedModel: models.NamespacedModel{
						Namespace: "default",
					},
					ManagedModel: models.ManagedModel{
						CreatedAt: actual.Meters[0].ManagedResource.CreatedAt,
						UpdatedAt: actual.Meters[0].ManagedResource.UpdatedAt,
					},
					Name: "m1",
				},
				Key:           "m1",
				Aggregation:   "SUM",
				EventType:     "api-calls",
				ValueProperty: lo.ToPtr("$.duration_ms"),
				GroupBy: map[string]string{
					"method": "$.method",
					"path":   "$.path",
				},
			},
		},
		Events: EventsConfiguration{
			SystemEvents: EventSubsystemConfiguration{
				Topic: "om_sys.api_events",
				AutoProvision: AutoProvisionConfiguration{
					Enabled:    true,
					Partitions: 4,
				},
			},
			IngestEvents: EventSubsystemConfiguration{
				Topic: "om_sys.ingest_events",
				AutoProvision: AutoProvisionConfiguration{
					Enabled:    true,
					Partitions: 8,
				},
			},
			BalanceWorkerEvents: EventSubsystemConfiguration{
				Topic: "om_sys.balance_worker_events",
				AutoProvision: AutoProvisionConfiguration{
					Enabled:    true,
					Partitions: 4,
				},
			},
		},
		BalanceWorker: BalanceWorkerConfiguration{
			ConsumerConfiguration: ConsumerConfiguration{
				ProcessingTimeout: 30 * time.Second,
				Retry: RetryConfiguration{
					MaxRetries:      10,
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
			StateStorage: BalanceWorkerStateStorageConfiguration{
				HighWatermarkCache: BalanceWorkerHighWatermarkCacheConfiguration{
					LRUCacheSize: 100_000,
				},
			},
		},
		ProductCatalog: ProductCatalogConfiguration{
			Subscription: SubscriptionConfiguration{
				MultiSubscriptionNamespaces: []string{
					"multi-subscription",
				},
			},
		},
		Notification: NotificationConfiguration{
			Consumer: ConsumerConfiguration{
				ProcessingTimeout: 30 * time.Second,
				Retry: RetryConfiguration{
					MaxRetries:      10,
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
			Webhook: WebhookConfiguration{
				EventTypeRegistrationTimeout:     notificationwebhook.DefaultRegistrationTimeout,
				SkipEventTypeRegistrationOnError: false,
			},
			ReconcileInterval: time.Minute,
			SendingTimeout:    time.Hour,
			PendingTimeout:    2 * time.Hour,
		},
		Svix: svix.SvixConfig{
			APIKey:    "test-svix-token",
			ServerURL: "http://127.0.0.1:8071",
			Debug:     true,
		},
		Termination: TerminationConfig{
			CheckInterval:           7 * time.Second,
			GracefulShutdownTimeout: 43 * time.Second,
			PropagationTimeout:      18 * time.Second,
		},
		ProgressManager: ProgressManagerConfiguration{
			Enabled:    false,
			Expiration: 5 * time.Minute,
			Redis: redis.Config{
				Address:  "127.0.0.1:6379",
				Database: 0,
				Username: "",
				Password: "",
				TLS: struct {
					Enabled            bool
					InsecureSkipVerify bool
				}{
					Enabled: false,
				},
			},
		},
		Customer: CustomerConfiguration{
			EnableSubjectHook: true,
			IgnoreErrors:      true,
		},
		ReservedEventTypes: []string{
			`^reserved\..*$`,
			`^_\..*$`,
			`^openmeter\..*$`,
		},
	}

	assert.Equal(t, expected, actual)
}
