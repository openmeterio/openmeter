# stats

<!-- archie:ai-start -->

> Typed Go representations of librdkafka's JSON statistics payload, providing JSON unmarshalling only — no OTel dependency. String-enum fields implement custom UnmarshalJSON + Int64() so the internal metrics layer records them as numeric gauges without parsing at record time.

## Patterns

**String enum type with UnmarshalJSON + Int64()** — Each librdkafka string-enum field is a named string type with: a const block of variants plus an Unknown sentinel, *T.UnmarshalJSON that lower-cases/trims and switches (defaulting to Unknown), and T.Int64() returning a numeric index with Unknown mapped to -1. (`func (s BrokerState) Int64() int64 { switch s { case BrokerStateUp: return 6; default: return -1 } }`)
**Struct fields tagged with exact librdkafka JSON keys** — Fields use json tags matching exact librdkafka snake_case names; Go names are idiomatic PascalCase. A wrong tag silently yields zero values. (`type BrokerStats struct { RequestsAwaitingTransmission int64 `json:"outbuf_cnt"`; StateAge int64 `json:"stateage"` }`)
**WindowStats embedded by value** — WindowStats (min/max/avg/sum/count/stddev/p50..p9999) is reused by value across BrokerStats (rtt, throttle) and TopicStats (batch_size, batch_count). P9999 maps to json tag 'p99_99'. (`type WindowStats struct { P9999 int64 `json:"p99_99"` } // tag is p99_99 not p9999`)
**Partitions as map[string]Partition with numeric Partition field** — TopicStats.Partitions is map[string]Partition (string partition ID key); Partition has a numeric Partition int64. Negative values mark the internal UA partition that internal-package callers skip. (`type Partition struct { Partition int64 `json:"partition"` } // negative = internal UA partition`)
**Test via embedded testdata/stats.json fixture** — stats_test.go //go:embed testdata/stats.json, unmarshals into Stats, and asserts known field values (ClientID, MessageCount, StoredOffset=-1001). New fields should add assertions. (`//go:embed testdata/stats.json
var statsJSON []byte
func TestStats(t *testing.T) { var s Stats; json.Unmarshal(statsJSON, &s); assert.Equal(t, "rdkafka", s.ClientID) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `stats.go` | Root Stats struct and shared WindowStats; Stats.Brokers/Topics are maps keyed by broker/topic name; ConsumerGroup embedded by value. | WindowStats.P9999 uses json tag 'p99_99' (librdkafka's name), not 'p9999' — common copy-paste mistake. |
| `broker.go` | BrokerStats plus BrokerSource and BrokerState string enums (full UnmarshalJSON/Int64) and BrokerTopicPartition; Latency='rtt', Throttle='throttle' WindowStats by value. | TopicPartitions keys are '<topic>-<partition>'. BrokerStats.Wakeups exists but is intentionally not recorded by the internal layer. |
| `topic.go` | TopicStats and Partition; Partition holds both consumer-side (CommittedOffset, ConsumerLag) and producer-side (MessagesSent) fields in one unified object. | StoredOffset=-1001 means 'no stored offset' and ConsumerLag=-1 means 'no committed offset' — these are valid sentinels, not errors. |
| `consumergroup.go` | ConsumerGroupStats; ConsumerGroupState (7 states, 0-6) and ConsumerGroupJoinState (9 states, 0-8) enums with UnmarshalJSON/Int64; Unknown returns -1. | JoinState has 9 variants vs State's 7 — do not assume shared cardinality when adding variants. |
| `stats_test.go` | White-box test unmarshalling testdata/stats.json, asserting key values including StoredOffset=-1001 and ConsumerLag=-1 sentinels. | Keep testdata/stats.json in sync with the librdkafka version in go.mod; a wrong tag silently passes with zero values — verify new fields produce non-zero assertions. |

## Anti-Patterns

- Int64() returning 0 for Unknown instead of -1 — the internal layer uses -1 to distinguish unknown from valid index 0.
- Embedding WindowStats by pointer — it is always present; value embedding avoids nil dereference in the internal layer.
- Changing json tags without updating testdata/stats.json assertions — the test silently passes with zero values.
- Treating Partition.Partition < 0 as an unmarshal error — negative IDs are valid internal librdkafka sentinels callers skip.
- Adding OTel imports or metric registration here — stats is a pure data/unmarshalling package; instrumentation lives in pkg/kafka/metrics/internal.

## Decisions

- **String-typed enums with custom UnmarshalJSON rather than iota int enums.** — librdkafka serializes enums as human-readable strings; typed string constants preserve debuggability while Int64() provides the numeric form for OTel without re-parsing.
- **Unknown enum sentinel returns -1 from Int64().** — -1 cleanly signals 'unrecognized state' in dashboards without colliding with valid indices (which start at 0), letting alerts distinguish missing data from known states.
- **Separate stats and internal packages with no stats→internal dependency.** — stats holds pure stdlib data types; keeping OTel out lets the data model be tested/inspected independently and prevents OTel leaking into callers who only inspect field values.

## Example: Add a new string enum field to a stats struct

```
const (
    MyEnumUnknown MyEnum = "unknown"
    MyEnumFoo     MyEnum = "foo"
)
type MyEnum string
func (s *MyEnum) UnmarshalJSON(data []byte) error {
    var value string
    if err := json.Unmarshal(data, &value); err != nil { return fmt.Errorf("failed to unmarshal value: %s", data) }
    switch strings.ToLower(strings.TrimSpace(value)) {
    case "foo": *s = MyEnumFoo
    default: *s = MyEnumUnknown
    }
    return nil
}
func (s MyEnum) Int64() int64 { switch s { case MyEnumFoo: return 0; default: return -1 } }
```

<!-- archie:ai-end -->
