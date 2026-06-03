# credits

<!-- archie:ai-start -->

> v3 HTTP handlers for the customer credit-grant lifecycle and balance queries: create/get/list grants, update external settlement status, get credit balance, and list ledger transactions. Primary constraint: gated by credits.enabled — wired to noop ledger/balance services when credits are off.

## Patterns

**Local customerBalanceFacade interface** — handler.go defines a narrow customerBalanceFacade (GetBalance, GetBalances, ListCreditTransactions) instead of importing the full customerbalance service, keeping the dependency surface explicit and mockable. (`type customerBalanceFacade interface { GetBalance(ctx, customerbalance.GetBalanceInput) (alpacadecimal.Decimal, error); ... }`)
**Settlement-type dispatch for purchase block** — toAPICreditGrantPurchase switches on settlement.Type() (invoice, external, promotional); promotional returns nil purchase, unknown types error, and the caller handles the nil case. (`switch settlement.Type() { case creditpurchase.SettlementTypePromotional: return nil, nil; ... default: return nil, fmt.Errorf("invalid settlement type") }`)
**Base64-JSON cursor for transaction pagination** — encode/decodeBillingCreditTransactionCursor marshal a {booked_at, created_at, id} struct to JSON then base64; decoded cursors are validated via ledgerCursor.Validate(). (`raw, _ := json.Marshal(payload); return base64.StdEncoding.EncodeToString(raw), nil`)
**ValidationIssue for domain errors without a local errorEncoder** — errors.go uses models.NewValidationIssue with a typed ErrorCode, WithCriticalSeverity, commonhttp.WithHTTPStatusCodeAttribute(400), WithFieldString so apierrors.GenericErrorEncoder maps it — no package-local encoder. (`models.NewValidationIssue(errCode, msg, models.WithCriticalSeverity(), commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest), models.WithFieldString("status"))`)
**Customer existence validation before balance query** — get_balance.go calls customerService.GetCustomer first so unknown customers return 404 before any balance lookup. (`_, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{CustomerID: &request.CustomerID}); if err != nil { return ..., err }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface (6 methods) and struct holding ledger, accountResolver, balanceFacade, creditGrantService; New() wires them. | ledger and accountResolver are noops when credits.enabled=false — handlers must not assume real implementations. |
| `convert.go` | Domain<->API mapping; toAPIBillingCreditGrant is the primary domain-to-API, fromAPICreateCreditGrantRequest the primary API-to-domain. | fromAPICreateCreditGrantRequest errors on feature filters (not yet supported) — keep the TODO, do not silently ignore. |
| `errors.go` | newCreditGrantExternalSettlementStatusInvalid via models.NewValidationIssue + HTTP status attribute; intentionally no local errorEncoder(). | Do not add an errorEncoder() — the ValidationIssue + WithHTTPStatusCodeAttribute pattern is the intended mechanism. |
| `externalsettlement.go` | PATCH external settlement status transitions; only authorized and settled are valid targets, pending is rejected in fromAPIBillingCreditPurchasePaymentSettlementStatus. | 'pending' must stay rejected — a grant cannot be downgraded to pending once advanced. |

## Anti-Patterns

- Assuming ledger or accountResolver are real implementations — they may be noops when credits.enabled=false
- Adding a package-local errorEncoder() instead of relying on ValidationIssue + WithHTTPStatusCodeAttribute
- Allowing the 'pending' settlement status to succeed in fromAPIBillingCreditPurchasePaymentSettlementStatus
- Returning a non-nil purchase block for SettlementTypePromotional grants
- Silently ignoring the feature filter in fromAPICreateCreditGrantRequest instead of returning an error

## Decisions

- **customerBalanceFacade is a local interface, not the full customerbalance.Service** — Narrows the dependency surface to the three needed methods, eases mocking, and prevents accidental use of write operations.
- **ValidationIssue + WithHTTPStatusCodeAttribute replaces a custom errorEncoder** — apierrors.GenericErrorEncoder already inspects the HTTP status attribute on ValidationIssues; a second encoder would be redundant and risk double-handling.

## Example: A domain validation error usable by GenericErrorEncoder without a custom encoder

```
import (
    "net/http"
    "github.com/openmeterio/openmeter/pkg/framework/commonhttp"
    "github.com/openmeterio/openmeter/pkg/models"
)

const errCodeExample models.ErrorCode = "example_error_code"

func newExampleError(value string) error {
    return models.NewValidationIssue(
        errCodeExample,
        fmt.Sprintf("unsupported value: %s", value),
        models.WithCriticalSeverity(),
        commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
        models.WithFieldString("field_name"),
// ...
```

<!-- archie:ai-end -->
