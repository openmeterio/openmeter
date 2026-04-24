# kafka

<!-- archie:ai-start -->

> Shared low-level Kafka utilities: typed config structs that produce kafka.ConfigMap, a TopicProvisioner with LRU cache, log bridging from librdkafka to slog, and an OTel metrics sub-package. Primary constraint: all code here is infrastructure plumbing — no domain or business logic.

## Patterns

**ConfigMapper + ConfigValidator interfaces** — Every config struct (CommonConfigParams, ConsumerConfigParams, ProducerConfigParams) implements both ConfigMapper (AsConfigMap() → kafka.ConfigMap) and ConfigValidator (Validate() error). Composite configs (ConsumerConfig, ProducerConfig) embed the parts and delegate to mergeConfigsToMap + sequential validators. (`type ConsumerConfig struct { CommonConfigParams; ConsumerConfigParams }; func (c ConsumerConfig) AsConfigMap() (kafka.ConfigMap, error) { return mergeConfigsToMap(c.CommonConfigParams, c.ConsumerConfigParams) }`)
**configValue interface for enum types** — Every string-based enum (BrokerAddressFamily, Partitioner, AutoOffsetReset, TimeDurationMilliSeconds) implements fmt.Stringer + encoding.TextUnmarshaler + json.Unmarshaler. Validation uses a ValidValues[T] slice with AsKeyMap() for O(1) lookup. (`var BrokerAddressFamilyValues = ValidValues[BrokerAddressFamily]{BrokerAddressFamilyAny, BrokerAddressFamilyIPv4, BrokerAddressFamilyIPv6}`)
**TopicProvisioner LRU cache + noop guard** — NewTopicProvisioner returns a TopicProvisionerNoop when no Kafka admin is available. The concrete topicProvisioner skips already-cached topics in Provision and skips protected topics in DeProvision. Always treat ErrTopicAlreadyExists and ErrUnknownTopicOrPart as success, not errors. (`case kafka.ErrNoError, kafka.ErrTopicAlreadyExists: _ = p.cache.Add(result.Topic, struct{}{})`)
**Sequential error returns in constructors** — NewTopicProvisioner validates each required field (AdminClient, Logger, Meter) with explicit nil checks before creating metrics, returning immediately on first failure. OTel metric registration errors are also returned sequentially, not collected. (`if config.AdminClient == nil { return nil, errors.New("kafka admin client is required") }`)
**LogProcessor oklog/run-compatible signature** — LogProcessor returns (execute func() error, interrupt func(error)) — the oklog/run pair. It starts an internal context for cancellation. ConsumeLogChannel is a simpler goroutine-targeted variant that ranges over the log channel directly. (`func LogProcessor(logEmitter LogEmitter, logger *slog.Logger) (execute func() error, interrupt func(error))`)
**TimeDurationMilliSeconds rendered as millisecond integer string** — TimeDurationMilliSeconds.String() returns the duration in milliseconds as a decimal string (e.g. "6000" for 6s). This is the format librdkafka expects for *.ms config keys. Never pass raw time.Duration to kafka.ConfigMap. (`func (d TimeDurationMilliSeconds) String() string { return strconv.FormatInt(int64(d/TimeDurationMilliSeconds(time.Millisecond)), 10) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `config.go` | Defines all Kafka config param structs and their ConfigMapper/ConfigValidator implementations. mergeConfigsToMap merges multiple ConfigMapper outputs into one kafka.ConfigMap. | metadata.max.age.ms is always set to 3× TopicMetadataRefreshInterval — do not set it independently. localhost/127.0.0.1 brokers auto-force BrokerAddressFamilyIPv4 if BrokerAddressFamily is empty. |
| `topicprovisioner.go` | TopicProvisioner interface + LRU-cached implementation + TopicProvisionerNoop. Owns topic creation/deletion against Kafka admin API with metrics. | ProtectedTopics prevents accidental deletion — always populate this list with system topics. Cache is expirable LRU; cache eviction fires a metrics counter via evictCallback, which must use a short-lived context, not the caller's ctx. |
| `log.go` | Bridges librdkafka syslog-level integers to slog.Level. Requires go.logs.channel.enable=true in kafka.ConfigMap (set automatically by ConsumerConfigParams and ProducerConfigParams). | go.logs.channel.enable and go.application.rebalance.enable are hardcoded in ConsumerConfigParams.AsConfigMap() — do not remove them or librdkafka logs will silently drop. |
| `kafka.go` | Package doc only — intentionally empty aside from the package declaration. | No init() or global state here; keep it that way. |
| `metrics/ (child package)` | OTel Int64Gauge metrics translated from librdkafka JSON stats snapshots. Separate sub-package so callers opt in. | All metrics are gauges (point-in-time snapshots), never counters/histograms. Nil guard required at top of Add because librdkafka may omit stats sections. |

## Anti-Patterns

- Setting kafka.ConfigMap keys directly instead of going through CommonConfigParams.AsConfigMap / mergeConfigsToMap — bypasses validation and the localhost IPv4 auto-fix.
- Using Int64Counter or Histogram for librdkafka-derived stats — they are rolling-window snapshots, not cumulative deltas; only Int64Gauge is correct.
- Passing raw time.Duration as a kafka.ConfigMap value for *.ms keys — must use TimeDurationMilliSeconds which renders as millisecond integer string.
- Accessing stats sections in metrics.Add without a nil guard — librdkafka JSON can omit broker/topic/consumer-group sections entirely.
- Mixing cooperative and eager PartitionAssignmentStrategy values — ConsumerConfigParams.Validate() rejects this combination explicitly.

## Decisions

- **ConfigMapper + ConfigValidator split into separate interfaces implemented by value receivers** — Composite configs (ConsumerConfig, ProducerConfig) embed multiple param structs; mergeConfigsToMap iterates them independently so each part owns its own key namespace and validation without coordinating with others.
- **TopicProvisioner uses an expirable LRU cache keyed by topic name** — Provision is called on every namespace creation; the cache prevents redundant Kafka admin round-trips for topics that already exist, while TTL-based expiry avoids stale cache entries after broker-side topic deletion.
- **OTel metrics live in a separate metrics/ sub-package behind functional options** — Not all consumers of pkg/kafka need stats instrumentation; separating it avoids pulling OTel metric dependencies into the base config package and lets callers opt in per sub-metric category.

## Example: Constructing and validating a ConsumerConfig then provisioning topics

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
    },
    ConsumerConfigParams: kafka.ConsumerConfigParams{
        ConsumerGroupID: "my-group",
        AutoOffsetReset: kafka.AutoOffsetResetEarliest,
    },
// ...
```

<!-- archie:ai-end -->
