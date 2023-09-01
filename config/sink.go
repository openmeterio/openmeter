package config

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

type SinkConfiguration struct {
	KafkaConnect KafkaConnectSinkConfiguration
}

func (c SinkConfiguration) Validate() error {
	if err := c.KafkaConnect.Validate(); err != nil {
		return fmt.Errorf("kafka connect: %w", err)
	}

	return nil
}

type KafkaConnectSinkConfiguration struct {
	Enabled         bool
	URL             string
	DeadLetterQueue DeadLetterQueueKafkaConnectSinkConfiguration
	ClickHouse      ClickHouseKafkaConnectSinkConfiguration
}

func (c KafkaConnectSinkConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.URL == "" {
		return errors.New("url is required")
	}

	if err := c.DeadLetterQueue.Validate(); err != nil {
		return fmt.Errorf("deadletterqueue: %w", err)
	}

	if err := c.ClickHouse.Validate(); err != nil {
		return fmt.Errorf("clickhouse: %w", err)
	}

	return nil
}

// Clickhouse configuration
// See: https://docs.confluent.io/platform/current/installation/configuration/connect/sink-connect-configs.html
type DeadLetterQueueKafkaConnectSinkConfiguration struct {
	TopicName         string
	ReplicationFactor int
	ContextHeaders    bool
}

func (c DeadLetterQueueKafkaConnectSinkConfiguration) Validate() error {
	if c.TopicName == "" {
		return errors.New("dead letter queue topic is required")
	}

	return nil
}

// Clickhouse configuration
// This may feel repetative but clikhouse sink and processor configs can be different,
// for example Kafka Connect ClickHouse plugin uses 8123 HTTP port while client uses native protocol's 9000 port.
// Hostname can be also different, as Kafka Connect and ClickHouse communicates inside the docker compose network.
// This why we default hostname in config to `clickhouse`.
type ClickHouseKafkaConnectSinkConfiguration struct {
	Hostname string
	Port     int
	SSL      bool
	Username string
	Password string
	Database string
}

func (c ClickHouseKafkaConnectSinkConfiguration) Validate() error {
	if c.Hostname == "" {
		return errors.New("hostname is required")
	}

	if c.Port == 0 {
		return errors.New("port is required")
	}

	if c.Username == "" {
		return errors.New("username is required")
	}

	if c.Database == "" {
		return errors.New("database is required")
	}

	return nil
}

// Configure configures some defaults in the Viper instance.
func configureSink(v *viper.Viper) {
	v.SetDefault("sink.kafkaConnect.enabled", false)
	v.SetDefault("sink.kafkaConnect.url", "http://127.0.0.1:8083")
	v.SetDefault("sink.kafkaConnect.deadLetterQueue.topicName", "om_deadletterqueue")
	v.SetDefault("sink.kafkaConnect.deadLetterQueue.replicationFactor", 3)
	v.SetDefault("sink.kafkaConnect.deadLetterQueue.contextHeaders", true)
	v.SetDefault("sink.kafkaConnect.clickhouse.hostname", "127.0.0.1")
	v.SetDefault("sink.kafkaConnect.clickhouse.port", 8123)
	v.SetDefault("sink.kafkaConnect.clickhouse.ssl", false)
	v.SetDefault("sink.kafkaConnect.clickhouse.database", "openmeter")
	v.SetDefault("sink.kafkaConnect.clickhouse.username", "default")
	v.SetDefault("sink.kafkaConnect.clickhouse.password", "default")
}
