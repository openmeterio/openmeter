# Enforcement: billing (2 rules)

Topic file. Loaded on demand when an agent works on something in the `billing` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Pattern Divergence (inform)

### `credit-001` — All effective times for credit grants and balance snapshots must be truncated to Granularity (time.Minute) before storage or computation. Never pass sub-minute timestamps into the credit engine.

*source: `deep_scan`*

**Why:** The openmeter/credit component description states: 'Granularity truncation (time.Minute) applied to all effective times.' The credit engine's burn-down arithmetic assumes all effective times are at minute boundaries. Sub-minute timestamps cause the engine to produce incorrect balance snapshots and period calculations that do not align with ClickHouse meter aggregations.

**Example:**

```
// Correct: truncate before passing to credit engine
effectiveAt := time.Now().Truncate(time.Minute)
return creditConnector.CreateGrant(ctx, credit.CreateGrantInput{EffectiveAt: effectiveAt})

// Wrong: passing sub-minute time
return creditConnector.CreateGrant(ctx, credit.CreateGrantInput{EffectiveAt: time.Now()})
```

**Path glob:** `openmeter/credit/**/*.go`, `openmeter/entitlement/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "CreateGrant|EffectiveAt"
    ],
    "must_not_match": [
      "Truncate\\(time\\.Minute\\)",
      "\\.Truncate\\("
    ]
  }
]
```

</details>

### `billing-010-split-line-group` — SplitLineGroup operations must go through billing.Service.SplitLineGroupService methods. Never manipulate SplitLineGroup fields directly on InvoiceLine structs — the group linkage must remain consistent with the billing adapter's diff computation.

*source: `deep_scan`*

**Why:** The openmeter/billing component description lists SplitLineGroupService as a composite sub-interface of billing.Service. The billing adapter's stdinvoicelinediff.go computes line hierarchies using SplitLineGroup linkage. Direct field manipulation on line structs bypasses the diff computation and can produce orphaned or duplicate line groups in invoice advancement.

**Example:**

```
// Correct: use billing.Service.SplitLineGroupService methods to manage line groups
// Wrong: setting line.SplitLineGroupID = someID directly without going through the service
```

**Path glob:** `openmeter/billing/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "SplitLineGroupID\\s*="
    ],
    "must_not_match": [
      "SplitLineGroupService",
      "// adapter"
    ]
  }
]
```

</details>
