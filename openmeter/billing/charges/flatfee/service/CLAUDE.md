# service

<!-- archie:ai-start -->

> Implements flatfee.Service: charge creation, state-machine-driven lifecycle advancement, line-engine integration, and invoice event hooks. The orchestration layer — it holds business logic but delegates transition rules to typed StateMachine structs (CreditsOnlyStateMachine, CreditThenInvoiceStateMachine) dispatched by settlement mode; its realizations/ child owns credit allocation/correction persistence only.

## Patterns

**Config+Validate before construction** — New(config) calls config.Validate() (errors.Join, never panic) and builds flatfeerealizations.Service internally; all dependencies enter through Config. (`func New(config Config) (flatfee.Service, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**transaction.Run wraps all multi-step adapter calls** — Any method that calls the adapter more than once or must be atomic wraps its body in transaction.Run / RunWithNoValue; never call the adapter bare from an exported method. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) ([]flatfee.ChargeWithGatheringLine, error) { ... s.adapter.CreateCharges(ctx, ...) ... })`)
**withLockedCharge for all charge mutations** — AdvanceCharge and TriggerPatch open a transaction AND acquire a pg_advisory lock via s.locker.LockForTX before fetching (with ExpandRealizations) and mutating the charge. (`return s.withLockedCharge(ctx, input.ChargeID, func(ctx context.Context, charge flatfee.Charge) (*flatfee.Charge, error) { sm, _ := s.newStateMachine(...); return sm.AdvanceUntilStateStable(ctx) })`)
**Settlement-mode dispatch via newStateMachine switch** — triggers.go switches on charge.Intent.SettlementMode to build the correct StateMachine subtype; unsupported modes return models.NewGenericNotImplementedError, not fmt.Errorf. (`case productcatalog.CreditOnlySettlementMode: return NewCreditsOnlyStateMachine(config)`)
**Input.Validate() before any adapter or state-machine call** — Every exported method validates its input struct first; use the typed Validate() result rather than raw fmt.Errorf for validation failures. (`if err := input.Validate(); err != nil { return nil, err }`)
**Currency rounding before any amount comparison or allocation** — Round amounts with currencyCalculator.RoundToPrecision before passing to realizations or comparing; raw DB amounts must not be used in arithmetic. (`amount := currencyCalculator.RoundToPrecision(s.Charge.State.AmountAfterProration)`)
**LineEngine as a thin wrapper over service** — LineEngine embeds a *service pointer and delegates heavy work back to service methods; var _ billing.LineEngine = (*LineEngine)(nil) enforces the interface at compile time. (`func (e *LineEngine) GetLineEngineType() billing.LineEngineType { return billing.LineEngineTypeChargeFlatFee }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config, Validate, New constructor; builds flatfeerealizations.Service internally; exposes GetLineEngine(). Single wiring point — all dependencies enter here. | Adding dependencies that bypass Config.Validate; constructing flatfeerealizations.Service anywhere else. |
| `triggers.go` | AdvanceCharge, TriggerPatch, newStateMachine, withLockedCharge — all charge lifecycle mutations flow through here. | Any mutation that skips withLockedCharge (missing advisory lock); adding a settlement-mode branch without a corresponding StateMachine type. |
| `statemachine.go` | Base stateMachine struct, StateMachineConfig, newStateMachineBase; wires Persistence callbacks (UpdateBase, Refetch) to the adapter. | Refetch must always pass ExpandRealizations; omitting it causes stale state in later state-machine steps. |
| `creditsonly.go` | CreditsOnlyStateMachine: Created→Active→Final→Deleted; Final's OnActive runs AllocateCredits then ClearAdvanceAfter. | AllocateCredits calls RoundToPrecision before Realizations — do not remove; DeleteCharge must RefetchCharge after adapter.DeleteCharge. |
| `creditheninvoice.go` | CreditThenInvoiceStateMachine: multi-step graph with realization, collection, issuing, payment settlement; handles extend/shrink patches with invoice line mutation. | generateInvoicePatches must handle three cases (pre-active no-run, mutable run, immutable run); UnsupportedExtend/ShrinkOperation must return GenericPreConditionFailedError. |
| `lineengine.go` | billing.LineEngine for LineEngineTypeChargeFlatFee; OnStandardInvoiceCreated / OnInvoiceIssued fire state-machine triggers per line; SplitGatheringLine always errors. | newStateMachineForStandardLine type-asserts to CreditThenInvoiceStateMachine — guard settlement mode before calling so CreditOnly charges don't reach it. |
| `payment.go` | postInvoicePaymentAuthorized / postInvoicePaymentSettled: ledger payment booking wrapped in transaction.RunWithNoValue via handler.OnPaymentAuthorized/OnPaymentSettled. | getPaymentTotal errors if AccruedUsage is nil/zero on fiat-backed runs — do not call it for NoFiatTransactionRequired runs. |
| `linemapper.go` | populateFlatFeeStandardLineFromRun maps a RealizationRun's detailed lines onto a StandardLine and asserts totals equality. | run.DetailedLines.IsAbsent() check must pass first — always fetch with ExpandRealizations. |

## Anti-Patterns

- Making charge status transitions inside create.go, payment.go, or lineengine.go — all transitions go through newStateMachine + AdvanceUntilStateStable or FireAndActivate
- Calling adapter methods outside a transaction.Run / RunWithNoValue block in any method that writes more than one row
- Mutating a charge via AdvanceCharge or TriggerPatch without the advisory lock from withLockedCharge
- Using raw (unrounded) currency amounts in allocation or comparison instead of currencyCalculator.RoundToPrecision
- Returning fmt.Errorf for unsupported settlement modes instead of models.NewGenericNotImplementedError

## Decisions

- **Settlement-mode dispatch at the service layer via newStateMachine switch, not inside the generic statemachine package** — Each settlement mode has a distinct state graph; keeping dispatch in triggers.go makes it easy to add new modes without modifying shared infrastructure.
- **flatfeerealizations.Service is constructed inside New() and never exposed directly** — Realization mechanics are an implementation detail; callers interact only through flatfee.Service, preventing direct realization calls that bypass lifecycle guards.
- **LineEngine is a separate struct holding a pointer to service, not an embedded interface** — Keeps the billing.LineEngine registration surface separate from the flatfee.Service surface so callers cannot confuse the two responsibilities.

## Example: Advancing a flat-fee charge through its state machine under the advisory lock

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
