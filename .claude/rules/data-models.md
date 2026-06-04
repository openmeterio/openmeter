OpenMeter persists all domain state (billing invoices/lines, charges, customers, subscriptions, entitlements, credit grants, double-entry ledger, meters, features, notifications, LLM cost prices) in one PostgreSQL database via ~35 hand-written Ent schema structs in openmeter/ent/schema that Atlas diffs into golang-migrate SQL files under tools/migrate/migrations; raw usage CloudEvents live append-only in a single shared ClickHouse MergeTree events table created by the connector at startup (openmeter/streaming/clickhouse/event_query.go), Redis provides TTL+SET-NX ingest deduplication (openmeter/dedupe), and Kafka (Watermill) is the cross-binary event bus. Every domain has a Service/Adapter pair and all writes go through entutils.TransactingRepo for ctx-bound transactions.

## Models

_Domain entities, DTOs, and value objects this codebase reads and writes._

### `BillingInvoice` *(table)*

An invoice progressing through a stateless state machine; status='gathering' is the single pending-line collector per customer+currency, while standard invoices clone profile/app settings and snapshot timestamps (see openmeter/ent/schema/billing.go:1170).

- **Location:** `openmeter/ent/schema/billing.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | String |  |
| `metadata` | jsonb |  |
| `voided_at` | Time |  |
| `issued_at` | Time | May be set in the future for pre-issued invoices (see openmeter/ent/schema/billing.go:1082). |
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
| `collection_at` | Time | On gathering invoices marks when pending lines are collected into a new draft; on standard invoices marks the post-creation metered-line collection/snapshot cutoff (see openmeter/ent/schema/billing.go:1148). |
| `payment_processing_entered_at` | Time | Timestamp the invoice first entered payment-processing state; used for staleness/fraud guards (see openmeter/ent/schema/billing.go:1160). |
| `schema_level` | Int | Schema level used when writing invoice data during the in-progress invoice-line migration (see openmeter/ent/schema/billing.go:1163). |

**Data Guarantees:**
- PK id (IDMixin)
- soft_delete (deleted_at, TimeMixin)
- audit (created_at, updated_at, TimeMixin)
- UNIQUE(namespace, customer_id, currency) WHERE deleted_at IS NULL AND status='gathering' — one gathering invoice per customer per currency (openmeter/ent/schema/billing.go:1192)
- INDEX(namespace, customer_id) (openmeter/ent/schema/billing.go:1175)
- INDEX(namespace, status) (openmeter/ent/schema/billing.go:1176)
- GIN INDEX(status_details_cache) (openmeter/ent/schema/billing.go:1182)

**Consumers:**
- `billing adapter (invoice.go)` — `openmeter/billing/adapter/invoice.go`: Ent read/write of invoices via TransactingRepo
- `billing service state machine` — `openmeter/billing/service/stdinvoicestate.go`: sole writer of status via the stateless state machine; direct status mutation forbidden

**Lifecycle:**
- *How to add:* Add a field to BillingInvoice.Fields() in openmeter/ent/schema/billing.go, run make generate to regenerate openmeter/ent/db/, then atlas migrate --env local diff <name> to emit the .up.sql/.down.sql pair and update atlas.sum. Commit schema + generated code + migration + atlas.sum together.

  ```
  // openmeter/ent/schema/billing.go
  field.Int("schema_level").Default(1),
  ```

- *How to modify:* Never edit a landed migration file or atlas.sum by hand. Change the Ent schema, regenerate, and produce a new timestamped migration via atlas migrate --env local diff. The invoice-line migration is mid-flight, so deprecated columns and the schema_level discriminator coexist and must be removed in lockstep.
- *How to read:* Always through billing.Service (composite interface) backed by the Ent adapter wrapped in entutils.TransactingRepo; never query openmeter/ent/db directly from a service.

  ```
  return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*Entity, error) { return tx.db.BillingInvoice.Query()... })
  ```

- *Tests:* `openmeter/billing/adapter/invoice.go`

### `Entitlement` *(table)*

A feature entitlement of one of three sub-types (metered/boolean/static); feature_key is validated to reject ULIDs, and usage_period_anchor keeps the original anchor while the effective anchor is derived from the last reset (see openmeter/ent/schema/entitlement.go:62).

- **Location:** `openmeter/ent/schema/entitlement.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/entitlement`

**Fields:**

| Field | Type | Description |
|---|---|---|
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

**Data Guarantees:**
- PK id (IDMixin), namespace (NamespaceMixin), metadata (MetadataMixin), soft_delete + audit (TimeMixin)
- INDEX(namespace, customer_id) (openmeter/ent/schema/entitlement.go:79)
- INDEX(current_usage_period_end, deleted_at) — collects entitlements with due resets (openmeter/ent/schema/entitlement.go:84)
- UNIQUE(created_at, id) (openmeter/ent/schema/entitlement.go:85)
- FK feature_id → features (required, immutable) (openmeter/ent/schema/entitlement.go:101)
- FK customer_id → customers (required, immutable) (openmeter/ent/schema/entitlement.go:107)
- cascade-delete of usage_reset, grant, balance_snapshot (openmeter/ent/schema/entitlement.go:91)

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

### `BillingInvoiceLine` *(table)*

A tagged-union invoice line (flat-fee, usage-based, or detailed) for a gathering or standard invoice; eventually intended to become the unified usage-based-pricing table, linking to charges, subscriptions, and split-line groups (see openmeter/ent/schema/billing.go:397).

- **Location:** `openmeter/ent/schema/billing.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/billing`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `id` | char(26) |  |
| `namespace` | String |  |
| `subscription_id` | String |  |
| `subscription_phase_id` | String |  |
| `subscription_item_id` | String |  |
| `subscription_billing_period_from` | Time |  |
| `subscription_billing_period_to` | Time |  |
| `split_line_group_id` | char(26) | Only valid for usage-based lines; this table is intended to eventually become the ubp table (see openmeter/ent/schema/billing.go:397). |
| `charge_id` | char(26) |  |
| `engine` | Enum (LineEngineType) |  |
| `line_ids` | char(26) | Deprecated; invoice discounts are now in line_discounts (see openmeter/ent/schema/billing.go:416). |
| `credits_applied` | jsonb |  |

**Data Guarantees:**
- PK id (IDMixin)
- soft_delete (deleted_at)
- audit (created_at, updated_at)
- UNIQUE(namespace, parent_line_id, child_unique_reference_id) WHERE child_unique_reference_id IS NOT NULL AND deleted_at IS NULL (openmeter/ent/schema/billing.go:441)
- INDEX(namespace, invoice_id) (openmeter/ent/schema/billing.go:436)
- INDEX(namespace, parent_line_id) (openmeter/ent/schema/billing.go:437)
- INDEX(namespace, subscription_id, subscription_phase_id, subscription_item_id) (openmeter/ent/schema/billing.go:443)
- FK invoice_id → billing_invoices (required) (openmeter/ent/schema/billing.go:448)

**Consumers:**
- `billing adapter (invoice.go)` — `openmeter/billing/adapter/invoice.go`: diff-based upsert of line hierarchies via TransactingRepo
- `subscriptionsync service` — `openmeter/billing/worker/subscriptionsync`: reconciles subscription views into invoice lines (SynchronizeSubscription)

**Lifecycle:**
- *How to add:* Add the field in BillingInvoiceLine.Fields() in openmeter/ent/schema/billing.go (it already has >30 fields), regenerate with make generate, then atlas migrate --env local diff <name>. Update the billing adapter line mapping for the new column.

  ```
  field.String("charge_id").SchemaType(map[string]string{dialect.Postgres: "char(26)"}).Optional().Nillable(),
  ```

- *How to modify:* Mark removed columns Deprecated(...) rather than dropping immediately (see line_ids), generate a new migration for each change, never hand-edit migrations.

  ```
  field.String("line_ids").Optional().Nillable().Deprecated("invoice discounts are deprecated, use line_discounts instead"),
  ```

- *How to read:* Through billing.Service; never construct billing.InvoiceLine{} literally — use NewStandardInvoiceLine/NewGatheringInvoiceLine so the private discriminator is set.

  ```
  line := billing.NewStandardInvoiceLine(billing.StandardInvoiceLineInput{...})
  ```

- *Tests:* `openmeter/billing/adapter/invoice.go`

### `Grant` *(table)*

A credit grant burned down for a metered entitlement; amount/rollover are numeric (immutable) and recurrence is an ISO duration with anchor (see openmeter/ent/schema/grant.go:38).

- **Location:** `openmeter/ent/schema/grant.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/credit`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `metadata` | jsonb |  |
| `owner_id` | char(26) | Entitlement that owns the grant (FK to entitlements.id) (see openmeter/ent/schema/grant.go:35). |
| `amount` | numeric | Immutable grant amount; the credit engine assumes minute-boundary effective times for burn-down (see openmeter/ent/schema/grant.go:38). |
| `priority` | Uint8 |  |
| `effective_at` | Time |  |
| `expiration` | jsonb |  |
| `expires_at` | Time |  |
| `voided_at` | Time |  |
| `reset_max_rollover` | numeric |  |
| `reset_min_rollover` | numeric |  |
| `recurrence_period` | ISODurationString |  |
| `recurrence_anchor` | Time |  |

**Data Guarantees:**
- PK id (IDMixin), namespace (NamespaceMixin), annotations (AnnotationsMixin), soft_delete + audit (TimeMixin)
- INDEX(namespace, owner_id) (openmeter/ent/schema/grant.go:61)
- INDEX(effective_at, expires_at) (openmeter/ent/schema/grant.go:62)
- FK owner_id → entitlements (required, immutable) (openmeter/ent/schema/grant.go:68)

**Consumers:**
- `credit engine` — `openmeter/credit/engine`: computes grant burn-down without I/O; all effective times truncated to time.Minute
- `balanceworker` — `openmeter/entitlement/balanceworker`: reads grants when recalculating entitlement balances

**Lifecycle:**
- *How to add:* Add field in Grant.Fields() (openmeter/ent/schema/grant.go), regenerate, atlas diff.

  ```
  field.Float("amount").Immutable().SchemaType(map[string]string{dialect.Postgres: "numeric"}),
  ```

- *How to modify:* Standard Ent + atlas diff. Always truncate grant effective times to time.Minute (Granularity) before storing/computing.

  ```
  effectiveAt := time.Now().Truncate(time.Minute)
  ```

- *How to read:* Through credit.CreditConnector (CreateGrant/VoidGrant/GetBalanceAt); never via the adapter directly.

  ```
  creditConnector.CreateGrant(ctx, credit.CreateGrantInput{EffectiveAt: effectiveAt})
  ```


### `RawEvent` *(entity)*

A raw CloudEvent usage row in the single shared ClickHouse MergeTree events table; written append-only by the sink worker, queried by meter aggregations; store_row_id (ULID) is the v2-pagination tie-breaker for same-second events (see openmeter/streaming/connector.go:34).

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
| `store_row_id` | String | Per-row ULID; cursor tie-breaker in v2 listing since DateTime is second-precision (see openmeter/streaming/connector.go:34). |
| `customer_id` | String (Nullable, query-time WITH map) |  |

**Data Guarantees:**
- ENGINE = MergeTree (no PK; not deduplicated by the engine — dedup is upstream in Redis) (openmeter/streaming/clickhouse/event_query.go:37)
- PARTITION BY toYYYYMM(time) (openmeter/streaming/clickhouse/event_query.go:38)
- ORDER BY (namespace, type, subject, toStartOfHour(time)) (openmeter/streaming/clickhouse/event_query.go:45)
- INDEX <table>_stored_at stored_at TYPE minmax GRANULARITY 4 (openmeter/streaming/clickhouse/event_query.go:35)
- CREATE TABLE IF NOT EXISTS — created by connector at startup, not Atlas (openmeter/streaming/clickhouse/event_query.go:25)
- append-only (no UPDATE/DELETE in the write path)

**Consumers:**
- `ClickHouseStorage.BatchInsert` — `openmeter/sink/storage.go`: sole writer; maps SinkMessage → RawEvent and batch-inserts via streaming.Connector
- `clickhouse meter_query / event_query` — `openmeter/streaming/clickhouse/meter_query.go`: reads via toSQL() query structs for aggregation and event listing

**Lifecycle:**
- *How to add:* Add a `ch:` tagged field to streaming.RawEvent in openmeter/streaming/connector.go, then add the column to the DDL in createEventsTable.toSQL() and to the INSERT column list in openmeter/streaming/clickhouse/event_query.go (column order must match the table exactly). CREATE TABLE IF NOT EXISTS means new columns on existing deployments need an explicit ALTER TABLE migration path.

  ```
  // event_query.go createEventsTable.toSQL()
  sb.Define("stored_at", "DateTime")
  sb.Define("store_row_id", "String")
  sb.SQL("ENGINE = MergeTree")
  sb.SQL("ORDER BY (namespace, type, subject, toStartOfHour(time))")
  ```

- *How to modify:* ClickHouse schema is created by the connector at startup (createTable), not by Atlas/golang-migrate. Changing column types requires coordinating the DDL builder and the INSERT/SELECT column lists; there is a single shared table across all namespaces.
- *How to read:* Always through streaming.Connector (QueryMeter/ListEvents/ListEventsV2); query logic lives in clickhouse/ query-structs with toSQL(), never inline SQL in connector method bodies.

  ```
  rows, err := connector.QueryMeter(ctx, namespace, meter, params)
  ```

- *Tests:* `openmeter/streaming/clickhouse/event_query_test.go`, `openmeter/streaming/clickhouse/event_query_v2_test.go`, `openmeter/streaming/clickhouse/meter_query_test.go`

### `Subscription` *(table)*

A customer subscription against a versioned plan, with default billing cadence/anchor and pro-rating config; settlement_mode (credit-then-invoice by default) is immutable (see openmeter/ent/schema/subscription.go:57).

- **Location:** `openmeter/ent/schema/subscription.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/subscription`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `name` | String |  |
| `description` | String |  |
| `plan_id` | String |  |
| `customer_id` | String |  |
| `currency` | varchar(3) |  |
| `billing_anchor` | Time |  |
| `billing_cadence` | ISODurationString | Default billing cadence for the subscription (see openmeter/ent/schema/subscription.go:42). |
| `pro_rating_config` | jsonb | Default pro-rating configuration; defaults to ProratePrices+enabled (see openmeter/ent/schema/subscription.go:44). |
| `settlement_mode` | Enum (SettlementMode) | Immutable; defaults to credit-then-invoice (see openmeter/ent/schema/subscription.go:57). |
| `active_from` | Time (CadencedMixin) |  |
| `active_to` | Time (CadencedMixin) |  |

**Data Guarantees:**
- PK id (IDMixin)
- namespace (NamespaceMixin), annotations (AnnotationsMixin), metadata (MetadataMixin)
- soft_delete (deleted_at), audit (created_at, updated_at) (TimeMixin)
- cadenced (active_from, active_to) (CadencedMixin, openmeter/ent/schema/subscription.go:29)
- INDEX(namespace, customer_id) (openmeter/ent/schema/subscription.go:67)
- FK customer_id → customers (required, immutable) (openmeter/ent/schema/subscription.go:74)
- FK plan_id → plans (openmeter/ent/schema/subscription.go:73)
- cascade-delete of phases, addons, billing_sync_state (openmeter/ent/schema/subscription.go:75,83,87)

**Consumers:**
- `subscription service` — `openmeter/subscription/service`: manages lifecycle via SubscriptionSpec + AppliesToSpec patches
- `SubscriptionBillingSyncState` — `openmeter/ent/schema/subscriptionbillingsync.go`: tracks billing sync reconciliation per subscription

**Lifecycle:**
- *How to add:* Add field in Subscription.Fields() (openmeter/ent/schema/subscription.go). SubscriptionItem deliberately duplicates RateCard fields for snapshot immutability, so RateCard changes must be mirrored manually. Regenerate + atlas diff.

  ```
  field.Enum("settlement_mode").GoType(productcatalog.SettlementMode("")).Default(string(productcatalog.CreditThenInvoiceSettlementMode)).Immutable(),
  ```

- *How to modify:* Mutate SubscriptionSpec only via the AppliesToSpec patch interface (ApplyTo); never modify spec fields directly. Schema changes go through atlas diff.
- *How to read:* Through subscription.Service (Get/GetView/List/ExpandViews) and subscriptionworkflow.Service for higher-level operations.

  ```
  subscription.Service.GetView(ctx, id)
  ```


### `Meter` *(table)*

An event-aggregation rule: event_type + aggregation function (COUNT/SUM/MAX/UNIQUE_COUNT) over an optional value_property and group_by JSON paths; event_from optionally bounds the included event window (see openmeter/ent/schema/meter.go:36).

- **Location:** `openmeter/ent/schema/meter.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/meter`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `event_type` | String |  |
| `value_property` | String | JSON path to the numeric value; optional for COUNT aggregation (see openmeter/ent/schema/meter.go:32). |
| `group_by` | jsonb |  |
| `aggregation` | Enum (MeterAggregation) |  |
| `event_from` | Time | If set, only events since this time are included (see openmeter/ent/schema/meter.go:36). |

**Data Guarantees:**
- PK id + namespace + key + name + metadata + soft_delete + audit via UniqueResourceMixin (openmeter/ent/schema/meter.go:21)
- annotations via AnnotationsMixin (openmeter/ent/schema/meter.go:23)
- UNIQUE(namespace, key) WHERE deleted_at IS NULL (openmeter/ent/schema/meter.go:43)
- INDEX(namespace, event_type) (openmeter/ent/schema/meter.go:48)

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

### `NotificationEvent` *(table)*

A notification event instance carrying a versioned jsonb payload for a rule; delivery is tracked separately via NotificationEventDeliveryStatus (see openmeter/ent/schema/notification.go:143).

- **Location:** `openmeter/ent/schema/notification.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/notification`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `created_at` | Time |  |
| `type` | Enum (EventType) |  |
| `rule_id` | char(26) |  |
| `payload` | jsonb | Version-pinned event payload; payload version is an API contract for webhook consumers (see openmeter/ent/schema/notification.go:143). |

**Data Guarantees:**
- PK id (IDMixin), namespace (NamespaceMixin), annotations (AnnotationsMixin)
- NO TimeMixin — only created_at is declared; no updated_at/deleted_at (openmeter/ent/schema/notification.go:122)
- FK rule_id → notification_rules (required, immutable) (openmeter/ent/schema/notification.go:154)

**Consumers:**
- `notification consumer` — `openmeter/notification/consumer`: builds payload and dispatches to webhook.Handler (Svix) via the system events topic
- `notification eventhandler` — `openmeter/notification/eventhandler`: runs Dispatch + Reconcile loops; Reconcile owns retry

**Lifecycle:**
- *How to add:* Add field in NotificationEvent.Fields() (openmeter/ent/schema/notification.go), regenerate, atlas diff. Channel/Rule config polymorphism uses ChannelConfigValueScanner/RuleConfigValueScanner type-switch serializers in the same file.

  ```
  field.String("payload").SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
  ```

- *How to modify:* Pin a new payload version per event family rather than changing struct shape in place; standard Ent + atlas diff.
- *How to read:* Through notification.Service (ListEvents/GetEvent/ResendEvent) and notification.EventHandler.Dispatch.

  ```
  notification.Service.GetEvent(ctx, id)
  ```


### `Customer` *(table)*

A billable customer; key is stored as empty string (not NULL) when unset so a partial unique index can enforce per-namespace key uniqueness (see openmeter/ent/schema/customer.go:33).

- **Location:** `openmeter/ent/schema/customer.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/customer`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `key` | String | Stored as empty string (not NULL) when unset because unique indexes can only be added on non-nullable fields (see openmeter/ent/schema/customer.go:33). |
| `primary_email` | String |  |
| `currency` | varchar(3) |  |

**Data Guarantees:**
- PK id + namespace + name + metadata via ResourceMixin (openmeter/ent/schema/customer.go:23)
- billing-prefixed address fields via CustomerAddressMixin (openmeter/ent/schema/customer.go:24)
- annotations via AnnotationsMixin (openmeter/ent/schema/customer.go:27)
- soft_delete (deleted_at), audit (created_at, updated_at) via ResourceMixin
- UNIQUE(namespace, key) WHERE deleted_at IS NULL (openmeter/ent/schema/customer.go:58)
- INDEX(namespace, key, deleted_at) (openmeter/ent/schema/customer.go:63)
- INDEX(name) (openmeter/ent/schema/customer.go:64)
- INDEX(primary_email) (openmeter/ent/schema/customer.go:65)
- INDEX(created_at) (openmeter/ent/schema/customer.go:67)
- cascade-delete of apps, subjects, billing_customer_override (openmeter/ent/schema/customer.go:73)

**Consumers:**
- `customer adapter` — `openmeter/customer/adapter/customer.go`: Ent read/write of customers with soft-delete semantics
- `RequestValidatorRegistry` — `openmeter/customer/requestvalidator.go`: pre-mutation cross-domain guards (billing/entitlement) run before any write

**Lifecycle:**
- *How to add:* Add field in Customer.Fields() in openmeter/ent/schema/customer.go, regenerate, then atlas migrate diff. Note the schema comment: v3 ILIKE filters on name/primary_email/key need pg_trgm GIN indexes via a custom SQL migration before exposing the list handler (see openmeter/ent/schema/customer.go:42-55).

  ```
  field.String("primary_email").Optional().Nillable(),
  ```

- *How to modify:* Standard Ent-schema-change + atlas diff flow; never edit landed migrations.
- *How to read:* Through customer.Service (ListCustomers/GetCustomer/GetCustomerByUsageAttribution); soft-deleted rows excluded by the partial index conventions.

  ```
  customer.Service.GetCustomer(ctx, input)
  ```

- *Tests:* `openmeter/customer/adapter/customer_test.go`

### `dedupe.Item` *(key_value)*

Composite ingest-dedup key (namespace-source-id) written to Redis with SET NX + TTL to suppress double-counting of retried CloudEvents; in keyhash mode hashed to a compact xxh3 key (see openmeter/dedupe/dedupe.go:39).

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
- key = namespace-source-id (Item.Key()) (openmeter/dedupe/dedupe.go:40)
- TTL = d.Expiration (config-driven) (openmeter/dedupe/redisdedupe/redisdedupe.go:84)
- SET NX (set-if-not-absent) — pre-existing key signals duplicate (openmeter/dedupe/redisdedupe/redisdedupe.go:83, Mode:"NX" at :142)
- value is empty string (only key presence matters) (openmeter/dedupe/redisdedupe/redisdedupe.go:83)

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

A meter-backed usage feature with optional unit-cost configuration (manual amount or LLM provider/model/token-type properties, enforced mutually-exclusive by CHECK constraints); archived_at provides archive instead of hard delete (see openmeter/ent/schema/feature.go:49).

- **Location:** `openmeter/ent/schema/feature.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/productcatalog/feature`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `namespace` | String |  |
| `name` | String |  |
| `description` | String |  |
| `key` | String |  |
| `meter_slug` | String | Deprecated; use meter_id, will be removed in Phase 2 (see openmeter/ent/schema/feature.go:35). |
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
| `archived_at` | Time | Archival timestamp; features are archived (not soft-deleted via deleted_at) to preserve billing references (see openmeter/ent/schema/feature.go:49). |

**Data Guarantees:**
- PK id (IDMixin), audit (created_at, updated_at) via TimeMixin, metadata via MetadataMixin
- NO deleted_at — Feature has no soft-delete; uses archived_at instead
- UNIQUE(namespace, key) WHERE archived_at IS NULL (openmeter/ent/schema/feature.go:66)
- CHECK unit_cost_llm_{provider,model,token_type}_mutual_exclusive (property xor literal) (openmeter/ent/schema/feature.go:55)
- FK meter_id → meters (openmeter/ent/schema/feature.go:90)

**Consumers:**
- `feature.FeatureConnector` — `openmeter/productcatalog/feature`: CreateFeature/ArchiveFeature/ResolveFeatureMeters

**Lifecycle:**
- *How to add:* Add field in Feature.Fields() (openmeter/ent/schema/feature.go); if it participates in mutual-exclusivity, add a CHECK in Feature.Annotations(). Regenerate + atlas diff.

  ```
  entsql.Checks(map[string]string{"unit_cost_llm_model_mutual_exclusive": "NOT (unit_cost_llm_model_property IS NOT NULL AND unit_cost_llm_model IS NOT NULL)"})
  ```

- *How to modify:* Standard Ent + atlas diff; features are archived (archived_at), not soft-deleted via deleted_at.
- *How to read:* Through feature.FeatureConnector (ListFeatures/GetFeature); use ArchiveFeature, never SetDeletedAt.

  ```
  featureConnector.ArchiveFeature(ctx, feature.ArchiveFeatureInput{ID: id})
  ```


### `SubscriptionItem` *(table)*

Lowest level of the subscription hierarchy: a snapshot of a plan RateCard (name, price, discounts, entitlement template, tax config) bound to a phase; deliberately duplicates RateCard fields for snapshot immutability (see openmeter/ent/schema/subscription.go:177).

- **Location:** `openmeter/ent/schema/subscription.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/subscription`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `annotations` | jsonb |  |
| `active_from` | Time | Mutable cadence (not CadencedMixin) because items can be re-cadenced by edits (see openmeter/ent/schema/subscription.go:164). |
| `active_to` | Time |  |
| `phase_id` | String |  |
| `key` | String |  |
| `entitlement_id` | String |  |
| `restarts_billing_period` | Bool |  |
| `active_from_override_relative_to_phase_start` | ISODurationString | Stores the intended cadence relative to phase start so it survives cancels/edits (see openmeter/ent/schema/subscription.go:171). |
| `active_to_override_relative_to_phase_start` | ISODurationString |  |
| `name` | String |  |
| `feature_key` | String |  |
| `entitlement_template` | jsonb |  |
| `tax_config` | jsonb |  |
| `billing_cadence` | ISODurationString |  |
| `price` | jsonb |  |
| `discounts` | jsonb |  |

**Data Guarantees:**
- PK id (IDMixin), namespace, audit, soft_delete (TimeMixin), metadata, TaxMixin (tax_code_id + tax_behavior)
- INDEX(namespace, phase_id, key) (openmeter/ent/schema/subscription.go:223)
- FK phase_id → subscription_phases (required, immutable) (openmeter/ent/schema/subscription.go:229)
- FK entitlement_id → entitlements with cascade (openmeter/ent/schema/subscription.go:230)

**Consumers:**
- `subscription service` — `openmeter/subscription/service`: writes item snapshots when materializing a SubscriptionSpec

**Lifecycle:**
- *How to add:* Add field in SubscriptionItem.Fields() (openmeter/ent/schema/subscription.go). Because it snapshots productcatalog RateCard fields, mirror RateCard schema changes here and in the adapter. Regenerate + atlas diff.

  ```
  field.String("price").GoType(&productcatalog.Price{}).ValueScanner(PriceValueScanner).SchemaType(map[string]string{dialect.Postgres: "jsonb"}).Optional().Nillable(),
  ```

- *How to modify:* Standard Ent + atlas diff; never collapse the RateCard duplication into a live FK (would retroactively change historical billing).
- *How to read:* Through subscription.Service views (GetView expands phases and items).

### `LLMCostPrice` *(table)*

Canonical LLM per-token pricing: a global synced price (namespace NULL) or a per-namespace override, effective over a [effective_from, effective_to) window; model IDs must be normalized via llmcost.NormalizeModelID before store/resolve (see openmeter/ent/schema/llmcostprice.go:32).

- **Location:** `openmeter/ent/schema/llmcostprice.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/llmcost`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `namespace` | String | Nil for global prices, set for namespace overrides (see openmeter/ent/schema/llmcostprice.go:33). |
| `provider` | String |  |
| `model_id` | String |  |
| `model_name` | String |  |
| `input_per_token` | numeric |  |
| `output_per_token` | numeric |  |
| `cache_read_per_token` | numeric |  |
| `reasoning_per_token` | numeric |  |
| `cache_write_per_token` | numeric |  |
| `currency` | String |  |
| `source` | String |  |
| `source_prices` | jsonb |  |
| `effective_from` | Time |  |
| `effective_to` | Time |  |

**Data Guarantees:**
- PK id (IDMixin), metadata (MetadataMixin), audit + soft_delete (TimeMixin)
- NO NamespaceMixin — namespace is a nullable field so global prices (NULL) and overrides coexist (openmeter/ent/schema/llmcostprice.go:30)
- UNIQUE(provider, model_id, namespace, effective_from) WHERE deleted_at IS NULL (openmeter/ent/schema/llmcostprice.go:85)
- INDEX(namespace, provider, model_id) WHERE deleted_at IS NULL (openmeter/ent/schema/llmcostprice.go:89)
- INDEX(provider, model_id) WHERE deleted_at IS NULL AND namespace IS NULL — global price lookup (openmeter/ent/schema/llmcostprice.go:92)

**Consumers:**
- `llmcost.Service` — `openmeter/llmcost`: ResolvePrice with namespace-override precedence; CreateOverride/DeleteOverride

**Lifecycle:**
- *How to add:* Add field in LLMCostPrice.Fields() (openmeter/ent/schema/llmcostprice.go), regenerate, atlas diff. namespace is a nullable field (not NamespaceMixin) to support global vs override rows.

  ```
  field.Other("cache_write_per_token", alpacadecimal.Decimal{}).SchemaType(map[string]string{dialect.Postgres: "numeric"}).Default(alpacadecimal.Decimal{}),
  ```

- *How to modify:* Standard Ent + atlas diff.
- *How to read:* Through llmcost.Service; always NormalizeModelID before resolving/storing.

  ```
  price, err := svc.ResolvePrice(ctx, llmcost.ResolvePriceInput{ModelID: llmcost.NormalizeModelID(raw)})
  ```


### `LedgerSubAccountRoute` *(table)*

Routes a ledger account to sub-accounts by a versioned routing key, denormalizing routing dimensions (currency, tax_code, tax_behavior, features, cost_basis, credit_priority, authorization status) as plain columns for query filtering — not FKs (see openmeter/ent/schema/ledger_account.go:116).

- **Location:** `openmeter/ent/schema/ledger_account.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/ledger`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `account_id` | char(26) |  |
| `routing_key_version` | Enum (ledger.RoutingKeyVersion) |  |
| `routing_key` | String |  |
| `currency` | String | Denormalized routing dimension, not a FK (see openmeter/ent/schema/ledger_account.go:116). |
| `tax_code` | String | Stores the TaxCode.Key string used as a routing dimension, not a FK to the tax_codes table (see openmeter/ent/schema/ledger_account.go:119). |
| `tax_behavior` | Enum (ledger.TaxBehavior) |  |
| `features` | Strings |  |
| `cost_basis` | numeric |  |
| `credit_priority` | Int |  |
| `transaction_authorization_status` | Enum (ledger.TransactionAuthorizationStatus) |  |

**Data Guarantees:**
- PK id (IDMixin), namespace, audit + soft_delete (TimeMixin)
- UNIQUE(namespace, account_id, routing_key_version, routing_key) (openmeter/ent/schema/ledger_account.go:136)
- FK account_id → ledger_accounts (required, immutable) (openmeter/ent/schema/ledger_account.go:142)
- all fields immutable

**Consumers:**
- `ledger resolvers/transactions` — `openmeter/ledger/transactions`: resolves sub-account routing for entry construction

**Lifecycle:**
- *How to add:* Add field in LedgerSubAccountRoute.Fields() (openmeter/ent/schema/ledger_account.go), regenerate, atlas diff. The most recent migration (20260520130500_add_ledger_tax_behavior) added the tax_behavior column this way.

  ```
  field.String("tax_behavior").GoType(ledger.TaxBehavior("")).Optional().Nillable().Immutable(),
  ```

- *How to modify:* Standard Ent + atlas diff. Routing dimensions are denormalized literals, kept in sync with the routing_key by application code, not DB FKs.

  ```
  -- tools/migrate/migrations/20260520130500_add_ledger_tax_behavior.up.sql
  ALTER TABLE "ledger_sub_account_routes" ADD COLUMN "tax_behavior" character varying NULL;
  ```

- *How to read:* Through ledger resolver adapters; written only when credits.enabled=true.

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
- PK id (IDMixin)
- soft_delete (deleted_at)
- UNIQUE(namespace, unique_reference_id) WHERE unique_reference_id IS NOT NULL AND deleted_at IS NULL (openmeter/ent/schema/charges.go:167)
- FK to exactly one of charge_flat_fee / charge_credit_purchase / charge_usage_based (immutable edges, openmeter/ent/schema/charges.go:141)
- NOTE: ChargesSearchV1 union view (ent.View) over the three sub-tables; chargesSearchV1Columns must list every column present in each charge sub-table (openmeter/ent/schema/charges.go:19)

**Consumers:**
- `charges adapter (search.go)` — `openmeter/billing/charges/adapter/search.go`: reads charges via the ChargesSearchV1 union view; helpers must wrap a.db in TransactingRepo

**Lifecycle:**
- *How to add:* Charge fields are added in openmeter/ent/schema/charges.go; per-type fields go in chargesflatfee.go / chargesusagebased.go / chargescreditpurchase.go. When adding a column to chargemeta.Mixin, also extend chargesSearchV1Columns in charges.go or the ChargesSearchV1 union view breaks. Run make generate then atlas migrate diff (and make generate-view-sql for the view).

  ```
  // charges.go: chargesSearchV1Columns must list every column present in each charge sub-table
  var chargesSearchV1Columns = []string{"id", "namespace", ... , "tax_behavior"}
  ```

- *How to modify:* ChargesSearchV1 is an ent.View; Atlas does not diff views, so view DDL changes require make generate-view-sql + an explicit SQL migration.
- *How to read:* Drive charge lifecycle exclusively through charges.Service (Create/AdvanceCharges/ApplyPatches); never construct charges.Charge{} literally and never call the adapter directly from outside the domain.

  ```
  charge := charges.NewCharge(flatfee.Charge{...}); fc, err := charge.AsFlatFeeCharge()
  ```

- *Tests:* `openmeter/billing/charges/adapter/search_test.go`

### `BalanceSnapshot` *(table)*

A point-in-time snapshot of grant balances/usage/overage for an entitlement owner, used to avoid recomputing burn-down from the beginning; intentionally omits IDMixin (no surrogate PK) (see openmeter/ent/schema/balance_snapshot.go:19).

- **Location:** `openmeter/ent/schema/balance_snapshot.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/credit`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `owner_id` | char(26) |  |
| `grant_balances` | jsonb |  |
| `usage` | jsonb |  |
| `balance` | numeric |  |
| `overage` | numeric |  |
| `at` | Time |  |

**Data Guarantees:**
- NO IDMixin — keyed by natural (owner, at) tuple, deliberate exception to the standard mixin triad (openmeter/ent/schema/balance_snapshot.go:19)
- namespace (NamespaceMixin), soft_delete + audit (TimeMixin)
- INDEX(namespace, owner_id, at) WHERE deleted_at IS NULL (openmeter/ent/schema/balance_snapshot.go:49)
- FK owner_id → entitlements (required, immutable) (openmeter/ent/schema/balance_snapshot.go:57)
- all value fields immutable (openmeter/ent/schema/balance_snapshot.go:31)

**Consumers:**
- `credit balance connector` — `openmeter/credit/balance`: reads/writes balance snapshots to bound burn-down recomputation

**Lifecycle:**
- *How to add:* Add field in BalanceSnapshot.Fields() (openmeter/ent/schema/balance_snapshot.go), regenerate, atlas diff. Note this entity omits IDMixin (no surrogate PK).

  ```
  field.Float("overage").Immutable().SchemaType(map[string]string{dialect.Postgres: "numeric"}),
  ```

- *How to modify:* Standard Ent + atlas diff; do not add IDMixin expecting the standard triad.
- *How to read:* Through credit.CreditConnector balance APIs.

### `CustomerSubjects` *(table)*

Link table mapping a customer to one or more subject keys; FK constraint to Subject.subject_key is intentionally absent because Ent cannot enforce FKs on non-ID fields (see openmeter/ent/schema/customer.go:147).

- **Location:** `openmeter/ent/schema/customer.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/customer`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `customer_id` | char(26) |  |
| `subject_key` | String |  |
| `created_at` | Time |  |
| `deleted_at` | Time |  |

**Data Guarantees:**
- namespace via NamespaceMixin (openmeter/ent/schema/customer.go:96)
- UNIQUE(namespace, subject_key) WHERE deleted_at IS NULL — one active mapping per subject (openmeter/ent/schema/customer.go:123)
- INDEX(namespace, customer_id, deleted_at) (openmeter/ent/schema/customer.go:122)
- FK customer_id → customers (required, immutable) (openmeter/ent/schema/customer.go:141)
- no FK on subject_key (Ent limitation on non-ID FK fields, openmeter/ent/schema/customer.go:147)

**Consumers:**
- `customer adapter` — `openmeter/customer/adapter/customer.go`: writes subject-key mappings on customer create/update

**Lifecycle:**
- *How to add:* Add field in CustomerSubjects.Fields() in openmeter/ent/schema/customer.go, regenerate, atlas diff.
- *How to modify:* Standard Ent + atlas diff.
- *How to read:* Through customer.Service usage-attribution APIs and ent .WithSubjects() edges.
- *Tests:* `openmeter/customer/adapter/customer_test.go`

### `LedgerEntry` *(table)*

A single signed amount posting against a ledger sub-account within a transaction; (transaction_id, sub_account_id, identity_key) is unique to make postings idempotent (see openmeter/ent/schema/ledger_entry.go:67).

- **Location:** `openmeter/ent/schema/ledger_entry.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/ledger`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `sub_account_id` | char(26) |  |
| `identity_key` | String | Dedup key making a posting idempotent within (transaction_id, sub_account_id) (see openmeter/ent/schema/ledger_entry.go:33). |
| `amount` | numeric |  |
| `transaction_id` | char(26) |  |

**Data Guarantees:**
- PK id (IDMixin), namespace (NamespaceMixin), annotations (AnnotationsMixin), soft_delete + audit (TimeMixin)
- UNIQUE(namespace, id) (openmeter/ent/schema/ledger_entry.go:64)
- UNIQUE(transaction_id, sub_account_id, identity_key) (openmeter/ent/schema/ledger_entry.go:67)
- INDEX(created_at, id) WHERE deleted_at IS NULL (openmeter/ent/schema/ledger_entry.go:68)
- FK transaction_id → ledger_transactions (required, immutable) (openmeter/ent/schema/ledger_entry.go:47)
- FK sub_account_id → ledger_sub_accounts (required, immutable) (openmeter/ent/schema/ledger_entry.go:53)
- all fields immutable (append-only postings)

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


### `NotificationChannel` *(table)*

A delivery channel (e.g. webhook) carrying a polymorphic jsonb config serialized by a type-switching ChannelConfigValueScanner; a new ChannelType requires updating both the marshal and unmarshal switches (see openmeter/ent/schema/notification.go:47).

- **Location:** `openmeter/ent/schema/notification.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/notification`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `type` | Enum (ChannelType) |  |
| `name` | String |  |
| `disabled` | Bool |  |
| `config` | jsonb | Polymorphic channel config serialized by ChannelConfigValueScanner type-switch; missing a case errors at runtime, not compile time (see openmeter/ent/schema/notification.go:47). |

**Data Guarantees:**
- PK id (IDMixin), namespace, audit + soft_delete (TimeMixin), annotations, metadata
- INDEX(namespace, id) (openmeter/ent/schema/notification.go:62)
- INDEX(namespace, type) (openmeter/ent/schema/notification.go:63)

**Consumers:**
- `notification adapter` — `openmeter/notification/adapter`: Ent read/write; serializes channel config via the value scanner

**Lifecycle:**
- *How to add:* Add field in NotificationChannel.Fields() (openmeter/ent/schema/notification.go), regenerate, atlas diff. A new ChannelType needs both V (marshal) and S (unmarshal) cases in ChannelConfigValueScanner.
- *How to modify:* Standard Ent + atlas diff.
- *How to read:* Through notification.Service (ChannelService).

### `LedgerCustomerAccount` *(table)*

Private linking table mapping a customer to their ledger accounts (one FBO and one Receivable per customer per namespace); intentionally has no edges/FKs to LedgerAccount to avoid import cycles (see openmeter/ent/schema/ledger_customer_account.go:43).

- **Location:** `openmeter/ent/schema/ledger_customer_account.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/ledger`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `customer_id` | String |  |
| `account_type` | Enum (ledger.AccountType) |  |
| `account_id` | String |  |

**Data Guarantees:**
- PK id (IDMixin), namespace, soft_delete + audit (TimeMixin)
- UNIQUE(namespace, id) (openmeter/ent/schema/ledger_customer_account.go:36)
- UNIQUE(namespace, customer_id, account_type) — one FBO and one Receivable per customer per namespace (openmeter/ent/schema/ledger_customer_account.go:38)
- no edges (Edges() returns nil) — deliberate, to avoid import cycles; referential integrity enforced only by application code (openmeter/ent/schema/ledger_customer_account.go:43)

**Consumers:**
- `ledger resolvers adapter` — `openmeter/ledger/resolvers/adapter/repo.go`: links customers to FBO/Receivable accounts

**Lifecycle:**
- *How to add:* Add field in LedgerCustomerAccount.Fields(), regenerate, atlas diff.
- *How to modify:* Standard Ent + atlas diff. account_id/customer_id are FK-less Strings; deleting an account leaves dangling rows undetected until a runtime read fails — add application-level integrity checks.
- *How to read:* Through ledger resolver adapters; rows only written when credits.enabled=true (otherwise concrete adapters must be constructed directly).

### `LedgerTransaction` *(table)*

A double-entry transaction grouping balanced ledger entries; belongs to a transaction group and records when it was booked (see openmeter/ent/schema/ledger_transaction.go:26).

- **Location:** `openmeter/ent/schema/ledger_transaction.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/ledger`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `group_id` | char(26) |  |
| `booked_at` | Time |  |

**Data Guarantees:**
- PK id (IDMixin), namespace, annotations, soft_delete + audit
- UNIQUE(namespace, id) (openmeter/ent/schema/ledger_transaction.go:49)
- INDEX(namespace, group_id) (openmeter/ent/schema/ledger_transaction.go:50)
- INDEX(namespace, booked_at) (openmeter/ent/schema/ledger_transaction.go:51)
- FK group_id → ledger_transaction_groups (required, immutable) (openmeter/ent/schema/ledger_transaction.go:37)

**Consumers:**
- `ledger transactions resolver` — `openmeter/ledger/transactions`: groups entries into a balanced transaction via ResolveTransactions/CommitGroup

**Lifecycle:**
- *How to add:* Add field in LedgerTransaction.Fields() (openmeter/ent/schema/ledger_transaction.go), regenerate, atlas diff.
- *How to modify:* Standard Ent + atlas diff.
- *How to read:* Through ledger.Ledger; never construct transaction inputs outside transactions.ResolveTransactions templates.

### `LedgerAccount` *(table)*

A double-entry ledger account of a typed kind (FBO/Receivable/Accrued for customers; Wash/Earnings/Brokerage for business), with sub-accounts routed by routing keys (see openmeter/ent/schema/ledger_account.go:30).

- **Location:** `openmeter/ent/schema/ledger_account.go`
- **Store:** `primary_postgres`
- **Owner:** `openmeter/ledger`

**Fields:**

| Field | Type | Description |
|---|---|---|
| `account_type` | Enum (ledger.AccountType) |  |

**Data Guarantees:**
- PK id (IDMixin), namespace, annotations, soft_delete + audit
- UNIQUE(namespace, id) (openmeter/ent/schema/ledger_account.go:36)
- account_type immutable (openmeter/ent/schema/ledger_account.go:30)

**Consumers:**
- `ledger AccountResolver` — `openmeter/ledger`: EnsureCustomerAccounts/EnsureBusinessAccounts provision typed accounts

**Lifecycle:**
- *How to add:* Add field in LedgerAccount.Fields() (openmeter/ent/schema/ledger_account.go), regenerate, atlas diff.

  ```
  field.String("account_type").GoType(ledger.AccountType("")).Immutable(),
  ```

- *How to modify:* Standard Ent + atlas diff.
- *How to read:* Through ledger.AccountResolver / ledger.Ledger; noop implementations are wired when credits.enabled=false.

## Persistence Stores

_Where data lives across process or session boundaries — databases, caches, queues, mobile local storage._

### `primary_postgres`

Authoritative relational store for all ~35 Ent entities; schema is defined in openmeter/ent/schema/*.go and applied at startup via golang-migrate over Atlas-generated SQL (see tools/migrate/migrate.go).

- **Engine:** PostgreSQL (atlas.hcl dev db docker://postgres/15; docker-compose base uses 14.20-alpine)
- **Role:** primary
- **Migrations dir:** `tools/migrate/migrations`
- **Lives here:** BillingInvoice, BillingInvoiceLine, Charge, Customer, CustomerSubjects, Subscription, SubscriptionItem, Entitlement, Grant, BalanceSnapshot, LedgerEntry, LedgerTransaction, LedgerAccount, LedgerSubAccountRoute, LedgerCustomerAccount, Meter, Feature, NotificationChannel, NotificationEvent, LLMCostPrice
- **Written by:** openmeter/billing (billing domain), openmeter/billing/charges (charges sub-domain), openmeter/credit (credit grants), openmeter/customer (customer domain), openmeter/entitlement (entitlement domain), openmeter/ledger (double-entry ledger), openmeter/llmcost (LLM cost prices), openmeter/meter + ingest + sink + streaming (usage pipeline), openmeter/notification (notification domain), openmeter/productcatalog (catalog domain), openmeter/subscription (subscription domain)

### `redis_dedupe`

TTL-based ingest deduplication store using SET NX on namespace-source-id keys; updated as the third (last) phase of the sink flush, strictly after Kafka offset commit (see openmeter/dedupe/redisdedupe/redisdedupe.go:83).

- **Engine:** Redis (go-redis/v9)
- **Role:** cache
- **Lives here:** dedupe.Item
- **Written by:** openmeter/dedupe (ingest dedup)

### `clickhouse_events`

Single shared append-only MergeTree events table across all namespaces (namespace is the leading ORDER BY column); table DDL is created by the connector at startup, not by Atlas (see openmeter/streaming/clickhouse/event_query.go:25).

- **Engine:** ClickHouse (clickhouse-go/v2)
- **Role:** analytics
- **Lives here:** RawEvent

### `kafka_topics`

Durable cross-binary event bus with three name-prefix-routed topics (ingest, system, balance-worker) via Watermill; sole inter-binary channel, also carries raw ingest CloudEvents consumed by the sink worker (see openmeter/watermill/eventbus/eventbus.go).

- **Engine:** Kafka (confluent-kafka-go v2 + Watermill)
- **Role:** queue