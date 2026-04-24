# routingrules

<!-- archie:ai-start -->

> Pure validation layer for double-entry ledger transactions: defines the RoutingRule interface, a composable Validator, and a DefaultValidator that encodes all permitted account-type pairings and route-field constraints. No DB access — validates []ledger.EntryInput before any persistence.

## Patterns

**RoutingRule interface + Validator composition** — Each constraint is a struct implementing RoutingRule{Validate(TxView) error}. Validator.ValidateEntries runs them sequentially; first failure returns immediately. (`type AllowedAccountSetsRule struct { Sets [][]ledger.AccountType }
func (r AllowedAccountSetsRule) Validate(tx TxView) error { ... }`)
**TxView as the validation projection** — NewTxView converts []ledger.EntryInput into []EntryView (with pre-decoded Route), exposing AccountTypes(), EntriesOf(accountType), HasAccountTypes(...) for rule predicates. Rules must use TxView — not the raw entries. (`view, err := NewTxView(entries)
for _, rule := range v.Rules { rule.Validate(view) }`)
**DefaultValidator as the canonical rule set** — DefaultValidator (defaults.go) is the single pre-built Validator used in production. New rules are added to its Rules slice; do not create ad-hoc Validator instances in application code. (`var DefaultValidator = Validator{Rules: []RoutingRule{AllowedAccountSetsRule{...}, RequireFlowDirectionRule{...}, ...}}`)
**RequireSameRouteRule for cross-account field consistency** — Ensures that entries on two different account types share the same route fields (currency, tax_code, features, cost_basis). Declared as Left/Right/Fields triplets. (`RequireSameRouteRule{Left: FBO, Right: Receivable, Fields: []RouteField{RouteFieldCurrency, RouteFieldTaxCode, ...}}`)
**FuncRule for ad-hoc rules in tests** — FuncRule wraps a plain function as a RoutingRule for quick test-only validation scenarios without defining a named struct. (`FuncRule(func(tx TxView) error { ... })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `routingrules.go` | Core types: RoutingRule interface, Validator struct (implements ledger.RoutingValidator), FuncRule, AllowedAccountSetsRule, RequireFlowDirectionRule, RequireSameRouteRule, RequireAccountAuthorizationStatusRule, RequireReceivableAuthorizationStageRule, RequireAccruedCostBasisTranslationRule, RequireFBOCostBasisTranslationRule. | All rules skip gracefully when the relevant account types are absent (tx.HasAccountTypes check first) — this is intentional permissive-when-unrelated behaviour. |
| `view.go` | EntryView (wraps EntryInput + pre-decoded Route), TxView (slice of EntryView with projection methods), helper comparators (optionalStringEqual, optionalDecimalEqual, etc.). | EntryView.Route() returns the decoded ledger.Route (not SubAccountRoute) — use this for route field comparisons. |
| `defaults.go` | DefaultValidator — the canonical production validator with all 12+ rules pre-composed. | Adding a new rule here affects every CommitGroup call in production; test bidirectionally with TestDefaultValidator_* cases. |
| `routingrules_test.go` | Table of acceptance/rejection scenarios for DefaultValidator; helper addressForRoute constructs PostingAddress from route for test entry inputs. | Test helper uses ledgeraccount.NewAddressFromData and ledger.BuildRoutingKey — new tests must follow the same pattern. |

## Anti-Patterns

- Accessing ledger.EntryInput.PostingAddress().Route().Route() directly in rules instead of using EntryView.Route() — bypasses the pre-decoded projection.
- Creating a new Validator with a subset of rules instead of extending DefaultValidator.Rules — silently removes production constraints.
- Adding DB calls or state inside RoutingRule.Validate — rules must be pure functions of TxView.
- Skipping the HasAccountTypes guard inside a rule — causes false positives when the rule's relevant accounts are absent from the transaction.

## Decisions

- **Rules are applied sequentially and fail-fast (first error returned) rather than collecting all violations.** — Simplifies rule logic; routing violations are programmer errors or abuse, not user validation flows that need full error accumulation.
- **TxView pre-decodes the Route from SubAccountRoute.Route() at construction time so each rule can compare plain ledger.Route fields.** — Avoids repeated decode calls inside each rule and makes field-comparison helpers straightforward.

## Example: Validate entries before booking a transaction

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
