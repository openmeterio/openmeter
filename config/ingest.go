package config

import (
	"errors"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/spf13/viper"
)

type IngestConfiguration struct {
	Kafka KafkaIngestConfiguration
}

// Validate validates the configuration.
func (c IngestConfiguration) Validate() error {
	if err := c.Kafka.Validate(); err != nil {
		return fmt.Errorf("kafka: %w", err)
	}

	return nil
}

type KafkaIngestConfiguration struct {
	Broker              string
	SecurityProtocol    string
	SaslMechanisms      string
	SaslUsername        string
	SaslPassword        string
	Partitions          int
	EventsTopicTemplate string
}

// CreateKafkaConfig creates a Kafka config map.
func (c KafkaIngestConfiguration) CreateKafkaConfig() kafka.ConfigMap {
	config := kafka.ConfigMap{
		"bootstrap.servers": c.Broker,

		// Required for logging
		"go.logs.channel.enable": true,
	}

	if c.SecurityProtocol != "" {
		config["security.protocol"] = c.SecurityProtocol
	}

	if c.SaslMechanisms != "" {
		config["sasl.mechanism"] = c.SaslMechanisms
	}

	if c.SaslUsername != "" {
		config["sasl.username"] = c.SaslUsername
	}

	if c.SaslPassword != "" {
		config["sasl.password"] = c.SaslPassword
	}

	return config
}

// Validate validates the configuration.
func (c KafkaIngestConfiguration) Validate() error {
	if c.Broker == "" {
		return errors.New("broker is required")
	}

	if c.EventsTopicTemplate == "" {
		return errors.New("events topic template is required")
	}

	return nil
}

// Configure configures some defaults in the Viper instance.
func ConfigureIngest(v *viper.Viper) {
	v.SetDefault("ingest.kafka.broker", "127.0.0.1:29092")
	v.SetDefault("ingest.kafka.securityProtocol", "")
	v.SetDefault("ingest.kafka.saslMechanisms", "")
	v.SetDefault("ingest.kafka.saslUsername", "")
	v.SetDefault("ingest.kafka.saslPassword", "")
	v.SetDefault("ingest.kafka.partitions", 1)
	v.SetDefault("ingest.kafka.eventsTopicTemplate", "om_%s_events")
}
