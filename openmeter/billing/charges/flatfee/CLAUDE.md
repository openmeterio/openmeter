# flatfee

<!-- archie:ai-start -->

> Public domain contract package for flat-fee charges: declares all domain types (ChargeBase, Charge, Intent, State, Status, Realizations, RealizationRun, DetailedLines), the composite Adapter interface, the Service interface, and the Handler interface for ledger/credit callbacks. It is the contract boundary between flatfee/service and flatfee/adapter — no business logic or DB access lives here.

## Patterns

**Input.Validate() before any operation** — Every Input struct (CreateChargesInput, GetByIDsInput, IntentWithInitialStatus, CreateCurrentRunInput, CreateInvoicedUsageInput, etc.) has Validate() collecting errors with errors.Join and wrapping via models.NewNillableGenericValidationError. Callers invoke it as the first statement. (`if err := input.Validate(); err != nil { return nil, err }`)
**Adapter is a composite of fine-grained sub-interfaces + entutils.TxCreator** — Adapter embeds ChargeAdapter, ChargeDetailedLineAdapter, ChargeCreditAllocationAdapter, ChargeRunAdapter, ChargeInvoicedUsageAdapter, ChargePaymentAdapter, and entutils.TxCreator. No direct methods live on the composite; new persistence methods go onto a sub-interface. (`type Adapter interface { ChargeAdapter; ChargeDetailedLineAdapter; ...; entutils.TxCreator }`)
**Currency rounding in Intent.Normalized() and CalculateAmountAfterProration** — Intent.Normalized() calls calc.RoundToPrecision on AmountBeforeProration; CalculateAmountAfterProration rounds the prorated result to currency precision. Never compare or store amounts without rounding first. (`i.AmountBeforeProration = calc.RoundToPrecision(i.AmountBeforeProration)`)
**Proration never increases the amount** — CalculateAmountAfterProration returns AmountBeforeProration when proration is disabled, mode != ProRatingModeProratePrices, a period is zero-length, or servicePeriodDuration >= fullServicePeriodDuration. (`if servicePeriodDuration == 0 || fullServicePeriodDuration == 0 || servicePeriodDuration >= fullServicePeriodDuration { return i.AmountBeforeProration, nil }`)
**Status as typed string with hierarchical dot notation** — Status has exhaustive Values(), Validate() (slices.Contains), and ToMetaChargeStatus() that splits on '.' to extract the top-level meta.ChargeStatus. New statuses must be added to Values() or validation rejects them. (`StatusActiveRealizationStarted Status = "active.realization.started"`)
**Charge value-semantics for the generic state machine** — Charge.WithStatus/WithBase return value copies, never pointer mutations — required by the generic Machine[CHARGE,BASE,STATUS] which updates its field by assignment. (`func (c Charge) WithStatus(status Status) Charge { c.Status = status; return c }`)
**Handler interface for external ledger/credit callbacks declared here** — Handler (OnAllocateCredits, OnInvoiceUsageAccrued, OnCorrectCreditAllocations, OnPaymentAuthorized/Settled/Uncollectible) lives here, not in flatfee/service, to break the import cycle: flatfee/service imports flatfee for types. (`OnAllocateCredits(ctx, OnAllocateCreditsInput) (creditrealization.CreateAllocationInputs, error)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Declares the Adapter composite + the six sub-interfaces and adapter-facing Input types (CreateChargesInput, GetByIDsInput, IntentWithInitialStatus, CreateCurrentRunInput, CreateInvoicedUsageInput); validateExpands enforces expand dependencies. | No conversion logic here — that belongs in flatfee/adapter/mapper.go. Every Input type must implement Validate(). |
| `charge.go` | Core domain types ChargeBase, Charge, Intent, State, Realizations. Intent.CalculateAmountAfterProration owns all proration math and rounds to currency precision. | WithStatus/WithBase must return value copies; proration must never increase beyond AmountBeforeProration. |
| `service.go` | Declares Service, CreateInput, AdvanceChargeInput, GetByIDsInput for the service layer plus InvoiceLifecycleHooks. | Never add billing.Service dependencies to this interface file. |
| `handler.go` | Handler interface for ledger/credit callbacks; each OnXxx input implements Validate() and some ValidateWith(currencyCalculator) when precision is required. | Handler implementations live outside this package (flatfee/service or ledger/chargeadapter); no business logic in the interface file. |
| `statemachine.go` | Status type, Values(), Validate(), ToMetaChargeStatus(). Statuses are hierarchical 'toplevel.substate' with dot notation. | New statuses must be added to Values(); ToMetaChargeStatus splits on '.' — follow the toplevel.substate convention. |
| `realizationrun.go` | RealizationRunBase/RealizationRun/RealizationRuns, RealizationRunType, UpdateRealizationRunInput (mo.Option fields), IsVoidedBillingHistory. | UpdateRealizationRunInput fields absent from a partial update must stay mo.None, not zero value; RealizationRunTypeInvalidDueToUnsupportedCreditNote must never be an InitialType. |
| `detailedline.go` | DetailedLine = type alias for stddetailedline.Base; DetailedLines slice with Clone() (deep copy via lo.Map) and Validate(). | DetailedLine is a type alias, not a named type — add helper methods to DetailedLines, not DetailedLine. |
| `prorating.go` | ProRatingModeAdapterEnum extends productcatalog.ProRatingMode with NoProratingAdapterMode ('no_prorate') for DB column mapping only. | NoProratingAdapterMode is adapter/DB-only; service-layer logic must use productcatalog.ProRatingConfig. |

## Anti-Patterns

- Adding business logic (state-machine transitions, DB calls) in adapter.go, service.go, or handler.go — these are interface/type definition files only
- Using raw (unrounded) amounts in allocation or comparison — always call calc.RoundToPrecision or Intent.CalculateAmountAfterProration first
- Omitting Validate() on a new Input type, or returning a raw error instead of models.NewNillableGenericValidationError
- Referencing meta.ChargeStatusXxx constants directly instead of the package-local Status constants (StatusCreated, StatusActive...)
- Using NoProratingAdapterMode in service-layer logic — it exists only for DB persistence mapping

## Decisions

- **Adapter is a composite of six fine-grained sub-interfaces rather than a flat method list** — Callers needing only ChargeAdapter or ChargePaymentAdapter depend on the narrower interface, reducing coupling and easing mocking.
- **ProRatingModeAdapterEnum lives here (not productcatalog) and adds NoProratingAdapterMode** — The DB enum must capture the 'no prorate' case productcatalog omits; keeping it here prevents productcatalog from gaining a persistence concern.
- **Handler interface declared here, not in flatfee/service** — flatfee/service imports flatfee for domain types; placing Handler in flatfee/service would force flatfee to import flatfee/service, creating a cycle.

## Example: Compute the prorated amount for a flat-fee intent before persisting a run

```
import "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"

intent = intent.Normalized()
if err := intent.Validate(); err != nil { return err }
amount, err := intent.CalculateAmountAfterProration()
if err != nil { return err }
runInput := flatfee.CreateCurrentRunInput{Charge: base, ServicePeriod: period, AmountAfterProration: amount}
if err := runInput.Validate(); err != nil { return err }
```

<!-- archie:ai-end -->
