# kafka

<!-- archie:ai-start -->

> Shared low-level Kafka infrastructure: typed config structs that compile to kafka.ConfigMap, a TopicProvisioner with LRU cache, librdkafka log bridging to slog, and an OTel metrics sub-package. Primary constraint: pure infrastructure plumbing — no domain or business logic.

## Patterns

**ConfigMapper + ConfigValidator interfaces** — Every config param struct (CommonConfigParams, ConsumerConfigParams, ProducerConfigParams) implements both ConfigMapper (AsConfigMap() → kafka.ConfigMap) and ConfigValidator (Validate() error). Composite configs (ConsumerConfig, ProducerConfig, AdminConfig) embed param structs and delegate to mergeConfigsToMap + sequential validators. (`type ConsumerConfig struct { CommonConfigParams; ConsumerConfigParams }
func (c ConsumerConfig) AsConfigMap() (kafka.ConfigMap, error) { return mergeConfigsToMap(c.CommonConfigParams, c.ConsumerConfigParams) }`)
**configValue interface for string enums** — Every string-based enum (BrokerAddressFamily, Partitioner, AutoOffsetReset, TimeDurationMilliSeconds) implements fmt.Stringer + encoding.TextUnmarshaler + json.Unmarshaler. Validation uses ValidValues[T] slice with AsKeyMap() for O(1) lookup. (`var BrokerAddressFamilyValues = ValidValues[BrokerAddressFamily]{BrokerAddressFamilyAny, BrokerAddressFamilyIPv4, BrokerAddressFamilyIPv6}`)
**TimeDurationMilliSeconds renders as millisecond integer string** — TimeDurationMilliSeconds.String() returns the duration in milliseconds as a decimal string (e.g. '6000' for 6s) — the format librdkafka expects for *.ms config keys. Never pass raw time.Duration to kafka.ConfigMap. (`func (d TimeDurationMilliSeconds) String() string { return strconv.FormatInt(int64(d/TimeDurationMilliSeconds(time.Millisecond)), 10) }`)
**TopicProvisioner LRU cache + noop guard** — NewTopicProvisioner returns a TopicProvisionerNoop when Kafka admin is unavailable. The concrete topicProvisioner skips cached topics in Provision and skips ProtectedTopics in DeProvision. Both ErrTopicAlreadyExists and ErrUnknownTopicOrPart are treated as success, not errors. (`case kafka.ErrNoError, kafka.ErrTopicAlreadyExists:
  _ = p.cache.Add(result.Topic, struct{}{})`)
**Sequential fail-fast validation in constructors** — NewTopicProvisioner validates AdminClient, Logger, and Meter with explicit nil checks before registering any OTel metrics, returning immediately on first failure. OTel registration errors are also returned sequentially. (`if config.AdminClient == nil { return nil, errors.New("kafka admin client is required") }`)
**LogProcessor oklog/run-compatible pair** — LogProcessor returns (execute func() error, interrupt func(error)) — the exact oklog/run.Group pair. ConsumeLogChannel is the simpler goroutine-targeted variant. Both require go.logs.channel.enable=true which ConsumerConfigParams and ProducerConfigParams set automatically in AsConfigMap. (`func LogProcessor(logEmitter LogEmitter, logger *slog.Logger) (execute func() error, interrupt func(error))`)
**metadata.max.age.ms always 3× TopicMetadataRefreshInterval** — CommonConfigParams.AsConfigMap sets metadata.max.age.ms to 3×TopicMetadataRefreshInterval whenever TopicMetadataRefreshInterval is non-zero. Never set metadata.max.age.ms independently — it must track refresh interval to avoid premature metadata cache expiry. (`m.SetKey("metadata.max.age.ms", 3*c.TopicMetadataRefreshInterval)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `config.go` | Defines all Kafka config param structs and their ConfigMapper/ConfigValidator implementations. mergeConfigsToMap merges multiple ConfigMapper outputs into one kafka.ConfigMap. | localhost/127.0.0.1 brokers auto-force BrokerAddressFamilyIPv4 when BrokerAddressFamily is empty — do not override this. Cooperative and eager PartitionAssignmentStrategy values must not be mixed — Validate() enforces this. StatsInterval minimum is 5s, TopicMetadataRefreshInterval minimum is 10s. |
| `topicprovisioner.go` | TopicProvisioner interface + expirable LRU-cached implementation + TopicProvisionerNoop. Manages Kafka topic creation and deletion via AdminClient with OTel metrics. | ProtectedTopics must include all system topics to prevent accidental deletion. evictCallback uses a short-lived context.WithTimeout — never use the caller's ctx here. ErrTopicAlreadyExists and ErrUnknownTopicOrPart must remain success cases. |
| `log.go` | Bridges librdkafka syslog-level integers to slog.Level. Exposes LogProcessor (oklog/run pair) and ConsumeLogChannel (goroutine variant). | go.logs.channel.enable and go.application.rebalance.enable are hardcoded in ConsumerConfigParams.AsConfigMap and ProducerConfigParams.AsConfigMap — do not remove them or librdkafka logs silently drop. |
| `kafka.go` | Package declaration only — intentionally empty aside from the package doc comment. | No init() or global state; keep it that way. |
| `metrics/ (child package)` | OTel Int64Gauge metrics translated from librdkafka JSON stats snapshots. Separate sub-package so callers opt in via functional options. | All metrics are Int64Gauge (point-in-time snapshots), never counters/histograms. Nil guard required at top of Add because librdkafka may omit broker/topic/consumer-group sections entirely. |

## Anti-Patterns

- Setting kafka.ConfigMap keys directly instead of through AsConfigMap/mergeConfigsToMap — bypasses validation and the localhost IPv4 auto-fix in CommonConfigParams.
- Using Int64Counter or Histogram for librdkafka-derived stats — they are rolling-window point-in-time snapshots, only Int64Gauge is correct.
- Passing raw time.Duration as a kafka.ConfigMap value for *.ms keys — must use TimeDurationMilliSeconds which renders as millisecond integer string.
- Setting metadata.max.age.ms independently without tying it to 3×TopicMetadataRefreshInterval — causes premature metadata cache expiry.
- Mixing cooperative and eager PartitionAssignmentStrategy values in ConsumerConfigParams — Validate() explicitly rejects this combination.

## Decisions

- **ConfigMapper + ConfigValidator split into separate interfaces implemented by value receivers** — Composite configs (ConsumerConfig, ProducerConfig) embed multiple param structs; mergeConfigsToMap iterates them independently so each part owns its own key namespace and validation without coordinating with others.
- **TopicProvisioner uses an expirable LRU cache keyed by topic name** — Provision is called on every namespace creation; the cache prevents redundant Kafka admin round-trips for topics that already exist, while TTL-based expiry avoids stale entries after broker-side topic deletion.
- **OTel metrics live in a separate metrics/ sub-package behind functional options** — Not all consumers of pkg/kafka need stats instrumentation; separating it avoids pulling OTel metric dependencies into the base config package and lets callers opt in per sub-metric category.

## Example: Constructing and validating a ConsumerConfig then provisioning a topic

```
import (
    "github.com/openmeterio/openmeter/pkg/kafka"
    "go.opentelemetry.io/otel/metric/noop"
    "time"
)

cfg := kafka.ConsumerConfig{
    CommonConfigParams: kafka.CommonConfigParams{
        Brokers:       "broker:9092",
        StatsInterval: kafka.TimeDurationMilliSeconds(10 * time.Second),
        TopicMetadataRefreshInterval: kafka.TimeDurationMilliSeconds(30 * time.Second),
    },
    ConsumerConfigParams: kafka.ConsumerConfigParams{
        ConsumerGroupID: "my-group",
        AutoOffsetReset: kafka.AutoOffsetResetEarliest,
// ...
```

<!-- archie:ai-end -->
