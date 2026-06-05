# service

<!-- archie:ai-start -->

> Service layer for flat-fee charges: owns the per-settlement-mode lifecycle state machines (credit_only, credit_then_invoice), the billing.LineEngine implementation that bridges charges to standard invoice lines, charge creation, and fiat payment authorization/settlement. It makes all lifecycle/state decisions and delegates realization mechanics (credit math, line/totals persistence) to the realizations/ child Service.

## Patterns

**Public methods are transaction.Run wrappers over the adapter** — Every exported service method validates its input then runs the body inside transaction.Run(ctx, s.adapter, ...) (or RunWithNoValue). Reads and writes both go through this so the adapter rebinds to the tx in ctx. (`func (s *service) GetByID(ctx, input) (...) { if err := input.Validate(); err != nil { return ... }; return transaction.Run(ctx, s.adapter, func(ctx) {...}) }`)
**One state machine type per settlement mode** — Each settlement mode has its own *StateMachine embedding *stateMachine. Constructors (NewCreditsOnlyStateMachine, NewCreditThenInvoiceStateMachine) hard-assert config.Charge.Intent.SettlementMode, call newStateMachineBase, then configureStates(). newStateMachine() in triggers.go dispatches by SettlementMode and errors with NewGenericNotImplementedError otherwise. (`if config.Charge.Intent.SettlementMode != productcatalog.CreditThenInvoiceSettlementMode { return nil, fmt.Errorf(...) }`)
**States declared via fluent Configure/Permit/OnActive DSL** — configureStates() wires transitions with s.Configure(status).Permit(meta.TriggerX, nextStatus, statelessx.BoolFn(guard)).InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).OnActive(action). Guards are bool methods; actions are ctx-taking methods on the machine. (`s.Configure(flatfee.StatusCreated).Permit(meta.TriggerNext, flatfee.StatusActive, statelessx.BoolFn(s.IsInsideServicePeriod))`)
**LineEngine drives the machine, never persists lines directly** — LineEngine handlers (OnStandardInvoiceCreated, OnInvoiceIssued, OnCollectionCompleted) load the charge, build a state machine, Fire/AdvanceUntilStateStable, then mutate the passed *billing.StandardLine in place via populateFlatFeeStandardLineFromRun and return it; billing persists. (`stateMachine.FireAndActivate(ctx, meta.TriggerFinalInvoiceCreated, billing.StandardLineWithInvoiceHeader{Line: stdLine, Invoice: input.Invoice})`)
**Mutate charge in-memory; the machine persists on stabilization** — Action methods set s.Charge.State/Intent/Status/Realizations.CurrentRun and AddInvoicePatch(...); they do NOT call UpdateCharge themselves (chargestatemachine.Persistence.UpdateBase handles it). AdvanceAfter is cleared via the StatusFinal ClearAdvanceAfter OnActive hook. (`s.Charge.Realizations.CurrentRun = &result.Run; s.Charge.State.AdvanceAfter = nil`)
**Charge mutation always under the charge lock** — AdvanceCharge and TriggerPatch route through withLockedCharge, which takes charges.NewLockKeyForCharge + locker.LockForTX inside a tx, re-fetches with meta.ExpandRealizations, then runs fn. Invoice patches are drained via stateMachine.DrainInvoicePatches() into the result. (`key, _ := charges.NewLockKeyForCharge(chargeID); s.locker.LockForTX(ctx, key)`)
**Round through the currency Calculator before compare/persist** — Amounts are rounded via intent.Currency.Calculator().RoundToPrecision before negativity checks, credit allocation, or totals equality assertions. Mapped line totals are reconciled against run.Totals and an error returned on mismatch. (`amount := currencyCalculator.RoundToPrecision(s.Charge.State.AmountAfterProration); if amount.IsNegative() { return fmt.Errorf(...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config/New constructor + service struct; wires realizations sub-Service, holds creditNotesSupported atomic.Bool; GetLineEngine() returns *LineEngine; asserts flatfee.FlatFeeService. | SetCreditNotesSupportedByLineUpdater requires *testing.T and must never run in production; New seeds creditNotesSupported from charges.CreditNotesSupportedByLineUpdater. |
| `statemachine.go` | stateMachine base embedding chargestatemachine.Machine; newStateMachineBase wires Persistence{UpdateBase, Refetch}; shared guards/actions (IsInsideServicePeriod, AdvanceAfterServicePeriodFrom/To, ClearAdvanceAfter). | Refetch always expands meta.ExpandRealizations; guards read clock.Now() so tests must clock.FreezeTime. |
| `creditheninvoice.go` | CreditThenInvoiceStateMachine: full realization lifecycle (active -> realization started/processing/issuing/completed -> awaiting payment settlement -> final); generateInvoicePatches handles extend/shrink and the immutable-invoice fallback. | When !CreditNotesSupported and currentRun.Immutable with a changed amount, only a delete patch is emitted to avoid double-charging; zero-amount patches drive StatusFinal/StatusCreated transitions explicitly. |
| `creditsonly.go` | CreditsOnlyStateMachine: created -> active -> final, AllocateCredits at final (via realizations.AllocateCreditsOnly), DeleteCharge honors CreditRefundPolicyCorrect. | Credit-only charges never touch the invoicing stack; DeleteCharge must CorrectAllCredits on existing lineage, not create raw negative rows. |
| `lineengine.go` | billing.LineEngine impl (LineEngineTypeChargeFlatFee): BuildStandardInvoiceLines, OnStandardInvoiceCreated, OnInvoiceIssued, OnCollectionCompleted, OnMutableStandardLinesDeleted, OnUnsupportedCreditNote, payment hooks; getChargesForStandardLineEvent batch-loads charges. | SplitGatheringLine returns an error (flat fee is not progressively billed); deletion handlers reject runs with AccruedUsage/Payment or that are still CurrentRun, and correct credits via realizations.CorrectAllCredits. |
| `payment.go` | postInvoicePaymentAuthorized/Settled: book fiat payment via handler.OnPaymentAuthorized/OnPaymentSettled and persist payment settlement rows; getPaymentTotal/validatePaymentRunForLine helpers. | getPaymentTotal errors on no-fiat runs, missing AccruedUsage, or zero totals; NoFiatTransactionRequired runs short-circuit before any payment booking. |
| `create.go` | Create(): bulk-builds flatfee.IntentWithInitialStatus, resolves feature IDs, calls adapter.CreateCharges, then builds gathering lines for credit_then_invoice non-zero charges via buildFlatFeeGatheringLine. | Credit-only and zero-amount charges return ChargeWithGatheringLine with no GatheringLineToCreate; buildFlatFeeGatheringLine rejects non-credit_then_invoice charges. |
| `triggers.go` | AdvanceCharge, TriggerPatch, newStateMachine dispatcher, withLockedCharge helper. | CreditNotesSupported is read via .Load() each time a machine is built; TriggerPatch drains invoice patches into meta.TriggerPatchResult. |

## Anti-Patterns

- Calling adapter writes (CreateCharges, CreatePayment, UpdateRealizationRun) outside a transaction.Run/RunWithNoValue block.
- Persisting the charge from inside an action method instead of mutating s.Charge and letting chargestatemachine.Persistence.UpdateBase write it on stabilization.
- Putting credit math, detailed-line construction, or totals persistence here instead of delegating to the realizations sub-Service (s.Realizations / s.service.realizations).
- Mutating or persisting invoice lines directly from a LineEngine handler instead of mutating the passed *billing.StandardLine and emitting invoiceupdater.Patch values.
- Mutating a charge without withLockedCharge, or skipping in.Validate()/RoundToPrecision before comparing or persisting amounts.

## Decisions

- **State machines are split per settlement mode rather than one parameterized machine.** — credit_only and credit_then_invoice have fundamentally different lifecycles (no invoicing stack vs. realization+payment), so each owns its own Configure graph and guards.
- **creditNotesSupported is an atomic.Bool read per state-machine build.** — Until the line updater can correct immutable invoice lines with credit notes, extend/shrink on already-invoiced periods must degrade to delete-only patches; the flag lets tests toggle the future behavior without rewiring.
- **The service makes lifecycle decisions; realizations/ is mechanics-only.** — Keeps the state graph and invoice-patch routing testable in one place while credit allocation/correction and detailed-line persistence stay a pure mechanics layer that cannot make transitions.

## Example: A LineEngine handler advances the charge state machine and maps the resulting run back onto the standard line.

```
func (e *LineEngine) OnInvoiceIssued(ctx context.Context, input billing.OnInvoiceIssuedInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating input: %w", err)
	}
	for _, stdLine := range input.Lines {
		stateMachine, err := e.newStateMachineForStandardLine(ctx, stdLine)
		if err != nil {
			return err
		}
		if err := stateMachine.FireAndActivate(ctx, meta.TriggerInvoiceIssued, billing.StandardLineWithInvoiceHeader{
			Line:    stdLine,
			Invoice: input.Invoice,
		}); err != nil {
			return fmt.Errorf("triggering invoice_issued for charge[%s]: %w", stateMachine.GetCharge().ID, err)
		}
// ...
```

<!-- archie:ai-end -->
