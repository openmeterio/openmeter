# Notifications

Notifications turn selected OpenMeter domain events into durable,
customer-configured webhook deliveries.

A source event is not itself a notification event. The notification consumer
matches it against active rules and creates notification events for the
matching rule and trigger. A single balance snapshot can produce separate
events for an active usage threshold and an active balance threshold. Each
event contains a payload snapshot and one delivery status for every active
channel assigned to that rule.

## Domain model

- A **channel** is a delivery destination. Webhooks are currently the only
  channel type; the channel owns its URL, headers, signing secret, and disabled
  state.
- A **rule** selects one notification event type, its type-specific conditions,
  and one to five channels. Rule and channel types are immutable.
- An **event** records that one rule matched one source fact. Its type, rule ID,
  payload, and creation time are immutable. The referenced rule remains mutable;
  its configuration and channel assignments are not copied into the event.
- A **delivery status** is the mutable per-event, per-channel delivery record.
  It contains the state, next attempt, failure reason, and provider attempts.

Rules and channels are namespaced and soft-deleted. A channel cannot be deleted
while a rule still references it. Events have no update or delete lifecycle;
payload compatibility and retention therefore matter for historical data.

Rule get and list responses include all assigned channels that have not been
deleted, including disabled channels. Disabling a channel stops delivery but
does not remove its rule assignment.

## Channel type storage compatibility

The service uses `webhook`; V1 HTTP uses `WEBHOOK`, while V3 uses `webhook`.
The Ent field scanner reads either stored spelling as the lowercase domain value.
New Channels store `webhook` in the type column. Updates preserve the stored
type. Channel type filters match both stored spellings using `in`/`nin`,
including exclusions and nested `and`/`or`.
Type filters accept exact comparisons and existence checks; pattern and range
operators are rejected. Other string filters retain their existing operators.

Ent exposes Channel config as `json.RawMessage`. The adapter decodes it using
the Channel's type, trying the legacy envelope first and falling back to flat
config if envelope decoding fails. Create and update write flat webhook config
with `url` and optional `custom_headers` and `signing_secret`. Reads preserve
the stored config and signing secret. Malformed configs propagate errors
through Channel, Rule, and Event reads.

The nullable `event_type_filter` column is reserved for V4 routing. Current
operations leave it unset on creation and preserve it on updates; Rule
assignments remain authoritative. There are no dual writes. Backfilling and
activating the filter, including catch-all behavior, belong to Phase 2.
The future V4 adapter must enforce unique filter entries, and the Phase 2
backfill must produce distinct values. No uniqueness validation is active yet.

Channel `provider_data` is nullable JSONB reserved for V4 provider-owned data,
separate from metadata and annotations. Current operations leave it unset on
creation and preserve it on updates. Provider identity adoption and backfilling
remain deferred; current provider lookups and annotations are unchanged.

Channel reconciliation fields are reserved for V4: `generation` defaults to `1`,
`reconcile_status` to `unknown`, and `reconcile_next_at` to NULL. Generations use
Go `int` / PostgreSQL `integer`, validated by Ent within `1..MaxInt32`. Statuses
are `unknown`, `pending`, `in_sync`, and `error`. Current updates and soft deletes
preserve these fields; they do not activate scheduling or change synchronous
provider operations. Migration defaults do not establish provider synchronization.

Roll out this writer from v1.0.0-beta.237, which reads both type spellings and both
config shapes. During rollout or rollback, beta.237 can write an envelope
when updating a Channel created by the new writer. Backfill legacy types and
configs after all legacy writers and their in-flight transactions have drained.

## Rule and Event type storage compatibility

Rule and Event creation writes the versioned names below. Reads accept both
stored spellings and return the legacy names to the service and V1/V3 APIs.
Webhook messages and JSON discriminators use the legacy names. Rule updates
preserve the existing type column.

| V1/V3 name | Versioned storage alias |
| --- | --- |
| `entitlements.balance.threshold` | `v1.entitlement.balance_threshold.triggered` |
| `entitlements.reset` | `v1.entitlement.usage_period.reset` |
| `invoice.created` | `v1.billing.invoice.created` |
| `invoice.updated` | `v1.billing.invoice.updated` |

V3 Rule and Event filters accept legacy names. Exact comparisons match either
stored spelling, and exclusions exclude both. Type sorting uses the V3 name
and then the resource ID in the same direction before pagination. The aliases
apply to type columns only. Ent stores plain strings. The service validates
public names before the adapter maps them to storage. Roll out from beta.237,
which reads both spellings. Historical-row backfills follow in Phase 3
after all legacy writers have drained. Payload and config conversion are separate
steps.

Ent exposes Rule config as `json.RawMessage`; the adapter decodes the legacy
envelope first, then the unwrapped existing contract using the Rule's type.
Flat configs reject unknown fields so malformed envelopes cannot become empty
invoice configs. Reads do not rewrite rows or revalidate historical conditions.
Create and update validate configs in the service and write the type-specific
object directly. Rule and config types must match, except that the two invoice
types share the same config contract. Invoice configs are `{}`. The beta.237 reader accepts both
representations during rollout and rollback. Decode errors propagate through
Rule reads and nested Event reads.

Rule create/update uses the product-catalog `FeatureResolver` to resolve feature
IDs or keys to active, same-namespace feature IDs before persistence. ID/key
aliases for the same feature collapse to one ID, keeping first-occurrence order. Missing references fail the write.
Caller-owned configuration is not modified. V1 responses keep their existing
`FeatureMeta` shape and populate `id` with the stored ID. Existing Rules can contain historical keys until
a later release backfills them. Deploy this writer on all replicas and drain
older writers before upgrading to that release.

## Event storage compatibility

Ent exposes Event payloads as `json.RawMessage`; the adapter continues writing
and decoding the existing typed JSON format, including invoice payload version
checks. Raw payloads remain JSON objects in PostgreSQL, without an extra JSON
string or base64 encoding. Reads do not rewrite historical payloads.

The Event `rule_id` column is nullable in preparation for system Events. Current
service and adapter creation paths still require an existing Rule and write its
ID; readers and workers still depend on that relation. The foreign key retains
`NO ACTION` on Rule deletion. Rule-free writes remain disabled until all readers
and delivery paths support them.

Event `rule_snapshot` is nullable JSONB reserved for the evaluated Rule state.
Current creation paths leave it unset, and existing operations preserve any
stored snapshot. It is immutable in Ent and is not exposed by the current API.
Historical snapshots must not be reconstructed from the live Rule.

Event `provider_data` is nullable JSONB reserved for V4 provider-owned data,
separate from annotations. Current creation paths leave it unset, and existing
operations preserve stored values. Existing provider identifiers remain
authoritative; there are no dual writes, backfills, or API exposure yet.

Delivery state uses a plain Ent string field. The service validates states
through `UpdateEventDeliveryStatusInput.Validate`. Creates and updates write
lowercase states, including resend, and omitted states default to `pending`.
The adapter reads either stored spelling as uppercase.
Event and delivery filters match both spellings, including the poller's check
for an active destination whose next attempt is due.

Delivery attempts are raw JSON in Ent and typed at the adapter boundary. Stored
arrays retain uppercase states and legacy response fields, including durations
in nanoseconds. Reads do not rewrite SQL NULL, JSON null, or empty arrays; an
update with nil attempts still writes JSON null. Invalid attempt data fails
both delivery reads and Event reads that include those deliveries.

Delivery-status `provider_data` is nullable JSONB reserved for V4. Current
creation leaves it unset, and delivery updates preserve stored values independently
of annotations and attempts. Provider lookups and reconciliation still use their
existing sources; the field is not exposed by the current API.

Delivery `dispatch_generation` uses Go `int` and PostgreSQL `integer`, with
default `1` and Ent validation within `1..MaxInt32`. This is a migration baseline,
not a reconstructed resend count. Current updates and resends preserve it;
generation tracking and provider adoption remain inactive until all writers
support them.

Delivery `dispatch_queued_at` is nullable and reserved for the current dispatch
generation. Phase 1 creation leaves it unset and delivery updates preserve it.
After activation, queueing a delivery or accepting an explicit resend starts its
timeout, including time spent waiting for the Channel to reach `in_sync`.
`next_attempt_at` continues to schedule the next delivery attempt.

## Ownership and runtime boundaries

- [Billing](../billing/README.md) owns invoice lifecycle and invoice domain
  events. Entitlements own balance calculation, usage periods, and snapshot
  events. Notifications do not reconstruct either domain's current state.
- The notification consumer owns rule matching, threshold evaluation,
  notification-event creation, and the mapping from source events to persisted
  notification payloads.
- Event creation atomically persists the event and its initial `PENDING`
  delivery statuses. It does not synchronously send the webhook.
- The notification worker consumes balance and invoice events and records
  notification events. A leader-elected reconciler in the API server delivers
  and reconciles them through Svix. See the
  [runtime architecture](../../docs/architecture.md).
- Svix owns outbound webhook attempts. OpenMeter owns the rule and channel
  model, the durable notification event, and its mirrored per-channel delivery
  state.

Each OpenMeter webhook channel is mirrored by a Svix endpoint. Rule assignment
is also reflected in Svix's endpoint filters. Channel and rule changes must
keep PostgreSQL and provider state consistent; changing only one side breaks
delivery or leaves delivery status unreconcilable.

Channel and rule mutation APIs return HTTP 400 for webhook validation failures,
including provider constraints checked after input validation. The response
uses fixed messages for explicitly mapped constraints and a generic message for
unknown validation failures. Provider details stay private; outages remain
server errors.

Svix disables an endpoint on its own after a prolonged delivery failure and
does not push that decision back. Reconciliation mirrors it onto the channel,
stamping the `channel.provider.disabled.timestamp` annotation so the reason is
visible. Re-enabling is a user action: updating the channel clears the
annotation and re-enables the endpoint at the provider.

## Source events and matching

| notification type | source | matching behavior |
| --- | --- | --- |
| `entitlements.balance.threshold` | entitlement snapshot update or reset | optional feature scope; evaluates configured usage and balance thresholds |
| `entitlements.reset` | entitlement reset snapshot | optional feature scope |
| `invoice.created` | billing standard-invoice created event | every active rule of the type |
| `invoice.updated` | billing standard-invoice updated event | every active rule of the type |

No active matching rule means no notification event is persisted. Invoice and
balance-threshold rules are evaluated independently, so matching rules can
produce separate events from the same source fact. Reset-event deduplication
currently uses a broader usage-period lookup and does not provide that per-rule
guarantee.

Balance-threshold notifications are evaluated only for active metered
entitlements with a complete balance and usage snapshot. Within each threshold
kind, the most advanced active threshold is selected. Deduplication is scoped
to the rule, entitlement, feature, usage period, measurement start, and
threshold kind. It suppresses repeated snapshots at the same active threshold,
not events from another rule.

This deduplication belongs to the balance-threshold handler; `CreateEvent` has
no generic source-event idempotency contract. Its hashes are persisted with the
event and legacy hash forms remain part of lookup compatibility.

Invoice notifications ignore invoices still in `gathering`: that state is an
incomplete billing aggregate, not a deliverable invoice snapshot.

## Payloads are historical snapshots

The persisted payload is the source for both notification API responses and
later webhook delivery. Delivery must not look up the current invoice,
entitlement, customer, feature, or subject and silently change what the event
means.

- Entitlement payloads contain API-shaped entitlement, feature, customer,
  subject, and calculated value data from the consumed snapshot.
- Invoice payloads are eagerly converted from the billing event to the v1 API
  `Invoice` model (`api/models`) before persistence. The notification package does not persist
  the internal billing aggregate.
- Current invoice payloads use persisted payload version `1`. Invoice reads
  reject unsupported versions. Entitlement payloads remain unversioned, so
  payload-version changes must account for existing rows by event type.

Adding or changing an event type is therefore a persistence and external
contract change, not only a new consumer branch. The domain type, rule and
payload validation, source-event mapping, API payload, stored JSON handling,
Svix event registration, and historical reads must agree.

Testing a rule is not a dry-run. It creates a normal persisted notification
event, marked with the `notification.rule.test` annotation, and its delivery
statuses enter the same reconciliation flow.

### Legacy payloads on the v3 API

The v3 Event endpoints return the legacy Event types (`invoice.created`,
`invoice.updated`, `entitlements.balance.threshold`, `entitlements.reset`) with
the same content as v1, so clients moving to v3 don't lose data. The v3 spec
can't import the legacy TypeSpec program, so
[`legacy-v1.tsp`](../../api/spec/packages/aip/src/notifications/legacy-v1.tsp)
holds copies of the v1 models these payloads use (`BillingNotificationEventV1*`).
The [v3 handler](../../api/v3/handlers/notification/events/convert.go) builds
the payload with the v1 mapping and only renames fields to snake_case, so v1
and v3 can't drift apart.

When changing the copies:

- Match what the v1 mapper sends, not the v1 schema. The copies only appear in
  responses, so they have no defaults and no patterns or length limits the
  server doesn't guarantee (for example, the subject `id`, `subject_key`, and
  the customer `id` and `name` of entitlement Events stored before snapshots
  carried a customer can be empty; an invoice's customer `subject_keys` and its
  workflow apps can be null), and they leave out union variants the mapper never
  sends.
- Keep the v1 enum spellings, including the deprecated `PERCENT` and `NUMBER`
  threshold types.
- Keep `metadata` and `annotations` instead of v3 `labels`. Labels are
  validated string-to-string maps, so converting would drop non-string
  annotation values and any key or value that fails label validation, and
  would add the `openmeter_` prefix to annotation keys. Keys of these maps are
  user data and keep their case.

The `data` of each legacy payload is also the payload of the versioned Event
type in the table above, and the type-specific fields of the v3 Rule are its
Rule configuration. [`legacy-v1-event-types.tsp`](../../api/spec/packages/aip/src/notifications/legacy-v1-event-types.tsp)
declares these v1 contracts, and their definitions are generated into
[`notification/v4/definitions`](v4/definitions). A changed copy therefore
changes a contract and fails the `definitions.sum` check. The v3 handler tests
validate every legacy fixture against its contract.

[`event_v3.go`](../../test/notification/event_v3.go) runs the real consumers,
reads the Events through both APIs and validates the v3 responses against the
spec. When a producer can send a new shape, add a case there.

Event types built from v3 models use `labels`; the copies exist only for these
legacy types.

## Delivery lifecycle

Delivery is tracked independently for each channel:

```text
PENDING -> SENDING -> SUCCESS
                   -> FAILED
     \-> FAILED

SENDING -> PENDING (missing provider message recovery)

SENDING, SUCCESS, or FAILED -> RESENDING -> SENDING
```

- `PENDING` means the event exists but the provider message is not yet known to
  be accepted. Repeated sends converge on the notification event ID at Svix.
- `SENDING` mirrors provider-side retries and attempt details.
- `SUCCESS` and `FAILED` are terminal until an explicit resend changes the
  selected statuses to `RESENDING`.
- Reconciliation handles transient provider errors using `next_attempt`.
  Missing provider message state can eventually return `SENDING` to `PENDING`;
  missing endpoints, missing rule assignments, disabled provider endpoints, and
  invalid configuration are terminal for that delivery.
- By default, a delivery that remains `PENDING` fails three hours after its
  delivery status was created, and one in `SENDING` fails 48 hours after
  creation. The current implementation does not restart that clock on resend,
  so an old delivery can reach the sending timeout immediately after a resend.
  These timeouts and the reconciliation interval are configurable.
- A failed delivery does not disable its rule or channel, with one exception:
  when the provider reports the endpoint as disabled, reconciliation disables
  the channel as well. Future matching source events remain independent.

Resend reuses the persisted event payload. It may target selected channels, but
only channels assigned to the event's rule and statuses already in
`SENDING`, `SUCCESS`, or `FAILED` are eligible.
