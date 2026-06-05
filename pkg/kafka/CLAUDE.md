# kafka

<!-- archie:ai-start -->

> Shared Kafka utility package: typed librdkafka config builders (Consumer/Producer/Admin), an LRU-cached idempotent TopicProvisioner, and slog bridging for librdkafka's log channel. Consumed by ingest/kafkaingest, sink, watermill/driver/kafka, and the cmd/* entrypoints to construct confluent-kafka-go clients from validated OpenMeter config.

## Patterns

**Config types implement ConfigMapper + ConfigValidator** — Every config struct provides AsConfigMap() (kafka.ConfigMap, error) and Validate() error, asserted at package scope via compile-time `var _ ConfigMapper = (*T)(nil)` / `var _ ConfigValidator = (*T)(nil)` checks. (`var (_ ConfigMapper = (*ConsumerConfigParams)(nil); _ ConfigValidator = (*ConsumerConfigParams)(nil))`)
**Composed config + mergeConfigsToMap** — Top-level ConsumerConfig/ProducerConfig embed CommonConfigParams plus a role-specific params struct; AsConfigMap delegates to mergeConfigsToMap(...) and Validate loops a []ConfigValidator returning the first error. (`func (c ConsumerConfig) AsConfigMap() (kafka.ConfigMap, error) { return mergeConfigsToMap(c.CommonConfigParams, c.ConsumerConfigParams) }`)
**Only-set-if-nonzero ConfigMap keys** — AsConfigMap only calls m.SetKey(...) for non-zero fields (string != "", duration > 0), leaving librdkafka defaults intact otherwise; every SetKey error is wrapped/returned. (`if c.ClientID != "" { if err := m.SetKey("client.id", c.ClientID); err != nil { return nil, err } }`)
**String-enum types with configValue interface** — Enums (BrokerAddressFamily, AutoOffsetReset, Partitioner, TimeDurationMilliSeconds) implement fmt.Stringer + encoding.TextUnmarshaler + json.Unmarshaler, with UnmarshalJSON delegating to UnmarshalText, plus a ValidValues[T] slice for validation. (`func (s *BrokerAddressFamily) UnmarshalJSON(data []byte) error { return s.UnmarshalText(data) }`)
**OTel metric creation with wrapped errors** — In NewTopicProvisioner each Meter.Int64Counter/Int64Gauge call is immediately followed by `if err != nil { return nil, fmt.Errorf("failed to create metric: <name>: %w", err) }`; metrics live in an anonymous struct field on topicProvisioner. (`provisioner.metrics.Errors, err = config.Meter.Int64Counter("topicprovisioner.errors", ...)`)
**Required-dependency constructor guards** — NewTopicProvisioner returns an error (never panics) when AdminClient, Logger, or Meter is nil; interface seams (AdminClient, TopicProvisioner, LogEmitter) enable mock tests. (`if config.Meter == nil { return nil, errors.New("meter is required") }`)
**Idempotent provisioning via LRU cache + benign-error tolerance** — Provision skips topics already in the expirable LRU cache; CreateTopics results with ErrNoError/ErrTopicAlreadyExists (or ErrNoError/ErrUnknownTopicOrPart on delete) are treated as success and cached, all other errors collected via errors.Join. DeProvision skips empty names and protectedTopics. (`case kafka.ErrNoError, kafka.ErrTopicAlreadyExists: _ = p.cache.Add(result.Topic, struct{}{})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `config.go` | All typed config structs (CommonConfigParams, ConsumerConfigParams, ProducerConfigParams, composed ConsumerConfig/ProducerConfig/AdminConfig), enum types, ValidValues[T], mergeConfigsToMap. | A new field needs both an AsConfigMap SetKey branch and (if validated) a Validate check; localhost/127.0.0.1 brokers auto-force broker.address.family=v4 when unset; setting TopicMetadataRefreshInterval also writes metadata.max.age.ms = 3x. |
| `topicprovisioner.go` | TopicProvisioner interface + topicProvisioner impl with LRU cache + OTel metrics; TopicConfig/TopicProvisionerConfig; TopicProvisionerNoop; AdminClient interface seam. | TopicConfig.Validate requires non-empty Name and Partitions > 0; ReplicationFactor/retention.ms only set when > 0; DeProvision silently skips protectedTopics — never bypass that guard. |
| `log.go` | Bridges librdkafka's go.logs.channel to *slog.Logger via LogEmitter, LogProcessor (execute/interrupt run-group pair), ConsumeLogChannel, and syslog->slog.Level mapping. | Requires go.logs.channel.enable=true (set in Consumer/Producer AsConfigMap); mapLogLevel maps 0-3->Error, 4->Warn, 5->Info, else Debug. |
| `kafka.go` | Package doc comment only. | No logic here; godoc anchor for the package. |
| `config_test.go` | Table-driven tests for config-map building, enum UnmarshalText/UnmarshalJSON round-trips, and Validate error messages. | New config keys must be added to ExpectedConfigMap; tests compare exact error strings, so changing a Validate message breaks them. |
| `topicprovisioner_test.go` | Tests provisioning via mockTopicProvisioner (an AdminClient mock) using a noop meter and testutils.NewDiscardLogger. | Uses noop.NewMeterProvider() and testutils.NewDiscardLogger(t) — mirror this for new provisioner tests. |

## Anti-Patterns

- Adding a config struct without the `var _ ConfigMapper`/`var _ ConfigValidator` assertions, or implementing only one of AsConfigMap/Validate.
- Unconditionally SetKey-ing a field (overriding librdkafka defaults) instead of guarding on the non-zero check, or ignoring the SetKey error.
- Creating an OTel instrument in NewTopicProvisioner without the immediate `fmt.Errorf("failed to create metric: <name>: %w", err)` guard.
- panicking or using slog.Default() in the provisioner instead of returning an error / requiring an injected *slog.Logger and metric.Meter.
- Treating ErrTopicAlreadyExists (Provision) or ErrUnknownTopicOrPart (DeProvision) as hard failures, or deleting topics in protectedTopics.

## Decisions

- **Config is split into a shared CommonConfigParams plus role-specific params structs, composed into ConsumerConfig/ProducerConfig/AdminConfig.** — Common broker/SASL/stats/debug settings are reused across roles while consumer- and producer-only tuning stays type-segregated and independently validatable.
- **TopicProvisioner keeps an expirable LRU cache of provisioned topic names.** — Lets callers invoke Provision repeatedly with the same topic set without an extra round-trip to Kafka brokers for already-known topics.
- **Localhost brokers default broker.address.family to IPv4 when unset.** — OSX resolvers return IPv6 first, breaking local dev against localhost brokers (openmeter issue #321).

## Example: Define a validated, config-mappable Kafka config struct

```
var (
	_ ConfigMapper    = (*ProducerConfigParams)(nil)
	_ ConfigValidator = (*ProducerConfigParams)(nil)
)

type ProducerConfigParams struct {
	Partitioner Partitioner
}

func (p ProducerConfigParams) Validate() error {
	if p.Partitioner != "" && !slices.Contains(PartitionerValues, p.Partitioner) {
		return errors.New("invalid partitioner config")
	}
	return nil
}
// ...
```

<!-- archie:ai-end -->
