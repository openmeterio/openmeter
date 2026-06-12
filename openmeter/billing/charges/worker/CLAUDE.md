# worker

<!-- archie:ai-start -->

> Structural parent for charge-advancement workers. No source files of its own; its two children split how usage-based charges are advanced past their advance-after watermark: advance/ is the batch sweep across all customers, asyncadvance/ is the per-event Kafka handler. Both are thin orchestration over charges.ChargeService with no business logic.

## Patterns

**Config.Validate() + New constructor with nil-dependency guards** — Both workers require Logger and charges.ChargeService via a validated Config; constructors reject nil rather than falling back to slog.Default() (`advance.AutoAdvancer New, asyncadvance.Handler New`)
**Delegation-only over charges.ChargeService** — Workers map their trigger (namespace scan / AdvanceChargesEvent) into charges.AdvanceChargesInput and delegate; no advancement logic lives here (`asyncadvance.Handle builds AdvanceChargesInput from the event`)
**Error handling differs by transport** — advance/ accumulates per-customer failures into errors.Join (best-effort batch); asyncadvance/ returns the error so the message bus retries (`advance.All aggregates; asyncadvance.Handle returns err`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `advance/` | AutoAdvancer.All batch sweep paginating customers with charges past their advance-after watermark via pagination.CollectAll | Must NOT return early on first customer error — aggregate into errors.Join so remaining customers still advance |
| `asyncadvance/` | Watermill/Kafka Handler advancing charges for a single customer per charges.AdvanceChargesEvent | Returning nil on failure silently drops advancement — the returned error is the retry signal |

## Anti-Patterns

- Adding charge-advancement business logic in either worker — it belongs in charges.ChargeService
- advance/: returning on the first customer error instead of accumulating into errors.Join
- asyncadvance/: swallowing the Handle error and returning nil (drops the message-bus retry)
- Constructing a worker directly instead of via New/Config.Validate (bypasses nil guards)
- Falling back to slog.Default() instead of requiring Logger through Config

## Decisions

- **Batch sweep (advance/) and per-event handler (asyncadvance/) are separate packages** — Different invocation models and error semantics — best-effort aggregation vs single-event retry — keep cleanly separated

<!-- archie:ai-end -->
