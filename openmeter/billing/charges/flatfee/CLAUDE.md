# flatfee

<!-- archie:ai-start -->

> Public domain package for flat-fee charges: declares domain types (Charge, Intent, State, Status, Realizations, DetailedLines), the Adapter interface, the Service interface, the Handler interface, and proration math. It is the contract boundary between the flatfee/service and flatfee/adapter sub-packages and any caller in openmeter/billing/charges.

## Patterns

**Input.Validate() before any operation** — Every Input struct (CreateChargesInput, GetByIDsInput, IntentWithInitialStatus, AdvanceChargeInput) has a Validate() method that collects errors with errors.Join and wraps via models.NewNillableGenericValidationError. (`if err := input.Validate(); err != nil { return nil, err }`)
**Adapter composed of fine-grained sub-interfaces + entutils.TxCreator** — Adapter embeds ChargeAdapter, ChargeDetailedLineAdapter, ChargeCreditAllocationAdapter, ChargeInvoicedUsageAdapter, ChargePaymentAdapter, and entutils.TxCreator — no direct methods on the composite. (`type Adapter interface { ChargeAdapter; ChargeDetailedLineAdapter; ...; entutils.TxCreator }`)
**Expand dependency validation** — validateExpands enforces that ExpandDetailedLines requires ExpandRealizations; violations return a plain fmt.Errorf wrapped by the caller's Validate(). (`if expands.Has(meta.ExpandDetailedLines) && !expands.Has(meta.ExpandRealizations) { return fmt.Errorf(...) }`)
**Currency rounding in Intent.Normalized()** — Intent.Normalized() calls calc.RoundToPrecision on AmountBeforeProration before storing; CalculateAmountAfterProration also rounds the prorated result to currency precision. (`i.AmountBeforeProration = calc.RoundToPrecision(i.AmountBeforeProration)`)
**Status defined as typed string with Values() + Validate()** — Status is a typed string with a Values() []string method and a Validate() that checks slices.Contains; ToMetaChargeStatus converts to the canonical meta.ChargeStatus. (`func (s Status) Validate() error { if !slices.Contains(s.Values(), string(s)) { return models.NewGenericValidationError(...) } }`)
**Handler interface for external ledger/credit callbacks** — Handler declares OnAssignedToInvoice, OnInvoiceUsageAccrued, OnCreditsOnlyUsageAccrued, OnCreditsOnlyUsageAccruedCorrection, OnPaymentAuthorized, OnPaymentSettled, OnPaymentUncollectible — all returning typed result structs, never raw errors only. (`OnAssignedToInvoice(ctx, OnAssignedToInvoiceInput) (creditrealization.CreateAllocationInputs, error)`)
**ProRatingModeAdapterEnum bridges productcatalog and DB enum** — prorating.go defines ProRatingModeAdapterEnum with a NoProratingAdapterMode value that has no counterpart in productcatalog; adapter layer uses this enum for DB persistence. (`NoProratingAdapterMode ProRatingModeAdapterEnum = "no_prorate"`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Declares the Adapter interface and all adapter-facing input types (CreateChargesInput, GetByIDInput, GetByIDsInput, IntentWithInitialStatus). New persistence methods must be added to one of the sub-interfaces here. | Do not add conversion logic here; that belongs in flatfee/adapter/mapper.go. |
| `charge.go` | Core domain types: ChargeBase, Charge, Intent, State, Realizations. Intent.CalculateAmountAfterProration owns all proration math. | CalculateAmountAfterProration must never increase AmountBeforeProration — check servicePeriodDuration >= fullServicePeriodDuration guard. |
| `service.go` | Declares Service (= FlatFeeService + GetLineEngine + InvoiceLifecycleHooks), CreateInput, AdvanceChargeInput, GetByIDsInput for the service layer. | InvoiceLifecycleHooks methods are called by the billing engine; never add billing.Service dependencies here. |
| `handler.go` | Declares Handler interface for ledger/credit event callbacks. All OnXxx input structs define ValidateWith(currencyCalculator) when currency precision is required. | Handler implementations live outside this package; do not add business logic in the interface file. |
| `statemachine.go` | Defines Status type, Values(), Validate(), ToMetaChargeStatus(). Flat-fee statuses are simpler than usage-based (no sub-states). | New statuses must be added to Values() or Validate() will reject them. |
| `detailedline.go` | DetailedLine = stddetailedline.Base type alias; DetailedLines slice with Clone() and Validate(). | Type alias means any change to stddetailedline.Base propagates here automatically. |
| `prorating.go` | ProRatingModeAdapterEnum with NoProratingAdapterMode — used by the adapter layer for DB enum mapping. | NoProratingAdapterMode has no productcatalog counterpart; do not use it in service-layer logic. |
| `charge_test.go` | Unit tests for Intent.CalculateAmountAfterProration covering all edge cases (proration disabled, equal periods, JPY rounding, zero-length periods, exceeding full period). | Tests live in package flatfee (not flatfee_test); they can access unexported helpers. |

## Anti-Patterns

- Adding business logic (state machine transitions, DB calls) directly in adapter.go, service.go, or handler.go — these are interface/type definition files only.
- Using raw (unrounded) amounts in allocation or comparison logic; always call currencyCalculator.RoundToPrecision or Intent.CalculateAmountAfterProration.
- Omitting Validate() on new Input types — every struct passed across the package boundary must be self-validating.
- Referencing meta.ChargeStatusXxx directly in service or adapter code instead of the package-local Status constants.
- Adding fields to Intent or ChargeBase without updating Normalized() — un-normalized timestamps cause subtle time-zone comparison bugs.

## Decisions

- **Adapter is a composite of fine-grained sub-interfaces rather than a flat method list.** — Callers that only need ChargeAdapter can depend on the narrower interface, reducing coupling and making mocking easier in tests.
- **ProRatingModeAdapterEnum lives in this package (not productcatalog) with an extra NoProratingAdapterMode value.** — The DB enum must capture the 'no prorate' case that productcatalog omits; keeping it here prevents productcatalog from gaining a persistence-layer concern.
- **Handler interface is declared here (not in flatfee/service) to allow the service sub-package to depend on it without import cycles.** — flatfee/service imports flatfee for domain types and Handler; placing Handler in flatfee breaks the cycle that would occur if flatfee imported flatfee/service.

## Example: Implementing a new flat-fee adapter method that writes two rows atomically

```
import (
	"context"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) CreatePaymentAndUsage(ctx context.Context, chargeID flatfee.GetByIDInput, ...) error {
	if err := chargeID.Validate(); err != nil {
		return err
	}
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		if _, err := tx.CreatePayment(ctx, chargeID.ChargeID, ...); err != nil {
			return err
		}
		return tx.CreateInvoicedUsage(ctx, chargeID.ChargeID, ...)
// ...
```

<!-- archie:ai-end -->
