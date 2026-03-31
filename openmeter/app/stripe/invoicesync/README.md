# Stripe Invoice Sync

Persistent sync plan pattern for synchronizing OpenMeter invoices to Stripe. Operations are persisted as an ordered plan and executed asynchronously with checkpointing, crash recovery, and idempotency.

## Overview

```text
 Billing State Machine                    Sync Plan System                          Stripe API
 =======================                  ================                          ==========

 draft.syncing ──────────> UpsertStandardInvoice()
                               │
                               ├─ Cancel ALL active sync plans for this invoice
                               ├─ Generate sync plan (ordered ops)
                               ├─ Persist sync plan + ops to DB (joins state machine tx)
                               └─ Publish ExecuteSyncPlanEvent ──────> Handler
                                    (deferred until after tx commits)
                                                                         │
                                                                    [advisory lock]
                                                                         │
                                                                    ExecuteNextOperation
                                                                         │
                                                              ┌──────────┴──────────┐
                                                              │                     │
                                                         Op pending            No more ops
                                                              │                     │
                                                         Call Stripe API       CompletePlan
                                                              │                     │
                                                         CompleteOperation     SyncDraftInvoice()
                                                              │                     │
                                                         SyncExternalIDs()     Write metadata
                                                              │                     │
                                                         Re-publish event      AdvanceStateMachine
                                                              │                     │
                                                         [next op...]               │
                                                                                    │
 draft.manual_approval <───────────────────────────────────────────── canDraftSyncAdvance() = true
```

## Plan Lifecycle

### Phases

Each sync plan maps to one billing invoice state machine phase:

| Phase      | Triggered by                  | Purpose                                  |
|------------|-------------------------------|------------------------------------------|
| `draft`    | `UpsertStandardInvoice()`     | Create/update invoice and lines in Stripe |
| `issuing`  | `FinalizeStandardInvoice()`   | Final sync + finalize invoice in Stripe   |
| `delete`   | `DeleteStandardInvoice()`     | Delete draft invoice from Stripe          |

### Plan Status Transitions

```text
 pending ────> executing ────> completed
                  │
                  └──────────> failed
```

- **pending**: Sync plan created, awaiting first execution event
- **executing**: At least one operation has been attempted
- **completed**: All operations finished successfully
- **failed**: A non-retryable error occurred; remaining ops canceled

### Operation Status Transitions

```text
 pending ────> completed     (success)
    │
    └────────> failed        (non-retryable error or plan failed)
```

## Operations by Phase

### Draft Phase: New Invoice

```text
Seq 0: InvoiceCreate ─────> Creates Stripe invoice
Seq 1: LineItemAdd   ─────> Adds all line items (if any)
```

### Draft Phase: Update Invoice

```text
Seq 0: InvoiceUpdate  ─────> Updates tax settings
Seq 1: LineItemRemove ─────> Removes deleted lines (before adding to avoid limits)
Seq 2: LineItemUpdate ─────> Updates changed lines
Seq 3: LineItemAdd    ─────> Adds new lines
```

Operations are only included if needed (e.g., no `LineItemRemove` if nothing to remove).

If a line references a Stripe ID that no longer exists on the Stripe invoice, the planner treats it as a new line and adds it (rather than skipping or erroring).

### Issuing Phase

```text
Seq 0..N-1: [Same as draft update]  ─────> Full re-sync to Stripe
Seq N:      InvoiceFinalize          ─────> Finalize + auto-advance
```

### Delete Phase

```text
Seq 0: InvoiceDelete ─────> Deletes Stripe invoice
```

If the invoice has no Stripe external ID, the plan is empty (no-op).

## Integration with Billing Invoice States

### Draft Sync

```text
 draft.validating
       │
       │ validation passes
       v
 draft.syncing ─────────────────────> [sync plan created + executed async]
       │                                        │
       │  canDraftSyncAdvance()                 │ SyncDraftInvoice() writes metadata:
       │  checks metadata key                   │   openmeter.io/billing/draft-sync-completed-at
       │                                        │
       v                                        v
 draft.manual_approval_needed    <──── state machine advances when
   OR                                  metadata key is present
 draft.waiting_auto_approval
```

When re-entering `draft.syncing` (e.g., after an invoice update), the state machine clears old sync completion metadata first. This ensures `canDraftSyncAdvance` returns false until the new sync plan completes — otherwise stale metadata from a previous sync would cause immediate advancement.

### Issuing Sync

```text
 draft.ready_to_issue
       │
       v
 issuing.syncing ───────────────────> [issuing plan created + executed async]
       │                                        │
       │  canIssuingSyncAdvance()               │ SyncIssuingInvoice() writes metadata:
       │  checks metadata key                   │   openmeter.io/billing/issuing-sync-completed-at
       │                                        │
       v                                        v
 issued                          <──── state machine advances when
                                       metadata key is present
```

### Failure

```text
 draft.syncing ──────> [plan fails] ──────> FailSyncInvoice ──────> draft.sync_failed
 issuing.syncing ────> [plan fails] ──────> FailSyncInvoice ──────> issuing.failed
```

The Stripe error is surfaced as a validation issue on the invoice via `FailSyncInvoice`, making it visible to API consumers. The operation that failed (sync, finalize, or delete) is recorded alongside the error.

## Execution Model

### One Operation Per Event

The handler processes exactly one operation per `ExecuteSyncPlanEvent`, then re-publishes the same event for the next operation. This enables:

- **Crash recovery**: If the worker dies mid-operation, the message is redelivered and execution resumes from the first non-completed operation
- **Checkpointing**: Each completed operation is persisted before moving to the next
- **Backpressure**: Operations flow through the normal Kafka consumer pipeline

```text
Event 1: Execute op 0 (InvoiceCreate)  ──> complete ──> SyncExternalIDs ──> re-publish
Event 2: Execute op 1 (LineItemAdd)    ──> complete ──> SyncExternalIDs ──> re-publish
Event 3: No pending ops                ──> CompletePlan ──> SyncDraftInvoice
```

### Advisory Lock

Each handler invocation acquires a PostgreSQL advisory lock scoped to `(namespace, invoiceID)` to prevent parallel workers from executing operations on the same invoice:

```text
Handler.Handle(event)
  │
  ├─ Pre-lock check (skip terminal plans without lock)
  │
  └─ BEGIN TRANSACTION
       │
       ├─ pg_advisory_xact_lock(hash(namespace:invoice_sync:invoiceID))
       │    └─ If locked by another worker: skip (ErrSyncPlanLocked)
       │
       ├─ Re-fetch plan (another worker may have advanced it)
       │    └─ If now terminal: skip
       │
       ├─ Check if a newer plan exists for this invoice (isSuperseded)
       │    └─ If superseded: cancel this plan and skip
       │
       ├─ Execute operation
       │
       └─ COMMIT (releases lock) ──> publish post-tx event (if any)
```

The superseded check is critical because `cancelAllActivePlans` runs at plan creation time, but the older plan's Kafka events may already be in flight. Without this check, both plans would execute concurrently between lock releases, causing duplicate Stripe API calls.

### Superseding Stale Sync Plans

When a new plan is created for an invoice, **all** active plans for that invoice are canceled regardless of phase. This prevents concurrent execution of draft and issuing plans for the same invoice:

```text
CreateDraftSyncPlan(invoice)
  │
  ├─ cancelAllActivePlans(namespace, invoiceID)
  │    └─ For each active plan (any phase): FailPlan("superseded by new sync plan")
  │
  ├─ buildPlanGeneratorInput (fetch Stripe customer data, existing lines)
  ├─ GenerateDraftSyncPlan (diff current state → ordered ops)
  └─ createAndPublish (persist + defer event until after tx commits)
```

Because external IDs are written back incrementally after each operation (see below), the invoice already has the correct Stripe state when a plan is canceled. The new plan naturally generates update operations instead of duplicate creates.

When a canceled plan's handler picks up the next event, it sees the plan is in `failed` status and skips execution. If the plan had already completed all its Stripe operations before cancellation, the `handlePlanCompletion` call may encounter an `InvoiceStateMismatchError` (invoice already advanced past the expected state), which is treated as a stale no-op.

### Incremental External ID Sync-Back

After each successful `InvoiceCreate` or `LineItemAdd` operation, the handler immediately writes external IDs back to the billing invoice via `billingService.SyncExternalIDs()`. This keeps the invoice's Stripe state current at all times:

```text
Operation completes
  │
  ├─ InvoiceCreate  ──> Write invoice.ExternalIDs.Invoicing = stripe_invoice_id
  │
  ├─ LineItemAdd    ──> Write line.ExternalIDs.Invoicing for each line + discount
  │
  └─ Other ops      ──> No external IDs to write
```

If the sync-back fails, the entire transaction rolls back (including the `CompleteOperation` call), and Kafka redelivers the event. The idempotency key ensures no duplicate Stripe API calls on retry.

### Idempotency

Each operation has a deterministic idempotency key:

```text
key = SHA256(length-prefixed(invoiceID) + length-prefixed(sessionID)
           + length-prefixed(opType) + int64(sequence))
```

- `sessionID` is a ULID generated per plan, scoping keys to a sync session
- Fields are length-prefixed to prevent aliasing across field boundaries
- If the same plan is re-executed (crash recovery), the same keys are used
- Stripe deduplicates requests with matching idempotency keys

### Stripe Invoice ID Resolution

For create flows, the Stripe invoice ID isn't known until `InvoiceCreate` completes. Subsequent operations resolve it via:

1. Check the operation's own payload (update/finalize/delete ops embed it)
2. Fall back to scanning completed `InvoiceCreate` responses in the same plan

### Transaction Participation and Post-Commit Publishing

`CreateSyncPlan` and `FailPlan` use `entutils.TransactingRepo` to join any existing transaction from the context. This is critical when the billing state machine calls `UpsertStandardInvoice` within its own transaction — the sync plan's FK to `billing_invoices` must see the uncommitted invoice row.

Event publishing is deferred until after the transaction commits:

- **Plan creation** (`service/service.go`): Uses `transaction.OnCommit()` to defer the initial Kafka publish until the outermost transaction commits, guaranteeing the sync plan is visible in the DB when the worker processes the event.
- **Handler re-publish** (`handler.go`): Uses `transaction.Run` to return a `postTxEvent` from the transaction body. The event is published after `Run` returns (i.e., after commit) using the original non-transaction context, keeping publish errors in the normal error-handling flow.

## Error Handling

### Retryable vs Non-Retryable

```text
Stripe Error                 Classification      Action
─────────────────────────────────────────────────────────
HTTP 429 (Rate Limited)      Retryable           Return error; Watermill retries
HTTP 500 (Server Error)      Retryable           Return error; Watermill retries
HTTP 502 (Bad Gateway)       Retryable           Return error; Watermill retries
HTTP 503 (Unavailable)       Retryable           Return error; Watermill retries
HTTP 504 (Timeout)           Retryable           Return error; Watermill retries
HTTP 400 (Bad Request)       Non-retryable       Fail operation + plan
HTTP 402 (Payment Required)  Non-retryable       Fail operation + plan
HTTP 404 (Not Found)         Non-retryable       Fail operation + plan
Non-Stripe error             Non-retryable       Fail operation + plan
```

### Non-Retryable Error Flow

```text
Operation fails (400)
  │
  ├─ Mark operation as failed (store error message)
  ├─ Mark all remaining pending ops as failed ("plan failed: ...")
  ├─ Mark plan as failed (store error message)
  │    └─ If any DB update fails: return error → transaction rolls back → Kafka retries
  │
  └─ Handler calls FailSyncInvoice (surfaces error as validation issue)
       └─ State machine transitions to sync_failed / issuing.failed
```

### Tax Location Error (Special Case)

During `InvoiceFinalize`, if Stripe returns a tax location error:

```text
FinalizeInvoice() → tax_location_invalid error
  │
  ├─ If tax IS enforced:
  │    └─ Non-retryable error (plan fails)
  │
  └─ If tax is NOT enforced:
       ├─ UpdateInvoice(automatic_tax: false)  ← disable tax
       └─ FinalizeInvoice() again              ← retry without tax
```

## Configuration

The sync plan is always on. `SyncPlanAdapter` and `Publisher` are required fields on the Stripe `App` struct — validated at construction time via `App.Validate()` and `service.Config.Validate()`. There is no synchronous fallback.

## Database Tables

| Table                             | Purpose                                    |
|-----------------------------------|--------------------------------------------|
| `app_stripe_invoice_sync_plans`   | One row per sync session (plan + status)   |
| `app_stripe_invoice_sync_ops`     | One row per Stripe API operation           |

Plans are linked to billing invoices via FK (cascade delete). Operations are linked to plans via FK (cascade delete).

## File Structure

```text
openmeter/app/stripe/invoicesync/
  types.go          ── Domain types, enums, payload/response structs, metadata keys
  service.go        ── Service interface (create/cancel sync plans)
  planner.go        ── Plan generation: diff invoice state → ordered operations
  executor.go       ── Sequential operation execution with error classification
  handler.go        ── Event handler: lock → execute → complete/fail → advance
  events.go         ── ExecuteSyncPlanEvent definition
  currency.go       ── Currency conversion helpers for Stripe amounts
  adapter.go        ── Persistence interface
  service/
    service.go      ── Service implementation (plan lifecycle + post-commit publishing)
  adapter/
    adapter.go      ── Ent/PostgreSQL implementation
```
