# routingrules

<!-- archie:ai-start -->

> Pure validation layer for double-entry ledger transactions: defines the RoutingRule interface, a composable Validator, and DefaultValidator encoding all permitted account-type pairings and route-field constraints. No DB access — validates []ledger.EntryInput before any persistence call.

## Patterns

**RoutingRule interface + Validator composition** — Each constraint is a named struct implementing RoutingRule{Validate(TxView) error}. Validator.ValidateEntries runs them sequentially, fail-fast on first error. New rules are appended to DefaultValidator.Rules in defaults.go — never create ad-hoc Validator instances in app code. (`type AllowedAccountSetsRule struct { Sets [][]ledger.AccountType }; func (r AllowedAccountSetsRule) Validate(tx TxView) error { ... }`)
**TxView as the rule's only input** — NewTxView converts []ledger.EntryInput into []EntryView (with pre-decoded Route), exposing AccountTypes(), EntriesOf(accountType), HasAccountTypes(...). Rules must use TxView — never the raw EntryInput slice. (`view, _ := NewTxView(entries); for _, rule := range v.Rules { rule.Validate(view) }`)
**HasAccountTypes guard before rule logic** — Every rule targeting specific account types checks tx.HasAccountTypes(r.From, r.To) first and returns nil when those types are absent. Omitting the guard causes false positives on unrelated transactions. (`func (r RequireFlowDirectionRule) Validate(tx TxView) error { if !tx.HasAccountTypes(r.From, r.To) { return nil }; ... }`)
**RequireSameRouteRule for cross-account field consistency** — Ensures entries on two account types share the same RouteField values (currency, tax_code, features, cost_basis). Declared as Left/Right/Fields triplets in DefaultValidator; uses requireMatchingRouteFields helper. (`RequireSameRouteRule{Left: ledger.AccountTypeCustomerFBO, Right: ledger.AccountTypeCustomerReceivable, Fields: []RouteField{RouteFieldCurrency, RouteFieldTaxCode}}`)
**FuncRule for test-only rules** — FuncRule wraps a plain function as a RoutingRule for test scenarios without a named struct. Never use in production code. (`FuncRule(func(tx TxView) error { return nil })`)
**ErrRoutingRuleViolated with WithAttrs for all errors** — All rule violations return ledger.ErrRoutingRuleViolated.WithAttrs(models.Attributes{...}) with a 'reason' key plus context. Never return plain fmt.Errorf from a RoutingRule. (`return ledger.ErrRoutingRuleViolated.WithAttrs(models.Attributes{"reason": "account_type_combination_not_allowed", "account_types": present})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `routingrules.go` | Core types: RoutingRule interface, Validator (implements ledger.RoutingValidator), FuncRule, and all concrete rule structs (AllowedAccountSetsRule, RequireFlowDirectionRule, RequireSameRouteRule, RequireAccountAuthorizationStatusRule, RequireReceivableAuthorizationStageRule, RequireAccruedCostBasisTranslationRule, RequireFBOCostBasisTranslationRule, RequireUniqueSubAccountsRule). | All rules skip gracefully when relevant account types are absent (HasAccountTypes first) — intentional permissive-when-unrelated behaviour; do not remove these guards. |
| `view.go` | EntryView (wraps EntryInput + pre-decoded Route), TxView (slice of EntryView with AccountTypes/EntriesOf/HasAccountTypes/HasAccountType projections), and optional field comparators (optionalStringEqual, optionalDecimalEqual). | EntryView.Route() returns the decoded route from entry.PostingAddress().Route().Route() — use it for route comparisons, not the raw SubAccountRoute. |
| `defaults.go` | DefaultValidator — the canonical production validator with all 12+ rules pre-composed in order; the single Validator used by historical.Ledger.CommitGroup. | Adding a rule here affects every CommitGroup in production. Test bidirectionally with TestDefaultValidator_* cases. |
| `routingrules_test.go` | Acceptance/rejection scenario table for DefaultValidator; addressForRoute constructs PostingAddress via ledger.BuildRoutingKey + ledgeraccount.NewAddressFromData. | New tests must use addressForRoute and require.ErrorContains(t, err, 'ledger routing rule violated') for rejections. |

## Anti-Patterns

- Accessing ledger.EntryInput.PostingAddress().Route().Route() directly in rule code instead of EntryView.Route().
- Creating a new Validator with a subset of rules instead of extending DefaultValidator.Rules — silently removes production constraints.
- Adding DB calls, external I/O, or mutable state inside RoutingRule.Validate — rules must be pure functions of TxView.
- Skipping the HasAccountTypes guard in a rule — false positives when relevant accounts are absent.
- Returning plain fmt.Errorf instead of ledger.ErrRoutingRuleViolated.WithAttrs — breaks the error-type contract used by callers.

## Decisions

- **Rules are applied sequentially and fail-fast (first error) rather than accumulating all violations.** — Routing violations are programmer errors or abuse, not user-facing validation needing full accumulation; fail-fast simplifies rule logic.
- **TxView pre-decodes the Route from SubAccountRoute.Route() at construction time.** — Avoids repeated decode calls inside each rule and simplifies field-comparison helpers.
- **DefaultValidator is a package-level var, not a constructor function.** — Rules are stateless structs; a single shared instance is safe and avoids per-CommitGroup allocation.

## Example: Validate entries before booking a transaction (production usage)

```
import (
  "github.com/openmeterio/openmeter/openmeter/ledger"
  "github.com/openmeterio/openmeter/openmeter/ledger/routingrules"
)

// In historical.Ledger.CommitGroup:
if err := ledger.ValidateTransactionInputWith(ctx, txInput, routingrules.DefaultValidator); err != nil {
  return nil, fmt.Errorf("invalid transaction: %w", err)
}
```

<!-- archie:ai-end -->
