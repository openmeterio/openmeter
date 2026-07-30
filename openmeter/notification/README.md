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
- Invoice payloads are eagerly converted from the billing event to
  `api.Invoice` before persistence. The notification package does not persist
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
- A failed delivery does not disable its rule or channel. Future matching
  source events remain independent.

Resend reuses the persisted event payload. It may target selected channels, but
only channels assigned to the event's rule and statuses already in
`SENDING`, `SUCCESS`, or `FAILED` are eligible.
