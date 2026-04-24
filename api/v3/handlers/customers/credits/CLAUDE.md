# credits

<!-- archie:ai-start -->

> HTTP handlers for customer credit-grant lifecycle and balance queries in the v3 API: create/get/list grants, update external settlement status, get credit balance, and list ledger transactions. Primary constraint: this package is gated by credits.enabled; the handler is wired to noop ledger/balance services when credits are off.

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
**models.ValidationIssue for domain-specific errors** — errors.go uses models.NewValidationIssue with a typed ErrorCode constant, WithCriticalSeverity, WithHTTPStatusCodeAttribute(400), and WithFieldString to produce structured errors that apierrors.GenericErrorEncoder handles without a custom error encoder in this package. (`return models.NewValidationIssue(errCodeCreditGrantExternalSettlementStatusInvalid, fmt.Sprintf("unsupported..."), models.WithCriticalSeverity(), commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest), models.WithFieldString("status"))`)
**Status-to-domain conversion with explicit unknown handling** — fromAPIBillingCreditGrantStatus maps API status strings to meta.ChargeStatus. 'expired' maps to ChargeStatusFinal (terminal). Unknown values return a plain fmt.Errorf (not a ValidationIssue) — the caller wraps it in apierrors.NewBadRequestError. (`case api.BillingCreditGrantStatusExpired:
	return meta.ChargeStatusFinal, nil`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines Handler interface (6 methods) and handler struct with ledger + accountResolver in addition to the balance facade and creditGrantService. New() wires all of them. | ledger and accountResolver are wired to noops by app/common when credits.enabled=false; handlers must not assume they are real implementations. |
| `convert.go` | All domain↔API mapping. toAPIBillingCreditGrant is the primary domain→API converter; fromAPICreateCreditGrantRequest is the primary API→domain converter. | fromAPICreateCreditGrantRequest returns an error for feature filters (not yet supported); add a TODO if that changes. |
| `errors.go` | Defines newCreditGrantExternalSettlementStatusInvalid using models.NewValidationIssue with HTTP status attribute, allowing GenericErrorEncoder to produce a 400 without a package-local error encoder. | Other packages in customers/* use a local errorEncoder(); this package intentionally does not. Keep it consistent. |
| `externalsettlement.go` | Handler for PATCH /customers/{id}/credits/{grant}/external-settlement. Calls creditGrantService.UpdateExternalSettlement. Only authorized→settled transitions are valid; pending is rejected in fromAPIBillingCreditPurchasePaymentSettlementStatus. | The 'pending' status must remain rejected; a grant cannot be downgraded to pending. |
| `get_balance.go` | Returns per-currency credit balances. Rejects feature filter queries (unsupported). Calls customerService.GetCustomer first to validate existence before querying balanceFacade. | The feature filter rejection is a hard constraint; do not silently ignore the filter. |

## Anti-Patterns

- Assuming ledger or accountResolver are real implementations — they may be noops when credits.enabled=false
- Adding a package-local errorEncoder() — this package relies on models.ValidationIssue + WithHTTPStatusCodeAttribute for error encoding
- Allowing the 'pending' settlement status to succeed in fromAPIBillingCreditPurchasePaymentSettlementStatus
- Returning a non-nil purchase block for SettlementTypePromotional grants
- Using context.Background() instead of the ctx passed to the handler operation

## Decisions

- **customerBalanceFacade is a local interface rather than importing the full customerbalance.Service** — Narrows the dependency surface to only the three methods this handler needs, making tests easier to mock and preventing accidental use of write operations.
- **ValidationIssue with WithHTTPStatusCodeAttribute replaces a custom errorEncoder for domain validation errors** — apierrors.GenericErrorEncoder already inspects the HTTP status attribute on ValidationIssues; adding a second encoder would be redundant and could cause double-handling.

<!-- archie:ai-end -->
