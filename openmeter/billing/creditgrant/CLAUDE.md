# creditgrant

<!-- archie:ai-start -->

> Credit-grant-centric facade over the charges layer exposing Create, Get, List, and UpdateExternalSettlement for creditpurchase.Charge values. Holds no Ent client — all persistence flows through the injected charges.Service; owns input validation and customer-ownership checks only.

## Patterns

**Config+Validate/New constructor** — New(c Config) calls c.Validate() before returning the service; dependencies enter only through Config. (`func New(c Config) (Service, error) { if err := c.Validate(); err != nil { return nil, err }; return &service{chargesService: c.ChargesService}, nil }`)
**input.Validate() at top of every method** — Each public method calls input.Validate() before any business logic; validation wraps errors in models.NewNillableGenericValidationError(errors.Join(errs...)). (`if err := input.Validate(); err != nil { return creditpurchase.Charge{}, err }`)
**Delegate all persistence to charges.Service** — No openmeter/ent/db imports; all reads/writes go through chargesService / creditpurchase pipeline. (`charge, err := s.chargesService.GetByID(ctx, charges.GetByIDInput{ID: input.ChargeID})`)
**Customer ownership check in service layer** — Get-style methods assert charge.Intent.CustomerID == input.CustomerID after fetching; not enforced downstream. (`if charge.Intent.CustomerID != input.CustomerID { return creditpurchase.Charge{}, models.NewGenericForbiddenError(...) }`)
**FundingMethod enum with Validate()** — FundingMethod.Validate() is invoked inside CreateInput.Validate(); funded grants (non-none) require non-nil Purchase terms. (`if i.FundingMethod != FundingMethodNone && i.Purchase == nil { errs = append(errs, errors.New("purchase terms are required")) }`)
**NoopService compile-time guard** — noop.go provides NoopService{} with var _ Service = NoopService{} for credits-disabled wiring. (`var _ Service = NoopService{}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/billing/creditgrant/service.go` | Service interface, all Input types with Validate(), FundingMethod enum, PurchaseTerms, GrantFilters, and the concrete service struct + New constructor. | Validate() must return models.NewNillableGenericValidationError(errors.Join(errs...)); FundingMethod.Validate() must run inside CreateInput.Validate(). |
| `openmeter/billing/creditgrant/noop.go` | No-op implementation returning zero-values for all methods when credits are disabled. | Must stay in sync with the Service interface — add a method here whenever Service grows. |

## Anti-Patterns

- Importing openmeter/ent/db or calling Ent builders directly — all DB access must flow through charges.Service or creditpurchase.Service
- Skipping input.Validate() at the top of a new method
- Returning a charge without verifying charge.Intent.CustomerID == input.CustomerID in Get-style methods
- Using context.Background() instead of the caller-supplied ctx
- Adding business logic inside toIntent()/toSettlement() converters beyond pure field mapping

## Decisions

- **Service delegates entirely to charges.Service rather than holding its own adapter** — Keeps a single authoritative write path for credit-purchase charges; avoids parallel DB access that could bypass transaction discipline.
- **Ownership check done in the service layer, not in charges.Service** — charges.Service is a generic multi-customer service; customer scoping is a caller responsibility enforced here.

## Example: Adding a service method with input validation, ownership check, and delegation

```
func (s *service) Cancel(ctx context.Context, input CancelInput) (creditpurchase.Charge, error) {
	if err := input.Validate(); err != nil {
		return creditpurchase.Charge{}, err
	}
	charge, err := s.chargesService.GetByID(ctx, charges.GetByIDInput{ID: input.ChargeID})
	if err != nil {
		return creditpurchase.Charge{}, err
	}
	if charge.Intent.CustomerID != input.CustomerID {
		return creditpurchase.Charge{}, models.NewGenericForbiddenError(fmt.Errorf("charge does not belong to customer"))
	}
	return s.chargesService.ApplyPatches(ctx, ...)
}
```

<!-- archie:ai-end -->
