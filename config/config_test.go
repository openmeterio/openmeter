package config

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/pkg/models"
)

func TestComplete(t *testing.T) {
	v, flags := viper.New(), pflag.NewFlagSet("OpenMeter", pflag.ExitOnError)

	Configure(v, flags)

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
		Address:     "127.0.0.1:8888",
		Environment: "local",
		Telemetry: TelemetryConfig{
			Address: "127.0.0.1:10000",
			Trace: TraceTelemetryConfig{
				Exporter: ExporterTraceTelemetryConfig{
					Enabled: true,
					Address: "127.0.0.1:4317",
				},
				Sampler: "always",
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
				Broker:              "127.0.0.1:9092",
				SecurityProtocol:    "SASL_SSL",
				SaslMechanisms:      "PLAIN",
				SaslUsername:        "user",
				SaslPassword:        "pass",
				Partitions:          1,
				EventsTopicTemplate: "om_%s_events",
			},
		},
		Processor: ProcessorConfiguration{
			ClickHouse: ClickHouseProcessorConfiguration{
				Address:  "127.0.0.1:9440",
				TLS:      true,
				Username: "default",
				Password: "default",
				Database: "openmeter",
			},
		},
		Sink: SinkConfiguration{
			KafkaConnect: KafkaConnectSinkConfiguration{
				Enabled: true,
				URL:     "http://127.0.0.1:8083",
				DeadLetterQueue: DeadLetterQueueKafkaConnectSinkConfiguration{
					TopicName:         "om_deadletterqueue",
					ReplicationFactor: 1,
					ContextHeaders:    true,
				},
				ClickHouse: ClickHouseKafkaConnectSinkConfiguration{
					Hostname: "127.0.0.1",
					Port:     8123,
					SSL:      true,
					Username: "default",
					Password: "default",
					Database: "openmeter",
				},
			},
		},
		Dedupe: DedupeConfiguration{
			Enabled: true,
			DedupeDriverConfiguration: DedupeDriverRedisConfiguration{
				Address:    "127.0.0.1:6379",
				Database:   0,
				Username:   "default",
				Password:   "pass",
				Expiration: 768 * time.Hour,
				TLS: struct {
					Enabled            bool
					InsecureSkipVerify bool
				}{
					Enabled: true,
				},
			},
		},
		Meters: []*models.Meter{
			{
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
	}

	assert.Equal(t, expected, actual)
}
