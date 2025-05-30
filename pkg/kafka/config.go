package kafka

import (
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/samber/lo"
)

type ConfigValidator interface {
	Validate() error
}

type ConfigMapper interface {
	AsConfigMap() (kafka.ConfigMap, error)
}

var (
	_ ConfigMapper    = (*CommonConfigParams)(nil)
	_ ConfigValidator = (*CommonConfigParams)(nil)
)

type CommonConfigParams struct {
	Brokers          string
	SecurityProtocol string
	SaslMechanisms   string
	SaslUsername     string
	SaslPassword     string

	StatsInterval TimeDurationMilliSeconds

	// StatsExtended defines if extended metrics are enabled.
	StatsExtended bool

	// BrokerAddressFamily defines the IP address family to be used for network communication with Kafka cluster
	BrokerAddressFamily BrokerAddressFamily
	// SocketKeepAliveEnable defines if TCP socket keep-alive is enabled to prevent closing idle connections
	// by Kafka brokers.
	SocketKeepAliveEnabled bool
	// TopicMetadataRefreshInterval defines how frequently the Kafka client needs to fetch metadata information
	// (brokers, topic, partitions, etc) from the Kafka cluster.
	// The 5 minutes default value is appropriate for mostly static Kafka clusters, but needs to be lowered
	// in case of large clusters where changes are more frequent.
	// This value must not be set to value lower than 10s.
	TopicMetadataRefreshInterval TimeDurationMilliSeconds

	// Enable contexts for extensive debugging of librdkafka.
	// See: https://github.com/confluentinc/librdkafka/blob/master/INTRODUCTION.md#debug-contexts
	DebugContexts DebugContexts

	// ClientID sets the Consumer/Producer identifier
	ClientID string
}

func (c CommonConfigParams) AsConfigMap() (kafka.ConfigMap, error) {
	m := kafka.ConfigMap{}

	if err := m.SetKey("bootstrap.servers", c.Brokers); err != nil {
		return nil, err
	}

	// This is needed when using localhost brokers on OSX,
	// since the OSX resolver will return the IPv6 addresses first.
	// See: https://github.com/openmeterio/openmeter/issues/321
	if c.BrokerAddressFamily != "" {
		if err := m.SetKey("broker.address.family", c.BrokerAddressFamily); err != nil {
			return nil, err
		}
	} else if strings.Contains(c.Brokers, "localhost") || strings.Contains(c.Brokers, "127.0.0.1") {
		if err := m.SetKey("broker.address.family", BrokerAddressFamilyIPv4); err != nil {
			return nil, err
		}
	}

	if c.SecurityProtocol != "" {
		if err := m.SetKey("security.protocol", c.SecurityProtocol); err != nil {
			return nil, err
		}
	}

	if c.SaslMechanisms != "" {
		if err := m.SetKey("sasl.mechanism", c.SaslMechanisms); err != nil {
			return nil, err
		}
	}

	if c.SaslUsername != "" {
		if err := m.SetKey("sasl.username", c.SaslUsername); err != nil {
			return nil, err
		}
	}

	if c.SaslPassword != "" {
		if err := m.SetKey("sasl.password", c.SaslPassword); err != nil {
			return nil, err
		}
	}

	if c.StatsInterval > 0 {
		if err := m.SetKey("statistics.interval.ms", c.StatsInterval); err != nil {
			return nil, err
		}
	}

	if c.SocketKeepAliveEnabled {
		if err := m.SetKey("socket.keepalive.enable", c.SocketKeepAliveEnabled); err != nil {
			return nil, err
		}
	}

	// The `topic.metadata.refresh.interval.ms` defines the frequency the Kafka client needs to retrieve metadata
	// from Kafka cluster. While `metadata.max.age.ms` defines the interval after the metadata cache maintained
	// on client side becomes invalid. Setting the former will automatically adjust the value of the latter to avoid
	// misconfiguration where the entries in metadata cache are evicted prior metadata refresh.
	if c.TopicMetadataRefreshInterval > 0 {
		if err := m.SetKey("topic.metadata.refresh.interval.ms", c.TopicMetadataRefreshInterval); err != nil {
			return nil, err
		}

		if err := m.SetKey("metadata.max.age.ms", 3*c.TopicMetadataRefreshInterval); err != nil {
			return nil, err
		}
	}

	if len(c.DebugContexts) > 0 {
		if err := m.SetKey("debug", c.DebugContexts.String()); err != nil {
			return nil, err
		}
	}

	if c.ClientID != "" {
		if err := m.SetKey("client.id", c.ClientID); err != nil {
			return nil, err
		}
	}

	return m, nil
}

func (c CommonConfigParams) Validate() error {
	if c.Brokers == "" {
		return errors.New("broker is required")
	}

	if c.StatsInterval > 0 && c.StatsInterval.Duration() < 5*time.Second {
		return errors.New("StatsInterval must be >=5s")
	}

	if c.TopicMetadataRefreshInterval > 0 && c.TopicMetadataRefreshInterval.Duration() < 10*time.Second {
		return errors.New("topic metadata refresh interval must be >=10s")
	}

	if c.BrokerAddressFamily != "" && !slices.Contains(BrokerAddressFamilyValues, c.BrokerAddressFamily) {
		return fmt.Errorf("invalid broker address family: %s", c.BrokerAddressFamily)
	}

	if len(c.DebugContexts) > 0 {
		keys := DebugContextValues.AsKeyMap()

		invalid := make([]DebugContext, 0, len(c.DebugContexts))
		for _, v := range c.DebugContexts {
			if _, ok := keys[v]; !ok {
				invalid = append(invalid, v)
			}
		}

		if len(invalid) > 0 {
			return fmt.Errorf("invalid debug contexts: %v", invalid)
		}
	}

	return nil
}

var (
	_ ConfigMapper    = (*ConsumerConfigParams)(nil)
	_ ConfigValidator = (*ConsumerConfigParams)(nil)
)

type ConsumerConfigParams struct {
	// ConsumerGroupID defines the group id. All clients sharing the same ConsumerGroupID belong to the same group.
	ConsumerGroupID string
	// ConsumerGroupInstanceID defines the instance id in consumer group. Setting this parameter enables static group membership.
	// Static group members are able to leave and rejoin a group within the configured SessionTimeout without prompting a group rebalance.
	// This should be used in combination with a larger session.timeout.ms to avoid group rebalances caused by transient unavailability (e.g. process restarts).
	ConsumerGroupInstanceID string

	// SessionTimeout defines the consumer group session and failure detection timeout.
	// The consumer sends periodic heartbeats (HeartbeatInterval) to indicate its liveness to the broker.
	// If no hearts are received by the broker for a group member within the session timeout,
	// the broker will remove the consumer from the group and trigger a rebalance.
	SessionTimeout TimeDurationMilliSeconds
	// Defines the consumer group session keepalive heartbeat interval.
	HeartbeatInterval TimeDurationMilliSeconds

	// EnableAutoCommit enables automatically and periodically commit offsets in the background.
	EnableAutoCommit bool
	// EnableAutoOffsetStore enables automatically store offset of last message provided to application.
	// The offset store is an in-memory store of the next offset to (auto-)commit for each partition.
	EnableAutoOffsetStore bool
	// AutoOffsetReset defines the action to take when there is no initial offset in offset store or the desired offset is out of range:
	// * "smallest","earliest","beginning": automatically reset the offset to the smallest offset
	// * "largest","latest","end": automatically reset the offset to the largest offset
	// * "error":  trigger an error (ERR__AUTO_OFFSET_RESET) which is retrieved by consuming messages and checking 'message->err'.
	AutoOffsetReset AutoOffsetReset

	// PartitionAssignmentStrategy defines one or more partition assignment strategies.
	// The elected group leader will use a strategy supported by all members of the group to assign partitions to group members.
	// If there is more than one eligible strategy, preference is determined by the order of this list (strategies earlier in the list have higher priority).
	// Cooperative and non-cooperative (eager) strategies must not be mixed.
	// Available strategies: PartitionAssignmentStrategyRange, PartitionAssignmentStrategyRoundRobin, PartitionAssignmentStrategyCooperativeSticky.
	PartitionAssignmentStrategy PartitionAssignmentStrategies

	// The maximum delay between invocations of poll() when using consumer group management.
	// This places an upper bound on the amount of time that the consumer can be idle before fetching more records.
	// If poll() is not called before expiration of this timeout, then the consumer is considered failed and
	// the group will rebalance in order to reassign the partitions to another member.
	// See https://docs.confluent.io/platform/current/installation/configuration/consumer-configs.html#max-poll-interval-ms
	MaxPollInterval TimeDurationMilliSeconds
}

func (c ConsumerConfigParams) Validate() error {
	if c.ConsumerGroupInstanceID != "" && c.ConsumerGroupID == "" {
		return errors.New("consumer group id is required if instance id is set")
	}

	if c.AutoOffsetReset != "" && !slices.Contains(AutoOffsetResetValues, c.AutoOffsetReset) {
		return errors.New("invalid auto offset reset")
	}

	if len(c.PartitionAssignmentStrategy) > 0 {
		if !lo.Every(PartitionAssignmentStrategyValues, c.PartitionAssignmentStrategy) {
			return errors.New("invalid partition assignment strategies")
		}

		if lo.Some(EagerPartitionAssignmentStrategyValues, c.PartitionAssignmentStrategy) &&
			lo.Some(CooperativePartitionAssignmentStrategyValues, c.PartitionAssignmentStrategy) {
			return errors.New("invalid partition assignment strategies: eager and cooperative strategies must no be mixed")
		}
	}

	return nil
}

func (c ConsumerConfigParams) AsConfigMap() (kafka.ConfigMap, error) {
	m := kafka.ConfigMap{
		// Required for logging
		"go.logs.channel.enable":          true,
		"go.application.rebalance.enable": true,
	}

	if c.ConsumerGroupID != "" {
		if err := m.SetKey("group.id", c.ConsumerGroupID); err != nil {
			return nil, err
		}
	}

	if c.ConsumerGroupInstanceID != "" {
		if err := m.SetKey("group.instance.id", c.ConsumerGroupInstanceID); err != nil {
			return nil, err
		}
	}

	if c.SessionTimeout > 0 {
		if err := m.SetKey("session.timeout.ms", c.SessionTimeout); err != nil {
			return nil, err
		}
	}

	if c.HeartbeatInterval > 0 {
		if err := m.SetKey("heartbeat.interval.ms", c.HeartbeatInterval); err != nil {
			return nil, err
		}
	}

	if c.MaxPollInterval > 0 {
		if err := m.SetKey("max.poll.interval.ms", c.MaxPollInterval); err != nil {
			return nil, err
		}
	}

	if err := m.SetKey("enable.auto.commit", c.EnableAutoCommit); err != nil {
		return nil, err
	}

	if err := m.SetKey("enable.auto.offset.store", c.EnableAutoOffsetStore); err != nil {
		return nil, err
	}

	if c.AutoOffsetReset != "" {
		if err := m.SetKey("auto.offset.reset", c.AutoOffsetReset); err != nil {
			return nil, err
		}
	}

	if len(c.PartitionAssignmentStrategy) > 0 {
		if err := m.SetKey("partition.assignment.strategy", c.PartitionAssignmentStrategy); err != nil {
			return nil, err
		}
	}

	return m, nil
}

var (
	_ ConfigMapper    = (*ProducerConfigParams)(nil)
	_ ConfigValidator = (*ProducerConfigParams)(nil)
)

type ProducerConfigParams struct {
	// Partitioner defines the algorithm used for assigning topic partition for produced message based its partition key.
	Partitioner Partitioner
}

func (p ProducerConfigParams) Validate() error {
	if p.Partitioner != "" && !slices.Contains(PartitionerValues, p.Partitioner) {
		return errors.New("invalid partitioner config")
	}

	return nil
}

func (p ProducerConfigParams) AsConfigMap() (kafka.ConfigMap, error) {
	m := kafka.ConfigMap{
		// Required for logging
		"go.logs.channel.enable": true,
	}

	if p.Partitioner != "" {
		if err := m.SetKey("partitioner", p.Partitioner); err != nil {
			return nil, err
		}
	}

	return m, nil
}

var (
	_ ConfigMapper    = (*ConsumerConfig)(nil)
	_ ConfigValidator = (*ConsumerConfig)(nil)
)

type ConsumerConfig struct {
	CommonConfigParams
	ConsumerConfigParams
}

func (c ConsumerConfig) AsConfigMap() (kafka.ConfigMap, error) {
	return mergeConfigsToMap(c.CommonConfigParams, c.ConsumerConfigParams)
}

func (c ConsumerConfig) Validate() error {
	validators := []ConfigValidator{
		c.CommonConfigParams,
		c.ConsumerConfigParams,
	}

	for _, validator := range validators {
		if err := validator.Validate(); err != nil {
			return err
		}
	}

	return nil
}

var (
	_ ConfigMapper    = (*ProducerConfig)(nil)
	_ ConfigValidator = (*ProducerConfig)(nil)
)

type ProducerConfig struct {
	CommonConfigParams
	ProducerConfigParams
}

func (c ProducerConfig) AsConfigMap() (kafka.ConfigMap, error) {
	return mergeConfigsToMap(c.CommonConfigParams, c.ProducerConfigParams)
}

func (c ProducerConfig) Validate() error {
	validators := []ConfigValidator{
		c.CommonConfigParams,
		c.ProducerConfigParams,
	}

	for _, validator := range validators {
		if err := validator.Validate(); err != nil {
			return err
		}
	}

	return nil
}

var (
	_ ConfigMapper    = (*AdminConfig)(nil)
	_ ConfigValidator = (*AdminConfig)(nil)
)

type AdminConfig struct {
	CommonConfigParams
}

func (c AdminConfig) AsConfigMap() (kafka.ConfigMap, error) {
	return c.CommonConfigParams.AsConfigMap()
}

func (c AdminConfig) Validate() error {
	validators := []ConfigValidator{
		c.CommonConfigParams,
	}

	for _, validator := range validators {
		if err := validator.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func mergeConfigsToMap(mappers ...ConfigMapper) (kafka.ConfigMap, error) {
	if len(mappers) == 0 {
		return nil, nil
	}

	configMap := kafka.ConfigMap{}

	for _, mapper := range mappers {
		m, err := mapper.AsConfigMap()
		if err != nil {
			return nil, err
		}

		for k, v := range m {
			configMap[k] = v
		}
	}

	return configMap, nil
}

type ValidValues[T comparable] []T

func (v ValidValues[T]) AsKeyMap() map[T]struct{} {
	m := make(map[T]struct{}, len(v))
	for _, k := range v {
		m[k] = struct{}{}
	}

	return m
}

type configValue interface {
	fmt.Stringer
	encoding.TextUnmarshaler
	json.Unmarshaler
}

var BrokerAddressFamilyValues = ValidValues[BrokerAddressFamily]{
	BrokerAddressFamilyAny,
	BrokerAddressFamilyIPv4,
	BrokerAddressFamilyIPv6,
}

const (
	BrokerAddressFamilyAny  BrokerAddressFamily = "any"
	BrokerAddressFamilyIPv4 BrokerAddressFamily = "v4"
	BrokerAddressFamilyIPv6 BrokerAddressFamily = "v6"
)

var _ configValue = (*BrokerAddressFamily)(nil)

type BrokerAddressFamily string

func (s *BrokerAddressFamily) UnmarshalText(text []byte) error {
	switch strings.ToLower(strings.TrimSpace(string(text))) {
	case "v4":
		*s = BrokerAddressFamilyIPv4
	case "v6":
		*s = BrokerAddressFamilyIPv6
	case "any":
		*s = BrokerAddressFamilyAny
	default:
		return fmt.Errorf("invalid value broker family address: %s", text)
	}

	return nil
}

func (s *BrokerAddressFamily) UnmarshalJSON(data []byte) error {
	return s.UnmarshalText(data)
}

func (s BrokerAddressFamily) String() string {
	return string(s)
}

var _ configValue = (*TimeDurationMilliSeconds)(nil)

type TimeDurationMilliSeconds time.Duration

func (d *TimeDurationMilliSeconds) UnmarshalText(text []byte) error {
	v, err := time.ParseDuration(strings.TrimSpace(string(text)))
	if err != nil {
		return fmt.Errorf("failed to parse time duration: %w", err)
	}

	*d = TimeDurationMilliSeconds(v)

	return nil
}

func (d *TimeDurationMilliSeconds) UnmarshalJSON(data []byte) error {
	return d.UnmarshalText(data)
}

func (d TimeDurationMilliSeconds) Duration() time.Duration {
	return time.Duration(d)
}

func (d TimeDurationMilliSeconds) String() string {
	return strconv.Itoa(int(time.Duration(d).Milliseconds()))
}

var DebugContextValues = ValidValues[DebugContext]{
	DebugContextGeneric,
	DebugContextBroker,
	DebugContextTopic,
	DebugContextMetadata,
	DebugContextFeature,
	DebugContextQueue,
	DebugContextMessage,
	DebugContextProtocol,
	DebugContextConsumerGroup,
	DebugContextSecurity,
	DebugContextFetch,
	DebugContextInterceptor,
	DebugContextPlugin,
	DebugContextConsumer,
	DebugContextAdmin,
	DebugContextIdempotentProducer,
	DebugContextMock,
	DebugContextAssignor,
	DebugContextConfig,
	DebugContextAll,
}

var _ configValue = (*DebugContext)(nil)

type DebugContext string

func (c DebugContext) String() string {
	return string(c)
}

func (c *DebugContext) UnmarshalText(text []byte) error {
	switch strings.ToLower(strings.TrimSpace(string(text))) {
	case "generic":
		*c = DebugContextGeneric
	case "broker":
		*c = DebugContextBroker
	case "topic":
		*c = DebugContextTopic
	case "metadata":
		*c = DebugContextMetadata
	case "feature":
		*c = DebugContextFeature
	case "queue":
		*c = DebugContextQueue
	case "msg":
		*c = DebugContextMessage
	case "protocol":
		*c = DebugContextProtocol
	case "cgrp":
		*c = DebugContextConsumerGroup
	case "security":
		*c = DebugContextSecurity
	case "fetch":
		*c = DebugContextFetch
	case "interceptor":
		*c = DebugContextInterceptor
	case "plugin":
		*c = DebugContextPlugin
	case "consumer":
		*c = DebugContextConsumer
	case "admin":
		*c = DebugContextAdmin
	case "eos":
		*c = DebugContextIdempotentProducer
	case "mock":
		*c = DebugContextMock
	case "assignor":
		*c = DebugContextAssignor
	case "conf":
		*c = DebugContextConfig
	case "all":
		*c = DebugContextAll
	default:
		return fmt.Errorf("invalid debug context: %s", text)
	}

	return nil
}

func (c *DebugContext) UnmarshalJSON(data []byte) error {
	return c.UnmarshalText(data)
}

const (
	// DebugContextGeneric enables generic client instance level debugging.
	// Includes initialization and termination debugging.
	// Client Type: producer, consumer
	DebugContextGeneric DebugContext = "generic"
	// DebugContextBroker enables broker and connection state debugging.
	// Client Type: producer, consumer
	DebugContextBroker DebugContext = "broker"
	// DebugContextTopic enables topic and partition state debugging. Includes leader changes.
	// Client Type: producer, consumer
	DebugContextTopic DebugContext = "topic"
	// DebugContextMetadata enables cluster and topic metadata retrieval debugging.
	// Client Type: producer, consumer
	DebugContextMetadata DebugContext = "metadata"
	// DebugContextFeature enables Kafka protocol feature support as negotiated with the broker.
	// Client Type: producer, consumer
	DebugContextFeature DebugContext = "feature"
	// DebugContextQueue enables message queue debugging.
	// Client Type: producer
	DebugContextQueue DebugContext = "queue"
	// DebugContextMessage enables message debugging. Includes information about batching, compression, sizes, etc.
	// Client Type: producer, consumer
	DebugContextMessage DebugContext = "msg"
	// DebugContextProtocol enables Kafka protocol request/response debugging. Includes latency (rtt) printouts.
	// Client Type: producer, consumer
	DebugContextProtocol DebugContext = "protocol"
	// DebugContextConsumerGroup enables low-level consumer group state debugging.
	// Client Type: consumer
	DebugContextConsumerGroup DebugContext = "cgrp"
	// DebugContextSecurity enables security and authentication debugging.
	// Client Type: producer, consumer
	DebugContextSecurity DebugContext = "security"
	// DebugContextFetch enables consumer message fetch debugging. Includes decision when and why messages are fetched.
	// Client Type: consumer
	DebugContextFetch DebugContext = "fetch"
	// DebugContextInterceptor enables interceptor interface debugging.
	// Client Type: producer, consumer
	DebugContextInterceptor DebugContext = "interceptor"
	// DebugContextPlugin enables plugin loading debugging.
	// Client Type: producer, consumer
	DebugContextPlugin DebugContext = "plugin"
	// DebugContextConsumer enables high-level consumer debugging.
	// Client Type: consumer
	DebugContextConsumer DebugContext = "consumer"
	// DebugContextAdmin enables admin API debugging.
	// Client Type: admin
	DebugContextAdmin DebugContext = "admin"
	// DebugContextIdempotentProducer enables idempotent Producer debugging.
	// Client Type: producer
	DebugContextIdempotentProducer DebugContext = "eos"
	// DebugContextMock enables mock cluster functionality debugging.
	// Client Type: producer, consumer
	DebugContextMock DebugContext = "mock"
	// DebugContextAssignor enables detailed consumer group partition assignor debugging.
	// Client Type: consumer
	DebugContextAssignor DebugContext = "assignor"
	// DebugContextConfig enables displaying set configuration properties on startup.
	// Client Type: producer, consumer
	DebugContextConfig DebugContext = "conf"
	// DebugContextAll enables all of the above.
	// Client Type: producer, consumer
	DebugContextAll DebugContext = "all"
)

var _ fmt.Stringer = (DebugContexts)(nil)

type DebugContexts []DebugContext

func (d DebugContexts) String() string {
	if len(d) > 0 {
		dd := make([]string, len(d))
		for idx, v := range d {
			dd[idx] = v.String()
		}

		return strings.Join(dd, ",")
	}

	return ""
}

var PartitionerValues = ValidValues[Partitioner]{
	PartitionerRandom,
	PartitionerConsistent,
	PartitionerConsistentRandom,
	PartitionerMurmur2,
	PartitionerMurmur2Random,
	PartitionerFnv1a,
	PartitionerFnv1aRandom,
}

const (
	// PartitionerRandom uses random distribution.
	PartitionerRandom Partitioner = "random"
	// PartitionerConsistent uses the CRC32 hash of key while Empty and NULL keys are mapped to single partition.
	PartitionerConsistent Partitioner = "consistent"
	// PartitionerConsistentRandom uses CRC32 hash of key while Empty and NULL keys are randomly partitioned.
	PartitionerConsistentRandom Partitioner = "consistent_random"
	// PartitionerMurmur2 uses Java Producer compatible Murmur2 hash of key while NULL keys are mapped to single partition.
	PartitionerMurmur2 Partitioner = "murmur2"
	// PartitionerMurmur2Random uses Java Producer compatible Murmur2 hash of key whileNULL keys are randomly partitioned.
	// This is functionally equivalent to the default partitioner in the Java Producer.
	PartitionerMurmur2Random Partitioner = "murmur2_random"
	// PartitionerFnv1a uses FNV-1a hash of key whileNULL keys are mapped to single partition.
	PartitionerFnv1a Partitioner = "fnv1a"
	// PartitionerFnv1aRandom uses FNV-1a hash of key whileNULL keys are randomly partitioned.
	PartitionerFnv1aRandom Partitioner = "fnv1a_random"
)

var _ configValue = (*Partitioner)(nil)

type Partitioner string

func (s *Partitioner) UnmarshalText(text []byte) error {
	switch strings.ToLower(strings.TrimSpace(string(text))) {
	case "random":
		*s = PartitionerRandom
	case "consistent":
		*s = PartitionerConsistent
	case "consistent_random":
		*s = PartitionerConsistentRandom
	case "murmur2":
		*s = PartitionerMurmur2
	case "murmur2_random":
		*s = PartitionerMurmur2Random
	case "fnv1a":
		*s = PartitionerFnv1a
	case "fnv1a_random":
		*s = PartitionerFnv1aRandom
	default:
		return fmt.Errorf("invalid partitioner: %s", text)
	}

	return nil
}

func (s *Partitioner) UnmarshalJSON(data []byte) error {
	return s.UnmarshalText(data)
}

func (s Partitioner) String() string {
	return string(s)
}

var AutoOffsetResetValues = []AutoOffsetReset{
	AutoOffsetResetSmallest,
	AutoOffsetResetEarliest,
	AutoOffsetResetBeginning,
	AutoOffsetResetLargest,
	AutoOffsetResetLatest,
	AutoOffsetResetEnd,
	AutoOffsetResetError,
}

const (
	// AutoOffsetResetSmallest automatically reset the offset to the smallest offset.
	AutoOffsetResetSmallest AutoOffsetReset = "smallest"
	// AutoOffsetResetEarliest automatically reset the offset to the smallest offset.
	AutoOffsetResetEarliest AutoOffsetReset = "earliest"
	// AutoOffsetResetBeginning automatically reset the offset to the smallest offset.
	AutoOffsetResetBeginning AutoOffsetReset = "beginning"
	// AutoOffsetResetLargest automatically reset the offset to the largest offset.
	AutoOffsetResetLargest AutoOffsetReset = "largest"
	// AutoOffsetResetLatest automatically reset the offset to the largest offset.
	AutoOffsetResetLatest AutoOffsetReset = "latest"
	// AutoOffsetResetEnd automatically reset the offset to the largest offset.
	AutoOffsetResetEnd AutoOffsetReset = "end"
	// AutoOffsetResetError trigger an error (ERR__AUTO_OFFSET_RESET) which is retrieved by
	// consuming messages and checking 'message->err'
	AutoOffsetResetError AutoOffsetReset = "error"
)

var _ configValue = (*AutoOffsetReset)(nil)

type AutoOffsetReset string

func (s *AutoOffsetReset) UnmarshalText(text []byte) error {
	switch strings.ToLower(strings.TrimSpace(string(text))) {
	case "smallest":
		*s = AutoOffsetResetSmallest
	case "earliest":
		*s = AutoOffsetResetEarliest
	case "beginning":
		*s = AutoOffsetResetBeginning
	case "largest":
		*s = AutoOffsetResetLargest
	case "latest":
		*s = AutoOffsetResetLatest
	case "end":
		*s = AutoOffsetResetEnd
	case "error":
		*s = AutoOffsetResetError
	default:
		return fmt.Errorf("invalid auto offset reset strategy: %s", text)
	}

	return nil
}

func (s *AutoOffsetReset) UnmarshalJSON(data []byte) error {
	return s.UnmarshalText(data)
}

func (s AutoOffsetReset) String() string {
	return string(s)
}

var PartitionAssignmentStrategyValues = ValidValues[PartitionAssignmentStrategy]{
	PartitionAssignmentStrategyRange,
	PartitionAssignmentStrategyRoundRobin,
	PartitionAssignmentStrategyCooperativeSticky,
}

var EagerPartitionAssignmentStrategyValues = ValidValues[PartitionAssignmentStrategy]{
	PartitionAssignmentStrategyRange,
	PartitionAssignmentStrategyRoundRobin,
}

var CooperativePartitionAssignmentStrategyValues = ValidValues[PartitionAssignmentStrategy]{
	PartitionAssignmentStrategyCooperativeSticky,
}

var _ configValue = (*PartitionAssignmentStrategy)(nil)

type PartitionAssignmentStrategy string

func (c PartitionAssignmentStrategy) String() string {
	return string(c)
}

func (c *PartitionAssignmentStrategy) UnmarshalText(text []byte) error {
	switch strings.ToLower(strings.TrimSpace(string(text))) {
	case "range":
		*c = PartitionAssignmentStrategyRange
	case "roundrobin":
		*c = PartitionAssignmentStrategyRoundRobin
	case "cooperative-sticky":
		*c = PartitionAssignmentStrategyCooperativeSticky

	default:
		return fmt.Errorf("invalid partition assignment strategy: %s", text)
	}

	return nil
}

func (c *PartitionAssignmentStrategy) UnmarshalJSON(data []byte) error {
	return c.UnmarshalText(data)
}

const (
	// PartitionAssignmentStrategyRange assigns partitions on a per-topic basis.
	PartitionAssignmentStrategyRange PartitionAssignmentStrategy = "range"
	// PartitionAssignmentStrategyRoundRobin assigns partitions to consumers in a round-robin fashion.
	PartitionAssignmentStrategyRoundRobin PartitionAssignmentStrategy = "roundrobin"
	// PartitionAssignmentStrategyCooperativeSticky guarantees an assignment that is maximally balanced while preserving
	// as many existing partition assignments as possible while allowing cooperative rebalancing.
	PartitionAssignmentStrategyCooperativeSticky PartitionAssignmentStrategy = "cooperative-sticky"
)

var _ fmt.Stringer = (PartitionAssignmentStrategies)(nil)

// PartitionAssignmentStrategies one or more partition assignment strategies.
// If there is more than one eligible strategy, preference is determined by the configured order of strategies.
// IMPORTANT: cooperative and non-cooperative (eager) strategies must NOT be mixed.
type PartitionAssignmentStrategies []PartitionAssignmentStrategy

func (p PartitionAssignmentStrategies) String() string {
	if len(p) > 0 {
		pp := make([]string, len(p))
		for idx, v := range p {
			pp[idx] = v.String()
		}

		return strings.Join(pp, ",")
	}

	return ""
}
