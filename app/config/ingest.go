package config

import (
	"errors"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/pkg/errorsx"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

type IngestConfiguration struct {
	Kafka KafkaIngestConfiguration
}

// Validate validates the configuration.
func (c IngestConfiguration) Validate() error {
	var errs []error

	if err := c.Kafka.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "kafka"))
	}

	return errors.Join(errs...)
}

type KafkaIngestConfiguration struct {
	KafkaConfiguration `mapstructure:",squash"`

	TopicProvisioner TopicProvisionerConfig

	Partitions          int
	EventsTopicTemplate string

	// NamespaceDeletionEnabled defines whether deleting namespaces are allowed or not.
	NamespaceDeletionEnabled bool
}

// Validate validates the configuration.
func (c KafkaIngestConfiguration) Validate() error {
	var errs []error

	if c.EventsTopicTemplate == "" {
		errs = append(errs, errors.New("events topic template is required"))
	}

	if err := c.KafkaConfiguration.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := c.TopicProvisioner.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

type KafkaConfiguration struct {
	Broker           string
	SecurityProtocol string
	TLSInsecure      bool
	SaslMechanisms   string
	SaslUsername     string
	SaslPassword     string

	StatsInterval pkgkafka.TimeDurationMilliSeconds

	// BrokerAddressFamily defines the IP address family to be used for network communication with Kafka cluster
	BrokerAddressFamily pkgkafka.BrokerAddressFamily
	// SocketKeepAliveEnable defines if TCP socket keep-alive is enabled to prevent closing idle connections
	// by Kafka brokers.
	SocketKeepAliveEnabled bool
	// TopicMetadataRefreshInterval defines how frequently the Kafka client needs to fetch metadata information
	// (brokers, topic, partitions, etc) from the Kafka cluster.
	// The 5 minutes default value is appropriate for mostly static Kafka clusters, but needs to be lowered
	// in case of large clusters where changes are more frequent.
	// This value must not be set to value lower than 10s.
	TopicMetadataRefreshInterval pkgkafka.TimeDurationMilliSeconds

	// Enable contexts for extensive debugging of librdkafka.
	// See: https://github.com/confluentinc/librdkafka/blob/master/INTRODUCTION.md#debug-contexts
	DebugContexts pkgkafka.DebugContexts
}

func (c KafkaConfiguration) Validate() error {
	var errs []error

	if c.Broker == "" {
		errs = append(errs, errors.New("broker is required"))
	}

	if c.StatsInterval > 0 && c.StatsInterval.Duration() < 5*time.Second {
		errs = append(errs, errors.New("StatsInterval must be >=5s"))
	}

	if c.TopicMetadataRefreshInterval > 0 && c.TopicMetadataRefreshInterval.Duration() < 10*time.Second {
		errs = append(errs, errors.New("topic metadata refresh interval must be >=10s"))
	}

	return errors.Join(errs...)
}

// CreateKafkaConfig creates a Kafka config map.
func (c KafkaConfiguration) CreateKafkaConfig() kafka.ConfigMap {
	config := kafka.ConfigMap{
		"bootstrap.servers": c.Broker,

		// Required for logging
		"go.logs.channel.enable": true,
	}

	// This is needed when using localhost brokers on OSX,
	// since the OSX resolver will return the IPv6 addresses first.
	// See: https://github.com/openmeterio/openmeter/issues/321
	if c.BrokerAddressFamily != "" {
		config["broker.address.family"] = c.BrokerAddressFamily
	} else if strings.Contains(c.Broker, "localhost") || strings.Contains(c.Broker, "127.0.0.1") {
		config["broker.address.family"] = pkgkafka.BrokerAddressFamilyIPv4
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

	if c.StatsInterval > 0 {
		config["statistics.interval.ms"] = c.StatsInterval
	}

	if c.SocketKeepAliveEnabled {
		config["socket.keepalive.enable"] = c.SocketKeepAliveEnabled
	}

	// The `topic.metadata.refresh.interval.ms` defines the frequency the Kafka client needs to retrieve metadata
	// from Kafka cluster. While `metadata.max.age.ms` defines the interval after the metadata cache maintained
	// on client side becomes invalid. Setting the former will automatically adjust the value of the latter to avoid
	// misconfiguration where the entries in metadata cache are evicted prior metadata refresh.
	if c.TopicMetadataRefreshInterval > 0 {
		config["topic.metadata.refresh.interval.ms"] = c.TopicMetadataRefreshInterval
		config["metadata.max.age.ms"] = 3 * c.TopicMetadataRefreshInterval
	}

	if len(c.DebugContexts) > 0 {
		config["debug"] = c.DebugContexts.String()
	}

	return config
}

func ConfigureIngestKafkaConfiguration(v *viper.Viper, prefixes ...string) {
	prefixer := NewViperKeyPrefixer(prefixes...)

	v.SetDefault(prefixer("kafka.broker"), "127.0.0.1:29092")
	v.SetDefault(prefixer("kafka.securityProtocol"), "")
	v.SetDefault(prefixer("kafka.saslMechanisms"), "")
	v.SetDefault(prefixer("kafka.saslUsername"), "")
	v.SetDefault(prefixer("kafka.saslPassword"), "")
	v.SetDefault(prefixer("kafka.statsInterval"), 15*time.Second)
	v.SetDefault(prefixer("kafka.brokerAddressFamily"), "v4")
	v.SetDefault(prefixer("kafka.socketKeepAliveEnabled"), true)
	v.SetDefault(prefixer("kafka.topicMetadataRefreshInterval"), time.Minute)
}

// Configure configures some defaults in the Viper instance.
func ConfigureIngest(v *viper.Viper) {
	v.SetDefault("ingest.kafka.partitions", 1)
	v.SetDefault("ingest.kafka.eventsTopicTemplate", "om_%s_events")
	v.SetDefault("ingest.kafka.namespaceDeletionEnabled", false)

	ConfigureIngestKafkaConfiguration(v, "ingest")
	ConfigureTopicProvisioner(v, "ingest", "kafka", "topicProvisioner")
}
