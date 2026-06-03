OpenMeter persists all domain state (billing invoices/lines, charges, customers, subscriptions, entitlements, credit grants, double-entry ledger, meters, features/plans, notifications, LLM cost prices) in a single PostgreSQL database via ~35 Ent schema structs that Atlas diffs into golang-migrate SQL files; raw usage events live append-only in a single shared ClickHouse MergeTree events table (queried by streaming.Connector), and Redis provides TTL-based ingest deduplication. Kafka (Watermill) is the cross-binary event bus. Every domain has a service/adapter pair; all writes go through entutils.TransactingRepo for ctx-bound transactions.

## Models

_Domain entities, DTOs, and value objects this codebase reads and writes._

### `BillingInvoice` *(table)*

An invoice that progresses through a stateless state machine; status='gathering' is the single pending-line collector per customer+currency, while standard invoices carry cloned profile/app settings and snapshot timestamps (see openmeter/ent/schema/billing.go:1190).

- **Location:** `openmeter/ent/schema/billing.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | String |  |
| `metadata` | jsonb |  |
| `supplier_name` | String |  |
| `supplier_tax_code` | String |  |
| `customer_key` | String |  |
| `customer_name` | String |  |
| `customer_usage_attribution` | jsonb |  |
| `number` | String |  |
| `type` | Enum (InvoiceType) |  |
| `description` | String |  |
| `customer_id` | char(26) |  |
| `source_billing_profile_id` | char(26) |  |
| `voided_at` | Time |  |
| `issued_at` | Time | May be set in the future for pre-issued invoices (see openmeter/ent/schema/billing.go:1087). |
| `sent_to_customer_at` | Time |  |
| `draft_until` | Time |  |
| `quantity_snapshoted_at` | Time |  |
| `currency` | varchar(3) |  |
| `due_at` | Time |  |
| `status` | Enum (StandardInvoiceStatus) |  |
| `status_details_cache` | jsonb |  |
| `workflow_config_id` | char(26) |  |
| `tax_app_id` | char(26) |  |
| `invoicing_app_id` | char(26) |  |
| `payment_app_id` | char(26) |  |
| `period_start` | Time |  |
| `period_end` | Time |  |
| `collection_at` | Time | On gathering invoices marks when pending lines are collected into a new draft; on standard invoices marks the post-creation metered-line collection/snapshot cutoff (see openmeter/ent/schema/billing.go:1154). |
| `payment_processing_entered_at` | Time | Timestamp the invoice first entered payment-processing state; used for staleness/fraud guards (see openmeter/ent/schema/billing.go:1164). |
| `schema_level` | Int | Schema level used when writing invoice data during the in-progress invoice-line migration (see openmeter/ent/schema/billing.go:1170). |
| `created_at` | Time |  |
| `updated_at` | Time |  |
| `deleted_at` | Time |  |

**Data Guarantees:**
- PK id
- soft_delete (deleted_at)
- audit (created_at, updated_at)
- UNIQUE(namespace, customer_id, currency) WHERE deleted_at IS NULL AND status='gathering' — one gathering invoice per customer per currency (openmeter/ent/schema/billing.go:1193)
- INDEX(namespace, customer_id)
- INDEX(namespace, status)
- GIN INDEX(status_details_cache)
- FK source_billing_profile_id → billing_profiles (immutable)
- FK customer_id → customers (immutable)

**Consumers:**
- `billing adapter (invoice.go)` — `openmeter/billing/adapter/invoice.go`: Ent read/write of invoices via TransactingRepo
- `billing service state machine` — `openmeter/billing/service/stdinvoicestate.go`: sole writer of status via the stateless state machine (FireAndActivate); direct status mutation forbidden

**Lifecycle:**
- *How to add:* Add a field to BillingInvoice.Fields() in openmeter/ent/schema/billing.go, run make generate to regenerate openmeter/ent/db/, then atlas migrate --env local diff <name> to emit the .up.sql/.down.sql pair and update atlas.sum. Commit schema + generated code + migration + atlas.sum together.

  ```
  // openmeter/ent/schema/billing.go
  field.Int("schema_level").Default(1),
  ```

- *How to modify:* Never edit a landed migration file or atlas.sum by hand. Change the Ent schema, regenerate, and produce a new timestamped migration via atlas migrate --env local diff. Field renames go through additive migrations.

  ```
  -- tools/migrate/migrations/20260520130500_add_ledger_tax_behavior.up.sql
  ALTER TABLE "ledger_sub_account_routes" ADD COLUMN "tax_behavior" character varying NULL;
  ```

- *How to read:* Always through billing.Service (composite interface) backed by the Ent adapter wrapped in entutils.TransactingRepo; never query openmeter/ent/db directly from a service.

  ```
  return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*Entity, error) { return tx.db.BillingInvoice.Query()... })
  ```

- *Tests:* `openmeter/billing/adapter/invoice.go`

### `BillingInvoiceLine` *(table)*

A tagged-union invoice line (flat-fee, usage-based, or detailed) for either a gathering or standard invoice; eventually intended to become the unified usage-based-pricing table, linking to charges, subscriptions, and split-line groups (see openmeter/ent/schema/billing.go:397).

- **Location:** `openmeter/ent/schema/billing.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | String |  |
| `name` | String |  |
| `description` | String |  |
| `metadata` | jsonb |  |
| `annotations` | jsonb |  |
| `currency` | varchar(3) |  |
| `tax_config` | jsonb |  |
| `period_start` | Time |  |
| `period_end` | Time |  |
| `invoice_id` | char(26) |  |
| `managed_by` | Enum (InvoiceLineManagedBy) |  |
| `parent_line_id` | char(26) |  |
| `invoice_at` | Time |  |
| `override_collection_period_end` | Time |  |
| `type` | Enum (InvoiceLineAdapterType) |  |
| `status` | Enum (InvoiceLineStatus) |  |
| `quantity` | numeric | Optional for usage-based lines; only persisted when the invoice is issued (see openmeter/ent/schema/billing.go:352). |
| `ratecard_discounts` | jsonb |  |
| `child_unique_reference_id` | String | Unique per parent line; used for upserting and matching lines created for the same reason (e.g. a price tier) across invoices (see openmeter/ent/schema/billing.go:370). |
| `subscription_id` | String |  |
| `subscription_phase_id` | String |  |
| `subscription_item_id` | String |  |
| `subscription_billing_period_from` | Time |  |
| `subscription_billing_period_to` | Time |  |
| `split_line_group_id` | char(26) |  |
| `charge_id` | char(26) |  |
| `engine` | Enum (LineEngineType) |  |
| `line_ids` | char(26) | Deprecated; invoice discounts are now in line_discounts (see openmeter/ent/schema/billing.go:416). |
| `credits_applied` | jsonb |  |
| `created_at` | Time |  |
| `updated_at` | Time |  |
| `deleted_at` | Time |  |

**Data Guarantees:**
- PK id
- soft_delete (deleted_at)
- audit (created_at, updated_at)
- UNIQUE(namespace, parent_line_id, child_unique_reference_id) WHERE child_unique_reference_id IS NOT NULL AND deleted_at IS NULL (openmeter/ent/schema/billing.go:440)
- INDEX(namespace, invoice_id)
- INDEX(namespace, subscription_id, subscription_phase_id, subscription_item_id)
- FK invoice_id → billing_invoices (required)
- FK charge_id → charges
- cascade-delete of flat_fee_line / usage_based_line / detailed_lines / discounts

**Consumers:**
- `billing adapter (invoice.go, invoicelinesplitgroup.go)` — `openmeter/billing/adapter/invoice.go`: diff-based upsert of line hierarchies via TransactingRepo
- `subscriptionsync service` — `openmeter/billing/worker/subscriptionsync`: reconciles subscription views into invoice lines (SynchronizeSubscription)

**Lifecycle:**
- *How to add:* Add the field in BillingInvoiceLine.Fields() in openmeter/ent/schema/billing.go (it already has >30 fields), regenerate with make generate, then atlas migrate --env local diff <name>. Update the billing adapter line mapping for the new column.

  ```
  field.String("charge_id").SchemaType(map[string]string{dialect.Postgres: "char(26)"}).Optional().Nillable(),
  ```

- *How to modify:* Mark removed columns Deprecated(...) rather than dropping immediately (see line_ids, several discount columns); generate a new migration for each change. Never hand-edit migrations.

  ```
  field.String("line_ids").Optional().Nillable().Deprecated("invoice discounts are deprecated, use line_discounts instead"),
  ```

- *How to read:* Through billing.Service; never construct billing.InvoiceLine{} literally — use NewStandardInvoiceLine/NewGatheringInvoiceLine so the private discriminator is set.

  ```
  line := billing.NewStandardInvoiceLine(billing.StandardInvoiceLineInput{...})
  ```


### `Entitlement` *(table)*

A feature entitlement of one of three sub-types (metered/boolean/static); feature_key is validated to reject ULIDs, and usage_period_anchor now keeps the original anchor while the effective anchor is derived from the last reset (see openmeter/ent/schema/entitlement.go:62).

- **Location:** `openmeter/ent/schema/entitlement.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/entitlement`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | String |  |
| `metadata` | jsonb |  |
| `entitlement_type` | Enum |  |
| `feature_id` | char(26) |  |
| `active_from` | Time |  |
| `active_to` | Time |  |
| `feature_key` | String | Validated to reject a value that parses as a ULID (see openmeter/ent/schema/entitlement.go:41). |
| `customer_id` | char(26) |  |
| `measure_usage_from` | Time |  |
| `issue_after_reset` | Float |  |
| `issue_after_reset_priority` | Uint8 |  |
| `is_soft_limit` | Bool |  |
| `preserve_overage_at_reset` | Bool |  |
| `config` | jsonb | JSON config value for static entitlements (see openmeter/ent/schema/entitlement.go:55). |
| `usage_period_interval` | ISODurationString |  |
| `usage_period_anchor` | Time | Original anchor time; effective anchor is populated from the last reset, queried dynamically (see openmeter/ent/schema/entitlement.go:62). |
| `current_usage_period_start` | Time |  |
| `current_usage_period_end` | Time |  |
| `annotations` | jsonb |  |
| `created_at` | Time |  |
| `updated_at` | Time |  |
| `deleted_at` | Time |  |

**Data Guarantees:**
- PK id
- soft_delete (deleted_at)
- audit (created_at, updated_at)
- INDEX(namespace, customer_id)
- INDEX(current_usage_period_end, deleted_at) — collects entitlements with due resets
- UNIQUE(created_at, id)
- FK feature_id → features (required, immutable)
- FK customer_id → customers (required, immutable)
- cascade-delete of usage_reset, grant, balance_snapshot

**Consumers:**
- `entitlement adapter` — `openmeter/entitlement/adapter/entitlement.go`: Ent read/write; acquires pg_advisory_lock per customer for multi-row mutations
- `balanceworker` — `openmeter/entitlement/balanceworker`: recalculates metered balances on Kafka lifecycle events using a high-watermark filter

**Lifecycle:**
- *How to add:* Add field in Entitlement.Fields() (openmeter/ent/schema/entitlement.go), regenerate, atlas diff. Sub-type behaviour lives in openmeter/entitlement/{metered,boolean,static}.

  ```
  field.Time("current_usage_period_end").Optional().Nillable(),
  ```

- *How to modify:* Standard Ent + atlas diff. Multi-row entitlement mutations for one customer must hold a pg_advisory_lock via lockr inside a transaction.
- *How to read:* Through entitlement.Service (GetEntitlement/GetEntitlementValue/GetAccess); metered balance uses the credit engine over ClickHouse usage.

  ```
  entitlement.Service.GetEntitlementValue(ctx, ...)
  ```

- *Tests:* `openmeter/entitlement/adapter/entitlement_test.go`

### `Grant` *(table)*

A credit grant burned down for a metered entitlement; amount/rollover are numeric and recurrence is an ISO duration with anchor (see openmeter/ent/schema/grant.go:28).

- **Location:** `openmeter/ent/schema/grant.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/credit`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | String |  |
| `annotations` | jsonb |  |
| `metadata` | jsonb |  |
| `owner_id` | char(26) | Entitlement that owns the grant (FK to entitlements.id) (see openmeter/ent/schema/grant.go:68). |
| `amount` | numeric |  |
| `priority` | Uint8 |  |
| `effective_at` | Time |  |
| `expiration` | jsonb |  |
| `expires_at` | Time |  |
| `voided_at` | Time |  |
| `reset_max_rollover` | numeric |  |
| `reset_min_rollover` | numeric |  |
| `recurrence_period` | ISODurationString |  |
| `recurrence_anchor` | Time |  |
| `created_at` | Time |  |
| `updated_at` | Time |  |
| `deleted_at` | Time |  |

**Data Guarantees:**
- PK id
- soft_delete (deleted_at)
- audit (created_at, updated_at)
- INDEX(namespace, owner_id)
- INDEX(effective_at, expires_at)
- FK owner_id → entitlements (required, immutable)

**Consumers:**
- `credit engine` — `openmeter/credit/engine`: computes grant burn-down without I/O; all effective times truncated to time.Minute
- `balanceworker` — `openmeter/entitlement/balanceworker`: reads grants when recalculating entitlement balances

**Lifecycle:**
- *How to add:* Add field in Grant.Fields() (openmeter/ent/schema/grant.go), regenerate, atlas diff.

  ```
  field.Float("amount").Immutable().SchemaType(map[string]string{dialect.Postgres: "numeric"}),
  ```

- *How to modify:* Standard Ent + atlas diff. Always truncate grant effective times to time.Minute before storing/computing.

  ```
  effectiveAt := time.Now().Truncate(time.Minute)
  ```

- *How to read:* Through credit.CreditConnector (CreateGrant/VoidGrant/GetBalanceAt); never via the adapter directly.

  ```
  creditConnector.CreateGrant(ctx, credit.CreateGrantInput{EffectiveAt: effectiveAt})
  ```


### `Subscription` *(table)*

A customer subscription against a versioned plan, with default billing cadence/anchor and pro-rating config; settlement_mode (credit-then-invoice by default) is immutable (see openmeter/ent/schema/subscription.go:57).

- **Location:** `openmeter/ent/schema/subscription.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/subscription`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | String |  |
| `annotations` | jsonb |  |
| `metadata` | jsonb |  |
| `name` | String |  |
| `description` | String |  |
| `plan_id` | String |  |
| `customer_id` | String |  |
| `currency` | varchar(3) |  |
| `billing_anchor` | Time |  |
| `billing_cadence` | ISODurationString |  |
| `pro_rating_config` | jsonb |  |
| `settlement_mode` | Enum (SettlementMode) | Immutable; defaults to credit-then-invoice (see openmeter/ent/schema/subscription.go:57). |
| `active_from` | Time (CadencedMixin) |  |
| `active_to` | Time (CadencedMixin) |  |
| `created_at` | Time |  |
| `updated_at` | Time |  |
| `deleted_at` | Time |  |

**Data Guarantees:**
- PK id
- soft_delete (deleted_at)
- audit (created_at, updated_at)
- cadenced (active_from, active_to)
- INDEX(namespace, customer_id)
- FK customer_id → customers (required, immutable)
- FK plan_id → plans
- cascade-delete of phases, addons, billing_sync_state

**Consumers:**
- `subscription service` — `openmeter/subscription/service`: manages lifecycle via SubscriptionSpec + AppliesToSpec patches
- `SubscriptionBillingSyncState` — `openmeter/ent/schema/subscriptionbillingsync.go`: tracks billing sync reconciliation per subscription

**Lifecycle:**
- *How to add:* Add field in Subscription.Fields() (openmeter/ent/schema/subscription.go); SubscriptionItem deliberately duplicates RateCard fields for snapshot immutability, so RateCard changes must be mirrored manually. Regenerate + atlas diff.

  ```
  field.Enum("settlement_mode").GoType(productcatalog.SettlementMode("")).Default(string(productcatalog.CreditThenInvoiceSettlementMode)).Immutable(),
  ```

- *How to modify:* Mutate SubscriptionSpec only via the AppliesToSpec patch interface (ApplyTo); never modify spec fields directly. Schema changes go through atlas diff.
- *How to read:* Through subscription.Service (Get/GetView/List/ExpandViews) and subscriptionworkflow.Service for higher-level operations.

  ```
  subscription.Service.GetView(ctx, id)
  ```


### `Meter` *(table)*

An event-aggregation rule: event_type + aggregation function (COUNT/SUM/MAX/UNIQUE_COUNT) over an optional value_property and group_by JSON paths; event_from optionally bounds the included event window (see openmeter/ent/schema/meter.go:30).

- **Location:** `openmeter/ent/schema/meter.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/meter`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | String |  |
| `key` | String |  |
| `name` | String |  |
| `description` | String |  |
| `metadata` | jsonb |  |
| `annotations` | jsonb |  |
| `event_type` | String |  |
| `value_property` | String | JSON path to the numeric value; optional for COUNT aggregation (see openmeter/ent/schema/meter.go:31). |
| `group_by` | jsonb |  |
| `aggregation` | Enum (MeterAggregation) |  |
| `event_from` | Time | If set, only events since this time are included (see openmeter/ent/schema/meter.go:35). |
| `created_at` | Time |  |
| `updated_at` | Time |  |
| `deleted_at` | Time |  |

**Data Guarantees:**
- PK id
- soft_delete (deleted_at)
- audit (created_at, updated_at)
- UNIQUE(namespace, key) WHERE deleted_at IS NULL (openmeter/ent/schema/meter.go:43)
- INDEX(namespace, event_type)

**Consumers:**
- `meter adapter` — `openmeter/meter/adapter/meter.go`: Ent CRUD of meter definitions
- `streaming clickhouse meter_query` — `openmeter/streaming/clickhouse/meter_query.go`: compiles a Meter into ClickHouse aggregation SQL over the shared events table

**Lifecycle:**
- *How to add:* Add field in Meter.Fields() (openmeter/ent/schema/meter.go), regenerate, atlas diff. Meter definitions also drive ClickHouse query construction, so map new aggregation/value semantics into openmeter/streaming/clickhouse/meter_query.go.

  ```
  field.Time("event_from").Optional().Nillable(),
  ```

- *How to modify:* Standard Ent + atlas diff; mutations publish MeterUpdateEvent.
- *How to read:* Through meter.Service (ListMeters/GetMeterByIDOrSlug) and meter.ManageService for mutation; usage queries go through streaming.Connector.QueryMeter.

  ```
  meter.Service.GetMeterByIDOrSlug(ctx, ns, slug)
  ```

- *Tests:* `openmeter/meter/adapter/adapter_test.go`

### `Customer` *(table)*

A billable customer; key is stored as empty string (not NULL) when unset so a partial unique index can enforce per-namespace key uniqueness (see openmeter/ent/schema/customer.go:33).

- **Location:** `openmeter/ent/schema/customer.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/customer`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | String |  |
| `name` | String |  |
| `metadata` | jsonb |  |
| `annotations` | jsonb |  |
| `key` | String |  |
| `primary_email` | String |  |
| `currency` | varchar(3) |  |
| `created_at` | Time |  |
| `updated_at` | Time |  |
| `deleted_at` | Time |  |

**Data Guarantees:**
- PK id
- soft_delete (deleted_at)
- audit (created_at, updated_at)
- UNIQUE(namespace, key) WHERE deleted_at IS NULL (openmeter/ent/schema/customer.go:57)
- INDEX(name)
- INDEX(primary_email)
- cascade-delete of apps, subjects, billing_customer_override

**Consumers:**
- `customer adapter` — `openmeter/customer/adapter/customer.go`: Ent read/write of customers with soft-delete semantics
- `RequestValidatorRegistry` — `openmeter/customer/requestvalidator.go`: pre-mutation cross-domain guards (billing/entitlement) run before any write

**Lifecycle:**
- *How to add:* Add field in Customer.Fields() in openmeter/ent/schema/customer.go, regenerate, then atlas migrate diff. Note the schema comment: v3 ILIKE filters on name/primary_email/key need pg_trgm GIN indexes via a custom SQL migration before exposing the list handler (see openmeter/ent/schema/customer.go:41).

  ```
  field.String("primary_email").Optional().Nillable(),
  ```

- *How to modify:* Standard Ent-schema-change + atlas diff flow; never edit landed migrations.
- *How to read:* Through customer.Service (ListCustomers/GetCustomer/GetCustomerByUsageAttribution); soft-deleted rows excluded by the partial index conventions.

  ```
  customer.Service.GetCustomer(ctx, input)
  ```

- *Tests:* `openmeter/customer/adapter/customer_test.go`

### `RawEvent` *(entity)*

A raw CloudEvent usage row in the single shared ClickHouse MergeTree events table; written append-only by the sink worker, queried by meter aggregations; store_row_id (ULID) is the v2-pagination tie-breaker for same-second events (see openmeter/streaming/connector.go:24, openmeter/streaming/clickhouse/event_query.go:36).

- **Location:** `openmeter/streaming/connector.go`
- **Store:** `clickhouse_events`
- **Owner:** `openmeter/streaming`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `namespace` | String |  |
| `id` | String |  |
| `type` | LowCardinality(String) |  |
| `source` | String |  |
| `subject` | String |  |
| `time` | DateTime |  |
| `data` | String |  |
| `ingested_at` | DateTime |  |
| `stored_at` | DateTime | Set when the sink writes the row; used for stored-at cutoff filtering in metered billing finalization (see openmeter/streaming/clickhouse/event_query.go:34). |
| `store_row_id` | String | Per-row ULID generated at insert; cursor tie-breaker in v2 listing since DateTime is second-precision (see openmeter/sink/storage.go:63). |
| `customer_id` | String (Nullable, query-time WITH map) |  |

**Data Guarantees:**
- ENGINE = MergeTree (no PK; not deduplicated by the engine — dedup is upstream in Redis)
- PARTITION BY toYYYYMM(time) (openmeter/streaming/clickhouse/event_query.go:38)
- ORDER BY (namespace, type, subject, toStartOfHour(time)) (openmeter/streaming/clickhouse/event_query.go:45)
- INDEX stored_at minmax GRANULARITY 4
- append-only (no UPDATE/DELETE in the write path)

**Consumers:**
- `ClickHouseStorage.BatchInsert` — `openmeter/sink/storage.go`: sole writer; maps SinkMessage → RawEvent and batch-inserts via streaming.Connector
- `clickhouse meter_query / event_query` — `openmeter/streaming/clickhouse/meter_query.go`: reads via toSQL() query structs for aggregation and event listing

**Lifecycle:**
- *How to add:* Add a `ch:` tagged field to streaming.RawEvent in openmeter/streaming/connector.go, then add the column to the DDL in createEventsTable.toSQL() and to the INSERT column list in openmeter/streaming/clickhouse/event_query.go (column order must match the table exactly). createEventsTable uses CREATE TABLE IF NOT EXISTS so new columns on existing deployments need an explicit ALTER TABLE migration path.

  ```
  // event_query.go createEventsTable.toSQL()
  sb.Define("stored_at", "DateTime")
  sb.Define("store_row_id", "String")
  sb.SQL("ENGINE = MergeTree")
  sb.SQL("PARTITION BY toYYYYMM(time)")
  sb.SQL("ORDER BY (namespace, type, subject, toStartOfHour(time))")
  ```

- *How to modify:* ClickHouse schema is created by the connector at startup (createTable), not by Atlas/golang-migrate. Changing column types requires coordinating the DDL builder and the INSERT/SELECT column lists; there is a single shared table across all namespaces.
- *How to read:* Always through streaming.Connector (QueryMeter/ListEvents/ListEventsV2); query logic lives in clickhouse/ query-structs with toSQL(), never inline SQL in connector method bodies.

  ```
  rows, err := connector.QueryMeter(ctx, namespace, meter, params)
  ```

- *Tests:* `openmeter/streaming/clickhouse/event_query_test.go`, `openmeter/streaming/clickhouse/event_query_v2_test.go`, `openmeter/streaming/clickhouse/meter_query_test.go`

### `NotificationEvent` *(table)*

A notification event instance carrying a versioned jsonb payload for a rule; delivery is tracked separately via NotificationEventDeliveryStatus (see openmeter/ent/schema/notification.go:130).

- **Location:** `openmeter/ent/schema/notification.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/notification`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | String |  |
| `annotations` | jsonb |  |
| `created_at` | Time |  |
| `type` | Enum (EventType) |  |
| `rule_id` | char(26) |  |
| `payload` | jsonb | Version-pinned event payload; payload version is an API contract for webhook consumers (see openmeter/ent/schema/notification.go:143). |

**Data Guarantees:**
- PK id
- INDEX(namespace, type)
- FK rule_id → notification_rules (required, immutable)

**Consumers:**
- `notification consumer` — `openmeter/notification/consumer`: builds payload and dispatches to webhook.Handler (Svix) via the system events topic
- `notification eventhandler` — `openmeter/notification/eventhandler`: runs Dispatch + Reconcile loops; Reconcile owns retry

**Lifecycle:**
- *How to add:* Add field in NotificationEvent.Fields() (openmeter/ent/schema/notification.go), regenerate, atlas diff. Channel/Rule config polymorphism uses ChannelConfigValueScanner/RuleConfigValueScanner type-switch serializers in the same file — a new ChannelType/EventType requires updating both the V (marshal) and S (unmarshal) switches.

  ```
  field.String("payload").SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
  ```

- *How to modify:* Pin a new payload version per event family rather than changing struct shape in place; standard Ent + atlas diff.
- *How to read:* Through notification.Service (ListEvents/GetEvent/ResendEvent) and notification.EventHandler.Dispatch.

  ```
  notification.Service.GetEvent(ctx, id)
  ```


### `dedupe.Item` *(key_value)*

Composite ingest-dedup key (namespace-source-id) written to Redis with NX+TTL to suppress double-counting of retried CloudEvents; hashed to a compact xxh3 key in keyhash mode (see openmeter/dedupe/redisdedupe/redisdedupe.go:45).

- **Location:** `openmeter/dedupe/dedupe.go`
- **Store:** `redis_dedupe`
- **Owner:** `openmeter/dedupe`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `Namespace` | string |  |
| `ID` | string |  |
| `Source` | string |  |

**Data Guarantees:**
- key = namespace-source-id (Item.Key())
- TTL d.Expiration (config-driven)
- SET NX (set-if-not-absent) — pre-existing key signals duplicate (openmeter/dedupe/redisdedupe/redisdedupe.go:82)
- value is empty/nil (only key presence matters)

**Consumers:**
- `redisdedupe.Deduplicator` — `openmeter/dedupe/redisdedupe/redisdedupe.go`: IsUnique/CheckUniqueBatch/Set — third phase of the sink flush, updated AFTER Kafka offset commit
- `memorydedupe.Deduplicator` — `openmeter/dedupe/memorydedupe/memorydedupe.go`: LRU fallback when Redis is not configured

**Lifecycle:**
- *How to add:* Dedup keys are not schema-migrated. Changing the key encoding requires adding a new DedupeMode and updating the mode-switch in every method (IsUnique/CheckUnique/Set/CheckUniqueBatch) plus a key rotation plan.

  ```
  switch d.Mode {
  case DedupeModeRawKey: keys = append(keys, item.Key())
  case DedupeModeKeyHash, DedupeModeKeyHashMigration: keys = append(keys, GetKeyHash(item.Key()))
  }
  ```

- *How to modify:* Never change GetKeyHash (xxh3-128 + base64url) silently — use keyhash-migration mode which checks both old rawkey and new hashed key during rollout.
- *How to read:* Through the dedupe.Deduplicator interface only; the sink updates Redis as the third flush phase strictly after the Kafka offset commit.

  ```
  isUnique, err := deduplicator.IsUnique(ctx, namespace, ev)
  ```

- *Tests:* `openmeter/dedupe/memorydedupe/memorydedupe_test.go`

### `Feature` *(table)*

A meter-backed usage feature with optional unit-cost configuration (manual amount or LLM provider/model/token-type properties, enforced mutually-exclusive by CHECK constraints); archived_at provides archive instead of hard delete (see openmeter/ent/schema/feature.go:53).

- **Location:** `openmeter/ent/schema/feature.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/productcatalog/feature`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | String |  |
| `metadata` | jsonb |  |
| `name` | String |  |
| `description` | String |  |
| `key` | String |  |
| `meter_slug` | String | Deprecated; use meter_id (see openmeter/ent/schema/feature.go:35). |
| `meter_id` | String |  |
| `meter_group_by_filters` | jsonb |  |
| `advanced_meter_group_by_filters` | jsonb |  |
| `unit_cost_type` | String |  |
| `unit_cost_manual_amount` | numeric |  |
| `unit_cost_llm_provider_property` | String |  |
| `unit_cost_llm_provider` | String |  |
| `unit_cost_llm_model_property` | String |  |
| `unit_cost_llm_model` | String |  |
| `unit_cost_llm_token_type_property` | String |  |
| `unit_cost_llm_token_type` | String |  |
| `archived_at` | Time |  |
| `created_at` | Time |  |
| `updated_at` | Time |  |

**Data Guarantees:**
- PK id
- audit (created_at, updated_at)
- UNIQUE(namespace, key) WHERE archived_at IS NULL (openmeter/ent/schema/feature.go:66)
- CHECK unit_cost_llm_{provider,model,token_type}_mutual_exclusive (property xor literal) (openmeter/ent/schema/feature.go:55)
- FK meter_id → meters

**Consumers:**
- `feature.FeatureConnector` — `openmeter/productcatalog/feature`: CreateFeature/ArchiveFeature/ResolveFeatureMeters

**Lifecycle:**
- *How to add:* Add field in Feature.Fields() (openmeter/ent/schema/feature.go); if it participates in mutual-exclusivity, add a CHECK in Feature.Annotations(). Regenerate + atlas diff.

  ```
  entsql.Checks(map[string]string{"unit_cost_llm_model_mutual_exclusive": "NOT (unit_cost_llm_model_property IS NOT NULL AND unit_cost_llm_model IS NOT NULL)"})
  ```

- *How to modify:* Standard Ent + atlas diff; features are archived (archived_at), not soft-deleted via deleted_at.
- *How to read:* Through feature.FeatureConnector (ListFeatures/GetFeature).

### `BalanceSnapshot` *(table)*

A point-in-time snapshot of grant balances/usage/overage for an entitlement owner, used to avoid recomputing burn-down from the beginning (see openmeter/ent/schema/balance_snapshot.go:31).

- **Location:** `openmeter/ent/schema/balance_snapshot.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/credit`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `namespace` | String |  |
| `owner_id` | char(26) |  |
| `grant_balances` | jsonb |  |
| `usage` | jsonb |  |
| `balance` | numeric |  |
| `overage` | numeric |  |
| `at` | Time |  |
| `created_at` | Time |  |
| `updated_at` | Time |  |
| `deleted_at` | Time |  |

**Data Guarantees:**
- soft_delete (deleted_at)
- audit (created_at, updated_at)
- INDEX(namespace, owner_id, at) WHERE deleted_at IS NULL
- FK owner_id → entitlements (required, immutable)

**Consumers:**
- `credit balance connector` — `openmeter/credit/balance`: reads/writes balance snapshots to bound burn-down recomputation

**Lifecycle:**
- *How to add:* Add field in BalanceSnapshot.Fields() (openmeter/ent/schema/balance_snapshot.go), regenerate, atlas diff. Note this entity omits IDMixin (no surrogate PK).

  ```
  field.Float("overage").Immutable().SchemaType(map[string]string{dialect.Postgres: "numeric"}),
  ```

- *How to modify:* Standard Ent + atlas diff.
- *How to read:* Through credit.CreditConnector balance APIs.

### `LedgerEntry` *(table)*

A single signed amount posting against a ledger sub-account within a transaction; (transaction_id, sub_account_id, identity_key) is unique to make postings idempotent (see openmeter/ent/schema/ledger_entry.go:67).

- **Location:** `openmeter/ent/schema/ledger_entry.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/ledger`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | String |  |
| `annotations` | jsonb |  |
| `sub_account_id` | char(26) |  |
| `identity_key` | String | Dedup key making a posting idempotent within (transaction_id, sub_account_id) (see openmeter/ent/schema/ledger_entry.go:67). |
| `amount` | numeric |  |
| `transaction_id` | char(26) |  |
| `created_at` | Time |  |
| `updated_at` | Time |  |
| `deleted_at` | Time |  |

**Data Guarantees:**
- PK id
- soft_delete (deleted_at)
- audit (created_at, updated_at)
- UNIQUE(namespace, id)
- UNIQUE(transaction_id, sub_account_id, identity_key) (openmeter/ent/schema/ledger_entry.go:67)
- FK transaction_id → ledger_transactions (required, immutable)
- FK sub_account_id → ledger_sub_accounts (required, immutable)

**Consumers:**
- `ledger transactions resolver` — `openmeter/ledger/transactions`: ResolveTransactions builds entries from typed templates; never hand-construct entries

**Lifecycle:**
- *How to add:* Add field in LedgerEntry.Fields() (openmeter/ent/schema/ledger_entry.go), regenerate, atlas diff. Ledger tables are only written when credits.enabled=true.

  ```
  field.Other("amount", alpacadecimal.Decimal{}).Immutable().SchemaType(map[string]string{dialect.Postgres: "numeric"}),
  ```

- *How to modify:* Standard Ent + atlas diff.
- *How to read:* Through ledger.Ledger (CommitGroup/QueryBalance); transaction inputs constructed only via transactions.ResolveTransactions with typed templates.

  ```
  entries, err := transactions.ResolveTransactions(ctx, ...); ledger.CommitGroup(ctx, entries)
  ```


### `Charge` *(table)*

Pivot entity holding one row per charge regardless of type, with an FK to exactly one of the three type-specific tables (flat_fee, credit_purchase, usage_based); enables a single FK from invoice lines to any charge type (see openmeter/ent/schema/charges.go:125).

- **Location:** `openmeter/ent/schema/charges.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing/charges`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | String |  |
| `created_at` | Time |  |
| `deleted_at` | Time |  |
| `unique_reference_id` | String |  |
| `type` | Enum (meta.ChargeType) |  |
| `charge_flat_fee_id` | String | FK to the flat-fee sub-table; exactly one of the three *_id fields is set per row, mandating sub-type existence (see openmeter/ent/schema/charges.go:125). |
| `charge_credit_purchase_id` | String |  |
| `charge_usage_based_id` | String |  |

**Data Guarantees:**
- PK id
- soft_delete (deleted_at)
- UNIQUE(namespace, unique_reference_id) WHERE unique_reference_id IS NOT NULL AND deleted_at IS NULL (openmeter/ent/schema/charges.go:167)
- FK to exactly one of charge_flat_fee / charge_credit_purchase / charge_usage_based (immutable)

**Consumers:**
- `charges adapter (search.go)` — `openmeter/billing/charges/adapter/search.go`: reads charges via the ChargesSearchV1 union view; helpers must wrap a.db in TransactingRepo

**Lifecycle:**
- *How to add:* Charge fields are added in openmeter/ent/schema/charges.go; per-type fields go in chargesflatfee.go / chargesusagebased.go / chargescreditpurchase.go. When adding a column to chargemeta.Mixin, also extend chargesSearchV1Columns in charges.go or the ChargesSearchV1 union view breaks. Run make generate then atlas migrate diff (and make generate-view-sql for the view).

  ```
  // charges.go: chargesSearchV1Columns must list every column present in each charge sub-table
  var chargesSearchV1Columns = []string{"id", "namespace", ... , "tax_behavior"}
  ```

- *How to modify:* ChargesSearchV1 is an ent.View; Atlas does not diff views, so view DDL changes require make generate-view-sql + an explicit SQL migration (see AGENTS.md ent.View caveat).
- *How to read:* Drive charge lifecycle exclusively through charges.Service (Create/AdvanceCharges/ApplyPatches); never construct charges.Charge{} literally and never call the adapter directly from outside the domain.

  ```
  charge := charges.NewCharge(flatfee.Charge{...}); fc, err := charge.AsFlatFeeCharge()
  ```

- *Tests:* `openmeter/billing/charges/adapter/search_test.go`

### `LedgerCustomerAccount` *(table)*

Private linking table mapping a customer to their ledger accounts (one FBO and one Receivable per customer per namespace); intentionally has no edges/FKs to LedgerAccount to avoid import cycles (see openmeter/ent/schema/ledger_customer_account.go:12).

- **Location:** `openmeter/ent/schema/ledger_customer_account.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/ledger`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | String |  |
| `customer_id` | String |  |
| `account_type` | Enum (ledger.AccountType) |  |
| `account_id` | String |  |
| `created_at` | Time |  |
| `updated_at` | Time |  |
| `deleted_at` | Time |  |

**Data Guarantees:**
- PK id
- soft_delete (deleted_at)
- audit (created_at, updated_at)
- UNIQUE(namespace, id)
- UNIQUE(namespace, customer_id, account_type) — one FBO and one Receivable per customer per namespace (openmeter/ent/schema/ledger_customer_account.go:38)
- no edges (deliberate, to avoid import cycles)

**Consumers:**
- `ledger resolvers adapter` — `openmeter/ledger/resolvers/adapter/repo.go`: links customers to FBO/Receivable accounts

**Lifecycle:**
- *How to add:* Add field in LedgerCustomerAccount.Fields(), regenerate, atlas diff.
- *How to modify:* Standard Ent + atlas diff.
- *How to read:* Through ledger resolver adapters; rows only written when credits.enabled=true (otherwise concrete adapters must be constructed directly per AGENTS.md).

### `LedgerAccount` *(table)*

A double-entry ledger account of a typed kind (FBO/Receivable/Accrued for customers; Wash/Earnings/Brokerage for business), with sub-accounts routed by routing keys (see openmeter/ent/schema/ledger_account.go:30).

- **Location:** `openmeter/ent/schema/ledger_account.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/ledger`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | String |  |
| `annotations` | jsonb |  |
| `account_type` | Enum (ledger.AccountType) |  |
| `created_at` | Time |  |
| `updated_at` | Time |  |
| `deleted_at` | Time |  |

**Data Guarantees:**
- PK id
- soft_delete (deleted_at)
- audit (created_at, updated_at)
- UNIQUE(namespace, id)

**Consumers:**
- `ledger AccountResolver` — `openmeter/ledger`: EnsureCustomerAccounts/EnsureBusinessAccounts provision typed accounts

**Lifecycle:**
- *How to add:* Add field in LedgerAccount.Fields() (openmeter/ent/schema/ledger_account.go), regenerate, atlas diff.

  ```
  field.String("account_type").GoType(ledger.AccountType("")).Immutable(),
  ```

- *How to modify:* Standard Ent + atlas diff.
- *How to read:* Through ledger.AccountResolver / ledger.Ledger; noop implementations are wired when credits.enabled=false.

### `LLMCostPrice` *(table)*

Persisted LLM model price: a global synced price or a per-namespace override; model IDs must be normalized via llmcost.NormalizeModelID before store/resolve (see .claude/rules/architecture.md openmeter/llmcost).

- **Location:** `openmeter/ent/schema/llmcostprice.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/llmcost`
**Data Guarantees:**
- PK id (IDMixin)
- namespace-scoped (NamespaceMixin)
- audit (created_at, updated_at) (TimeMixin)

**Consumers:**
- `llmcost.Service` — `openmeter/llmcost`: ResolvePrice with namespace-override precedence; CreateOverride/DeleteOverride

**Lifecycle:**
- *How to add:* Add field in LLMCostPrice.Fields() (openmeter/ent/schema/llmcostprice.go), regenerate, atlas diff. (Field list not enumerated here — fields are in the schema file.)
- *How to modify:* Standard Ent + atlas diff.
- *How to read:* Through llmcost.Service; always NormalizeModelID before resolving/storing.

  ```
  price, err := svc.ResolvePrice(ctx, llmcost.ResolvePriceInput{ModelID: llmcost.NormalizeModelID(raw)})
  ```


## Persistence Stores

_Where data lives across process or session boundaries — databases, caches, queues, mobile local storage._

### `primary_postgres`

Authoritative relational store for all ~35 Ent entities; schema is defined in openmeter/ent/schema/*.go and applied at startup via golang-migrate over Atlas-generated SQL (see tools/migrate/migrate.go).

- **Engine:** PostgreSQL 14.20-alpine (dev compose; docs reference 15)
- **Role:** primary
- **Migrations dir:** `tools/migrate/migrations`
- **Lives here:** BillingInvoice, BillingInvoiceLine, Charge, Customer, Subscription, Entitlement, Grant, BalanceSnapshot, LedgerEntry, LedgerAccount, LedgerCustomerAccount, Meter, Feature, NotificationEvent, LLMCostPrice
- **Written by:** openmeter/billing (billing domain), openmeter/credit (credit grants), openmeter/customer (customer domain), openmeter/entitlement (entitlement domain), openmeter/ledger (double-entry ledger), openmeter/meter + ingest + sink + streaming (usage pipeline), openmeter/notification (notification domain), openmeter/productcatalog (catalog domain), openmeter/subscription (subscription domain)

### `redis_dedupe`

TTL-based ingest deduplication store using SET NX on namespace-source-id keys; updated as the third (last) phase of the sink flush, strictly after Kafka offset commit (see openmeter/dedupe/redisdedupe/redisdedupe.go:82).

- **Engine:** Redis 7.4.7
- **Role:** cache
- **Lives here:** dedupe.Item

### `clickhouse_events`

Single shared append-only MergeTree events table across all namespaces (namespace is the leading ORDER BY column); table DDL is created by the connector at startup, not by Atlas (see openmeter/streaming/clickhouse/event_query.go:20).

- **Engine:** ClickHouse 25.12.3-alpine
- **Role:** analytics
- **Lives here:** RawEvent

### `kafka_topics`

Durable cross-binary event bus with three name-prefix-routed topics (ingest, system, balance-worker) via Watermill; sole inter-binary channel, also carries raw ingest CloudEvents consumed by the sink worker (see .claude/rules/architecture.md openmeter/watermill).

- **Engine:** Kafka (confluentinc/cp-kafka 8.0.3, confluent-kafka-go v2.14.1)
- **Role:** queue