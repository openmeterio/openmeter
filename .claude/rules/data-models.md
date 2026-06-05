OpenMeter persists its control-plane state in PostgreSQL via the Ent ORM (~70 tables across customer, entitlement/credit, productcatalog, subscription, billing, charges, ledger, notification, and app domains; source of truth is openmeter/ent/schema/*.go, Atlas migrations under tools/migrate/migrations/). Raw usage events live in ClickHouse (per-namespace om_<ns>_events MergeTree table) and flow there through per-namespace Kafka ingest topics consumed by the sink-worker; Redis backs event deduplication and async query progress tracking. Almost every Postgres table is multi-tenant (namespace-scoped), uses ULID char(26) ids, and soft-deletes via deleted_at.

## Models

_Domain entities, DTOs, and value objects this codebase reads and writes._

### `RawEvent` *(document)*

The CloudEvents-shaped usage event row stored in the ClickHouse om_<ns>_events table; data is the raw JSON payload, store_row_id a generated ULID per stored copy and customer_id an optional attribution (see openmeter/streaming/connector.go:24).

- **Location:** `openmeter/streaming/connector.go`
- **Store:** `clickhouse_events`
- **Owner:** `openmeter/streaming`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `namespace` | String |  |
| `id` | String |  |
| `type` | LowCardinality(String) |  |
| `subject` | String |  |
| `source` | String |  |
| `time` | DateTime |  |
| `data` | String | Raw JSON event payload from which meters extract values via value_property JSONPath (see openmeter/streaming/connector.go:31). |
| `ingested_at` | DateTime |  |
| `stored_at` | DateTime | When the row landed in ClickHouse; used as the late-arriving-usage cutoff in usage-based billing realization runs (see openmeter/sink/storage.go:62). |
| `store_row_id` | String | Per-stored-row ULID assigned by the sink on insert; distinguishes physical copies of the same logical event id (see openmeter/sink/storage.go:63). |
| `customer_id` | String |  |

**Data Guarantees:**
- ENGINE = MergeTree PARTITION BY toYYYYMM(time) (see openmeter/streaming/clickhouse/event_query.go:38)
- ORDER BY (namespace, type, subject, toStartOfHour(time)) (see openmeter/streaming/clickhouse/event_query.go:45)
- minmax skip-index on stored_at
- no soft-delete; append-only; logical dedup happens upstream in Redis before insert

**Consumers:**
- `ClickHouseStorage.BatchInsert` — `openmeter/sink/storage.go`: sole writer — maps SinkMessages to RawEvents and batch-inserts them into ClickHouse, stamping stored_at/store_row_id
- `createEventsTable.toSQL` — `openmeter/streaming/clickhouse/event_query.go`: emits the CREATE TABLE DDL for the per-namespace events table
- `queryEventsTable` — `openmeter/streaming/clickhouse/event_query.go`: builds list/count queries filtering by namespace/time/subject

**Lifecycle:**
- *How to add:* Add a column to RawEvent in openmeter/streaming/connector.go (with ch:"..." tag) AND to the createEventsTable.Define(...) calls in openmeter/streaming/clickhouse/event_query.go; the table is created idempotently (IF NOT EXISTS) on connector startup. There is no Atlas migration for ClickHouse.

  ```
  sb.Define("store_row_id", "String")
  ```

- *How to modify:* ClickHouse columns are added via the IF NOT EXISTS create path or ALTER TABLE; changing ORDER BY/partitioning is a breaking schema change requiring a new table and backfill.
- *How to read:* Always through the streaming.Connector (ListEvents/ListEventsV2/CountEvents/QueryMeter), never raw ClickHouse from handlers.

  ```
  events, err := connector.ListEvents(ctx, namespace, params)
  ```

- *Tests:* `openmeter/streaming/clickhouse/event_query_test.go`, `openmeter/streaming/clickhouse/event_query_v2_test.go`, `openmeter/streaming/clickhouse/meter_query_test.go`

### `dedupe.Item` *(key_value)*

The deduplication key (namespace + event id + source) hashed into a Redis key with TTL to drop duplicate ingest events (see openmeter/dedupe/redisdedupe/redisdedupe.go:45).

- **Location:** `openmeter/dedupe/dedupe.go`
- **Store:** `redis`
- **Owner:** `openmeter/dedupe`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `Namespace` | string |  |
| `ID` | string |  |
| `Source` | string |  |

**Data Guarantees:**
- TTL = Deduplicator.Expiration (see openmeter/dedupe/redisdedupe/redisdedupe.go:84)
- SET NX semantics — first writer wins, returns unique=true (see openmeter/dedupe/redisdedupe/redisdedupe.go:82)
- key may be raw key or sha-hashed depending on DedupeMode (rawkey/keyhash/keyhash-migration) (see openmeter/dedupe/redisdedupe/redisdedupe.go:51)

**Consumers:**
- `Deduplicator.IsUnique` — `openmeter/dedupe/redisdedupe/redisdedupe.go`: checks-and-sets the dedup key atomically (SET NX); returns whether the event is new
- `Deduplicator.CheckUniqueBatch` — `openmeter/dedupe/redisdedupe/redisdedupe.go`: MGET batch read path classifying items into unique vs already-processed

**Lifecycle:**
- *How to add:* Dedup keys are produced from dedupe.Item.Key(); the Redis value is empty — only key presence matters. Change the key shape in dedupe.Item / keyhash.go.

  ```
  d.Redis.SetArgs(ctx, key, "", redis.SetArgs{TTL: d.Expiration, Mode: "nx"})
  ```

- *How to modify:* Changing the hashing format requires a migration mode (DedupeModeKeyHashMigration) that checks both old and new key formats during transition.
- *How to read:* Via Deduplicator.IsUnique/CheckUnique/CheckUniqueBatch; never touch Redis keys directly.

### `Progress` *(key_value)*

Async-query progress state cached in Redis under key progress:<namespace>:<id> with a TTL, used to report long-running event-query progress (see openmeter/progressmanager/adapter/progress.go:16).

- **Location:** `openmeter/progressmanager/entity/progressmanager.go`
- **Store:** `redis`
- **Owner:** `openmeter/progressmanager`
**Data Guarantees:**
- key = progress:<namespace>:<id> (optionally prefixed) (see openmeter/progressmanager/adapter/progress.go:65)
- JSON-serialized value with TTL = adapter.expiration (see openmeter/progressmanager/adapter/progress.go:56)
- missing key → NotFound (redis.Nil) (see openmeter/progressmanager/adapter/progress.go:29)

**Consumers:**
- `adapter.UpsertProgress` — `openmeter/progressmanager/adapter/progress.go`: writes JSON progress with TTL (SET)
- `adapter.GetProgress` — `openmeter/progressmanager/adapter/progress.go`: reads/unmarshals progress; returns NotFound on missing key

**Lifecycle:**
- *How to add:* Edit the Progress entity in openmeter/progressmanager/entity/progressmanager.go; it is JSON-marshaled wholesale into the Redis value.

  ```
  a.redis.Set(ctx, a.getKey(input.ProgressID), data, a.expiration)
  ```

- *How to read:* Via progressmanager service GetProgress; a noop adapter exists when Redis is not configured.

### `SinkMessage` *(value_object)*

In-flight per-event message inside the sink-worker carrying the deserialized CloudEvent plus namespace and ingested/stored timestamps before it is mapped to a RawEvent and persisted (see openmeter/sink/storage.go:52).

- **Location:** `openmeter/sink/models/models.go`
- **Owner:** `openmeter/sink`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `Namespace` | string |  |
| `Serialized` | *serializer.CloudEventsKafkaPayload |  |
| `IngestedAt` | *time.Time |  |
| `StoredAt` | *time.Time |  |

**Data Guarantees:**
- transient; not persisted as-is — mapped to RawEvent for ClickHouse insert (see openmeter/sink/storage.go:53)

**Consumers:**
- `ClickHouseStorage.BatchInsert` — `openmeter/sink/storage.go`: consumes SinkMessages and maps them to ClickHouse RawEvents

**Lifecycle:**
- *How to add:* Edit the SinkMessage struct in openmeter/sink/models/models.go.

### `BillingInvoice` *(table)*

An invoice in gathering or standard state, snapshotting customer/supplier identity, addresses, cloned profile settings and lifecycle timestamps; status_details_cache is GIN-indexed and collection_at has dual gathering/standard semantics (see openmeter/ent/schema/billing.go:1018).

- **Location:** `openmeter/ent/schema/billing.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `metadata` | jsonb |  |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ |  |
| `supplier_name` | TEXT |  |
| `supplier_tax_code` | TEXT |  |
| `customer_key` | TEXT |  |
| `customer_name` | TEXT |  |
| `customer_usage_attribution` | jsonb |  |
| `number` | TEXT |  |
| `type` | enum |  |
| `description` | TEXT |  |
| `customer_id` | char(26) |  |
| `source_billing_profile_id` | char(26) |  |
| `voided_at` | TIMESTAMPTZ |  |
| `issued_at` | TIMESTAMPTZ | Can be in the future for pre-issued invoices (see openmeter/ent/schema/billing.go:1087). |
| `sent_to_customer_at` | TIMESTAMPTZ |  |
| `draft_until` | TIMESTAMPTZ |  |
| `quantity_snapshoted_at` | TIMESTAMPTZ |  |
| `currency` | varchar(3) |  |
| `due_at` | TIMESTAMPTZ |  |
| `status` | enum |  |
| `status_details_cache` | jsonb |  |
| `workflow_config_id` | char(26) |  |
| `tax_app_id` | char(26) |  |
| `invoicing_app_id` | char(26) |  |
| `payment_app_id` | char(26) |  |
| `period_start` | TIMESTAMPTZ |  |
| `period_end` | TIMESTAMPTZ |  |
| `collection_at` | TIMESTAMPTZ | Dual purpose: on gathering invoices defines when pending lines collect into a draft; on standard invoices the post-creation collection/snapshot cutoff for metered lines (see openmeter/ent/schema/billing.go:1154). |
| `payment_processing_entered_at` | TIMESTAMPTZ | When the invoice first entered payment-processing; used for staleness/fraud guarding (see openmeter/ent/schema/billing.go:1164). |
| `schema_level` | INT | Write schema version used during the in-progress invoice-line migration (see openmeter/ent/schema/billing.go:1170). |
| `supplier_address_*` | TEXT |  |
| `customer_address_*` | TEXT |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, customer_id, currency) WHERE deleted_at IS NULL AND status = 'gathering' — one gathering invoice per customer+currency (see openmeter/ent/schema/billing.go:1193)
- GIN INDEX(status_details_cache)
- INDEX(namespace, status), INDEX(namespace, issued_at)
- FK source_billing_profile_id immutable (profile change forces void)

**Lifecycle:**
- *How to add:* Edit BillingInvoice.Fields() in billing.go and regenerate. Invoice has supplier AND customer address mixins (two CustomerAddressMixin prefixes).
- *How to modify:* See the /billing skill for the invoice state machine.

### `BillingInvoiceLine` *(table)*

The primary invoice line table (flat-fee or usage-based) carrying service/billing period, managed_by, parent_line hierarchy, subscription+charge linkage and totals; intended to eventually absorb the ubp line config (see openmeter/ent/schema/billing.go:303).

- **Location:** `openmeter/ent/schema/billing.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `name/description/metadata/annotations` | TEXT/jsonb |  |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ |  |
| `currency` | varchar(3) |  |
| `tax_config` | jsonb |  |
| `tax_code_id` | char(26) |  |
| `tax_behavior` | enum |  |
| `period_start` | TIMESTAMPTZ |  |
| `period_end` | TIMESTAMPTZ |  |
| `invoice_id` | char(26) |  |
| `managed_by` | enum |  |
| `parent_line_id` | char(26) |  |
| `invoice_at` | TIMESTAMPTZ |  |
| `override_collection_period_end` | TIMESTAMPTZ |  |
| `type` | enum |  |
| `status` | enum |  |
| `quantity` | numeric | Optional; for usage-based billing it is only persisted when the invoice is issued (see openmeter/ent/schema/billing.go:352). |
| `ratecard_discounts` | jsonb |  |
| `child_unique_reference_id` | TEXT | Stable key per parent line for upserting/identifying lines created for the same reason (e.g. tiered price tier) across invoices (see openmeter/ent/schema/billing.go:370). |
| `subscription_id/subscription_phase_id/subscription_item_id` | TEXT |  |
| `subscription_billing_period_from/to` | TIMESTAMPTZ |  |
| `split_line_group_id` | char(26) |  |
| `charge_id` | char(26) |  |
| `engine` | enum |  |
| `line_ids` | char(26) | Deprecated: invoice discounts moved to line_discounts (see openmeter/ent/schema/billing.go:416). |
| `credits_applied` | jsonb |  |
| `amount/totals fields` | numeric |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, parent_line_id, child_unique_reference_id) WHERE child_unique_reference_id IS NOT NULL AND deleted_at IS NULL (see openmeter/ent/schema/billing.go:440)
- FK invoice_id → billing_invoices.id (required)
- detailed_lines/discount children ON DELETE CASCADE

**Lifecycle:**
- *How to add:* Edit BillingInvoiceLine.Fields() in billing.go and regenerate. This table aggregates several mixins (Annotations, Resource, InvoiceLineBase, Tax, totals, externalid.Line).

### `SubscriptionItem` *(table)*

A ratecard-instance inside a phase carrying its own mutable cadence and effective price/entitlement/discount config; active_*_override_relative_to_phase_start preserves intended cadence across edits/cancels (see openmeter/ent/schema/subscription.go:141).

- **Location:** `openmeter/ent/schema/subscription.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/subscription`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `created_at` | TIMESTAMPTZ |  |
| `updated_at` | TIMESTAMPTZ |  |
| `deleted_at` | TIMESTAMPTZ |  |
| `metadata` | jsonb |  |
| `tax_code_id` | char(26) |  |
| `tax_behavior` | enum |  |
| `annotations` | jsonb |  |
| `active_from` | TIMESTAMPTZ |  |
| `active_to` | TIMESTAMPTZ |  |
| `phase_id` | TEXT |  |
| `key` | TEXT |  |
| `entitlement_id` | TEXT |  |
| `restarts_billing_period` | BOOL |  |
| `active_from_override_relative_to_phase_start` | TEXT | ISO duration offset from phase start preserving the intended item cadence across edits/cancels (see openmeter/ent/schema/subscription.go:173). |
| `active_to_override_relative_to_phase_start` | TEXT |  |
| `name` | TEXT |  |
| `description` | TEXT |  |
| `feature_key` | TEXT |  |
| `entitlement_template` | jsonb |  |
| `tax_config` | jsonb |  |
| `billing_cadence` | TEXT |  |
| `price` | jsonb |  |
| `discounts` | jsonb |  |

**Data Guarantees:**
- PK id
- INDEX(namespace, phase_id, key)
- FK phase_id → subscription_phases.id, FK entitlement_id → entitlements.id ON DELETE CASCADE

**Lifecycle:**
- *How to add:* Edit SubscriptionItem.Fields() and regenerate. Note items are intentionally NOT cadenced via CadencedMixin because their cadence is mutable.

### `Entitlement` *(table)*

A customer's right to consume a feature (metered, boolean, or static); anchors usage periods and links to grants, usage resets and balance snapshots (see openmeter/ent/schema/entitlement.go:20).

- **Location:** `openmeter/ent/schema/entitlement.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/entitlement`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `metadata` | jsonb |  |
| `created_at` | TIMESTAMPTZ |  |
| `updated_at` | TIMESTAMPTZ |  |
| `deleted_at` | TIMESTAMPTZ |  |
| `entitlement_type` | enum |  |
| `feature_id` | char(26) |  |
| `active_from` | TIMESTAMPTZ |  |
| `active_to` | TIMESTAMPTZ |  |
| `feature_key` | TEXT | Validated to reject valid ULIDs so feature keys never collide with ids (see openmeter/ent/schema/entitlement.go:41). |
| `customer_id` | char(26) |  |
| `measure_usage_from` | TIMESTAMPTZ |  |
| `issue_after_reset` | FLOAT |  |
| `issue_after_reset_priority` | uint8 |  |
| `is_soft_limit` | BOOL |  |
| `preserve_overage_at_reset` | BOOL |  |
| `config` | jsonb |  |
| `usage_period_interval` | TEXT | ISO8601 duration string defining the recurring usage period (see openmeter/ent/schema/entitlement.go:61). |
| `usage_period_anchor` | TIMESTAMPTZ | Original anchor time; no longer overwritten on reset — the effective anchor is derived from the last UsageReset queried dynamically (see openmeter/ent/schema/entitlement.go:62). |
| `current_usage_period_start` | TIMESTAMPTZ | Denormalized current-period start; schema TODO marks it for removal in favor of calculation (see openmeter/ent/schema/entitlement.go:63). |
| `current_usage_period_end` | TIMESTAMPTZ |  |
| `annotations` | jsonb |  |

**Data Guarantees:**
- PK id
- UNIQUE(created_at, id) (see openmeter/ent/schema/entitlement.go:85)
- INDEX(current_usage_period_end, deleted_at) — used to collect entitlements with due resets (see openmeter/ent/schema/entitlement.go:84)
- FK feature_id → features.id, FK customer_id → customers.id
- child usage_reset/grant/balance_snapshot ON DELETE CASCADE

**Lifecycle:**
- *How to add:* Edit Entitlement.Fields() in entitlement.go and regenerate; many fields are Immutable and Nillable. The current_usage_period_* columns are slated for removal in favor of dynamic computation.

### `Feature` *(table)*

A purchasable/grantable capability tied to a meter; holds unit-cost configuration (manual amount or LLM-provider/model/token pricing) and is archived via archived_at rather than deleted (see openmeter/ent/schema/feature.go:17).

- **Location:** `openmeter/ent/schema/feature.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/productcatalog/feature`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `created_at` | TIMESTAMPTZ |  |
| `updated_at` | TIMESTAMPTZ |  |
| `deleted_at` | TIMESTAMPTZ |  |
| `metadata` | jsonb |  |
| `namespace` | TEXT |  |
| `name` | TEXT |  |
| `description` | TEXT |  |
| `key` | TEXT |  |
| `meter_slug` | TEXT | Deprecated meter reference by slug; to be removed in Phase 2 in favor of meter_id (see openmeter/ent/schema/feature.go:35). |
| `meter_id` | TEXT |  |
| `meter_group_by_filters` | jsonb |  |
| `advanced_meter_group_by_filters` | jsonb |  |
| `unit_cost_type` | TEXT |  |
| `unit_cost_manual_amount` | numeric |  |
| `unit_cost_llm_provider_property` | TEXT |  |
| `unit_cost_llm_provider` | TEXT |  |
| `unit_cost_llm_model_property` | TEXT |  |
| `unit_cost_llm_model` | TEXT |  |
| `unit_cost_llm_token_type_property` | TEXT |  |
| `unit_cost_llm_token_type` | TEXT |  |
| `archived_at` | TIMESTAMPTZ | Soft-archive timestamp; features are uniqueness-scoped to archived_at IS NULL rather than deleted_at (see openmeter/ent/schema/feature.go:67). |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, key) WHERE archived_at IS NULL (see openmeter/ent/schema/feature.go:66)
- CHECK unit_cost_llm_{provider,model,token_type} property vs literal mutually exclusive (see openmeter/ent/schema/feature.go:55)

**Lifecycle:**
- *How to add:* Edit Feature.Fields() and Annotations() (for CHECK constraints) in feature.go, regenerate, diff a migration.

  ```
  entsql.Checks(map[string]string{
    "unit_cost_llm_provider_mutual_exclusive": "NOT (unit_cost_llm_provider_property IS NOT NULL AND unit_cost_llm_provider IS NOT NULL)",
  })
  ```


### `LedgerBreakageRecord` *(table)*

Projection row recording credit breakage (expired/unused credit) with full source-vs-breakage transaction lineage; intentionally keeps ledger references as plain IDs because it is a projection, not accounting source of truth (see openmeter/ent/schema/ledger_breakage_record.go:15).

- **Location:** `openmeter/ent/schema/ledger_breakage_record.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/ledger/breakage`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | TEXT |  |
| `annotations` | jsonb |  |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ |  |
| `kind` | enum |  |
| `amount` | numeric |  |
| `customer_id` | char(26) |  |
| `currency` | varchar(3) |  |
| `credit_priority` | INT |  |
| `expires_at` | TIMESTAMPTZ |  |
| `source_kind` | enum |  |
| `source_transaction_group_id` | char(26) |  |
| `source_transaction_id` | char(26) |  |
| `source_entry_id` | char(26) |  |
| `breakage_transaction_group_id` | char(26) |  |
| `breakage_transaction_id` | char(26) |  |
| `fbo_sub_account_id` | char(26) |  |
| `breakage_sub_account_id` | char(26) |  |
| `plan_id` | char(26) |  |
| `release_id` | char(26) |  |

**Data Guarantees:**
- PK id
- INDEX(namespace, customer_id, currency, credit_priority, expires_at, id)
- references stored as plain IDs (no FKs) by design (see openmeter/ent/schema/ledger_breakage_record.go:19)

**Lifecycle:**
- *How to add:* Edit LedgerBreakageRecord.Fields() and regenerate. Added via migration 20260517121831_add_credit_expiration_breakage.

### `PlanRateCard` *(table)*

A ratecard within a plan phase (shared RateCard mixin: type, price, entitlement_template, discounts, tax) referencing an optional feature (see openmeter/ent/schema/productcatalog.go:141).

- **Location:** `openmeter/ent/schema/productcatalog.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/productcatalog`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `metadata` | jsonb |  |
| `created_at` | TIMESTAMPTZ |  |
| `updated_at` | TIMESTAMPTZ |  |
| `deleted_at` | TIMESTAMPTZ |  |
| `name` | TEXT |  |
| `description` | TEXT |  |
| `key` | TEXT |  |
| `tax_code_id` | char(26) |  |
| `tax_behavior` | enum |  |
| `type` | enum |  |
| `feature_key` | TEXT |  |
| `entitlement_template` | jsonb |  |
| `tax_config` | jsonb |  |
| `billing_cadence` | TEXT |  |
| `price` | jsonb |  |
| `discounts` | jsonb |  |
| `phase_id` | TEXT |  |
| `feature_id` | TEXT |  |

**Data Guarantees:**
- PK id
- UNIQUE(phase_id, key) WHERE deleted_at IS NULL
- UNIQUE(phase_id, feature_key) WHERE deleted_at IS NULL (see openmeter/ent/schema/productcatalog.go:193)

**Lifecycle:**
- *How to add:* RateCard fields come from the shared RateCard mixin in ratecard.go; phase_id/feature_id are added in PlanRateCard.Fields(). Edit and regenerate.

### `Customer` *(table)*

A billable end-customer; carries billing address snapshot fields and an optional currency, and fans out to subscriptions, entitlements, charges and billing overrides (see openmeter/ent/schema/customer.go:16).

- **Location:** `openmeter/ent/schema/customer.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/customer`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `metadata` | jsonb |  |
| `created_at` | TIMESTAMPTZ |  |
| `updated_at` | TIMESTAMPTZ |  |
| `deleted_at` | TIMESTAMPTZ |  |
| `name` | TEXT |  |
| `description` | TEXT |  |
| `annotations` | jsonb |  |
| `billing_address_country` | char(2) |  |
| `billing_address_postal_code` | TEXT |  |
| `billing_address_state` | TEXT |  |
| `billing_address_city` | TEXT |  |
| `billing_address_line1` | TEXT |  |
| `billing_address_line2` | TEXT |  |
| `billing_address_phone_number` | TEXT |  |
| `key` | TEXT | External-facing stable customer key; stored as empty string (not NULL) when unset so a unique index can be applied (see openmeter/ent/schema/customer.go:33). |
| `primary_email` | TEXT |  |
| `currency` | char(3) |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, key) WHERE deleted_at IS NULL (see openmeter/ent/schema/customer.go:58)
- INDEX(name), INDEX(primary_email), INDEX(created_at)
- soft_delete (deleted_at)
- audit (created_at, updated_at)
- FK apps/subjects/billing_customer_override → ON DELETE CASCADE (see openmeter/ent/schema/customer.go:73)

**Lifecycle:**
- *How to add:* Add the column to Customer.Fields() in openmeter/ent/schema/customer.go, run `make generate`, then `atlas migrate --env local diff <name>` to emit the SQL under tools/migrate/migrations/.

  ```
  field.String("key").Optional(),
  ```

- *How to modify:* Never edit historical migrations; add a new ent field/index and generate a new migration. The committed v3 list-filter TODO in customer.go warns to add pg_trgm GIN indexes via a custom SQL migration before exposing ILIKE contains filters.
- *How to read:* Through the customer adapter/service; the schema notes the v3 list API exposes case-insensitive contains filters compiling to ILIKE.

### `BillingCustomerOverride` *(table)*

Per-customer overrides of billing profile / workflow settings (one per customer); NULL override fields fall back to the profile (see openmeter/ent/schema/billing.go:178).

- **Location:** `openmeter/ent/schema/billing.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `created_at` | TIMESTAMPTZ |  |
| `updated_at` | TIMESTAMPTZ |  |
| `deleted_at` | TIMESTAMPTZ |  |
| `tax_code_id` | char(26) |  |
| `tax_behavior` | enum |  |
| `customer_id` | char(26) |  |
| `billing_profile_id` | char(26) |  |
| `collection_alignment` | enum |  |
| `anchored_alignment_detail` | jsonb |  |
| `line_collection_period` | TEXT |  |
| `invoice_auto_advance` | BOOL |  |
| `invoice_draft_period` | TEXT |  |
| `invoice_due_after` | TEXT |  |
| `invoice_collection_method` | enum |  |
| `invoice_progressive_billing` | BOOL |  |
| `invoice_default_tax_config` | jsonb |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, customer_id) (see openmeter/ent/schema/billing.go:258)
- FK customer_id → customers.id (required, immutable)

**Lifecycle:**
- *How to add:* Edit BillingCustomerOverride.Fields() and regenerate.

### `BillingWorkflowConfig` *(table)*

Cloneable invoicing workflow settings (collection alignment, draft/due periods, auto-advance, progressive billing, tax enable/enforce) referenced by profiles, overrides and invoices (see openmeter/ent/schema/billing.go:110).

- **Location:** `openmeter/ent/schema/billing.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `created_at` | TIMESTAMPTZ |  |
| `updated_at` | TIMESTAMPTZ |  |
| `deleted_at` | TIMESTAMPTZ |  |
| `tax_code_id` | char(26) |  |
| `tax_behavior` | enum |  |
| `collection_alignment` | enum |  |
| `anchored_alignment_detail` | jsonb |  |
| `line_collection_period` | TEXT |  |
| `invoice_auto_advance` | BOOL |  |
| `invoice_draft_period` | TEXT |  |
| `invoice_due_after` | TEXT |  |
| `invoice_collection_method` | enum |  |
| `invoice_progressive_billing` | BOOL |  |
| `invoice_default_tax_settings` | jsonb |  |
| `tax_enabled` | BOOL |  |
| `tax_enforced` | BOOL |  |

**Data Guarantees:**
- PK id
- INDEX(namespace, id)

**Lifecycle:**
- *How to add:* Edit BillingWorkflowConfig.Fields() and regenerate.

### `Grant` *(table)*

A credit grant against an entitlement (owner_id) with amount, priority, expiration and recurrence; voided_at soft-voids it (see openmeter/ent/schema/grant.go:15).

- **Location:** `openmeter/ent/schema/grant.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/credit/grant`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `annotations` | jsonb |  |
| `created_at` | TIMESTAMPTZ |  |
| `updated_at` | TIMESTAMPTZ |  |
| `deleted_at` | TIMESTAMPTZ |  |
| `metadata` | jsonb |  |
| `owner_id` | char(26) | Entitlement id this grant belongs to (FK to entitlements via the grant edge) (see openmeter/ent/schema/grant.go:67). |
| `amount` | numeric |  |
| `priority` | uint8 |  |
| `effective_at` | TIMESTAMPTZ |  |
| `expiration` | jsonb |  |
| `expires_at` | TIMESTAMPTZ |  |
| `voided_at` | TIMESTAMPTZ |  |
| `reset_max_rollover` | numeric |  |
| `reset_min_rollover` | numeric |  |
| `recurrence_period` | TEXT |  |
| `recurrence_anchor` | TIMESTAMPTZ |  |

**Data Guarantees:**
- PK id
- INDEX(namespace, owner_id), INDEX(effective_at, expires_at)
- FK owner_id → entitlements.id ON DELETE CASCADE

**Lifecycle:**
- *How to add:* Edit Grant.Fields() in grant.go, regenerate, diff migration.

### `Subscription` *(table)*

An active customer subscription instance with cadenced active window, currency, billing anchor/cadence, pro-rating and settlement mode; root of phases/items and billing/charge edges (see openmeter/ent/schema/subscription.go:18).

- **Location:** `openmeter/ent/schema/subscription.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/subscription`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `annotations` | jsonb |  |
| `created_at` | TIMESTAMPTZ |  |
| `updated_at` | TIMESTAMPTZ |  |
| `deleted_at` | TIMESTAMPTZ |  |
| `metadata` | jsonb |  |
| `active_from` | TIMESTAMPTZ |  |
| `active_to` | TIMESTAMPTZ |  |
| `name` | TEXT |  |
| `description` | TEXT |  |
| `plan_id` | TEXT |  |
| `customer_id` | TEXT |  |
| `currency` | char(3) |  |
| `billing_anchor` | TIMESTAMPTZ |  |
| `billing_cadence` | TEXT |  |
| `pro_rating_config` | jsonb |  |
| `settlement_mode` | enum |  |

**Data Guarantees:**
- PK id
- INDEX(namespace, customer_id)
- FK customer_id → customers.id (required, immutable)
- phases/addons/billing_sync_state ON DELETE CASCADE

**Lifecycle:**
- *How to add:* Edit Subscription.Fields() in subscription.go and regenerate. Subscription is cadenced (active_from/active_to via CadencedMixin).
- *How to modify:* See the /subscription skill for the sync algorithm; subscription edits drive billing/charge reconciliation.

### `LLMCostPrice` *(table)*

Canonical per-token LLM pricing rows; namespace NULL means a global default, a set namespace is an override (see openmeter/ent/schema/llmcostprice.go:16).

- **Location:** `openmeter/ent/schema/llmcostprice.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/llmcost`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `metadata` | jsonb |  |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ |  |
| `namespace` | TEXT | NULL for global prices, set for namespace overrides (see openmeter/ent/schema/llmcostprice.go:32). |
| `provider` | TEXT |  |
| `model_id` | TEXT |  |
| `model_name` | TEXT |  |
| `input_per_token` | numeric |  |
| `output_per_token` | numeric |  |
| `cache_read_per_token` | numeric |  |
| `reasoning_per_token` | numeric |  |
| `cache_write_per_token` | numeric |  |
| `currency` | TEXT |  |
| `source` | TEXT |  |
| `source_prices` | jsonb |  |
| `effective_from` | TIMESTAMPTZ |  |
| `effective_to` | TIMESTAMPTZ |  |

**Data Guarantees:**
- PK id
- UNIQUE(provider, model_id, namespace, effective_from) WHERE deleted_at IS NULL (see openmeter/ent/schema/llmcostprice.go:85)
- partial index for global lookups WHERE namespace IS NULL

**Lifecycle:**
- *How to add:* Edit LLMCostPrice.Fields() in llmcostprice.go and regenerate. This table notably does NOT use NamespaceMixin (namespace is nullable).
- *How to read:* Override resolution: look up namespace override first, then fall back to the global (namespace IS NULL) row.

### `BillingInvoiceSplitLineGroup` *(table)*

Groups invoice lines that result from splitting a single usage-based service period across progressive-billing invoices; tax fields are deprecated here (see openmeter/ent/schema/billing.go:611).

- **Location:** `openmeter/ent/schema/billing.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `name` | TEXT |  |
| `currency` | varchar(3) |  |
| `tax_config` | jsonb | Deprecated: split groups no longer carry tax config (see openmeter/ent/schema/billing.go:631). |
| `tax_code_id` | char(26) |  |
| `tax_behavior` | enum |  |
| `service_period_start` | TIMESTAMPTZ |  |
| `service_period_end` | TIMESTAMPTZ |  |
| `unique_reference_id` | TEXT |  |
| `ratecard_discounts` | jsonb |  |
| `feature_key` | TEXT |  |
| `price` | jsonb |  |
| `subscription_id/subscription_phase_id/subscription_item_id` | TEXT |  |
| `subscription_billing_period_from/to` | TIMESTAMPTZ |  |
| `charge_id` | char(26) |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, unique_reference_id) WHERE unique_reference_id IS NOT NULL AND deleted_at IS NULL (see openmeter/ent/schema/billing.go:717)

**Lifecycle:**
- *How to add:* Edit and regenerate.

### `BillingProfile` *(table)*

A billing configuration tying supplier identity + tax/invoicing/payment apps + workflow config; one default per namespace (see openmeter/ent/schema/billing.go:28).

- **Location:** `openmeter/ent/schema/billing.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `metadata` | jsonb |  |
| `created_at` | TIMESTAMPTZ |  |
| `updated_at` | TIMESTAMPTZ |  |
| `deleted_at` | TIMESTAMPTZ |  |
| `name` | TEXT |  |
| `description` | TEXT |  |
| `supplier_address_*` | TEXT |  |
| `tax_app_id` | char(26) |  |
| `invoicing_app_id` | char(26) |  |
| `payment_app_id` | char(26) |  |
| `workflow_config_id` | TEXT |  |
| `default` | BOOL |  |
| `supplier_name` | TEXT |  |
| `supplier_tax_code` | TEXT |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, default) WHERE "default" AND deleted_at IS NULL — at most one default profile per namespace (see openmeter/ent/schema/billing.go:102)
- FK tax_app/invoicing_app/payment_app → apps.id (required, immutable)

**Lifecycle:**
- *How to add:* Edit BillingProfile.Fields() in billing.go and regenerate.

### `Plan` *(table)*

A versioned product-catalog plan (key + version) with default billing cadence, pro-rating config, settlement mode and effective window; phases hang off it (see openmeter/ent/schema/productcatalog.go:16).

- **Location:** `openmeter/ent/schema/productcatalog.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/productcatalog`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `metadata` | jsonb |  |
| `created_at` | TIMESTAMPTZ |  |
| `updated_at` | TIMESTAMPTZ |  |
| `deleted_at` | TIMESTAMPTZ |  |
| `name` | TEXT |  |
| `description` | TEXT |  |
| `key` | TEXT |  |
| `version` | INT |  |
| `currency` | TEXT |  |
| `billing_cadence` | TEXT |  |
| `pro_rating_config` | jsonb |  |
| `effective_from` | TIMESTAMPTZ |  |
| `effective_to` | TIMESTAMPTZ |  |
| `settlement_mode` | enum | Defaults to credit-then-invoice settlement (see openmeter/ent/schema/productcatalog.go:57). |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, key, version) WHERE deleted_at IS NULL (see openmeter/ent/schema/productcatalog.go:78)
- version >= 1
- phases/addons ON DELETE CASCADE

**Lifecycle:**
- *How to add:* Edit Plan.Fields() in productcatalog.go and regenerate.

### `Addon` *(table)*

A versioned add-on product (single or multi instance) that can be attached to plans and subscriptions; annotations are GIN-indexed (see openmeter/ent/schema/addon.go:16).

- **Location:** `openmeter/ent/schema/addon.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/productcatalog`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `metadata` | jsonb |  |
| `created_at` | TIMESTAMPTZ |  |
| `updated_at` | TIMESTAMPTZ |  |
| `deleted_at` | TIMESTAMPTZ |  |
| `name` | TEXT |  |
| `description` | TEXT |  |
| `key` | TEXT |  |
| `version` | INT |  |
| `currency` | TEXT |  |
| `instance_type` | enum |  |
| `effective_from` | TIMESTAMPTZ |  |
| `effective_to` | TIMESTAMPTZ |  |
| `annotations` | jsonb |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, key, version) WHERE deleted_at IS NULL (see openmeter/ent/schema/addon.go:72)
- GIN INDEX(annotations)
- ratecards/plans/subscription_addons ON DELETE CASCADE

**Lifecycle:**
- *How to add:* Edit Addon.Fields() in addon.go and regenerate.

### `ChargeFlatFee` *(table)*

Flat-fee charge subtype: payment term, pro-rating, pre/post-proration amounts, settlement and detailed status; has realization runs (see openmeter/ent/schema/chargesflatfee.go:23).

- **Location:** `openmeter/ent/schema/chargesflatfee.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing/charges/flatfee`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id (chargemeta.Mixin)` | char(26) |  |
| `namespace` | TEXT |  |
| `customer_id` | char(26) |  |
| `payment_term` | TEXT |  |
| `invoice_at` | TIMESTAMPTZ |  |
| `settlement_mode` | enum |  |
| `discounts` | jsonb |  |
| `pro_rating` | enum |  |
| `feature_key` | TEXT |  |
| `feature_id` | char(26) |  |
| `amount_before_proration` | numeric |  |
| `amount_after_proration` | numeric |  |
| `current_realization_run_id` | char(26) |  |
| `status_detailed` | enum |  |
| `tax_code_id` | char(26) |  |

**Data Guarantees:**
- PK id
- INDEX(tax_code_id)
- tax_code FK ON DELETE SET NULL

**Lifecycle:**
- *How to add:* Edit ChargeFlatFee.Fields() in chargesflatfee.go and regenerate.

### `Meter` *(table)*

Definition of how to aggregate raw usage events (event_type + aggregation + optional value_property/group_by) into a metered value; event_from bounds which events count (see openmeter/ent/schema/meter.go:14).

- **Location:** `openmeter/ent/schema/meter.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/meter`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `key` | TEXT |  |
| `name` | TEXT |  |
| `description` | TEXT |  |
| `metadata` | jsonb |  |
| `annotations` | jsonb |  |
| `created_at` | TIMESTAMPTZ |  |
| `updated_at` | TIMESTAMPTZ |  |
| `deleted_at` | TIMESTAMPTZ |  |
| `event_type` | TEXT | CloudEvents `type` value this meter filters on when scanning the ClickHouse events table (see openmeter/ent/schema/meter.go:30). |
| `value_property` | TEXT | JSONPath into the event data extracting the numeric value; optional for COUNT aggregation (see openmeter/ent/schema/meter.go:32). |
| `group_by` | jsonb |  |
| `aggregation` | enum |  |
| `event_from` | TIMESTAMPTZ | If set, only events at/after this time are included in the meter (see openmeter/ent/schema/meter.go:36). |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, key) WHERE deleted_at IS NULL (see openmeter/ent/schema/meter.go:43)
- INDEX(namespace, event_type)
- soft_delete (deleted_at)

**Lifecycle:**
- *How to add:* Edit Meter.Fields() and regenerate. Meters are the bridge between Postgres definitions and the ClickHouse om_<ns>_events table queried by openmeter/streaming/clickhouse/meter_query.go.
- *How to read:* Meter aggregation queries run against ClickHouse via the streaming connector, not Postgres.

### `ChargeUsageBasedRuns` *(table)*

A realization run for a usage-based charge; stored_at_lt is the late-arriving-event cutoff and metered_quantity the measured usage for the period (see openmeter/ent/schema/chargesusagebased.go:143).

- **Location:** `openmeter/ent/schema/chargesusagebased.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing/charges/usagebased`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | TEXT |  |
| `charge_id` | char(26) |  |
| `feature_id` | char(26) |  |
| `type` | enum |  |
| `initial_type` | enum |  |
| `stored_at_lt` | TIMESTAMPTZ | Upper bound on event stored_at included in this run, implementing the late-arriving usage cutoff (see openmeter/ent/schema/chargesusagebased.go:179). |
| `service_period_to` | TIMESTAMPTZ |  |
| `detailed_lines_present` | BOOL |  |
| `line_id` | char(26) |  |
| `invoice_id` | char(26) |  |
| `metered_quantity` | numeric |  |
| `no_fiat_transaction_required` | BOOL |  |
| `totals fields` | numeric |  |

**Data Guarantees:**
- PK id
- INDEX(namespace, charge_id)
- FK feature_id → features.id (required)

**Lifecycle:**
- *How to add:* Edit ChargeUsageBasedRuns.Fields() and regenerate.

### `ChargesSearchV1` *(value_object)*

Postgres VIEW UNION-ALLing charge_credit_purchases, charge_flat_fees and charge_usage_based into a unified searchable charge-meta surface; declared as ent.View with no mixins (see openmeter/ent/schema/charges.go:48).

- **Location:** `openmeter/ent/schema/charges.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing/charges/meta`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `type` | enum |  |
| `id` | char(26) |  |
| `namespace` | TEXT |  |
| `customer_id` | char(26) |  |
| `status` | TEXT |  |
| `unique_reference_id` | TEXT |  |
| `currency` | TEXT |  |
| `managed_by` | TEXT |  |
| `subscription_id/subscription_phase_id/subscription_item_id` | TEXT |  |
| `advance_after` | TIMESTAMPTZ |  |
| `service_period_from/to` | TIMESTAMPTZ |  |
| `billing_period_from/to` | TIMESTAMPTZ |  |
| `tax_code_id` | char(26) |  |
| `tax_behavior` | TEXT |  |

**Data Guarantees:**
- read-only view; per AGENTS.md ent.View DDL may need explicit SQL migration (not in migrate.Tables)

**Lifecycle:**
- *How to add:* Modify buildChargesSearchV1TableSelector / chargesSearchV1Columns in charges.go. Per the AGENTS.md ent.View caveat, view DDL may need a hand-written SQL migration since views don't appear in generated migrate.Tables.

### `BillingInvoiceLineDiscount` *(table)*

Amount discount applied to an invoice line (reason + amount + rounding); several quantity fields are deprecated after splitting amount vs usage discount tables (see openmeter/ent/schema/billing.go:755).

- **Location:** `openmeter/ent/schema/billing.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ |  |
| `line_id` | char(26) |  |
| `child_unique_reference_id` | TEXT |  |
| `description` | TEXT |  |
| `reason` | enum |  |
| `amount` | numeric |  |
| `rounding_amount` | numeric |  |
| `source_discount` | jsonb |  |
| `type` | TEXT | Deprecated after splitting amount/usage discount tables (see openmeter/ent/schema/billing.go:793). |
| `quantity` | numeric |  |
| `pre_line_period_quantity` | numeric |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, line_id, child_unique_reference_id) WHERE child_unique_reference_id IS NOT NULL AND deleted_at IS NULL (see openmeter/ent/schema/billing.go:820)
- FK line_id → billing_invoice_lines.id (required)

**Lifecycle:**
- *How to add:* Edit and regenerate.

### `ChargeFlatFeeRun` *(table)*

A realization run for a flat-fee charge over a service period, producing detailed lines / payment / invoiced-usage and linking to invoice lines (see openmeter/ent/schema/chargesflatfee.go:141).

- **Location:** `openmeter/ent/schema/chargesflatfee.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing/charges/flatfee`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | TEXT |  |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ |  |
| `charge_id` | char(26) |  |
| `type` | enum |  |
| `initial_type` | enum |  |
| `service_period_from/to` | TIMESTAMPTZ |  |
| `line_id` | char(26) |  |
| `invoice_id` | char(26) |  |
| `amount_after_proration` | numeric |  |
| `no_fiat_transaction_required` | BOOL |  |
| `immutable` | BOOL |  |
| `totals fields` | numeric |  |

**Data Guarantees:**
- PK id
- INDEX(namespace, charge_id)
- line_id/invoice_id FK ON DELETE SET NULL

**Lifecycle:**
- *How to add:* Edit ChargeFlatFeeRun.Fields() and regenerate.

### `ChargeUsageBased` *(table)*

Usage-based charge subtype tied to a feature with rating engine, price JSON and realization runs; table explicitly named charge_usage_based (see openmeter/ent/schema/chargesusagebased.go:24).

- **Location:** `openmeter/ent/schema/chargesusagebased.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing/charges/usagebased`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | TEXT |  |
| `customer_id` | char(26) |  |
| `invoice_at` | TIMESTAMPTZ |  |
| `settlement_mode` | enum |  |
| `discounts` | jsonb |  |
| `feature_key` | TEXT |  |
| `feature_id` | char(26) |  |
| `rating_engine` | enum |  |
| `price` | jsonb |  |
| `current_realization_run_id` | char(26) |  |
| `status_detailed` | enum |  |
| `tax_code_id` | char(26) |  |

**Data Guarantees:**
- PK id
- INDEX(tax_code_id)
- FK feature_id → features.id (required)

**Lifecycle:**
- *How to add:* Edit ChargeUsageBased.Fields() in chargesusagebased.go and regenerate. Note Annotations() forces table name to charge_usage_based.

### `AddonRateCard` *(table)*

RateCard belonging to an Addon (same RateCard mixin shape as PlanRateCard) (see openmeter/ent/schema/addon.go:87).

- **Location:** `openmeter/ent/schema/addon.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/productcatalog`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `key` | TEXT |  |
| `name` | TEXT |  |
| `tax_code_id` | char(26) |  |
| `tax_behavior` | enum |  |
| `type` | enum |  |
| `feature_key` | TEXT |  |
| `price` | jsonb |  |
| `discounts` | jsonb |  |
| `addon_id` | TEXT |  |
| `feature_id` | TEXT |  |

**Data Guarantees:**
- PK id
- UNIQUE(addon_id, key) WHERE deleted_at IS NULL
- UNIQUE(addon_id, feature_key) WHERE deleted_at IS NULL (see openmeter/ent/schema/addon.go:139)

**Lifecycle:**
- *How to add:* Edit AddonRateCard.Fields() and regenerate.

### `LedgerSubAccountRoute` *(table)*

Routing rule mapping a routing_key (currency/tax/feature/cost-basis/priority dimensions, denormalized as literal columns, not FKs) to a sub-account within an account (see openmeter/ent/schema/ledger_account.go:96).

- **Location:** `openmeter/ent/schema/ledger_account.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/ledger`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | TEXT |  |
| `account_id` | char(26) |  |
| `routing_key_version` | TEXT |  |
| `routing_key` | TEXT |  |
| `currency` | TEXT | Literal routing value denormalized from routing_key for filtering, not a FK (see openmeter/ent/schema/ledger_account.go:117). |
| `tax_code` | TEXT | Stores the TaxCode.Key string as a routing dimension, not a FK to tax_codes (see openmeter/ent/schema/ledger_account.go:119). |
| `tax_behavior` | TEXT |  |
| `features` | text[] |  |
| `cost_basis` | numeric |  |
| `credit_priority` | INT |  |
| `transaction_authorization_status` | TEXT |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, account_id, routing_key_version, routing_key) (see openmeter/ent/schema/ledger_account.go:140)
- FK account_id (required)

**Lifecycle:**
- *How to add:* Edit LedgerSubAccountRoute.Fields() and regenerate. The features text[] column was changed via migration 20260604143000_ledger_route_features_text_array; tax_behavior added in 20260520130500.

### `PlanPhase` *(table)*

An ordered phase within a Plan (index + optional duration) that groups ratecards (see openmeter/ent/schema/productcatalog.go:86).

- **Location:** `openmeter/ent/schema/productcatalog.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/productcatalog`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `metadata` | jsonb |  |
| `created_at` | TIMESTAMPTZ |  |
| `updated_at` | TIMESTAMPTZ |  |
| `deleted_at` | TIMESTAMPTZ |  |
| `name` | TEXT |  |
| `description` | TEXT |  |
| `key` | TEXT |  |
| `plan_id` | TEXT |  |
| `index` | uint8 |  |
| `duration` | TEXT |  |

**Data Guarantees:**
- PK id
- UNIQUE(plan_id, key) WHERE deleted_at IS NULL
- UNIQUE(plan_id, index) WHERE deleted_at IS NULL (see openmeter/ent/schema/productcatalog.go:133)
- ratecards ON DELETE CASCADE

**Lifecycle:**
- *How to add:* Edit PlanPhase.Fields() and regenerate.

### `SubscriptionPhase` *(table)*

A time-bounded phase of a subscription; sort_hint disambiguates phases that share an active_from (zero-length phases) (see openmeter/ent/schema/subscription.go:95).

- **Location:** `openmeter/ent/schema/subscription.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/subscription`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `created_at` | TIMESTAMPTZ |  |
| `updated_at` | TIMESTAMPTZ |  |
| `deleted_at` | TIMESTAMPTZ |  |
| `metadata` | jsonb |  |
| `subscription_id` | TEXT |  |
| `key` | TEXT |  |
| `name` | TEXT |  |
| `description` | TEXT |  |
| `active_from` | TIMESTAMPTZ |  |
| `sort_hint` | uint8 | Tie-breaker for ordering phases with equal active_from (see openmeter/ent/schema/subscription.go:115). |

**Data Guarantees:**
- PK id
- INDEX(namespace, subscription_id, key)
- FK subscription_id → subscriptions.id ON DELETE CASCADE
- items ON DELETE CASCADE

**Lifecycle:**
- *How to add:* Edit SubscriptionPhase.Fields() and regenerate.

### `ChargeCreditPurchase` *(table)*

Credit-purchase charge subtype: credit amount, effective/expiry window, priority, feature filters, settlement JSON and detailed status; links to a granted credit + payment edges (see openmeter/ent/schema/chargescreditpurchase.go:21).

- **Location:** `openmeter/ent/schema/chargescreditpurchase.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing/charges/creditpurchase`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | TEXT |  |
| `customer_id` | char(26) |  |
| `credit_amount` | numeric |  |
| `effective_at` | TIMESTAMPTZ |  |
| `expires_at` | TIMESTAMPTZ |  |
| `priority` | INT |  |
| `feature_filters` | text[] | Restricts which features the purchased credit applies to (added in migration 20260605120000) (see openmeter/ent/schema/chargescreditpurchase.go:50). |
| `settlement` | jsonb |  |
| `status_detailed` | enum |  |
| `tax_code_id` | char(26) |  |

**Data Guarantees:**
- PK id
- INDEX(tax_code_id)
- FK customer_id → customers.id (required)

**Lifecycle:**
- *How to add:* Edit ChargeCreditPurchase.Fields() in chargescreditpurchase.go and regenerate. feature_filters was added via tools/migrate/migrations/20260605120000_add_credit_feature_filters.up.sql.

  ```
  ALTER TABLE "charge_credit_purchases" ADD COLUMN "feature_filters" text[] NULL;
  ```


### `NotificationEventDeliveryStatus` *(table)*

Per-(event, channel) delivery state machine row with reason, next_attempt_at and an attempts history; drives the reconciliation/retry loop (see openmeter/ent/schema/notification.go:170).

- **Location:** `openmeter/ent/schema/notification.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/notification`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | TEXT |  |
| `annotations` | jsonb |  |
| `created_at` | TIMESTAMPTZ |  |
| `updated_at` | TIMESTAMPTZ |  |
| `event_id` | TEXT |  |
| `channel_id` | TEXT |  |
| `state` | enum |  |
| `reason` | TEXT |  |
| `next_attempt_at` | TIMESTAMPTZ | When the reconciliation loop should next retry delivery (see openmeter/ent/schema/notification.go:201). |
| `attempts` | jsonb |  |

**Data Guarantees:**
- PK id
- INDEX(namespace, event_id, channel_id), INDEX(namespace, state, next_attempt_at) — drives retry polling (see openmeter/ent/schema/notification.go:220)

**Lifecycle:**
- *How to add:* Edit NotificationEventDeliveryStatus.Fields() and regenerate.
- *How to read:* Reconciliation loop polls state + next_attempt_at; see the /notification skill.

### `PlanAddon` *(table)*

Join table declaring which Addon is available for a Plan, from which phase and up to what quantity (see openmeter/ent/schema/planaddon.go:13).

- **Location:** `openmeter/ent/schema/planaddon.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/productcatalog`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `metadata` | jsonb |  |
| `annotations` | jsonb |  |
| `created_at` | TIMESTAMPTZ |  |
| `updated_at` | TIMESTAMPTZ |  |
| `deleted_at` | TIMESTAMPTZ |  |
| `plan_id` | TEXT |  |
| `addon_id` | TEXT |  |
| `from_plan_phase` | TEXT |  |
| `max_quantity` | INT |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, plan_id, addon_id) WHERE deleted_at IS NULL (see openmeter/ent/schema/planaddon.go:65)

**Lifecycle:**
- *How to add:* Edit PlanAddon.Fields() and regenerate.

### `BalanceSnapshot` *(table)*

Cached point-in-time credit balance for an entitlement owner (grant_balances + usage + overage at a timestamp), used to avoid recomputing the full ledger from scratch (see openmeter/ent/schema/balance_snapshot.go:15).

- **Location:** `openmeter/ent/schema/balance_snapshot.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/credit/balance`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `namespace` | TEXT |  |
| `created_at` | TIMESTAMPTZ |  |
| `updated_at` | TIMESTAMPTZ |  |
| `deleted_at` | TIMESTAMPTZ |  |
| `owner_id` | char(26) |  |
| `grant_balances` | jsonb |  |
| `usage` | jsonb |  |
| `balance` | numeric |  |
| `overage` | numeric |  |
| `at` | TIMESTAMPTZ | The instant this balance was snapshotted; queried as the high-water mark for resuming balance computation (see openmeter/ent/schema/balance_snapshot.go:43). |

**Data Guarantees:**
- INDEX(namespace, owner_id, at) WHERE deleted_at IS NULL (see openmeter/ent/schema/balance_snapshot.go:49)
- FK owner_id → entitlements.id ON DELETE CASCADE
- all value fields Immutable

**Lifecycle:**
- *How to add:* Edit BalanceSnapshot.Fields() and regenerate.

### `BillingInvoiceLineUsageDiscount` *(table)*

Usage (quantity) discount applied to a usage-based invoice line (see openmeter/ent/schema/billing.go:838).

- **Location:** `openmeter/ent/schema/billing.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ |  |
| `line_id` | char(26) |  |
| `child_unique_reference_id` | TEXT |  |
| `description` | TEXT |  |
| `reason` | enum |  |
| `quantity` | numeric |  |
| `pre_line_period_quantity` | numeric |  |
| `reason_details` | jsonb |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, line_id, child_unique_reference_id) WHERE child_unique_reference_id IS NOT NULL AND deleted_at IS NULL
- FK line_id → billing_invoice_lines.id (required)

**Lifecycle:**
- *How to add:* Edit and regenerate.

### `UsageReset` *(table)*

Records an entitlement usage-period reset event (reset_time + anchor + interval); the entitlement's effective anchor is derived from the latest reset rather than from the entitlement row (see openmeter/ent/schema/usage_reset.go:14).

- **Location:** `openmeter/ent/schema/usage_reset.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/entitlement`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `annotations` | jsonb |  |
| `created_at` | TIMESTAMPTZ |  |
| `updated_at` | TIMESTAMPTZ |  |
| `deleted_at` | TIMESTAMPTZ |  |
| `entitlement_id` | char(26) |  |
| `reset_time` | TIMESTAMPTZ |  |
| `anchor` | TIMESTAMPTZ |  |
| `usage_period_interval` | TEXT |  |

**Data Guarantees:**
- PK id
- INDEX(namespace, entitlement_id, reset_time)
- FK entitlement_id → entitlements.id ON DELETE CASCADE
- all fields Immutable

**Lifecycle:**
- *How to add:* Edit UsageReset.Fields() and regenerate.

### `AppStripe` *(table)*

Stripe app configuration (account id, livemode, webhook id, and sensitive api_key/webhook_secret) (see openmeter/ent/schema/app_stripe.go:15).

- **Location:** `openmeter/ent/schema/app_stripe.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/app/stripe`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | TEXT |  |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ |  |
| `stripe_account_id` | TEXT |  |
| `stripe_livemode` | BOOL |  |
| `api_key` | TEXT | Ent Sensitive() field — redacted in logs/serialization (see openmeter/ent/schema/app_stripe.go:31). |
| `masked_api_key` | TEXT |  |
| `stripe_webhook_id` | TEXT |  |
| `webhook_secret` | TEXT | Ent Sensitive() field — redacted in logs/serialization (see openmeter/ent/schema/app_stripe.go:34). |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, stripe_account_id, stripe_livemode) WHERE deleted_at IS NULL (see openmeter/ent/schema/app_stripe.go:40)

**Lifecycle:**
- *How to add:* Edit AppStripe.Fields() and regenerate. Use .Sensitive() for secret columns.

### `BillingInvoiceValidationIssue` *(table)*

A validation issue attached to an invoice, deduped per invoice via a 32-byte dedupe_hash (see openmeter/ent/schema/billing.go:1249).

- **Location:** `openmeter/ent/schema/billing.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `invoice_id` | char(26) |  |
| `severity` | enum |  |
| `code` | TEXT |  |
| `message` | TEXT |  |
| `path` | TEXT |  |
| `component` | TEXT |  |
| `dedupe_hash` | bytea(32) | 32-byte hash that dedupes identical issues per invoice via the unique index (see openmeter/ent/schema/billing.go:1286). |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, invoice_id, dedupe_hash) (see openmeter/ent/schema/billing.go:1294)
- FK invoice_id → billing_invoices.id ON DELETE CASCADE

**Lifecycle:**
- *How to add:* Edit and regenerate.

### `Charge` *(table)*

Polymorphic parent row for a billable charge; exactly one of the three per-type FK columns (flat_fee/credit_purchase/usage_based) points to the subtype, with unique_reference_id for idempotent creation (see openmeter/ent/schema/charges.go:96).

- **Location:** `openmeter/ent/schema/charges.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing/charges`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `created_at` | TIMESTAMPTZ |  |
| `deleted_at` | TIMESTAMPTZ |  |
| `unique_reference_id` | TEXT | Idempotency key for a charge; uniqueness is enforced only when set and not deleted (see openmeter/ent/schema/charges.go:116). |
| `type` | enum |  |
| `charge_flat_fee_id` | TEXT |  |
| `charge_credit_purchase_id` | TEXT |  |
| `charge_usage_based_id` | TEXT |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, unique_reference_id) WHERE unique_reference_id IS NOT NULL AND deleted_at IS NULL (see openmeter/ent/schema/charges.go:167)
- exactly one of the three subtype FK columns is set

**Lifecycle:**
- *How to add:* Edit Charge.Fields() in charges.go and regenerate. See the /charges skill for charge creation and advancement.
- *How to read:* The ChargesSearchV1 Postgres VIEW (UNION ALL of the three subtype tables) provides a unified read surface (see openmeter/ent/schema/charges.go:48).

### `CreditRealizationLineage` *(table)*

Tracks the lineage of a realized credit allocation for a charge (root_realization_id + origin_kind + advance_features), with closeable segments (see openmeter/ent/schema/creditrealizationlineage.go:24).

- **Location:** `openmeter/ent/schema/creditrealizationlineage.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing/charges/creditrealization`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | TEXT |  |
| `charge_id` | char(26) |  |
| `root_realization_id` | char(26) |  |
| `customer_id` | char(26) |  |
| `currency` | varchar(3) |  |
| `origin_kind` | enum |  |
| `advance_features` | text[] | Feature filters carried by the originating advance (added in migration 20260605120000) (see openmeter/ent/schema/creditrealizationlineage.go:65). |
| `created_at` | TIMESTAMPTZ |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, root_realization_id) (see openmeter/ent/schema/creditrealizationlineage.go:91)
- FK charge_id → charges.id (required)

**Lifecycle:**
- *How to add:* Edit CreditRealizationLineage.Fields() and regenerate.

### `CreditRealizationLineageSegment` *(table)*

A segment of a credit realization lineage tracking amount + state transitions and backing transaction groups; closed_at marks settlement (see openmeter/ent/schema/creditrealizationlineage.go:97).

- **Location:** `openmeter/ent/schema/creditrealizationlineage.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing/charges/creditrealization`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `lineage_id` | char(26) |  |
| `amount` | numeric |  |
| `state` | enum |  |
| `backing_transaction_group_id` | char(26) |  |
| `source_state` | enum |  |
| `source_backing_transaction_group_id` | char(26) |  |
| `closed_at` | TIMESTAMPTZ |  |
| `created_at` | TIMESTAMPTZ |  |

**Data Guarantees:**
- PK id
- INDEX(lineage_id, closed_at)
- FK lineage_id → credit_realization_lineages.id (required)

**Lifecycle:**
- *How to add:* Edit and regenerate. This table has no namespace/time mixins.

### `NotificationChannel` *(table)*

A delivery channel (currently webhook) holding a type-tagged config JSON resolved by a custom value scanner (see openmeter/ent/schema/notification.go:21).

- **Location:** `openmeter/ent/schema/notification.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/notification`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | TEXT |  |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ |  |
| `annotations` | jsonb |  |
| `metadata` | jsonb |  |
| `type` | enum |  |
| `name` | TEXT |  |
| `disabled` | BOOL |  |
| `config` | jsonb | Type-discriminated channel config (e.g. webhook), serialized via ChannelConfigValueScanner (see openmeter/ent/schema/notification.go:230). |

**Data Guarantees:**
- PK id
- INDEX(namespace, type)

**Lifecycle:**
- *How to add:* Edit NotificationChannel.Fields() in notification.go and regenerate. New config variants need a case added to ChannelConfigValueScanner. See the /notification skill.

### `NotificationRule` *(table)*

A rule binding an event type to channels with a type-tagged config (balance threshold, entitlement reset, invoice created/updated) (see openmeter/ent/schema/notification.go:67).

- **Location:** `openmeter/ent/schema/notification.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/notification`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | TEXT |  |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ |  |
| `annotations` | jsonb |  |
| `metadata` | jsonb |  |
| `type` | enum |  |
| `name` | TEXT |  |
| `disabled` | BOOL |  |
| `config` | jsonb | Type-discriminated rule config serialized via RuleConfigValueScanner (see openmeter/ent/schema/notification.go:290). |

**Data Guarantees:**
- PK id
- INDEX(namespace, type)

**Lifecycle:**
- *How to add:* Edit NotificationRule.Fields() and regenerate. New event-type configs require a case in RuleConfigValueScanner.

### `Subject` *(table)*

The metered identity (e.g. a user/account key) that events are attributed to; carries optional stripe_customer_id and free-form metadata (see openmeter/ent/schema/subject.go:13).

- **Location:** `openmeter/ent/schema/subject.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/subject`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `created_at` | TIMESTAMPTZ |  |
| `updated_at` | TIMESTAMPTZ |  |
| `deleted_at` | TIMESTAMPTZ |  |
| `key` | TEXT |  |
| `display_name` | TEXT |  |
| `stripe_customer_id` | TEXT |  |
| `metadata` | jsonb |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, key) WHERE deleted_at IS NULL (see openmeter/ent/schema/subject.go:45)
- UNIQUE(namespace, id)
- INDEX(display_name), INDEX(created_at, id)
- soft_delete (deleted_at)

**Lifecycle:**
- *How to add:* Edit Subject.Fields() in subject.go, regenerate and diff a migration.

### `TaxCode` *(table)*

A reusable tax code resource with optional per-app mappings, referenced by ratecards, lines, charges and org defaults (see openmeter/ent/schema/taxcode.go:18).

- **Location:** `openmeter/ent/schema/taxcode.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/taxcode`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | TEXT |  |
| `key` | TEXT |  |
| `name` | TEXT |  |
| `description` | TEXT |  |
| `metadata` | jsonb |  |
| `annotations` | jsonb |  |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ |  |
| `app_mappings` | jsonb |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, key) WHERE deleted_at IS NULL (see openmeter/ent/schema/taxcode.go:44)

**Lifecycle:**
- *How to add:* Edit TaxCode.Fields() and regenerate. Migration 20260527120000_dedupe_tax_codes_by_app_mapping consolidated duplicate tax codes.

### `App` *(table)*

An installed integration/app of a given type and status (e.g. stripe, custom-invoicing); linked to customers and used as tax/invoicing/payment provider on billing profiles and invoices (see openmeter/ent/schema/app.go:16).

- **Location:** `openmeter/ent/schema/app.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/app`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | TEXT |  |
| `metadata` | jsonb |  |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ |  |
| `name` | TEXT |  |
| `description` | TEXT |  |
| `type` | TEXT |  |
| `status` | TEXT |  |

**Data Guarantees:**
- PK id
- INDEX(namespace, type)
- customer_apps ON DELETE CASCADE

**Lifecycle:**
- *How to add:* Edit App.Fields() in app.go and regenerate.

### `BillingInvoiceUsageBasedLineConfig` *(table)*

Usage-based line config: price type, feature key, price JSON and pre-line/metered quantities (see openmeter/ent/schema/billing.go:541).

- **Location:** `openmeter/ent/schema/billing.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `price_type` | enum |  |
| `feature_key` | TEXT |  |
| `price` | jsonb |  |
| `pre_line_period_quantity` | numeric |  |
| `metered_pre_line_period_quantity` | numeric |  |
| `metered_quantity` | numeric |  |

**Data Guarantees:**
- PK id

**Lifecycle:**
- *How to add:* Edit and regenerate.

### `BillingStandardInvoiceDetailedLineAmountDiscount` *(table)*

Amount discount on a standard-invoice detailed line (see openmeter/ent/schema/billing.go:956).

- **Location:** `openmeter/ent/schema/billing.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `line_id` | char(26) |  |
| `child_unique_reference_id` | TEXT |  |
| `reason` | enum |  |
| `amount` | numeric |  |
| `rounding_amount` | numeric |  |
| `source_discount` | jsonb |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, line_id, child_unique_reference_id) WHERE child_unique_reference_id IS NOT NULL AND deleted_at IS NULL (see openmeter/ent/schema/billing.go:996)

**Lifecycle:**
- *How to add:* Edit and regenerate.

### `ChargeUsageBasedRunDetailedLine` *(table)*

Detailed (priced) line produced by a usage-based realization run; corrects_run_id links a correction line back to a superseded run (see openmeter/ent/schema/chargesusagebased.go:260).

- **Location:** `openmeter/ent/schema/chargesusagebased.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing/charges/usagebased`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | TEXT |  |
| `charge_id` | char(26) |  |
| `run_id` | char(26) |  |
| `pricer_reference_id` | TEXT |  |
| `corrects_run_id` | char(26) |  |
| `child_unique_reference_id (stddetailedline.Mixin)` | TEXT |  |
| `tax_code_id` | char(26) |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, charge_id, run_id, child_unique_reference_id) WHERE deleted_at IS NULL (see openmeter/ent/schema/chargesusagebased.go:324)
- table explicitly named charge_usage_based_run_detailed_line

**Lifecycle:**
- *How to add:* Edit and regenerate.

### `LedgerEntry` *(table)*

A single signed amount posting against a sub-account within a transaction; identity_key dedupes entries within a (transaction, sub_account) (see openmeter/ent/schema/ledger_entry.go:15).

- **Location:** `openmeter/ent/schema/ledger_entry.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/ledger`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | TEXT |  |
| `annotations` | jsonb |  |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ |  |
| `sub_account_id` | char(26) |  |
| `identity_key` | TEXT | Per-(transaction, sub_account) dedup key enforced by a unique index; defaults to empty string (see openmeter/ent/schema/ledger_entry.go:33). |
| `amount` | numeric |  |
| `transaction_id` | char(26) |  |

**Data Guarantees:**
- PK id
- UNIQUE(transaction_id, sub_account_id, identity_key) (see openmeter/ent/schema/ledger_entry.go:67)
- amount immutable
- FK transaction_id, sub_account_id (required, immutable)

**Lifecycle:**
- *How to add:* Edit LedgerEntry.Fields() and regenerate. Entries are append-only (immutable amount).

### `SubscriptionAddon` *(table)*

An instantiated addon attached to a subscription; quantity history lives in SubscriptionAddonQuantity (see openmeter/ent/schema/subscription_addon.go:14).

- **Location:** `openmeter/ent/schema/subscription_addon.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/subscription`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `metadata` | jsonb |  |
| `created_at` | TIMESTAMPTZ |  |
| `updated_at` | TIMESTAMPTZ |  |
| `deleted_at` | TIMESTAMPTZ |  |
| `addon_id` | TEXT |  |
| `subscription_id` | TEXT |  |

**Data Guarantees:**
- PK id
- FK subscription_id → subscriptions.id (required), FK addon_id → addons.id (required)
- quantities ON DELETE CASCADE

**Lifecycle:**
- *How to add:* Edit SubscriptionAddon.Fields() and regenerate.

### `SubscriptionAddonQuantity` *(table)*

Append-only quantity-over-time records for a SubscriptionAddon (active_from + immutable quantity) (see openmeter/ent/schema/subscription_addon.go:56).

- **Location:** `openmeter/ent/schema/subscription_addon.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/subscription`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `created_at` | TIMESTAMPTZ |  |
| `updated_at` | TIMESTAMPTZ |  |
| `deleted_at` | TIMESTAMPTZ |  |
| `active_from` | TIMESTAMPTZ |  |
| `quantity` | INT |  |
| `subscription_addon_id` | TEXT |  |

**Data Guarantees:**
- PK id
- quantity >= 0, immutable
- FK subscription_addon_id → subscription_addons.id ON DELETE CASCADE

**Lifecycle:**
- *How to add:* Edit SubscriptionAddonQuantity.Fields() and regenerate.

### `CurrencyCostBasis` *(table)*

Time-effective exchange rate of a custom currency to a fiat code (see openmeter/ent/schema/custom_currencies.go:61).

- **Location:** `openmeter/ent/schema/custom_currencies.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/currency`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | TEXT |  |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ |  |
| `custom_currency_id` | char(26) |  |
| `fiat_code` | TEXT |  |
| `rate` | numeric |  |
| `effective_from` | TIMESTAMPTZ |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, custom_currency_id, fiat_code, effective_from) WHERE deleted_at IS NULL (see openmeter/ent/schema/custom_currencies.go:108)
- FK custom_currency_id → custom_currencies.id (required)

**Lifecycle:**
- *How to add:* Edit and regenerate.

### `NotificationEvent` *(table)*

A fired notification event instance carrying a JSON payload and the rule_id that produced it (see openmeter/ent/schema/notification.go:118).

- **Location:** `openmeter/ent/schema/notification.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/notification`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | TEXT |  |
| `annotations` | jsonb |  |
| `created_at` | TIMESTAMPTZ |  |
| `type` | enum |  |
| `rule_id` | char(26) |  |
| `payload` | jsonb |  |

**Data Guarantees:**
- PK id
- INDEX(namespace, type)
- FK rule_id → notification_rules.id (required)

**Lifecycle:**
- *How to add:* Edit NotificationEvent.Fields() and regenerate. Note: only created_at (no updated/deleted) — events are append-only.

### `AppStripeCustomer` *(table)*

Links a Stripe app to a Customer with the Stripe customer id and default payment method (see openmeter/ent/schema/app_stripe.go:62).

- **Location:** `openmeter/ent/schema/app_stripe.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/app/stripe`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `namespace` | TEXT |  |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ |  |
| `app_id` | char(26) |  |
| `customer_id` | char(26) |  |
| `stripe_customer_id` | TEXT |  |
| `stripe_default_payment_method_id` | TEXT |  |

**Data Guarantees:**
- UNIQUE(namespace, app_id, customer_id) WHERE deleted_at IS NULL
- UNIQUE(app_id, stripe_customer_id) WHERE deleted_at IS NULL (see openmeter/ent/schema/app_stripe.go:93)

**Lifecycle:**
- *How to add:* Edit and regenerate.

### `BillingInvoiceFlatFeeLineConfig` *(table)*

Per-unit amount + category + payment term config for a flat-fee invoice line (see openmeter/ent/schema/billing.go:511).

- **Location:** `openmeter/ent/schema/billing.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `per_unit_amount` | numeric |  |
| `category` | enum |  |
| `payment_term` | enum |  |
| `index` | INT | Sort order, only meaningful for detailed lines (see openmeter/ent/schema/billing.go:534). |

**Data Guarantees:**
- PK id

**Lifecycle:**
- *How to add:* Edit and regenerate.

### `BillingStandardInvoiceDetailedLine` *(table)*

Detailed (child) line of a standard invoice produced by the v2 line model, linked to a parent BillingInvoiceLine (see openmeter/ent/schema/billing.go:898).

- **Location:** `openmeter/ent/schema/billing.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `invoice_id` | char(26) |  |
| `parent_line_id` | char(26) |  |
| `child_unique_reference_id` | TEXT |  |
| `tax_code_id` | char(26) |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, parent_line_id, child_unique_reference_id) WHERE deleted_at IS NULL (see openmeter/ent/schema/billing.go:926)
- FK invoice_id, parent_line_id (required)

**Lifecycle:**
- *How to add:* Fields come mostly from stddetailedline.Mixin. Edit and regenerate.

### `CustomCurrency` *(table)*

A namespace-defined non-fiat currency (code/name/symbol) with a cost-basis history; e.g. for credit-token style billing (see openmeter/ent/schema/custom_currencies.go:16).

- **Location:** `openmeter/ent/schema/custom_currencies.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/currency`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | TEXT |  |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ |  |
| `code` | TEXT |  |
| `name` | TEXT |  |
| `symbol` | TEXT |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, code) WHERE deleted_at IS NULL (see openmeter/ent/schema/custom_currencies.go:51)
- cost_basis_history ON DELETE CASCADE

**Lifecycle:**
- *How to add:* Edit and regenerate.

### `LedgerCustomerAccount` *(table)*

Private link table mapping a customer to their ledger accounts — one FBO and one Receivable account per customer per namespace (see openmeter/ent/schema/ledger_customer_account.go:14).

- **Location:** `openmeter/ent/schema/ledger_customer_account.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/ledger`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | TEXT |  |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ |  |
| `customer_id` | TEXT |  |
| `account_type` | enum |  |
| `account_id` | TEXT |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, customer_id, account_type) — one FBO + one Receivable per customer per namespace (see openmeter/ent/schema/ledger_customer_account.go:38)

**Lifecycle:**
- *How to add:* Edit and regenerate. AGENTS.md notes that when credits.enabled is false, ledger account services are noop; backfills writing real ledger_customer_accounts rows must construct concrete adapters directly.

### `LedgerSubAccount` *(table)*

A sub-account keyed by (account_id, route_id) that holds ledger entries (see openmeter/ent/schema/ledger_account.go:48).

- **Location:** `openmeter/ent/schema/ledger_account.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/ledger`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | TEXT |  |
| `annotations` | jsonb |  |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ |  |
| `account_id` | char(26) |  |
| `route_id` | char(26) |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, account_id, route_id)
- FK account_id, route_id (required)

**Lifecycle:**
- *How to add:* Edit and regenerate.

### `LedgerTransaction` *(table)*

A booked transaction within a group, dated by booked_at, made up of balanced entries (see openmeter/ent/schema/ledger_transaction.go:13).

- **Location:** `openmeter/ent/schema/ledger_transaction.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/ledger`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | TEXT |  |
| `annotations` | jsonb |  |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ |  |
| `group_id` | char(26) |  |
| `booked_at` | TIMESTAMPTZ |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, id)
- INDEX(namespace, booked_at)
- FK group_id (required)

**Lifecycle:**
- *How to add:* Edit and regenerate.

### `SubscriptionBillingSyncState` *(table)*

Per-subscription bookkeeping for the billing sync bridge: whether it has billables, when it was last synced, and when the next sync is due (see openmeter/ent/schema/subscriptionbillingsync.go:13).

- **Location:** `openmeter/ent/schema/subscriptionbillingsync.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing/worker/subscriptionsync`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `subscription_id` | char(26) |  |
| `has_billables` | BOOL |  |
| `synced_at` | TIMESTAMPTZ |  |
| `next_sync_after` | TIMESTAMPTZ | Schedules the next reconciliation pass; NULL means no pending re-sync (see openmeter/ent/schema/subscriptionbillingsync.go:28). |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, subscription_id) (see openmeter/ent/schema/subscriptionbillingsync.go:41)
- FK subscription_id → subscriptions.id ON DELETE CASCADE

**Lifecycle:**
- *How to add:* Edit SubscriptionBillingSyncState.Fields() and regenerate. This table has no TimeMixin (no created/updated/deleted_at).

### `AppCustomInvoicing` *(table)*

Custom-invoicing app config toggling draft/issuing sync hooks (see openmeter/ent/schema/appcustominvoicing.go:15).

- **Location:** `openmeter/ent/schema/appcustominvoicing.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/app/custominvoicing`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | TEXT |  |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ |  |
| `enable_draft_sync_hook` | BOOL |  |
| `enable_issuing_sync_hook` | BOOL |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, id)

**Lifecycle:**
- *How to add:* Edit and regenerate.

### `AppCustomInvoicingCustomer` *(table)*

Links a custom-invoicing app to a Customer (see openmeter/ent/schema/appcustominvoicing.go:54).

- **Location:** `openmeter/ent/schema/appcustominvoicing.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/app/custominvoicing`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `namespace` | TEXT |  |
| `metadata` | jsonb |  |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ |  |
| `app_id` | char(26) |  |
| `customer_id` | char(26) |  |

**Data Guarantees:**
- UNIQUE(namespace, app_id, customer_id) WHERE deleted_at IS NULL

**Lifecycle:**
- *How to add:* Edit and regenerate.

### `CustomerSubjects` *(table)*

Join table linking a Customer to one or more subject keys (the metering identity); FK to Subject.key is intentionally absent because Ent cannot FK non-ID fields (see openmeter/ent/schema/customer.go:147).

- **Location:** `openmeter/ent/schema/customer.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/customer`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `namespace` | TEXT |  |
| `customer_id` | char(26) |  |
| `subject_key` | TEXT |  |
| `created_at` | TIMESTAMPTZ |  |
| `deleted_at` | TIMESTAMPTZ |  |

**Data Guarantees:**
- UNIQUE(namespace, subject_key) WHERE deleted_at IS NULL (see openmeter/ent/schema/customer.go:123)
- FK customer_id → customers.id ON DELETE CASCADE
- soft_delete (deleted_at)

**Lifecycle:**
- *How to add:* Edit CustomerSubjects in customer.go and regenerate.

### `LedgerAccount` *(table)*

A double-entry ledger account of a given account_type; parent of sub-accounts and routing rows (see openmeter/ent/schema/ledger_account.go:16).

- **Location:** `openmeter/ent/schema/ledger_account.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/ledger`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | TEXT |  |
| `annotations` | jsonb |  |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ |  |
| `account_type` | enum |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, id)

**Lifecycle:**
- *How to add:* Edit LedgerAccount.Fields() in ledger_account.go and regenerate. See the /ledger skill.

### `OrganizationDefaultTaxCodes` *(table)*

Per-namespace default tax codes for invoicing and credit grants; namespace declared explicitly (not via mixin) to keep a unique constraint (see openmeter/ent/schema/organizationdefaulttaxcodes.go:15).

- **Location:** `openmeter/ent/schema/organizationdefaulttaxcodes.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/taxcode`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ |  |
| `namespace` | TEXT |  |
| `invoicing_tax_code_id` | char(26) |  |
| `credit_grant_tax_code_id` | char(26) |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace) WHERE deleted_at IS NULL (see openmeter/ent/schema/organizationdefaulttaxcodes.go:57)
- FK both tax_code ids → tax_codes.id (required)

**Lifecycle:**
- *How to add:* Edit and regenerate.

### `AppCustomer` *(table)*

Join table linking an App install to a Customer (see openmeter/ent/schema/app.go:53).

- **Location:** `openmeter/ent/schema/app.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/app`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `namespace` | TEXT |  |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ |  |
| `app_id` | char(26) |  |
| `customer_id` | char(26) |  |

**Data Guarantees:**
- UNIQUE(namespace, app_id, customer_id) WHERE deleted_at IS NULL (see openmeter/ent/schema/app.go:77)

**Lifecycle:**
- *How to add:* Edit and regenerate.

### `LedgerTransactionGroup` *(table)*

Groups related ledger transactions that must be booked atomically (see openmeter/ent/schema/ledger_transaction_group.go:11).

- **Location:** `openmeter/ent/schema/ledger_transaction_group.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/ledger`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | TEXT |  |
| `annotations` | jsonb |  |
| `created_at/updated_at/deleted_at` | TIMESTAMPTZ |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, id)

**Lifecycle:**
- *How to add:* Edit and regenerate.

### `BillingCustomerLock` *(table)*

Row used as a per-customer advisory lock during billing mutations (one row per customer) (see openmeter/ent/schema/billing.go:1334).

- **Location:** `openmeter/ent/schema/billing.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) ULID |  |
| `namespace` | TEXT |  |
| `customer_id` | char(26) |  |

**Data Guarantees:**
- PK id
- UNIQUE(namespace, customer_id) (see openmeter/ent/schema/billing.go:1356)

**Lifecycle:**
- *How to add:* Edit and regenerate.
- *How to read:* Locked via SELECT ... FOR UPDATE during billing mutations.

### `BillingSequenceNumbers` *(table)*

Monotonic counter per (namespace, scope) for generating invoice numbers (see openmeter/ent/schema/billing.go:1308).

- **Location:** `openmeter/ent/schema/billing.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `namespace` | TEXT |  |
| `scope` | TEXT |  |
| `last` | numeric |  |

**Data Guarantees:**
- UNIQUE(namespace, scope) (see openmeter/ent/schema/billing.go:1330)

**Lifecycle:**
- *How to add:* Edit and regenerate. No id/audit columns — keyed by (namespace, scope).

### `BillingInvoiceWriteSchemaLevel` *(table)*

Temporary single-row-keyed table tracking the active write schema level for billing invoices during the line-model migration (see openmeter/ent/schema/billing.go:1363).

- **Location:** `openmeter/ent/schema/billing.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | TEXT |  |
| `schema_level` | INT |  |

**Data Guarantees:**
- PK id (UNIQUE)

**Lifecycle:**
- *How to add:* Temporary migration table; no mixins. Edit and regenerate.

## Persistence Stores

_Where data lives across process or session boundaries — databases, caches, queues, mobile local storage._

### `primary_postgres`

Source-of-truth control-plane store; schema is defined in Ent (openmeter/ent/schema/*.go) and migrated with Atlas-generated golang-migrate SQL. Almost all tables are namespace-scoped with ULID char(26) ids and deleted_at soft-deletes (see openmeter/ent/schema/customer.go:21).

- **Engine:** PostgreSQL (Ent ORM + Atlas migrations + pgx)
- **Role:** primary
- **Migrations dir:** `tools/migrate/migrations`
- **Lives here:** Customer, CustomerSubjects, Subject, Meter, Feature, Entitlement, Grant, BalanceSnapshot, UsageReset, Plan, PlanPhase, PlanRateCard, Addon, AddonRateCard, PlanAddon, Subscription, SubscriptionPhase, SubscriptionItem, SubscriptionAddon, SubscriptionAddonQuantity, SubscriptionBillingSyncState, BillingProfile, BillingWorkflowConfig, BillingCustomerOverride, BillingInvoice, BillingInvoiceLine, BillingInvoiceFlatFeeLineConfig, BillingInvoiceUsageBasedLineConfig, BillingInvoiceSplitLineGroup, BillingInvoiceLineDiscount, BillingInvoiceLineUsageDiscount, BillingStandardInvoiceDetailedLine, BillingStandardInvoiceDetailedLineAmountDiscount, BillingInvoiceValidationIssue, BillingSequenceNumbers, BillingCustomerLock, BillingInvoiceWriteSchemaLevel, Charge, ChargesSearchV1, ChargeFlatFee, ChargeFlatFeeRun, ChargeUsageBased, ChargeUsageBasedRuns, ChargeUsageBasedRunDetailedLine, ChargeCreditPurchase, CreditRealizationLineage, CreditRealizationLineageSegment, LedgerAccount, LedgerSubAccount, LedgerSubAccountRoute, LedgerTransactionGroup, LedgerTransaction, LedgerEntry, LedgerCustomerAccount, LedgerBreakageRecord, App, AppCustomer, AppStripe, AppStripeCustomer, AppCustomInvoicing, AppCustomInvoicingCustomer, TaxCode, OrganizationDefaultTaxCodes, CustomCurrency, CurrencyCostBasis, LLMCostPrice, NotificationChannel, NotificationRule, NotificationEvent, NotificationEventDeliveryStatus
- **Written by:** App / marketplace integrations, Billing domain, Charges sub-system, Credit & grant domain, Customer domain, Entitlement domain, Ledger domain, Meter domain, Notification domain, Product catalog domain, Subscription domain

### `redis`

Cache/coordination store: ingest event deduplication via SET NX keys with TTL (openmeter/dedupe/redisdedupe) and async query progress under progress:<ns>:<id> keys with TTL (openmeter/progressmanager/adapter) (see openmeter/dedupe/redisdedupe/redisdedupe.go:32).

- **Engine:** Redis (go-redis v9)
- **Role:** cache
- **Lives here:** dedupe.Item, Progress

### `clickhouse_events`

Append-only analytics store holding raw usage events in per-namespace MergeTree tables named om_<ns>_events; meters aggregate over it and usage-based billing reads metered quantities from it. Tables are created idempotently on connector startup, not via Atlas (see openmeter/streaming/clickhouse/event_query.go:20).

- **Engine:** ClickHouse (MergeTree)
- **Role:** analytics
- **Lives here:** RawEvent
- **Written by:** Streaming / usage query (ClickHouse)

### `kafka_ingest`

Per-namespace ingest event topics (template om_%s_events) that the API ingest path produces to and the sink-worker consumes (subscribing via regexp ^om_<ns>_events$) before writing to ClickHouse; plus system event topics om_sys.api_events / om_sys.ingest_events / om_sys.balance_worker_events and the notification DLQ om_sys.notification_service_dlq (see app/config/ingest.go:180, app/config/events.go:230).

- **Engine:** Kafka (confluent-kafka-go + Watermill)
- **Role:** queue