# Enforcement: data-modeling (8 rules)

Topic file. Loaded on demand when an agent works on something in the `data-modeling` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Pitfalls (block)

### `pf-010-fkless-ledger-clickhouse-drift` — Cross-aggregate references in the ledger (LedgerCustomerAccount.account_id/customer_id) are deliberately FK-less and ClickHouse tables are migration-less, so referential integrity and column/struct alignment must be guarded by application-level checks, never assumed from the database.

*source: `deep_scan`*

**Why:** Pitfall pf_0010: ledger_customer_account.go:42 Edges() returns nil and account_id/customer_id are field.String(...).Immutable() with no FK to LedgerAccount or Customer (intentional, to avoid import cycles). ClickHouse is not under Atlas/golang-migrate (connector.go:78 createTable only when !SkipCreateTables, event_query.go:25 uses IfNotExists), so a column change on an already-provisioned deployment leaves existing tables unaltered and silently drifts from the Go struct.

**Example:**

```
// Never rely on a DB FK here — validate in application code:
if _, err := accountResolver.GetByID(ctx, lca.AccountID); err != nil {
    return fmt.Errorf("dangling ledger customer account: %w", err)
}
```

**Path glob:** `openmeter/ent/schema/ledger_customer_account.go`, `openmeter/ent/schema/ledger_account.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "func.*Edges\\(\\).*\\[\\]ent\\.Edge"
    ],
    "must_not_match": [
      "edge\\.To|edge\\.From"
    ]
  }
]
```

</details>

## Pattern Divergence (inform)

### `data-charges-searchview-sync` — When adding a column to chargemeta.Mixin, also extend chargesSearchV1Columns in charges.go, and regenerate the ChargesSearchV1 ent.View DDL via make generate-view-sql plus an explicit SQL migration — Atlas does not diff views.

*source: `deep_scan`*

**Why:** The Charge data model lifecycle states: 'When adding a column to chargemeta.Mixin, also extend chargesSearchV1Columns in charges.go or the ChargesSearchV1 union view breaks' and 'ChargesSearchV1 is an ent.View; Atlas does not diff views, so view DDL changes require make generate-view-sql + an explicit SQL migration (see AGENTS.md ent.View caveat).' Forgetting the view column or the view migration leaves the search view stale or undeployed in production.

**Example:**

```
// After adding a column to chargemeta.Mixin:
// 1. add it to chargesSearchV1Columns in openmeter/ent/schema/charges.go
// 2. make generate-view-sql
// 3. atlas migrate --env local diff add_charges_search_col (or hand-add the view SQL migration)
```

**Path glob:** `openmeter/ent/schema/charges.go`, `openmeter/ent/schema/chargemeta.go`, `openmeter/ent/schema/charges*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "chargesSearchV1Columns|ChargesSearchV1"
    ]
  }
]
```

</details>

### `data-feature-archived-not-deleted` — Features are archived via archived_at, not soft-deleted via deleted_at. Use feature.FeatureConnector.ArchiveFeature and filter on archived_at — never set or query deleted_at for features.

*source: `deep_scan`*

**Why:** The Feature data model lifecycle states: 'features are archived (archived_at), not soft-deleted via deleted_at.' The Feature entity diverges from the standard TimeMixin soft-delete convention: archival preserves history for billing references while removing the feature from active catalogs. Treating archived_at as deleted_at (or vice versa) breaks plan/rate-card references that still point at archived features.

**Example:**

```
if err := featureConnector.ArchiveFeature(ctx, feature.ArchiveFeatureInput{ID: id}); err != nil { return err }
```

**Path glob:** `openmeter/productcatalog/feature/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "Feature.*\\.SetDeletedAt|feature.*deleted_at"
    ]
  }
]
```

</details>

### `data-subscriptionitem-snapshot-mirror` — SubscriptionItem deliberately duplicates RateCard fields for snapshot immutability; when RateCard fields change, mirror the change into SubscriptionItem manually — do not collapse the duplication into a live reference.

*source: `deep_scan`*

**Why:** The Subscription data model lifecycle states: 'SubscriptionItem deliberately duplicates RateCard fields for snapshot immutability, so RateCard changes must be mirrored manually.' The snapshot is intentional: a subscription must bill against the rate card as it existed at creation, not the current plan. Replacing the snapshot with a live RateCard FK would retroactively change historical billing.

**Example:**

```
field.Enum("settlement_mode").GoType(productcatalog.SettlementMode("")) // mirrored from RateCard for snapshot immutability
```

**Path glob:** `openmeter/ent/schema/subscription.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "SubscriptionItem"
    ]
  }
]
```

</details>

### `data-notification-config-scanner-typeswitch` — A new notification ChannelType or EventType requires updating both ChannelConfigValueScanner and RuleConfigValueScanner type-switch serializers in openmeter/ent/schema/notification.go; never add a polymorphic config variant without extending both scanners.

*source: `deep_scan`*

**Why:** The NotificationEvent data model lifecycle states: 'Channel/Rule config polymorphism uses ChannelConfigValueScanner/RuleConfigValueScanner type-switch serializers in the same file — a new ChannelType/EventType requires updating both.' These scanners serialize the polymorphic config column; a missing case silently drops or fails to decode the new variant's config.

**Example:**

```
// In notification.go, both scanners must handle the new type:
case ChannelTypeWebhookV2:
    return json.Marshal(cfg.WebhookV2)
```

**Path glob:** `openmeter/ent/schema/notification.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "ChannelType|EventType|ChannelConfigValueScanner|RuleConfigValueScanner"
    ]
  }
]
```

</details>

### `data-balancesnapshot-no-idmixin` — BalanceSnapshot intentionally omits IDMixin (no surrogate primary key); do not add IDMixin to it expecting the standard mixin triad, and read it only through credit.CreditConnector balance APIs.

*source: `deep_scan`*

**Why:** The BalanceSnapshot data model lifecycle states: 'Note this entity omits IDMixin (no surrogate PK).' This is a deliberate exception to the otherwise-universal IDMixin+NamespaceMixin+TimeMixin convention; the snapshot is keyed by its natural (owner, time) tuple rather than a ULID. Adding IDMixin would change the primary key and break the credit engine's snapshot lookups.

**Example:**

```
// openmeter/ent/schema/balance_snapshot.go intentionally has NO entutils.IDMixin{} in Mixin()
```

**Path glob:** `openmeter/ent/schema/balance_snapshot.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "IDMixin"
    ]
  }
]
```

</details>

### `data-customersubjects-fkless` — Validate CustomerSubjects subject_key links in application code, not via a DB foreign key

*source: `deep_scan`*

**Why:** CustomerSubjects is a link table mapping a customer to one or more subject keys; the FK constraint to Subject.subject_key is intentionally absent because Ent cannot enforce FKs on non-ID fields (see openmeter/ent/schema/customer.go:147). Referential integrity between a customer's subject keys and the Subject table is therefore guarded only by application code, never assumed from the schema.

**Path glob:** `openmeter/ent/schema/customer.go`

### `data-ledgertransaction-balanced` — Create LedgerTransaction rows only via transactions.ResolveTransactions so entries stay balanced

*source: `deep_scan`*

**Why:** A LedgerTransaction groups balanced double-entry ledger entries within a transaction group and records when it was booked (see openmeter/ent/schema/ledger_transaction.go:26). The debit=credit invariant is enforced only by constructing inputs through transactions.ResolveTransactions with typed templates; hand-building LedgerTransaction or LedgerEntry rows can persist an unbalanced transaction that no schema constraint rejects.

**Example:**

```
entries, err := transactions.ResolveTransactions(ctx, /* typed templates */)
if err != nil { return err }
return ledger.CommitGroup(ctx, entries)
```

**Path glob:** `openmeter/ledger/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "LedgerTransaction\\{|\\.LedgerTransaction\\.Create\\("
    ],
    "must_not_match": [
      "ResolveTransactions"
    ]
  }
]
```

</details>
