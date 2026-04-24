# service

<!-- archie:ai-start -->

> Implements the flatfee.Service interface: charge creation, state-machine-driven lifecycle advancement, line engine integration, and invoice event hooks. It is the orchestration layer — it holds business logic but never owns state-machine transition rules (those live in typed StateMachine structs per settlement mode).

## Patterns

**Config+Validate before construction** — Every constructor accepts a typed Config struct and calls Config.Validate() before building the service. Missing dependencies are returned as joined errors, never panics. (`func New(config Config) (flatfee.Service, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**transaction.Run wraps all multi-step adapter calls** — Any method that calls the adapter more than once (or must be atomic) wraps the body in transaction.Run / transaction.RunWithNoValue, never calls the adapter bare. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) ([]flatfee.ChargeWithGatheringLine, error) { ... s.adapter.CreateCharges(ctx, ...) ... })`)
**withLockedCharge for mutation triggers** — AdvanceCharge and TriggerPatch always open a transaction AND acquire a pg_advisory lock via s.locker.LockForTX before fetching and mutating the charge. Never mutate a charge without both. (`return s.withLockedCharge(ctx, input.ChargeID, func(ctx context.Context, charge flatfee.Charge) (*flatfee.Charge, error) { ... stateMachine.AdvanceUntilStateStable(ctx) })`)
**Settlement-mode dispatch via newStateMachine** — triggers.go dispatches to the correct StateMachine subtype (e.g. CreditsOnlyStateMachine) by switching on charge.Intent.SettlementMode. Unsupported modes return models.NewGenericNotImplementedError. (`case productcatalog.CreditOnlySettlementMode: return NewCreditsOnlyStateMachine(config)`)
**Input.Validate() before any adapter call** — Every exported method validates its input struct before any DB or state-machine operation. Returning a raw fmt.Errorf for validation is wrong — use the typed Validate() result. (`if err := input.Validate(); err != nil { return nil, err }`)
**LineEngine as a thin wrapper over service** — LineEngine embeds *service and implements billing.LineEngine. It delegates heavy work (credit allocation, detailed-line generation, persistence) back to service methods rather than containing logic itself. (`var _ billing.LineEngine = (*LineEngine)(nil)
func (e *LineEngine) GetLineEngineType() billing.LineEngineType { return billing.LineEngineTypeChargeFlatFee }`)
**Currency rounding before any amount comparison or allocation** — Before passing an amount to handler or realizations, round with currencyCalculator.RoundToPrecision. Raw amounts from the DB must not be used in arithmetic comparisons. (`amount := currencyCalculator.RoundToPrecision(s.Charge.State.AmountAfterProration)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines Config, Validate, and New constructor; builds the flatfeerealizations.Service internally; exposes GetLineEngine(). This is the single wiring point — all dependencies enter here. | Adding dependencies that bypass Config.Validate; constructing flatfeerealizations.Service anywhere else. |
| `triggers.go` | Houses AdvanceCharge, TriggerPatch, newStateMachine, and withLockedCharge. All charge lifecycle mutations flow through here. | Any mutation that skips withLockedCharge (missing advisory lock); adding a settlement-mode branch without a corresponding StateMachine type. |
| `statemachine.go` | Defines the base stateMachine struct, StateMachineConfig, and newStateMachineBase. Wires Persistence callbacks (UpdateBase, Refetch) to the adapter. | Refetch must always pass ExpandRealizations expand; omitting it causes stale state in subsequent state-machine steps. |
| `creditsonly.go` | CreditsOnlyStateMachine: the only currently supported settlement mode. Defines state graph (Created→Active→Final→Deleted) and OnActive handlers that call Realizations. | AllocateCredits calls RoundToPrecision before passing to Realizations — do not remove. DeleteCharge must call RefetchCharge after adapter.DeleteCharge. |
| `invoice.go` | PostLineAssignedToInvoice and PostInvoiceIssued: hooks called by the line engine when a standard invoice is created or issued. Both run inside transactions. | PostInvoiceIssued must guard against accrued usage already existing (lifecycle violation check). Zero-total lines must skip ledger accrual. |
| `lineengine.go` | Implements billing.LineEngine for LineEngineTypeChargeFlatFee. OnStandardInvoiceCreated fetches charges, calls PostLineAssignedToInvoice, then CalculateLines, then persistDetailedLines. | CalculateLines requires a non-empty invoice ID and at least one line; returns fmt.Errorf if missing. SplitGatheringLine always errors — flat fee is not progressively billed. |
| `create.go` | Bulk charge creation: validates input, computes proration, resolves feature IDs, calls adapter.CreateCharges, then maps to ChargeWithGatheringLine. CreditOnly charges skip the gathering-line path. | intent.Normalized() must be called before CalculateAmountAfterProration. GatheringLine must carry Engine: billing.LineEngineTypeChargeFlatFee. |

## Anti-Patterns

- Making charge status transitions (state-machine decisions) inside create.go, invoice.go, or lineengine.go — all transitions must go through newStateMachine + AdvanceUntilStateStable or FireAndActivate
- Calling adapter methods outside a transaction.Run / transaction.RunWithNoValue block in any method that writes more than one row
- Mutating a charge via AdvanceCharge or TriggerPatch without the advisory lock from withLockedCharge
- Using raw (unrounded) currency amounts in allocation or comparison logic instead of calling currencyCalculator.RoundToPrecision first
- Returning fmt.Errorf for unsupported settlement modes instead of models.NewGenericNotImplementedError

## Decisions

- **Settlement-mode dispatch at the service layer via newStateMachine switch, not inside the generic statemachine package** — Each settlement mode has a distinct state graph; keeping dispatch in triggers.go makes it easy to add new modes without modifying shared infrastructure.
- **flatfeerealizations.Service is constructed inside New() and never exposed directly** — Realization mechanics are an implementation detail of this service; callers interact only through flatfee.Service, preventing direct realization calls that bypass lifecycle guards.
- **LineEngine is a separate struct that holds a pointer to service, not an embedded interface** — Keeps the billing.LineEngine registration surface separate from the flatfee.Service surface so callers cannot confuse the two responsibilities.

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
		})
		if err != nil {
			return nil, fmt.Errorf("new state machine: %w", err)
		}
		return stateMachine.AdvanceUntilStateStable(ctx)
// ...
```

<!-- archie:ai-end -->
