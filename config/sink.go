package config

import (
	"errors"
	"fmt"

	"github.com/mitchellh/mapstructure"
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
	Enabled bool
	URL     string

	Connectors []ConnectorKafkaConnectSinkConfiguration
}

func (c KafkaConnectSinkConfiguration) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.URL == "" {
		return errors.New("url is required")
	}

	for _, connector := range c.Connectors {
		if err := connector.Validate(); err != nil {
			return err
		}
	}

	return nil
}

type ConnectorKafkaConnectSinkConfiguration struct {
	Name string

	ConnectorTypeKafkaConnectSinkConfiguration
}

type ConnectorTypeKafkaConnectSinkConfiguration interface {
	ConnectorConfig() (map[string]string, error)
	Validate() error
}

func (c ConnectorKafkaConnectSinkConfiguration) ConnectorConfig() (map[string]string, error) {
	if c.ConnectorTypeKafkaConnectSinkConfiguration == nil {
		return nil, errors.New("sink: kafka connect: connector: missing configuration")
	}

	return c.ConnectorTypeKafkaConnectSinkConfiguration.ConnectorConfig()
}

func (c ConnectorKafkaConnectSinkConfiguration) Validate() error {
	if c.ConnectorTypeKafkaConnectSinkConfiguration == nil {
		return errors.New("missing connector configuration")
	}

	if err := c.ConnectorTypeKafkaConnectSinkConfiguration.Validate(); err != nil {
		return fmt.Errorf("connector(%s): %w", c.Name, err)
	}

	return nil
}

type rawConnectorKafkaConnectSinkConfiguration struct {
	Name   string
	Type   string
	Config map[string]any
}

func (c *ConnectorKafkaConnectSinkConfiguration) DecodeMap(v map[string]any) error {
	var rawConfig rawConnectorKafkaConnectSinkConfiguration

	err := mapstructure.Decode(v, &rawConfig)
	if err != nil {
		return err
	}

	c.Name = rawConfig.Name

	switch rawConfig.Type {
	case "clickhouse":
		var connectorConfig ClickHouseConnectorTypeKafkaConnectSinkConfiguration

		decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			Metadata:         nil,
			Result:           &connectorConfig,
			WeaklyTypedInput: true,
			DecodeHook: mapstructure.ComposeDecodeHookFunc(
				mapstructure.StringToTimeDurationHookFunc(),
			),
		})
		if err != nil {
			return fmt.Errorf("sink: kafka connect: creating decoder: %w", err)
		}

		err = decoder.Decode(rawConfig.Config)
		if err != nil {
			return fmt.Errorf("dedupe: decoding redis driver config: %w", err)
		}

		c.ConnectorTypeKafkaConnectSinkConfiguration = connectorConfig

	default:
		c.ConnectorTypeKafkaConnectSinkConfiguration = unknownConnectorTypeKafkaConnectSinkConfiguration{
			connector: rawConfig.Type,
		}
	}

	return nil
}

// This may feel repetative but clikhouse sink and processor configs can be different,
// for example Kafka Connect ClickHouse plugin uses 8123 HTTP port while client uses native protocol's 9000 port.
// Hostname can be also different, as Kafka Connect and ClickHouse communicates inside the docker compose network.
type ClickHouseConnectorTypeKafkaConnectSinkConfiguration struct {
	Hostname string
	Port     int
	SSL      bool
	Username string
	Password string
	Database string

	DeadLetterQueue DeadLetterQueueKafkaConnectSinkConfiguration
}

func (c ClickHouseConnectorTypeKafkaConnectSinkConfiguration) ConnectorConfig() (map[string]string, error) {
	return map[string]string{
		"connector.class":                   "com.clickhouse.kafka.connect.ClickHouseSinkConnector",
		"database":                          c.Database,
		"errors.retry.timeout":              "30",
		"hostname":                          c.Hostname,
		"port":                              fmt.Sprint(c.Port),
		"ssl":                               fmt.Sprint(c.SSL),
		"username":                          c.Username,
		"password":                          c.Password,
		"key.converter":                     "org.apache.kafka.connect.storage.StringConverter",
		"value.converter":                   "org.apache.kafka.connect.json.JsonConverter",
		"value.converter.schemas.enable":    "false",
		"schemas.enable":                    "false",
		"topics.regex":                      "^om_[A-Za-z0-9]+(?:_[A-Za-z0-9]+)*_events$",
		"errors.tolerance":                  "all",
		"errors.deadletterqueue.topic.name": c.DeadLetterQueue.TopicName,
		"errors.deadletterqueue.topic.replication.factor": fmt.Sprint(c.DeadLetterQueue.ReplicationFactor),
		"errors.deadletterqueue.context.headers.enable":   fmt.Sprint(c.DeadLetterQueue.ContextHeaders),
	}, nil
}

func (c ClickHouseConnectorTypeKafkaConnectSinkConfiguration) Validate() error {
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

	if err := c.DeadLetterQueue.Validate(); err != nil {
		return fmt.Errorf("deadletterqueue: %w", err)
	}

	return nil
}

type unknownConnectorTypeKafkaConnectSinkConfiguration struct {
	connector string
}

func (c unknownConnectorTypeKafkaConnectSinkConfiguration) ConnectorConfig() (map[string]string, error) {
	return nil, fmt.Errorf("sink: kafka connect: unknown connector: %s", c.connector)
}

func (c unknownConnectorTypeKafkaConnectSinkConfiguration) Validate() error {
	return fmt.Errorf("unknown connector: %s", c.connector)
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

	if c.ReplicationFactor < 1 {
		return errors.New("dead letter queue replication factor is required")
	}

	return nil
}

// Configure configures some defaults in the Viper instance.
func configureSink(v *viper.Viper) {
	v.SetDefault("sink.kafkaConnect.enabled", false)
	v.SetDefault("sink.kafkaConnect.url", "http://127.0.0.1:8083")
}
