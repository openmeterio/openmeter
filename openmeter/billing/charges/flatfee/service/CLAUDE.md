# service

<!-- archie:ai-start -->

> Implements flatfee.Service: charge creation, state-machine-driven lifecycle advancement, line engine integration, and invoice event hooks. It is the orchestration layer — it holds business logic but delegates state-machine transition rules to typed StateMachine structs (CreditsOnlyStateMachine, CreditThenInvoiceStateMachine) dispatched by settlement mode.

## Patterns

**Config+Validate before construction** — Constructor accepts a typed Config struct and calls Config.Validate() before building the service. Missing dependencies accumulate via errors.Join, never panic. (`func New(config Config) (flatfee.Service, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**transaction.Run wraps all multi-step adapter calls** — Any method that calls the adapter more than once or must be atomic wraps the body in transaction.Run / transaction.RunWithNoValue. Never call the adapter bare from exported methods. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) ([]flatfee.ChargeWithGatheringLine, error) { ... s.adapter.CreateCharges(ctx, ...) ... })`)
**withLockedCharge for all charge mutations** — AdvanceCharge and TriggerPatch always open a transaction AND acquire a pg_advisory lock via s.locker.LockForTX before fetching and mutating the charge. The helper fetches with ExpandRealizations expand. (`return s.withLockedCharge(ctx, input.ChargeID, func(ctx context.Context, charge flatfee.Charge) (*flatfee.Charge, error) { stateMachine, _ := s.newStateMachine(...); return stateMachine.AdvanceUntilStateStable(ctx) })`)
**Settlement-mode dispatch via newStateMachine switch** — triggers.go dispatches to the correct StateMachine subtype by switching on charge.Intent.SettlementMode. Unsupported modes return models.NewGenericNotImplementedError, not fmt.Errorf. (`case productcatalog.CreditOnlySettlementMode: return NewCreditsOnlyStateMachine(config)`)
**Input.Validate() before any adapter call** — Every exported method validates its input struct before any DB or state-machine operation. Raw fmt.Errorf for validation failures is wrong — use the typed Validate() result. (`if err := input.Validate(); err != nil { return nil, err }`)
**Currency rounding before any amount comparison or allocation** — Before passing an amount to realizations or comparing amounts, round with currencyCalculator.RoundToPrecision. Raw amounts from the DB must not be used in arithmetic comparisons. (`amount := currencyCalculator.RoundToPrecision(s.Charge.State.AmountAfterProration)`)
**LineEngine as a thin wrapper over service** — LineEngine embeds *service and implements billing.LineEngine. It delegates heavy work back to service methods rather than containing logic itself. var _ billing.LineEngine = (*LineEngine)(nil) enforces the interface at compile time. (`func (e *LineEngine) GetLineEngineType() billing.LineEngineType { return billing.LineEngineTypeChargeFlatFee }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines Config, Validate, and New constructor; builds flatfeerealizations.Service internally; exposes GetLineEngine(). Single wiring point — all dependencies enter here. | Adding dependencies that bypass Config.Validate; constructing flatfeerealizations.Service anywhere else. |
| `triggers.go` | Houses AdvanceCharge, TriggerPatch, newStateMachine, and withLockedCharge. All charge lifecycle mutations flow through here. | Any mutation that skips withLockedCharge (missing advisory lock); adding a settlement-mode branch without a corresponding StateMachine type. |
| `statemachine.go` | Defines the base stateMachine struct, StateMachineConfig, and newStateMachineBase. Wires Persistence callbacks (UpdateBase, Refetch) to the adapter. | Refetch must always pass ExpandRealizations expand; omitting it causes stale state in subsequent state-machine steps. |
| `creditsonly.go` | CreditsOnlyStateMachine: state graph Created→Active→Final→Deleted. OnActive for Final state calls AllocateCredits then ClearAdvanceAfter. | AllocateCredits calls RoundToPrecision before passing to Realizations — do not remove. DeleteCharge must call RefetchCharge after adapter.DeleteCharge. |
| `creditheninvoice.go` | CreditThenInvoiceStateMachine: multi-step state graph with realization, collection, issuing, and payment settlement phases. Handles extend/shrink patches with invoice line mutation. | generateInvoicePatches must handle three cases: pre-active (no run), mutable run, and immutable run. UnsupportedExtendOperation / UnsupportedShrinkOperation must return GenericPreConditionFailedError. |
| `lineengine.go` | Implements billing.LineEngine for LineEngineTypeChargeFlatFee. OnStandardInvoiceCreated and OnInvoiceIssued fire state machine triggers per line. SplitGatheringLine always errors — flat fee is not progressively billed. | newStateMachineForStandardLine type-asserts to CreditThenInvoiceStateMachine and panics on CreditOnly charges; guard settlement mode before calling. |
| `payment.go` | postInvoicePaymentAuthorized and postInvoicePaymentSettled: ledger payment booking wrapped in transaction.RunWithNoValue. Both call handler.OnPaymentAuthorized/OnPaymentSettled for ledger postings. | getPaymentTotal errors if AccruedUsage is nil or zero on fiat-backed runs — do not call it for NoFiatTransactionRequired runs. |
| `linemapper.go` | populateFlatFeeStandardLineFromRun maps RealizationRun detailed lines onto a StandardLine and asserts totals equality. | run.DetailedLines.IsAbsent() check must pass before mapping — always fetch with ExpandRealizations. |

## Anti-Patterns

- Making charge status transitions (state-machine decisions) inside create.go, payment.go, or lineengine.go — all transitions must go through newStateMachine + AdvanceUntilStateStable or FireAndActivate
- Calling adapter methods outside a transaction.Run / transaction.RunWithNoValue block in any method that writes more than one row
- Mutating a charge via AdvanceCharge or TriggerPatch without the advisory lock from withLockedCharge
- Using raw (unrounded) currency amounts in allocation or comparison logic instead of calling currencyCalculator.RoundToPrecision first
- Returning fmt.Errorf for unsupported settlement modes instead of models.NewGenericNotImplementedError

## Decisions

- **Settlement-mode dispatch at the service layer via newStateMachine switch, not inside the generic statemachine package** — Each settlement mode has a distinct state graph; keeping dispatch in triggers.go makes it easy to add new modes without modifying shared infrastructure.
- **flatfeerealizations.Service is constructed inside New() and never exposed directly** — Realization mechanics are an implementation detail of this service; callers interact only through flatfee.Service, preventing direct realization calls that bypass lifecycle guards.
- **LineEngine is a separate struct holding a pointer to service, not an embedded interface** — Keeps the billing.LineEngine registration surface separate from the flatfee.Service surface so callers cannot confuse the two responsibilities.

## Example: Advancing a credit-only flat-fee charge through its state machine with advisory lock

```
// triggers.go
func (s *service) AdvanceCharge(ctx context.Context, input flatfee.AdvanceChargeInput) (*flatfee.Charge, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}
	return s.withLockedCharge(ctx, input.ChargeID, func(ctx context.Context, charge flatfee.Charge) (*flatfee.Charge, error) {
		stateMachine, err := s.newStateMachine(StateMachineConfig{
			Charge:       charge,
			Adapter:      s.adapter,
			Realizations: s.realizations,
			Service:      s,
		})
		if err != nil {
			return nil, fmt.Errorf("new state machine: %w", err)
		}
// ...
```

<!-- archie:ai-end -->
