# stats

<!-- archie:ai-start -->

> Typed Go representations of librdkafka's JSON statistics payload (https://github.com/confluentinc/librdkafka/blob/v2.4.0/STATISTICS.md), providing JSON unmarshalling only — no OTel dependency. String-enum fields implement custom UnmarshalJSON + Int64() so the internal metrics layer can record them as numeric gauges without string parsing at record time.

## Patterns

**String enum type with UnmarshalJSON + Int64()** — Each librdkafka string-enum field is a named string type with: (a) package-level const block of all variants plus an Unknown sentinel, (b) *T UnmarshalJSON that lower-cases+trims the JSON string and switches to assign the typed constant defaulting to Unknown, (c) T.Int64() that returns a numeric index with Unknown mapping to -1. (`type BrokerState string
func (s *BrokerState) UnmarshalJSON(data []byte) error { var value string; json.Unmarshal(data, &value); switch strings.ToLower(strings.TrimSpace(value)) { case "up": *s = BrokerStateUp; default: *s = BrokerStateUnknown }; return nil }
func (s BrokerState) Int64() int64 { switch s { case BrokerStateUp: return 6; default: return -1 } }`)
**Struct fields tagged with exact librdkafka JSON keys** — All struct fields use json tags matching exact librdkafka statistics JSON field names (snake_case as documented in STATISTICS.md). Go field names are idiomatic PascalCase; the json tag bridges to the librdkafka name. Deviation silently produces zero values. (`type BrokerStats struct { RequestsAwaitingTransmission int64 `json:"outbuf_cnt"`; StateAge int64 `json:"stateage"` }`)
**WindowStats embedded by value for rolling-window histogram data** — WindowStats (min/max/avg/sum/count/stddev/p50/p75/p90/p95/p99/p9999) is reused across BrokerStats (Latency='rtt', Throttle='throttle') and TopicStats (BatchSize='batch_size', BatchCount='batch_count') by value embedding, not pointer. P9999 maps to json tag 'p99_99' not 'p9999'. (`type BrokerStats struct { Latency WindowStats `json:"rtt"`; Throttle WindowStats `json:"throttle"` }
type WindowStats struct { P9999 int64 `json:"p99_99"` } // NOTE: tag is p99_99 not p9999`)
**Partitions as map[string]Partition with numeric Partition field** — TopicStats.Partitions is map[string]Partition (string key is the string partition ID from librdkafka JSON). The Partition struct has a Partition int64 field for the numeric id. Negative values (e.g. -1) indicate the internal UA/UnAssigned partition; callers in the internal package must skip them. (`type TopicStats struct { Partitions map[string]Partition `json:"partitions"` }
type Partition struct { Partition int64 `json:"partition"` } // negative = internal UA partition`)
**Test via embedded testdata/stats.json fixture** — stats_test.go embeds testdata/stats.json via //go:embed and unmarshals into Stats to verify field mapping. Tests assert on specific known field values (ClientID, MessageCount, broker RequestsSent, partition StoredOffset=-1001). Adding a new struct field should include a corresponding assertion. (`//go:embed testdata/stats.json
var statsJSON []byte
func TestStats(t *testing.T) { var stats Stats; json.Unmarshal(statsJSON, &stats); assert.Equal(t, "rdkafka", stats.ClientID) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `stats.go` | Top-level Stats struct (root of librdkafka JSON payload) and shared WindowStats. Stats.Brokers and Stats.Topics are maps[string] keyed by broker name and topic name respectively. ConsumerGroup is embedded by value. | WindowStats.P9999 uses json tag 'p99_99' (matching librdkafka's field name), not 'p9999' — a common copy-paste mistake from field naming. |
| `broker.go` | BrokerStats struct (all broker-level metrics), BrokerSource and BrokerState string enums with full UnmarshalJSON/Int64 implementations, and BrokerTopicPartition. Latency='rtt' and Throttle='throttle' are WindowStats embedded by value. | BrokerStats.TopicPartitions is map[string]BrokerTopicPartition where the key is '<topic>-<partition>' in librdkafka output. BrokerStats.Wakeups field exists in the struct but is not recorded by the internal metrics layer — intentional omission. |
| `topic.go` | TopicStats struct and Partition struct. Partition contains both consumer-side fields (CommittedOffset, ConsumerLag) and producer-side fields (MessagesSent) in the same struct — librdkafka outputs a unified partition object. | StoredOffset sentinel value -1001 indicates 'no stored offset' in librdkafka; the test asserts this explicitly. ConsumerLag = -1 when there is no committed offset. These are not error conditions. |
| `consumergroup.go` | ConsumerGroupStats struct, ConsumerGroupState (7 states, 0-6) and ConsumerGroupJoinState (9 states, 0-8) enums, both with UnmarshalJSON/Int64. Unknown sentinel returns -1 in both. | ConsumerGroupJoinState has 9 variants (indices 0-8) versus ConsumerGroupState's 7 (0-6) — do not assume they share the same cardinality when adding new variants. |
| `stats_test.go` | White-box integration test (package stats) that unmarshals testdata/stats.json and validates key field values including StoredOffset=-1001 and ConsumerLag=-1 sentinels. Must be updated when adding new fields to stats structs. | testdata/stats.json must stay in sync with the librdkafka version pinned in go.mod. A wrong json tag causes silent zero values in assertions — always verify new fields produce non-zero test assertions. |

## Anti-Patterns

- Int64() returning 0 for Unknown variants instead of -1 — the internal metrics layer uses -1 as the sentinel for 'unknown' to distinguish from valid index 0
- Embedding WindowStats by pointer — it is always present in the librdkafka JSON and must be embedded by value to avoid nil dereference in the internal metrics layer
- Changing json tags without updating testdata/stats.json assertions — the test silently passes with zero values when a tag is wrong
- Treating Partition.Partition < 0 as an unmarshal error — negative partition IDs are valid internal librdkafka sentinels that callers must skip
- Adding OTel imports or metric registration to this package — stats is a pure data/unmarshalling package; OTel instrumentation belongs exclusively in pkg/kafka/metrics/internal

## Decisions

- **String-typed enums with custom UnmarshalJSON rather than iota int enums** — librdkafka serializes enum fields as human-readable strings ('up', 'auth-handshake'); typed string constants preserve debuggability in logs and error messages while Int64() provides the numeric form needed by OTel gauges without re-parsing at record time.
- **Unknown enum sentinel returns -1 from Int64()** — OTel Int64Gauge records these as numeric values; -1 cleanly signals 'unrecognized state' in dashboards without colliding with any valid state index (which start at 0), enabling alert rules to distinguish missing data from known states.
- **Separate stats and internal packages with no cross-dependency from stats to internal** — stats holds pure data types with only stdlib dependencies (encoding/json, strings, fmt); internal holds OTel instrument registration. Keeping them separate lets the data model be tested and inspected independently and prevents OTel from leaking into callers who only need to inspect stats field values.

## Example: Add a new string enum field to a stats struct

```
// In stats/broker.go (or appropriate file):
const (
	MyEnumUnknown MyEnum = "unknown"
	MyEnumFoo     MyEnum = "foo"
	MyEnumBar     MyEnum = "bar"
)

type MyEnum string

func (s *MyEnum) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("failed to unmarshal value: %s", data)
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
// ...
```

<!-- archie:ai-end -->
