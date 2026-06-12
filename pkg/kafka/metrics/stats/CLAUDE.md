# stats

<!-- archie:ai-start -->

> Plain Go structs that unmarshal librdkafka's STATISTICS.md JSON (the rdkafka stats callback payload) into typed BrokerStats/TopicStats/Partition/ConsumerGroupStats/Stats, with string-enum types that expose stable .Int64() codes for metric recording. Pure data/parsing layer with no OTel dependency.

## Patterns

**Struct fields map librdkafka JSON keys via json tags** — Every field carries a `json:"..."` tag matching librdkafka's stats schema (e.g. RequestsSent int64 `json:"tx"`, MessagesInQueue `json:"msgq_cnt"`). Field names are human-readable; tags are the cryptic rdkafka keys. New fields must use the exact rdkafka key from STATISTICS.md. (`RequestsAwaitingTransmission int64 `json:"outbuf_cnt"``)
**String enums with UnmarshalJSON + Int64 projection** — Enum types (BrokerSource, BrokerState, ConsumerGroupState, ConsumerGroupJoinState) are `type X string` with const values, a case-insensitive UnmarshalJSON (strings.ToLower(strings.TrimSpace(value))) defaulting to the Unknown variant, and an Int64() method returning a fixed integer code (Unknown => -1). (`func (s BrokerState) Int64() int64 { switch s { case BrokerStateUp: i = 6; ...; default: i = -1 }; return i }`)
**WindowStats for rolling percentile windows** — Any rolling-window stat (rtt, throttle, batch_size, batch_count) is typed as WindowStats with Min/Max/Avg/Sum/Count/StdDev/P50..P9999 fields. The p99.99 field tag is `json:"p99_99"`. (`Latency WindowStats `json:"rtt"`; Throttle WindowStats `json:"throttle"``)
**Nested collections keyed by string** — Aggregates use map[string]T for child collections: Stats.Brokers map[string]BrokerStats, Stats.Topics map[string]TopicStats, TopicStats.Partitions map[string]Partition, keyed by rdkafka's string identifiers (broker name, topic name, partition id as string). (`Topics map[string]TopicStats `json:"topics"``)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `stats.go` | Top-level Stats struct (the full rdkafka payload root) plus the shared WindowStats type used for all percentile windows. | Stats is the unmarshal entrypoint; WindowStats.P9999 maps to json key `p99_99`, not `p9999`. |
| `broker.go` | BrokerStats plus BrokerSource and BrokerState enums (with UnmarshalJSON + Int64). | BrokerState.Int64 maps up=6/update=7 and unknown=-1; keep the int codes stable since internal/broker.go documents them in metric descriptions. |
| `consumergroup.go` | ConsumerGroupStats plus ConsumerGroupState and ConsumerGroupJoinState enums. | JoinState has 9 variants (init..steady, codes 0-8); the wait-incr-unassign-to-complete json value is abbreviated, not the full Go const name. |
| `topic.go` | TopicStats and Partition structs (no enums). Partition holds offsets, lag, queue depths, throughput. | Partition id -1 denotes the internal UA/UnAssigned partition (callers in internal/ skip it); offset fields like StoredOffset use librdkafka sentinel -1001. |
| `stats_test.go` | Round-trip test unmarshalling testdata/stats.json and asserting broker/topic/partition fields. | Uses //go:embed testdata/stats.json; new fields with parsing logic should be asserted here against a representative payload. |

## Anti-Patterns

- Adding a field without the correct librdkafka json tag (the Go field name will silently not bind to the rdkafka key)
- Adding an enum variant without updating both UnmarshalJSON (lowercase value) and Int64() (stable code), or omitting the default->Unknown/-1 case
- Changing an existing enum Int64() code — the integer mappings are documented in internal/ metric descriptions and consumed downstream
- Importing OTel or any metric machinery here — this package is pure data and must stay dependency-light

## Decisions

- **librdkafka stats are modeled as typed Go structs with explicit json tags rather than a generic map** — Gives type-safe access and self-documenting field names while still binding to librdkafka's terse STATISTICS.md keys.
- **Enums carry an Int64() projection alongside string values** — OTel gauges are numeric, so the sibling internal package records enum state as a stable integer code; keeping the projection on the type centralizes the mapping.

## Example: Defining a librdkafka string enum with case-insensitive unmarshal and numeric projection

```
type BrokerState string

const (
	BrokerStateUnknown BrokerState = "unknown"
	BrokerStateUp      BrokerState = "up"
)

func (s *BrokerState) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("failed to unmarshal value: %s", data)
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "up":
		*s = BrokerStateUp
// ...
```

<!-- archie:ai-end -->
