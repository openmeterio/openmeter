# stats

<!-- archie:ai-start -->

> Typed Go representations of librdkafka's JSON statistics payload (https://github.com/confluentinc/librdkafka/blob/v2.4.0/STATISTICS.md). Each string-enum stat field implements custom JSON unmarshalling and an Int64() method that maps enum variants to the integers recorded by the internal metrics layer.

## Patterns

**String enum type with UnmarshalJSON + Int64()** — Each librdkafka string-enum field (BrokerSource, BrokerState, ConsumerGroupState, ConsumerGroupJoinState) is a named string type with: (a) package-level const block of all variants including an Unknown sentinel, (b) *T UnmarshalJSON that lower-cases and trims the JSON string then switches to assign the typed constant, defaulting to Unknown, (c) T.Int64() that switches on the typed value returning an integer index for OTel recording with Unknown mapping to -1. (`type BrokerState string
func (s *BrokerState) UnmarshalJSON(data []byte) error { ... switch strings.ToLower(strings.TrimSpace(value)) { case "up": *s = BrokerStateUp ... default: *s = BrokerStateUnknown } }
func (s BrokerState) Int64() int64 { switch s { case BrokerStateUp: return 6 ... default: return -1 } }`)
**Struct fields tagged with librdkafka JSON keys** — All struct fields use json tags matching the exact librdkafka statistics JSON field names (e.g. 'stateage', 'outbuf_cnt', 'consumer_lag_stored'). Go field names are idiomatic PascalCase; the json tag bridges to the snake_case librdkafka names. (`type BrokerStats struct { RequestsAwaitingTransmission int64 `json:"outbuf_cnt"` ... }`)
**WindowStats reused for all rolling-window histogram data** — The WindowStats struct captures min/max/avg/sum/count/stddev/p50/p75/p90/p95/p99/p9999 — it is embedded by value in BrokerStats (Latency, Throttle fields) and TopicStats (BatchSize, BatchCount fields) rather than duplicated. (`type BrokerStats struct { Latency WindowStats `json:"rtt"`; Throttle WindowStats `json:"throttle"` }`)
**Partitions as map[string]Partition not slice** — TopicStats.Partitions is map[string]Partition (string key is the partition ID) to match librdkafka's JSON object shape. Partition.Partition int64 field is the numeric id; negative values indicate the internal UA partition which callers must skip. (`type TopicStats struct { Partitions map[string]Partition `json:"partitions"` }`)
**Test via embedded testdata JSON** — stats_test.go embeds testdata/stats.json via //go:embed and unmarshals it into Stats to verify field mapping is correct. Tests assert on specific field values to catch json-tag regressions. (`//go:embed testdata/stats.json
var statsJSON []byte
func TestStats(t *testing.T) { var stats Stats; json.Unmarshal(statsJSON, &stats); assert.Equal(t, "rdkafka", stats.ClientID) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `stats.go` | Top-level Stats struct (the root of the librdkafka JSON payload) and WindowStats. Stats.Brokers and Stats.Topics are maps keyed by broker name / topic name respectively. | WindowStats.P9999 maps to json tag 'p99_99' (not 'p9999') — this is the librdkafka JSON name. |
| `broker.go` | BrokerStats struct, BrokerSource and BrokerState enums with UnmarshalJSON/Int64, and BrokerTopicPartition. Latency and Throttle are WindowStats embedded by value. | BrokerStats.TopicPartitions is map[string]BrokerTopicPartition; the key is '<topic>-<partition>' in librdkafka output. |
| `topic.go` | TopicStats and Partition structs. Partition has both consumer-side (CommittedOffset, ConsumerLag) and producer-side (MessagesSent) fields in the same struct. | Partition.StoredOffset json tag is 'stored_offset'; librdkafka uses -1001 as the sentinel 'no stored offset' value — tests assert this. |
| `consumergroup.go` | ConsumerGroupStats, ConsumerGroupState, and ConsumerGroupJoinState enums. Both have full UnmarshalJSON/Int64 implementations. | ConsumerGroupJoinState has 9 variants (0-8) vs ConsumerGroupState's 7 (0-6); Unknown maps to -1 in both. |
| `stats_test.go` | Integration test that unmarshals testdata/stats.json and validates key field values. Adding a new field to a stats struct should include an assertion here. | Test file lives in package stats (white-box). testdata/stats.json is the real stats fixture; it must stay in sync with the librdkafka version pinned in go.mod. |

## Anti-Patterns

- Adding Int64() methods that return 0 for unknown variants instead of -1 — the metrics layer uses -1 as the sentinel for 'unknown' to distinguish from valid 0 values
- Embedding WindowStats by pointer — it is always present in the JSON and must be embedded by value to avoid nil dereference in the internal metrics layer
- Changing json tags without updating testdata/stats.json — the test will silently pass with zero values if the tag is wrong
- Treating Partition.Partition < 0 as an error — negative partition IDs are valid internal librdkafka sentinels that callers must skip, not unmarshal errors

## Decisions

- **String-typed enums with custom UnmarshalJSON rather than iota int enums** — librdkafka serializes enum fields as human-readable strings ('up', 'init', 'auth-handshake'); typed string constants preserve debuggability while Int64() provides the numeric form needed by OTel gauges.
- **Unknown enum sentinel returns -1 from Int64()** — The internal metrics layer records these values as Int64Gauge; -1 cleanly signals 'unrecognised state' to dashboards without colliding with any valid state index (which start at 0).
- **Separate stats and internal packages** — stats holds pure data types (JSON unmarshalling only); internal holds OTel instrument registration and recording. Keeping them separate means the data model can be tested independently and the OTel dependency does not leak into callers who only need to inspect stats values.

## Example: Add a new string enum field to BrokerStats

```
// In stats/broker.go:
const (
	MyEnumUnknown MyEnum = "unknown"
	MyEnumFoo     MyEnum = "foo"
)
type MyEnum string
func (s *MyEnum) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("failed to unmarshal value: %s", data)
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "foo":
		*s = MyEnumFoo
	default:
// ...
```

<!-- archie:ai-end -->
