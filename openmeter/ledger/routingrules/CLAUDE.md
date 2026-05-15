# routingrules

<!-- archie:ai-start -->

> Pure validation layer for double-entry ledger transactions: defines the RoutingRule interface, composable Validator, and DefaultValidator encoding all permitted account-type pairings and route-field constraints. No DB access — validates []ledger.EntryInput before any persistence call.

## Patterns

**RoutingRule interface + Validator composition** — Each constraint is a named struct implementing RoutingRule{Validate(TxView) error}. Validator.ValidateEntries runs them sequentially, fail-fast on first error. New rules are appended to DefaultValidator.Rules in defaults.go — never create ad-hoc Validator instances in application code. (`type AllowedAccountSetsRule struct { Sets [][]ledger.AccountType }
func (r AllowedAccountSetsRule) Validate(tx TxView) error { ... }`)
**TxView as the rule's only input** — NewTxView converts []ledger.EntryInput into []EntryView (with pre-decoded Route via entry.PostingAddress().Route().Route()), exposing AccountTypes(), EntriesOf(accountType), HasAccountTypes(...). Rules must use TxView — never the raw EntryInput slice. (`view, err := NewTxView(entries)
for _, rule := range v.Rules { rule.Validate(view) }`)
**HasAccountTypes guard before rule logic** — Every rule that targets specific account types checks tx.HasAccountTypes(r.From, r.To) first and returns nil when those types are absent. Omitting the guard causes false positives on unrelated transactions. (`func (r RequireFlowDirectionRule) Validate(tx TxView) error {
    if !tx.HasAccountTypes(r.From, r.To) { return nil }
    // ... enforcement logic
}`)
**RequireSameRouteRule for cross-account field consistency** — Ensures entries on two different account types share the same RouteField values (currency, tax_code, features, cost_basis). Declared as Left/Right/Fields triplets in DefaultValidator. Use requireMatchingRouteFields helper internally. (`RequireSameRouteRule{Left: ledger.AccountTypeCustomerFBO, Right: ledger.AccountTypeCustomerReceivable, Fields: []RouteField{RouteFieldCurrency, RouteFieldTaxCode}}`)
**FuncRule for test-only rules** — FuncRule wraps a plain function as a RoutingRule for test-only validation scenarios without defining a named struct. Never use FuncRule in production code. (`FuncRule(func(tx TxView) error { return nil })`)
**ErrRoutingRuleViolated with WithAttrs for all errors** — All rule violations return ledger.ErrRoutingRuleViolated.WithAttrs(models.Attributes{...}) with a 'reason' key and relevant context. Never return plain fmt.Errorf from a RoutingRule. (`return ledger.ErrRoutingRuleViolated.WithAttrs(models.Attributes{"reason": "account_type_combination_not_allowed", "account_types": present})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `routingrules.go` | Core types: RoutingRule interface, Validator (implements ledger.RoutingValidator), FuncRule, and all concrete rule structs (AllowedAccountSetsRule, RequireFlowDirectionRule, RequireSameRouteRule, RequireAccountAuthorizationStatusRule, RequireReceivableAuthorizationStageRule, RequireAccruedCostBasisTranslationRule, RequireFBOCostBasisTranslationRule, RequireUniqueSubAccountsRule). | All rules skip gracefully when the relevant account types are absent (HasAccountTypes check first) — this is intentional permissive-when-unrelated behaviour. Do not remove these guards. |
| `view.go` | EntryView (wraps EntryInput + pre-decoded Route), TxView (slice of EntryView with projection methods AccountTypes/EntriesOf/HasAccountTypes/HasAccountType), and optional field comparators (optionalStringEqual, optionalDecimalEqual, etc.). | EntryView.Route() returns the decoded ledger.Route from entry.PostingAddress().Route().Route() — use this for route field comparisons, not the raw SubAccountRoute. |
| `defaults.go` | DefaultValidator — the canonical production validator with all 12+ rules pre-composed in order. This is the single Validator used by historical.Ledger.CommitGroup in production. | Adding a new rule here affects every CommitGroup call in production. Test bidirectionally with TestDefaultValidator_* cases in routingrules_test.go. |
| `routingrules_test.go` | Table of acceptance/rejection scenarios for DefaultValidator. Helper addressForRoute constructs PostingAddress from route via ledger.BuildRoutingKey + ledgeraccount.NewAddressFromData. | New tests must use the addressForRoute helper and require.ErrorContains(t, err, 'ledger routing rule violated') for rejection cases. |

## Anti-Patterns

- Accessing ledger.EntryInput.PostingAddress().Route().Route() directly in rule code instead of using EntryView.Route() — bypasses the pre-decoded projection.
- Creating a new Validator with a subset of rules instead of extending DefaultValidator.Rules — silently removes production constraints.
- Adding DB calls, external I/O, or mutable state inside RoutingRule.Validate — rules must be pure functions of TxView.
- Skipping the HasAccountTypes guard in a rule — causes false positives when the rule's relevant accounts are absent from the transaction.
- Returning plain fmt.Errorf from a RoutingRule instead of ledger.ErrRoutingRuleViolated.WithAttrs — breaks the error-type contract used by callers.

## Decisions

- **Rules are applied sequentially and fail-fast (first error returned) rather than collecting all violations.** — Routing violations are programmer errors or abuse, not user-facing validation flows that need full error accumulation. Fail-fast simplifies rule logic.
- **TxView pre-decodes the Route from SubAccountRoute.Route() at construction time.** — Avoids repeated decode calls inside each rule and makes field-comparison helpers in requireMatchingRouteFields straightforward.
- **DefaultValidator is a package-level var, not a constructor function.** — Rules are stateless structs; a single shared instance is safe and avoids repeated allocation on every CommitGroup call.

## Example: Validate entries before booking a transaction (production usage pattern)

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
