# flatfee

<!-- archie:ai-start -->

> Public domain contract package for flat-fee charges: declares all domain types (Charge, Intent, Status, Realizations, DetailedLines, RealizationRun), the Adapter composite interface, the Service interface, and the Handler interface for ledger/credit callbacks. It is the contract boundary between flatfee/service and flatfee/adapter sub-packages — no business logic or DB access lives here.

## Patterns

**Input.Validate() before any operation** — Every Input struct (CreateChargesInput, GetByIDsInput, IntentWithInitialStatus, AdvanceChargeInput, CreateCurrentRunInput, etc.) implements Validate() that collects errors with errors.Join and wraps via models.NewNillableGenericValidationError. Callers must invoke Validate() as the first statement before any adapter or service call. (`if err := input.Validate(); err != nil { return nil, err }`)
**Adapter is a composite of fine-grained sub-interfaces + entutils.TxCreator** — Adapter embeds ChargeAdapter, ChargeDetailedLineAdapter, ChargeCreditAllocationAdapter, ChargeRunAdapter, ChargeInvoicedUsageAdapter, ChargePaymentAdapter, and entutils.TxCreator. Callers that only need a sub-interface can depend on the narrower type. No direct methods live on the Adapter composite. (`type Adapter interface { ChargeAdapter; ChargeDetailedLineAdapter; ...; entutils.TxCreator }`)
**Expand dependency validation** — validateExpands enforces that ExpandDetailedLines requires ExpandRealizations, and ExpandDeletedRealizations requires ExpandRealizations. Violations return fmt.Errorf wrapped by the caller's Validate(). (`if expands.Has(meta.ExpandDetailedLines) && !expands.Has(meta.ExpandRealizations) { return fmt.Errorf(...) }`)
**Currency rounding in Intent.Normalized() and CalculateAmountAfterProration** — Intent.Normalized() calls calc.RoundToPrecision on AmountBeforeProration. CalculateAmountAfterProration also rounds the prorated result to currency precision. Never compare or store amounts without rounding first. (`i.AmountBeforeProration = calc.RoundToPrecision(i.AmountBeforeProration)`)
**Status as typed string with Values() + Validate() + ToMetaChargeStatus()** — Status is a typed string with an exhaustive Values() []string method and a Validate() that checks slices.Contains. ToMetaChargeStatus() splits on '.' to extract the top-level meta.ChargeStatus. New statuses must be added to Values() or Validate() will reject them. (`func (s Status) Validate() error { if !slices.Contains(s.Values(), string(s)) { return models.NewGenericValidationError(...) } }`)
**Handler interface for external ledger/credit callbacks declared here, not in flatfee/service** — Handler declares OnAssignedToInvoice, OnInvoiceUsageAccrued, OnCreditsOnlyUsageAccrued, OnCreditsOnlyUsageAccruedCorrection, OnPaymentAuthorized, OnPaymentSettled, OnPaymentUncollectible — all returning typed result structs. Placing Handler here (not in flatfee/service) breaks the import cycle: flatfee/service imports flatfee for types. (`OnAssignedToInvoice(ctx context.Context, input OnAssignedToInvoiceInput) (creditrealization.CreateAllocationInputs, error)`)
**ProRatingModeAdapterEnum with NoProratingAdapterMode for DB persistence** — ProRatingModeAdapterEnum extends productcatalog.ProRatingMode with a NoProratingAdapterMode value ('no_prorate') that has no productcatalog counterpart. Only the adapter layer uses this enum for DB column mapping; service-layer logic must use productcatalog.ProRatingConfig. (`NoProratingAdapterMode ProRatingModeAdapterEnum = "no_prorate"`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Declares the Adapter composite interface and all adapter-facing input types (CreateChargesInput, GetByIDInput, GetByIDsInput, IntentWithInitialStatus, CreateCurrentRunInput, CreateInvoicedUsageInput). New persistence methods must be added to one of the six sub-interfaces here. | Do not add conversion logic here; that belongs in flatfee/adapter/mapper.go. All input types must implement Validate(). |
| `charge.go` | Core domain types: ChargeBase, Charge, Intent, State, Realizations. Intent.CalculateAmountAfterProration owns all proration math — it rounds results to currency precision and never increases AmountBeforeProration when servicePeriod >= fullServicePeriod. | CalculateAmountAfterProration guard: servicePeriodDuration >= fullServicePeriodDuration returns the unprorated amount. Charge.WithStatus/WithBase return value copies (not pointer mutations) — required by the generic state machine. |
| `service.go` | Declares Service (= FlatFeeService + GetLineEngine + InvoiceLifecycleHooks is implied via flatfee/service embedding), CreateInput, AdvanceChargeInput, GetByIDsInput for the service layer. InvoiceLifecycleHooks methods are called by the billing engine. | Never add billing.Service dependencies to this interface file. ChargeWithGatheringLine wraps a Charge with an optional GatheringLine for atomic creation. |
| `handler.go` | Declares Handler interface for ledger/credit event callbacks. Each OnXxx input struct implements Validate() and some implement ValidateWith(currencyCalculator) when currency precision is required. | Handler implementations live outside this package (in flatfee/service or ledger/chargeadapter). Do not add business logic in the interface file. |
| `statemachine.go` | Defines Status type, Values(), Validate(), ToMetaChargeStatus(). Flat-fee statuses are hierarchical: sub-statuses use dot notation (e.g. 'active.realization.started'); ToMetaChargeStatus() extracts the top-level prefix. | New statuses must be added to Values() or runtime validation will reject them. ToMetaChargeStatus splits on '.' — ensure new sub-statuses follow the 'toplevel.substate' convention. |
| `realizationrun.go` | RealizationRunBase, RealizationRun, RealizationRuns, RealizationRunID, UpdateRealizationRunInput. RealizationRun uses mo.Option[DetailedLines] so detailed lines are distinguished from 'not loaded' vs 'empty'. IsVoidedBillingHistory covers both soft-deleted and unsupported-credit-note runs. | UpdateRealizationRunInput uses mo.Option fields — a field absent from a partial update must remain mo.None, not zero value. RealizationRunTypeInvalidDueToUnsupportedCreditNote must never be an InitialType. |
| `detailedline.go` | DetailedLine is a type alias for stddetailedline.Base; changes to stddetailedline.Base propagate here automatically. DetailedLines.Clone() performs a deep copy via lo.Map. | Because DetailedLine is a type alias (not a named type), you cannot add methods to it directly — add helpers to DetailedLines (the slice type) instead. |
| `charge_test.go` | Unit tests for Intent.CalculateAmountAfterProration covering proration disabled, equal periods, half-period, zero-length periods, JPY rounding, and servicePeriod exceeding fullServicePeriod. Lives in package flatfee (not flatfee_test) to access unexported helpers. | Tests use datetime.MustParseTimeInLocation — always supply UTC to avoid timezone-dependent rounding surprises in assertions. |

## Anti-Patterns

- Adding business logic (state machine transitions, DB calls) directly in adapter.go, service.go, or handler.go — these files are interface and type definition files only.
- Using raw (unrounded) amounts in allocation or comparison logic — always call currencyCalculator.RoundToPrecision or Intent.CalculateAmountAfterProration before storing or comparing.
- Omitting Validate() on new Input types — every struct passed across the package boundary must be self-validating via models.NewNillableGenericValidationError.
- Referencing meta.ChargeStatusXxx constants directly in service or adapter code instead of the package-local Status constants (StatusCreated, StatusActive, etc.).
- Using NoProratingAdapterMode in service-layer logic — this enum value exists only for DB persistence mapping and has no productcatalog counterpart.

## Decisions

- **Adapter is a composite of six fine-grained sub-interfaces rather than a flat method list.** — Callers that only need ChargeAdapter or ChargePaymentAdapter can depend on the narrower interface, reducing coupling and making mocking easier in tests.
- **ProRatingModeAdapterEnum lives in this package (not productcatalog) and adds NoProratingAdapterMode.** — The DB enum must capture the 'no prorate' case that productcatalog omits; keeping it here prevents productcatalog from gaining a persistence-layer concern.
- **Handler interface is declared here (not in flatfee/service) to allow flatfee/service to implement it without import cycles.** — flatfee/service imports flatfee for domain types; placing Handler in flatfee/service would force flatfee to import flatfee/service, creating a cycle.

## Example: Implementing a new flat-fee adapter method that writes two rows atomically inside a ctx-propagated transaction

```
import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) CreatePaymentAndUsage(ctx context.Context, input flatfee.CreateInvoicedUsageInput) error {
	if err := input.Validate(); err != nil {
		return err
	}
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		if _, err := tx.CreatePayment(ctx, input.RunID, paymentData); err != nil {
			return fmt.Errorf("create payment: %w", err)
// ...
```

<!-- archie:ai-end -->
