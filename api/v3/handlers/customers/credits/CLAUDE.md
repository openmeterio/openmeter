# credits

<!-- archie:ai-start -->

> v3 HTTP handlers for customer credit-grant lifecycle and balance queries: create/get/list grants, update external settlement status, get credit balance, and list ledger transactions. Primary constraint: this package is gated by credits.enabled; the handler is wired to noop ledger/balance services when credits are off.

## Patterns

**Local customerBalanceFacade interface** — handler.go defines a narrow customerBalanceFacade interface (GetBalance, GetBalances, ListCreditTransactions) rather than importing the full customerbalance service. This keeps the dependency surface explicit and testable. (`type customerBalanceFacade interface {
	GetBalance(ctx context.Context, input customerbalance.GetBalanceInput) (alpacadecimal.Decimal, error)
	GetBalances(ctx context.Context, input customerbalance.GetBalancesInput) ([]customerbalance.BalanceByCurrency, error)
	ListCreditTransactions(ctx context.Context, input customerbalance.ListCreditTransactionsInput) (customerbalance.ListCreditTransactionsResult, error)
}`)
**Settlement-type dispatch for purchase block** — toAPICreditGrantPurchase switches on settlement.Type() (invoice, external, promotional). Promotional grants return nil purchase. Unknown types return an error. The outer toAPIBillingCreditGrant handles the nil case gracefully. (`switch settlement.Type() {
case creditpurchase.SettlementTypePromotional: return nil, nil
case creditpurchase.SettlementTypeInvoice: ...
case creditpurchase.SettlementTypeExternal: ...
default: return nil, fmt.Errorf("invalid settlement type")
}`)
**Base64-JSON cursor for transaction pagination** — encodeBillingCreditTransactionCursor/decodeBillingCreditTransactionCursor marshal a {booked_at, created_at, id} struct to JSON then base64. Cursors are validated after decoding via ledgerCursor.Validate(). (`raw, _ := json.Marshal(payload)
return base64.StdEncoding.EncodeToString(raw), nil`)
**models.ValidationIssue for domain-specific errors without a custom errorEncoder** — errors.go uses models.NewValidationIssue with a typed ErrorCode constant, WithCriticalSeverity, commonhttp.WithHTTPStatusCodeAttribute(400), and WithFieldString. This lets apierrors.GenericErrorEncoder handle it without a package-local error encoder. (`return models.NewValidationIssue(errCodeCreditGrantExternalSettlementStatusInvalid, fmt.Sprintf("unsupported..."), models.WithCriticalSeverity(), commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest), models.WithFieldString("status"))`)
**Customer existence validation before balance query** — get_balance.go calls h.customerService.GetCustomer first to validate existence before querying balanceFacade.GetBalances. This ensures a 404 is returned for unknown customers before attempting a balance lookup. (`_, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{CustomerID: &request.CustomerID})
if err != nil { return GetCustomerCreditBalanceResponse{}, err }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines Handler interface (6 methods) and handler struct with ledger, accountResolver, balanceFacade, creditGrantService. New() wires all of them. | ledger and accountResolver are wired to noops by app/common when credits.enabled=false; handlers must not assume they are real implementations. |
| `convert.go` | All domain-to-API and API-to-domain mapping. toAPIBillingCreditGrant is the primary domain-to-API converter; fromAPICreateCreditGrantRequest is the primary API-to-domain converter. | fromAPICreateCreditGrantRequest returns an error for feature filters (not yet supported); keep the TODO and do not silently ignore the filter. |
| `errors.go` | Defines newCreditGrantExternalSettlementStatusInvalid using models.NewValidationIssue with HTTP status attribute. This package intentionally has no custom errorEncoder() — unlike sibling packages. | Do not add an errorEncoder() function here; the ValidationIssue+WithHTTPStatusCodeAttribute pattern is the intended mechanism. |
| `externalsettlement.go` | Handler for PATCH external settlement status transitions. Only authorized and settled are valid target statuses; pending is explicitly rejected in fromAPIBillingCreditPurchasePaymentSettlementStatus. | The 'pending' status must remain rejected; a grant cannot be downgraded to pending once advanced. |

## Anti-Patterns

- Assuming ledger or accountResolver are real implementations — they may be noops when credits.enabled=false
- Adding a package-local errorEncoder() — this package relies on models.ValidationIssue + WithHTTPStatusCodeAttribute for error encoding
- Allowing the 'pending' settlement status to succeed in fromAPIBillingCreditPurchasePaymentSettlementStatus
- Returning a non-nil purchase block for SettlementTypePromotional grants
- Silently ignoring the feature filter in fromAPICreateCreditGrantRequest instead of returning an error

## Decisions

- **customerBalanceFacade is a local interface rather than importing the full customerbalance.Service** — Narrows the dependency surface to only the three methods this handler needs, making tests easier to mock and preventing accidental use of write operations.
- **ValidationIssue with WithHTTPStatusCodeAttribute replaces a custom errorEncoder for domain validation errors** — apierrors.GenericErrorEncoder already inspects the HTTP status attribute on ValidationIssues; adding a second encoder would be redundant and could cause double-handling.

## Example: Create a domain-specific validation error usable by GenericErrorEncoder without a custom error encoder

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
