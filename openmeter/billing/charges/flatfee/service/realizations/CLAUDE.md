# realizations

<!-- archie:ai-start -->

> Owns flat-fee credit allocation and correction mechanics: persisting credit realization records and their lineage segments. It must not make state-machine decisions — all charge lifecycle transitions happen in the parent charges.Service layer.

## Patterns

**Config struct with Validate() before construction** — All dependencies are declared in a Config struct; Config.Validate() collects all nil-check errors via errors.Join before New() returns a *Service. Never construct Service directly. (`func New(config Config) (*Service, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Input structs carry a Validate() method** — Each method input (AllocateCreditsOnlyInput, CorrectAllCreditRealizationsInput) has a Validate() that checks zero-values and negative amounts before any adapter call. Call Validate() at the top of every service method. (`if err := in.Validate(); err != nil { return AllocateCreditsOnlyResult{}, err }`)
**Currency rounding before validation** — RoundToPrecision is called on monetary amounts before Validate() to avoid false negatives from floating-point comparisons. Pattern: round first, then validate, then assert sum equality. (`in.Amount = in.CurrencyCalculator.RoundToPrecision(in.Amount); if err := in.Validate(); err != nil { ... }`)
**Lineage persistence is always paired with adapter writes** — CreateCreditAllocations calls adapter.CreateCreditAllocations, then lineage.CreateInitialLineages, then lineage.PersistCorrectionLineageSegments — all three must succeed or the caller's transaction rolls back. Never skip lineage steps. (`realizations, err := s.adapter.CreateCreditAllocations(...); ... s.lineage.CreateInitialLineages(...); s.lineage.PersistCorrectionLineageSegments(...)`)
**Handler delegates external credit allocation decisions** — The Service does not compute how credits are split — it delegates to flatfee.Handler.OnCreditsOnlyUsageAccrued / OnCreditsOnlyUsageAccruedCorrection. Service only validates sum equality after the fact. (`creditAllocations, err := s.handler.OnCreditsOnlyUsageAccrued(ctx, input)`)
**Sum equality assertion after allocation** — After handler returns allocations, the sum is re-rounded and checked against the original amount. Mismatch returns models.NewGenericValidationError, not an internal error. (`if !allocated.Equal(in.Amount) { return ..., models.NewGenericValidationError(fmt.Errorf("credit allocations do not match total ...")) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | The entire package is this single file. Defines Service struct, Config, all input/result types, and three exported methods: CreateCreditAllocations, AllocateCreditsOnly, CorrectAllCredits. | Adding state-machine logic here violates the layer contract — state transitions belong in the parent charges service. Also watch for missing lineage.PersistCorrectionLineageSegments calls after adapter writes. |

## Anti-Patterns

- Making state-machine decisions (charge status transitions) inside this package — it must remain a pure realization mechanics layer
- Calling adapter.CreateCreditAllocations without following up with both lineage.CreateInitialLineages and lineage.PersistCorrectionLineageSegments
- Skipping Validate() on input structs before adapter calls
- Performing currency arithmetic without rounding via CurrencyCalculator.RoundToPrecision first
- Returning a raw internal error instead of models.NewGenericValidationError when allocation sums don't match

## Decisions

- **Service delegates allocation-splitting to flatfee.Handler rather than computing it internally** — Different flat-fee billing strategies (immediate full allocation vs. pro-rated) need to vary the split logic without touching realization persistence code.
- **Lineage persistence is mandatory and co-located in CreateCreditAllocations** — Realization records without lineage segments break correction lookups in CorrectAllCredits; coupling the two writes in one method makes it impossible to forget lineage.
- **Config+Validate pattern instead of variadic options** — Consistent with the broader billing/charges constructor pattern; makes missing-dependency errors surface at startup rather than at first call.

## Example: Allocate credits for a flat-fee charge amount, persist realizations and lineage

```
import (
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service/realizations"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

svc, _ := realizations.New(realizations.Config{
	Adapter: adapter,
	Handler: handler,
	Lineage: lineageSvc,
})

result, err := svc.AllocateCreditsOnly(ctx, realizations.AllocateCreditsOnlyInput{
	Charge:             charge,
	Amount:             amount,
// ...
```

<!-- archie:ai-end -->
