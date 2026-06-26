# service

<!-- archie:ai-start -->

> Implements the creditgrant.Service interface as a thin orchestration facade that maps credit-grant operations onto the lower-level charges/creditpurchase layer. Its primary constraint: it owns no persistence — every read/write delegates to chargesService or creditPurchaseService, and it only adds customer-existence checks, intent construction, and credit-purchase-charge typing.

## Patterns

**Config-validated constructor returning interface** — New(config Config) calls config.Validate() (which requires CreditPurchaseService, ChargesService, CustomerService non-nil) before constructing the unexported *service, and returns the creditgrant.Service interface, not the concrete struct. (`func New(config Config) (creditgrant.Service, error) { if err := config.Validate(); err != nil { return nil, fmt.Errorf("invalid config: %w", err) }; return &service{...}, nil }`)
**Validate-then-delegate per method** — Every Service method begins with input.Validate() and wraps the error as 'invalid input: %w', then delegates to a charges/creditpurchase call. No method touches Ent or a transaction directly. (`if err := input.Validate(); err != nil { return creditpurchase.Charge{}, fmt.Errorf("invalid input: %w", err) }`)
**Charge-to-credit-purchase typing via AsCreditPurchaseCharge** — After fetching a generic charge with chargesService.GetByID, the result is narrowed with charge.AsCreditPurchaseCharge(); a non-credit-purchase charge is surfaced as an error rather than returned. (`cpCharge, err := charge.AsCreditPurchaseCharge(); if err != nil { return creditpurchase.Charge{}, fmt.Errorf("charge is not a credit purchase: %w", err) }`)
**Expand realizations on every read** — Get/Create reads pass Expands: meta.Expands{meta.ExpandRealizations} to GetByID, and List sets the same Expands on ListChargesInput, so returned credit-purchase charges always carry realization data. (`Expands: meta.Expands{meta.ExpandRealizations}`)
**Input mapping via package-local to* helpers** — CreateInput is translated into a creditpurchase.Intent by toIntent, which itself calls toSettlement and calculateExpiresAt. FundingMethod selects the settlement variant (Invoice/External/Promotional) in a switch. (`intent := toIntent(input); result, err := s.chargesService.Create(ctx, charges.CreateInput{ Namespace: input.Namespace, Intents: charges.ChargeIntents{charges.NewChargeIntent(intent)} })`)
**Customer-scoped not-found and ownership enforcement** — Create verifies the customer via customerService.GetCustomer; Get cross-checks cpCharge.Intent.CustomerID against input.CustomerID and returns models.NewGenericNotFoundError when mismatched, so charges are namespaced+customer-scoped. (`if cpCharge.Intent.CustomerID != input.CustomerID { return creditpurchase.Charge{}, fmt.Errorf("get charge: %w", models.NewGenericNotFoundError(...)) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Whole service implementation: Config/Validate, New, the *service struct, the four Service methods (Create, Get, List, UpdateExternalSettlement), and the toIntent/toSettlement/calculateExpiresAt mapping helpers. | Create expects chargesService.Create to return exactly one charge (errors on len != 1). UpdateExternalSettlement rejects non-external settlements with a models.NewValidationIssue (code credit_grant_external_settlement_not_supported, 400) — only external-funded grants may transition payment state. toIntent currently sets ServicePeriod/BillingPeriod/FullServicePeriod to a zero-width [effectiveAt, effectiveAt] period (TODO marker). |

## Anti-Patterns

- Adding Ent queries, transactions, or a billing.Adapter dependency here — this layer must stay a pure orchestration facade over charges/creditpurchase.
- Returning the concrete *service from New instead of the creditgrant.Service interface.
- Skipping input.Validate() or the AsCreditPurchaseCharge() type narrowing and returning a raw generic charge.
- Constructing creditpurchase.Settlement inline in a method instead of routing FundingMethod through toSettlement.
- Bypassing the customer-existence / Intent.CustomerID ownership checks, which would leak charges across customers.

## Decisions

- **creditgrant.Service is a separate facade over the charges layer rather than methods on charges.Service.** — It exposes a credit-grant-centric vocabulary (FundingMethod, Amount, ExpiresAfter, customer scoping) and hides the generic charge/intent machinery from API handlers and app/common.
- **List delegates to creditPurchaseService.List while Get/Create use the generic chargesService.GetByID/Create.** — creditPurchaseService already provides a credit-purchase-typed list with status/currency/customer filters, avoiding manual narrowing of a heterogeneous charge list.

## Example: Create a credit grant by building an intent and delegating to the charges layer, then narrowing the result.

```
intent := toIntent(input)
result, err := s.chargesService.Create(ctx, charges.CreateInput{
	Namespace: input.Namespace,
	Intents:   charges.ChargeIntents{charges.NewChargeIntent(intent)},
})
if err != nil { return creditpurchase.Charge{}, fmt.Errorf("create credit grant charge: %w", err) }
if len(result) != 1 { return creditpurchase.Charge{}, fmt.Errorf("expected 1 created charge, got %d", len(result)) }
createdChargeID, err := result[0].GetChargeID()
charge, err := s.chargesService.GetByID(ctx, charges.GetByIDInput{ ChargeID: createdChargeID, Expands: meta.Expands{meta.ExpandRealizations} })
return charge.AsCreditPurchaseCharge()
```

<!-- archie:ai-end -->
