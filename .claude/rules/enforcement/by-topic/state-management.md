# Enforcement: state-management (1 rule)

Topic file. Loaded on demand when an agent works on something in the `state-management` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Pattern Divergence (inform)

### `balance-001` — The balance-worker high-watermark filter must be used to skip ClickHouse queries for entitlement IDs that have already been recalculated at or after the event timestamp. Never bypass the high-watermark check for performance optimization.

*source: `deep_scan`*

**Why:** The openmeter/entitlement/balanceworker component description states: 'Uses LRU caches and high-watermark filter to avoid redundant ClickHouse queries.' The high-watermark filter in the balance-worker is the primary mechanism preventing excessive ClickHouse load under burst conditions. Bypassing it converts every ingest event into a ClickHouse query, which at high throughput exhausts ClickHouse connections.

**Example:**

```
// Correct: check high-watermark before querying ClickHouse
if w.highWatermark.HasPassedFor(entitlementID, eventTime) {
    return nil // already recalculated at or after this time
}
// ... proceed with ClickHouse query ...
```

**Path glob:** `openmeter/entitlement/balanceworker/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "QueryMeter|GetBalanceAt|handleEntitlementEvent"
    ],
    "must_not_match": [
      "highWatermark|HighWatermark|watermark"
    ]
  }
]
```

</details>
