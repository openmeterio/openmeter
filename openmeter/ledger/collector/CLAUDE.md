# collector

<!-- archie:ai-start -->

> Implements FBO-to-accrued credit collection and its correction (reversal) as a standalone service used by chargeadapter's flat-fee and usage-based handlers. Encapsulates the plan-then-execute pattern for multi-step accrual corrections with lineage-aware unwinding.

## Patterns

**Service interface with two operations** — collector.Service exposes exactly CollectToAccrued and CorrectCollectedAccrued; implementation delegates to private accrualCollector and accrualCorrector structs. Constructor is NewService(Config). (`func NewService(config Config) Service { return &service{collector: &accrualCollector{...}, corrector: &accrualCorrector{...}} }`)
**Plan-then-execute for corrections** — accrualCorrector.correct first builds a []plannedAction slice (plannedTransactionCorrection or plannedDirectInputs) for each correction item, then merges overlapping corrections by transaction ID, then calls transactions.CorrectTransaction for each merged plan in correctionOrder. (`actions := planCorrection(...); resolvedInputs := resolvePlannedInputs(...); c.ledger.CommitGroup(ctx, transactions.GroupInputs(...))`)
**Advance shortfall issuance in collect** — After resolving FBO→accrued templates, collect checks if settlement mode is CreditOnly and FBO didn't cover the full amount; if so it resolves IssueCustomerReceivable + TransferCustomerFBOAdvanceToAccrued for the shortfall and appends those to the same group. (`if shortfall := input.Amount.Sub(collectedFBOAmount); c.shouldAdvanceShortfall(input, shortfall) { advanceInputs, _ := c.resolveAdvanceInputs(ctx, input, shortfall); inputs = append(inputs, advanceInputs...) }`)
**Credit realization rows from FBO debits** — toCreditRealizations scans resolved inputs and creates one creditrealization.CreateAllocationInput per entry that debits AccountTypeCustomerFBO (negative amount on FBO). SortHint position in the slice maps corrections back to originals. (`for _, entry := range input.EntryInputs() { if entry.Amount().IsNegative() && entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO { out = append(out, CreateAllocationInput{Amount: entry.Amount().Abs(), ...}) } }`)
**Lineage-aware segment correction** — When correction lineage segments are present, planCorrection dispatches to planSegmentCorrection based on segment.State (RealCredit, AdvanceUncovered, AdvanceBackfilled). Backfilled advances require unwinding the backing credit-purchase group too. (`switch segment.State { case LineageSegmentStateRealCredit: return plannedSourceCorrectionActions(source, amount, false) ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Public interface Service, Config, CollectToAccruedInput, CorrectCollectedAccruedInput, and NewService constructor. All external callers use this file. | CollectToAccruedInput.Annotations is optional; if nil, ChargeAnnotations is synthesized from Namespace+ChargeID. CorrectCollectedAccruedInput.LineageSegmentsByRealization may be empty for legacy data. |
| `collect.go` | accrualCollector — collects FBO→accrued. Returns creditrealization.CreateAllocationInputs (one row per FBO debit). Private, used only by service.go. | shouldAdvanceShortfall only returns true for CreditOnlySettlementMode; CreditThenInvoice shortfall is not issued as advance. |
| `correct.go` | accrualCorrector — reverses prior collections using lineage-aware planning. Most complex file; plan/execute split prevents double-counting when multiple corrections touch the same source transaction. | reissueBackfilledCredit re-issues purchased credits back to FBO without triggering a new backfill sweep (intentional design). |

## Anti-Patterns

- Calling ledger.CommitGroup directly from charge handlers for FBO collection instead of routing through collector.Service
- Modifying the correction plan execution order (correctionOrder preserves insertion order; changing it breaks merging logic)
- Assuming CorrectCollectedAccrued is idempotent — it must only be called once per correction batch

## Decisions

- **Correction uses plan-then-execute with merged per-transaction amounts.** — Multiple realization corrections may reference the same original ledger transaction; merging before execution ensures the correction template receives one aggregated amount and produces a single well-formed correction entry per original transaction.
- **Backfilled advance corrections re-issue to FBO rather than immediately redirecting to another uncovered advance.** — Prevents correction from triggering a cascading backfill pass that could affect other outstanding advances unpredictably.

<!-- archie:ai-end -->
