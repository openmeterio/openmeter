# creditgrant

<!-- archie:ai-start -->

> Credit-grant-centric facade over the charges layer exposing Create, Get, List, and UpdateExternalSettlement for creditpurchase.Charge values. Owns input validation and customer-ownership checks but holds no Ent client — all persistence flows through the injected charges.Service.

## Patterns

**Config+Validate/New constructor** — Every concrete implementation uses a Config struct with a Validate() method; New() calls Validate() before returning the service. (`func New(c Config) (Service, error) { if err := c.Validate(); err != nil { return nil, err }; return &service{chargesService: c.ChargesService}, nil }`)
**input.Validate() called at top of every method** — Each public service method calls input.Validate() before any business logic and wraps errors in models.NewNillableGenericValidationError. (`func (s *service) Get(ctx context.Context, input GetInput) (creditpurchase.Charge, error) { if err := input.Validate(); err != nil { return creditpurchase.Charge{}, err }; ... }`)
**Delegate all persistence to charges.Service** — No Ent imports appear here; all DB access flows through charges.Service or creditpurchase.Service injected via Config. (`charge, err := s.chargesService.GetByID(ctx, charges.GetByIDInput{ID: input.ChargeID})`)
**Customer ownership check in service layer** — Get-style methods assert cpCharge.Intent.CustomerID == input.CustomerID after fetching; not enforced by charges.Service. (`if charge.Intent.CustomerID != input.CustomerID { return creditpurchase.Charge{}, models.NewGenericForbiddenError(fmt.Errorf("charge does not belong to customer")) }`)
**NoopService compile-time guard** — noop.go provides NoopService{} with var _ Service = NoopService{} ensuring interface compliance without business logic. (`var _ Service = NoopService{}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/billing/creditgrant/service.go` | Defines the Service interface, all Input types with Validate(), FundingMethod enum, PurchaseTerms, GrantFilters, and the concrete service struct with New constructor. | Input.Validate() must call models.NewNillableGenericValidationError(errors.Join(errs...)) — never return raw errors from validation. FundingMethod.Validate() must be called inside CreateInput.Validate(). |
| `openmeter/billing/creditgrant/noop.go` | No-op implementation used when credit grants are disabled; returns zero-values with nil errors for all methods. | Must stay in sync with the Service interface; update when new methods are added. |

## Anti-Patterns

- Importing openmeter/ent/db or calling Ent builders directly — all DB access must flow through charges.Service
- Skipping input.Validate() at the top of a new method
- Returning a charge without verifying CustomerID ownership in Get-style methods
- Using context.Background() instead of the caller-supplied ctx
- Adding business logic inside toIntent()/toSettlement() conversion helpers beyond pure field mapping

## Decisions

- **Service delegates entirely to charges.Service rather than holding its own adapter** — Keeps a single authoritative write path for credit-purchase charges; avoids parallel DB access patterns that could bypass transaction discipline.
- **Ownership check done in the service layer, not in charges.Service** — charges.Service is a generic multi-customer service; customer scoping is a caller responsibility enforced here.

## Example: Adding a new service method with input validation, ownership check, and delegation

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
