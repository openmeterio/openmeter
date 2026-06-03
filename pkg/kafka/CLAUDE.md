# kafka

<!-- archie:ai-start -->

> Shared low-level Kafka infrastructure: typed config-param structs that compile to confluent-kafka-go kafka.ConfigMap, a TopicProvisioner with an expirable LRU cache, librdkafka log bridging to slog, and an OTel metrics sub-package. Primary constraint: pure infrastructure plumbing — no domain or business logic.

## Patterns

**ConfigMapper + ConfigValidator interfaces on every param struct** — Each param struct (CommonConfigParams, ConsumerConfigParams, ProducerConfigParams) implements both AsConfigMap() (kafka.ConfigMap, error) and Validate() error. Composite configs (ConsumerConfig, ProducerConfig, AdminConfig) embed the param structs and delegate via mergeConfigsToMap + a sequential validator loop. (`type ConsumerConfig struct { CommonConfigParams; ConsumerConfigParams }
func (c ConsumerConfig) AsConfigMap() (kafka.ConfigMap, error) { return mergeConfigsToMap(c.CommonConfigParams, c.ConsumerConfigParams) }`)
**configValue interface for string enums** — Every string-based enum (BrokerAddressFamily, Partitioner, AutoOffsetReset, TimeDurationMilliSeconds) implements fmt.Stringer + encoding.TextUnmarshaler + json.Unmarshaler. Validation uses a ValidValues[T] slice with AsKeyMap() for O(1) lookup, or slices.Contains. (`var BrokerAddressFamilyValues = ValidValues[BrokerAddressFamily]{BrokerAddressFamilyAny, BrokerAddressFamilyIPv4, BrokerAddressFamilyIPv6}`)
**TimeDurationMilliSeconds renders as a millisecond integer string** — TimeDurationMilliSeconds.String() returns the duration in milliseconds as a decimal string (e.g. '6000' for 6s) — the format librdkafka expects for *.ms keys. Never pass a raw time.Duration as a kafka.ConfigMap value for *.ms config keys. (`func (d TimeDurationMilliSeconds) String() string { return strconv.FormatInt(int64(d/TimeDurationMilliSeconds(time.Millisecond)), 10) }`)
**localhost auto-forces BrokerAddressFamilyIPv4** — CommonConfigParams.AsConfigMap sets broker.address.family to IPv4 when BrokerAddressFamily is empty and Brokers contains 'localhost' or '127.0.0.1' (OSX IPv6 resolver workaround). Do not override or remove this branch. (`} else if strings.Contains(c.Brokers, "localhost") || strings.Contains(c.Brokers, "127.0.0.1") { m.SetKey("broker.address.family", BrokerAddressFamilyIPv4) }`)
**metadata.max.age.ms always 3x TopicMetadataRefreshInterval** — When TopicMetadataRefreshInterval is non-zero, AsConfigMap also sets metadata.max.age.ms to 3x that value so the metadata cache never expires before a refresh. Never set metadata.max.age.ms independently. (`m.SetKey("metadata.max.age.ms", 3*c.TopicMetadataRefreshInterval)`)
**TopicProvisioner LRU cache + noop + error-as-success** — topicProvisioner skips cache-hit topics in Provision and ProtectedTopics in DeProvision; ErrTopicAlreadyExists (provision) and ErrUnknownTopicOrPart (de-provision) are treated as success alongside ErrNoError. NewTopicProvisioner validates AdminClient/Logger/Meter (nil checks, fail-fast) before registering OTel metrics. (`switch result.Error.Code() { case kafka.ErrNoError, kafka.ErrTopicAlreadyExists: _ = p.cache.Add(result.Topic, struct{}{}) }`)
**LogProcessor as an oklog/run-compatible pair** — LogProcessor returns (execute func() error, interrupt func(error)) for run.Group; ConsumeLogChannel is the goroutine variant. Both require go.logs.channel.enable=true, which ConsumerConfigParams and ProducerConfigParams hardcode in AsConfigMap. (`func LogProcessor(logEmitter LogEmitter, logger *slog.Logger) (execute func() error, interrupt func(error))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `config.go` | All Kafka config param structs (Common/Consumer/Producer/Admin) with ConfigMapper/ConfigValidator impls, string-enum types, ValidValues[T], and mergeConfigsToMap. | localhost/127.0.0.1 auto-forces IPv4 when BrokerAddressFamily is empty. Cooperative and eager PartitionAssignmentStrategy values must not be mixed — Validate() rejects this. StatsInterval min is 5s, TopicMetadataRefreshInterval min is 10s. |
| `topicprovisioner.go` | TopicProvisioner interface + expirable-LRU-cached topicProvisioner + TopicProvisionerNoop. Provisions/de-provisions topics via AdminClient with OTel metrics. | ProtectedTopics must list every system topic to prevent accidental deletion. evictCallback uses a fresh context.WithTimeout(3s) — never the caller ctx. ErrTopicAlreadyExists and ErrUnknownTopicOrPart must remain success cases. |
| `log.go` | Bridges librdkafka syslog-level integers to slog.Level (mapLogLevel); exposes LogProcessor (oklog/run pair) and ConsumeLogChannel (goroutine variant). | go.logs.channel.enable (and go.application.rebalance.enable in the consumer) are hardcoded in AsConfigMap — removing them silently drops librdkafka logs. |
| `kafka.go` | Package declaration only — intentionally empty aside from the package doc comment. | No init() or global state; keep it that way. |

## Anti-Patterns

- Setting kafka.ConfigMap keys directly instead of through AsConfigMap/mergeConfigsToMap — bypasses validation and the localhost IPv4 auto-fix
- Passing a raw time.Duration as a kafka.ConfigMap value for *.ms keys — must use TimeDurationMilliSeconds which renders as a millisecond integer string
- Setting metadata.max.age.ms independently instead of tying it to 3x TopicMetadataRefreshInterval — causes premature metadata cache expiry
- Mixing cooperative and eager PartitionAssignmentStrategy values in ConsumerConfigParams — Validate() explicitly rejects this combination
- Using Int64Counter or Histogram for librdkafka-derived stats in metrics/ — they are rolling-window point-in-time snapshots; only Int64Gauge is correct

## Decisions

- **ConfigMapper + ConfigValidator split into separate interfaces implemented by value receivers** — Composite configs embed multiple param structs; mergeConfigsToMap iterates them independently so each part owns its own key namespace and validation without coordinating with others.
- **TopicProvisioner uses an expirable LRU cache keyed by topic name** — Provision is called on every namespace creation; the cache prevents redundant Kafka admin round-trips for already-existing topics, while TTL expiry avoids stale entries after broker-side topic deletion.
- **OTel stats metrics live in a separate metrics/ sub-package behind functional options** — Not all consumers of pkg/kafka need stats instrumentation; separating it avoids pulling OTel metric dependencies into the base config package and lets callers opt in per sub-metric category.

## Example: Construct and validate a ConsumerConfig then provision a topic

```
import (
	"time"
	"github.com/openmeterio/openmeter/pkg/kafka"
	"go.opentelemetry.io/otel/metric/noop"
)

cfg := kafka.ConsumerConfig{
	CommonConfigParams: kafka.CommonConfigParams{
		Brokers:                      "broker:9092",
		StatsInterval:                kafka.TimeDurationMilliSeconds(10 * time.Second),
		TopicMetadataRefreshInterval: kafka.TimeDurationMilliSeconds(30 * time.Second),
	},
	ConsumerConfigParams: kafka.ConsumerConfigParams{ConsumerGroupID: "my-group", AutoOffsetReset: kafka.AutoOffsetResetEarliest},
}
if err := cfg.Validate(); err != nil { return err }
// ...
```

<!-- archie:ai-end -->
