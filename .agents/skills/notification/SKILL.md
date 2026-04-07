---
name: notification
description: Work with the OpenMeter notification package. Use when modifying notification event creation, delivery, reconciliation, rules, channels, webhooks, event payloads, or the Kafka consumer handlers. Trigger this skill whenever the task touches `openmeter/notification/...`, `cmd/notification-service/`, Svix integration, or notification-related tests.
user-invocable: true
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent
---

# Notification

Guidance for working with the OpenMeter notification package (`openmeter/notification/`).

## Package Layout

```
openmeter/notification/
├── (root)              — Domain types, interfaces, payload union types, event type constants
├── service/            — Service implementation (business logic)
├── adapter/            — Ent ORM-backed Repository implementation (Postgres persistence)
├── consumer/           — Kafka consumer handlers (entitlement snapshots, invoice events)
├── eventhandler/       — Async dispatch + background reconciliation loop
│   └── noop/          — No-op EventHandler for testing / non-notification services
├── webhook/            — Webhook provider abstraction
│   ├── svix/          — Svix production implementation
│   ├── noop/          — No-op implementation for testing
│   └── secret/        — Signing secret validation
├── httpdriver/         — HTTP API handlers and domain-to-API mapping
├── internal/           — Internal helpers (test event generator for TestRule endpoint)
└── (subdomain files)
    ├── service.go         — Service interface (ChannelService, RuleService, EventService)
    ├── repository.go      — Repository interface (ChannelRepo, RuleRepo, EventRepo)
    ├── eventhandler.go    — EventHandler interface (Dispatch, Reconcile, Start, Close)
    ├── event.go           — Event domain type, EventType enum, input types
    ├── eventpayload.go    — EventPayload union type with version field
    ├── entitlements.go    — BalanceThresholdPayload, EntitlementResetPayload
    ├── invoice.go         — InvoicePayload (embeds api.Invoice), InvoiceRuleConfig
    ├── channel.go         — Channel, ChannelConfig, WebHookChannelConfig
    ├── rule.go            — Rule, RuleConfig, BalanceThresholdRuleConfig
    ├── deliverystatus.go  — EventDeliveryStatus, state constants, attempt tracking
    ├── errors.go          — NotFoundError, UpdateAfterDeleteError
    ├── annotations.go     — Standard annotation key constants for events
    ├── defaults.go        — Default timeouts, pagination, and limits
    └── utils.go           — Channel helpers, delivery status helpers
```

HTTP API entry points: `openmeter/notification/httpdriver/`

DI wiring: `app/common/notification.go` — `NewNotificationService`, `NewNotificationEventHandler`, `NewNotificationWebhookHandler`

Standalone service: `cmd/notification-service/` — Kafka consumer + telemetry

## Core Domain Model

### Event Types

| EventType constant               | Source event                                  | Consumer handler                  |
|----------------------------------|-----------------------------------------------|-----------------------------------|
| `entitlements.balance.threshold` | Entitlement balance snapshot                  | `consumer/entitlementsnapshot.go` |
| `entitlements.reset`             | Entitlement balance snapshot (reset detected) | `consumer/entitlementsnapshot.go` |
| `invoice.created`                | `billing.StandardInvoiceCreatedEvent`         | `consumer/invoice.go`             |
| `invoice.updated`                | `billing.StandardInvoiceUpdatedEvent`         | `consumer/invoice.go`             |

### Entity Hierarchy

```
Channel        — webhook endpoint (URL, custom headers, signing secret)
Rule           — event type + config + channels (M:N, max 5 channels per rule)
Event          — notification record (payload, rule ref, dedup hash)
  └── EventDeliveryStatus[]  — per-channel delivery tracking (state, attempts)
```

### Event Payloads (Union Type)

`EventPayload` is a union type with a `Type` discriminator and `Version` field:

```go
type EventPayload struct {
    EventPayloadMeta                                          // Type + Version
    BalanceThreshold *BalanceThresholdPayload `json:"balanceThreshold,omitempty"`
    EntitlementReset *EntitlementResetPayload `json:"entitlementReset,omitempty"`
    Invoice          *InvoicePayload          `json:"invoice,omitempty"`
}
```

**Versioning:** The `Version int` field in `EventPayloadMeta` tracks the JSONB schema version:
- `Version: 0` (absent/zero) — legacy events (no longer written; only v0 invoice payloads existed)
- `Version: 1` — current; all new events are written with `EventPayloadVersionCurrent`

Invoice payloads use `api.Invoice` (transformed eagerly in the consumer), NOT internal billing types. Entitlement payloads already use API types (`api.EntitlementMetered`, `api.Feature`, `api.Subject`, `api.Customer`).

Reference: `openmeter/notification/eventpayload.go`, `openmeter/notification/invoice.go`

### Delivery Status State Machine

| State       | Meaning                             | Transition                                   |
|-------------|-------------------------------------|----------------------------------------------|
| `PENDING`   | Created, not yet sent               | → SENDING (on send) or → FAILED (3h timeout) |
| `SENDING`   | Sent to Svix, awaiting confirmation | → SUCCESS or → FAILED (48h timeout)          |
| `RESENDING` | User-initiated resend               | → SENDING (on resend)                        |
| `SUCCESS`   | Delivered successfully              | Terminal                                     |
| `FAILED`    | Delivery failed                     | Terminal                                     |

Reference: `openmeter/notification/deliverystatus.go`, `openmeter/notification/eventhandler/reconcile.go`

## Service Interface

```go
type Service interface {
    FeatureService   // ListFeature
    ChannelService   // CRUD channels
    RuleService      // CRUD rules
    EventService     // CRUD events + delivery statuses
}
```

Every event mutating operation:
1. Validates input (rule exists, rule is not disabled, payload matches type)
2. Persists event + per-channel delivery statuses atomically via adapter
3. Dispatches asynchronously via `EventHandler.Dispatch()`

Reference: `openmeter/notification/service.go`, `openmeter/notification/service/event.go`

## Event Flow (End-to-End)

```
Kafka (om_sys.api_events)
  ↓
Consumer handler (consumer/)
  ├── Entitlement snapshots → balance threshold / reset path
  └── Invoice events → per-rule event creation
  ↓
Service.CreateEvent()
  ├── Validates rule + payload
  ├── adapter.CreateEvent() → writes notification_event + delivery_status rows
  └── EventHandler.Dispatch() → async goroutine (30s timeout)
       ↓
       reconcileEvent() → reconcileWebhookEvent()
       ├── webhook.SendMessage() → Svix API
       └── Updates delivery status based on Svix response
  ↓
Background Reconcile() loop (every 15s)
  ├── Lists PENDING/SENDING/RESENDING delivery statuses
  ├── Fetches status from Svix for each
  └── Updates to SUCCESS/FAILED or retries
```

## Consumer Handlers

### Entitlement Snapshot Handler (`consumer/entitlementsnapshot.go`, `consumer/entitlementbalancethreshold.go`, `consumer/entitlementreset.go`)

1. Receives `snapshot.SnapshotEvent` from Kafka
2. Lists active rules matching the event type and (optionally) the feature
3. For balance thresholds: evaluates each threshold, checks if it is active based on usage/balance values
4. **Deduplicates** using a hash stored in event annotations:
   - Hash inputs: usage period, rule ID, threshold kind, namespace, subject, entitlement, feature
   - Dual-version hashing: SHA256 v1 (`bsnap_v1_*`) + xxHash v2 (`bsnap_v2_*`)
   - Queries `ListEvents(DeduplicationHashes: [v1, v2])` to check for existing events in the same period
5. Calls `Service.CreateEvent()` per matching rule + threshold

### Invoice Event Handler (`consumer/invoice.go`)

1. Receives `billing.EventStandardInvoice` from Kafka
2. Skips if invoice status is `gathering` (incomplete)
3. **Eagerly transforms** `billing.EventStandardInvoice → api.Invoice` via `billinghttp.MapEventInvoiceToAPI()`
4. Sets `Version: EventPayloadVersionCurrent` on the payload
5. Lists matching rules for the event type
6. Calls `Service.CreateEvent()` per matching rule

**Important:** The consumer performs the billing → API type transformation. The rest of the notification package (`adapter/`, `httpdriver/`, `eventhandler/`) works exclusively with `api.Invoice`. This mirrors how entitlement events work.

Reference: `openmeter/notification/consumer/invoice.go`

## Event Handler & Reconciliation (`eventhandler/`)

The `EventHandler` interface provides:
- `Dispatch(ctx, event)` — async dispatch (goroutine with 30s timeout)
- `Reconcile(ctx)` — background loop every 15s, distributed-lock-protected via `lockr`
- `Start()` / `Close()` — lifecycle management

### Reconciliation Logic

For each non-terminal delivery status:

**PENDING:**
1. Check 3h timeout → FAILED if exceeded
2. Fetch webhook endpoints matching the channel
3. Send message to Svix if not yet sent
4. Update status based on Svix response

**SENDING:**
1. Check 48h timeout → FAILED if exceeded
2. Fetch message from Svix to get delivery status updates
3. Sync attempts (response codes, timestamps, durations)
4. Wait until all attempt data is collected before finalizing

**RESENDING:**
1. Resend message via Svix
2. Transition to SENDING

Reference: `openmeter/notification/eventhandler/reconcile.go`, `openmeter/notification/eventhandler/webhook.go`

## Webhook Integration (`webhook/`)

Abstracted behind the `webhook.Handler` interface with two implementations:
- `webhook/svix/` — production (talks to Svix API)
- `webhook/noop/` — no-op for tests and non-notification services

Key operations:
- `CreateWebhook` / `UpdateWebhook` — manages Svix application endpoints
- `SendMessage` — delivers event payload to Svix for webhook dispatch
- `GetMessage` — fetches delivery status from Svix
- `ResendMessage` — triggers a resend
- `RegisterEventTypes` — registers `notification.NotificationEventTypes` with the Svix app

Webhook error types (`webhook/errors.go`):
- `ValidationError` — input validation
- `NotFoundError` — resource not found in Svix
- `RetryableError` — transient, has `RetryAfter()` duration
- `UnrecoverableError` — permanent failure
- `MessageAlreadyExistsError` — duplicate message (idempotent send)

Reference: `openmeter/notification/webhook/webhook.go`, `openmeter/notification/webhook/svix/svix.go`

## HTTP API (`httpdriver/`)

Handlers implement the oapi-codegen server interface:

| Resource | Operations                                  |
|----------|---------------------------------------------|
| Channels | List, Create, Get, Update, Delete           |
| Rules    | List, Create, Get, Update, Delete, TestRule |
| Events   | List, Get, Resend                           |

Mapping functions in `httpdriver/mapping.go` convert between domain types and API types. Invoice and entitlement payload mapping is a trivial pass-through since payloads already use API types.

Reference: `openmeter/notification/httpdriver/handler.go`, `openmeter/notification/httpdriver/mapping.go`

## Adapter / Persistence (`adapter/`)

Ent ORM-backed implementation of `notification.Repository`:

### Database Tables

| Table                                | Purpose                                                       |
|--------------------------------------|---------------------------------------------------------------|
| `notification_channel`               | Webhook endpoints (URL, headers, signing secret)              |
| `notification_rule`                  | Event rules with type-specific config; M:N join to channels   |
| `notification_event`                 | Event records; payload stored as JSONB; FK to rule            |
| `notification_event_delivery_status` | Per-channel delivery tracking (state, attempts, next_attempt) |

### Entity Mapping (`adapter/entitymapping.go`)

`eventPayloadFromJSON(data []byte)` deserializes JSONB payloads:
- First-pass: unmarshal meta to get `type` + `version`
- For invoice types: guards `version == EventPayloadVersionCurrent` (rejects v0/unknown)
- Second-pass: full unmarshal into `EventPayload`

Reference: `openmeter/notification/adapter/entitymapping.go`

### What IS and IS NOT Persisted

**Persisted:** Every event that matches at least one active rule is atomically written to `notification_event` with per-channel `notification_event_delivery_status` rows.

**NOT persisted:** If no active rule matches, the system event is silently dropped. It exists only in Kafka.

**Events are never deleted:** There is no TTL, retention policy, or cleanup mechanism. All `notification_event` rows accumulate indefinitely. This has implications for any data migration work.

## Key Constants and Defaults

```go
DefaultReconcileInterval           = 15 * time.Second
DefaultDispatchTimeout             = 30 * time.Second
DefaultDeliveryStatePendingTimeout = 3 * time.Hour
DefaultDeliveryStateSendingTimeout = 48 * time.Hour

MaxChannelsPerRule     = 5
MaxChannelsPerWebhook  = 10

DefaultPageSize = 100
```

Reference: `openmeter/notification/defaults.go`

## Standard Annotations (`annotations.go`)

| Key                                         | Purpose                                     |
|---------------------------------------------|---------------------------------------------|
| `notification.rule.test`                    | Marks a test event (from TestRule endpoint) |
| `event.feature.key` / `event.feature.id`    | Feature metadata on entitlement events      |
| `event.subject.key` / `event.subject.id`    | Subject metadata                            |
| `event.customer.id` / `event.customer.key`  | Customer metadata                           |
| `event.balance.dedupe.hash`                 | Balance threshold dedup hash                |
| `event.invoice.id` / `event.invoice.number` | Invoice metadata                            |
| `event.resend.timestamp`                    | Timestamp of manual resend                  |

## DI Wiring

Two wire sets in `app/common/notification.go`:

- `Notification` — full production wiring (Svix webhook handler + real event handler with reconciliation loop)
- `NotificationService` — service-only wiring with no-op webhook and event handler (used by non-notification services like `cmd/server`)

```go
func NewNotificationService(
    logger *slog.Logger,
    adapter notification.Repository,
    webhook notificationwebhook.Handler,
    eventHandler notification.EventHandler,
    featureConnector feature.FeatureConnector,
) (notification.Service, error)
```

The `cmd/notification-service/` standalone worker wires the full consumer + Kafka subscriber + production Svix handler.

## Non-Obvious Pitfalls

- **Rules and channels are NOT auto-disabled after delivery failures.** Svix retries for up to 48h; after that the delivery status is `FAILED` but the rule stays active for future events.
- **Balance threshold dedup is rule-scoped.** The dedup hash includes `ruleID` — if a second rule is added for the same event type, it will independently trigger (no cross-rule dedup).
- **Invoice events skip `gathering` status.** The consumer explicitly checks `event.Invoice.Status` and skips if the invoice is still being assembled.
- **`notification_event` rows are never deleted.** Any migration or schema change must account for the full historical dataset. There is no natural expiry.
- **The reconciler is invisible to rule-less events.** The `HasDeliveryStatusesWith` EXISTS filter in the adapter means events with no delivery status rows (e.g., future persistence-only events) never appear in reconciliation queries.
- **Event payload versioning.** The `Version` field in `EventPayloadMeta` tracks the JSONB schema. The v0 → v1 migration for invoice payloads is complete; `eventPayloadFromJSON` rejects v0 invoice events. Non-invoice types pass through without version checks.
- **Svix channel mapping.** Each notification channel maps to a Svix endpoint. Channel IDs are stored in Svix message metadata (`ChannelIDMetadataKey = "om-channel-id"`). A `NullChannel` (`"__null_channel"`) is used as a Svix filter placeholder when no specific channel is targeted.

## Testing

### Running Tests

```bash
# All notification tests
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic -v ./openmeter/notification/...

# Adapter tests (including entity mapping unit tests)
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic -v ./openmeter/notification/adapter/...

# HTTP driver mapping tests
go test -v ./openmeter/notification/httpdriver/...
```

### Test Event Generator (`internal/`)

`TestEventGenerator` in `openmeter/notification/internal/rule.go` generates realistic test payloads for all event types. Used by the `TestRule` HTTP endpoint. For invoice events, it calls `billinghttp.MapEventInvoiceToAPI()` to produce an `InvoicePayload`.

### Adapter Entity Mapping Tests (`adapter/entitymapping_test.go`)

Unit tests for `eventPayloadFromJSON`:
- `TestEventPayloadFromJSON_V1Invoice` — v1 invoice round-trip
- `TestEventPayloadFromJSON_UnsupportedVersion` — rejects v0 and unknown versions
- `TestEventPayloadFromJSON_NonInvoiceType` — non-invoice types pass without version check
- `TestEventPayloadV1JSONShape` — verifies flat JSON structure (no `"Invoice"` nesting)

### HTTP Driver Mapping Tests (`httpdriver/mapping_test.go`)

- `TestFromEventAsInvoiceCreatedPayload` — happy path + nil guard
- `TestFromEventAsInvoiceUpdatedPayload` — happy path + nil guard

## Editing Checklist

When modifying consumer handlers:
- Each handler receives a decoded system event and calls `Service.CreateEvent()` per matching rule
- Entitlement events: check dedup hash logic; balance threshold dedup is rule-scoped
- Invoice events: transformation to API types happens in the consumer (eager), not later
- Always set `Version: EventPayloadVersionCurrent` on new payloads

When modifying event payloads:
- `EventPayload` is a union type — only one payload field should be non-nil per event type
- If adding a new event type: add the constant, the payload type, the `Validate()` case, and the consumer handler
- Register new event types in `webhook/svix/` (`NotificationEventTypes`)

When modifying the reconciliation loop:
- The reconciler runs every 15s with a distributed lock
- Delivery statuses flow: PENDING → SENDING → SUCCESS/FAILED
- Timeouts: 3h for PENDING, 48h for SENDING
- The reconciler is payload-blind except at `sendWebhookMessage` time

When modifying the adapter:
- `eventPayloadFromJSON` is the single deserialization entry point for all event payloads from the DB
- Invoice events require `version == EventPayloadVersionCurrent`; unknown versions are rejected
- Non-invoice types currently have no version guard

When modifying webhook integration:
- Svix is the external provider; all webhook calls go through the `webhook.Handler` interface
- Handle `RetryableError` (transient) vs `UnrecoverableError` (permanent) appropriately
- `MessageAlreadyExistsError` is expected on duplicate sends (idempotent)
