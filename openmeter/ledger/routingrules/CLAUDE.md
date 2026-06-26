# routingrules

<!-- archie:ai-start -->

> Declarative routing-rule validator for ledger transactions: given a set of ledger.EntryInput postings, verify the combination of account types, signs, and route dimensions forms a legal double-entry flow. The package's single constraint is that it is pure validation logic over decoded entries — no DB, no service, no context.

## Patterns

**RoutingRule interface + struct rules** — Every rule is a struct implementing `RoutingRule` with one method `Validate(tx TxView) error`. New rules are added as struct types (e.g. RequireFlowDirectionRule) and registered in DefaultValidator.Rules, not as ad-hoc functions. (`type RequireFlowDirectionRule struct { From, To ledger.AccountType }; func (r RequireFlowDirectionRule) Validate(tx TxView) error {...}`)
**Validator implements ledger.RoutingValidator** — Validator{Rules []RoutingRule} satisfies `var _ ledger.RoutingValidator = (*Validator)(nil)` via ValidateEntries, which builds a TxView once then runs every rule in order, short-circuiting on first error. (`func (v Validator) ValidateEntries(entries []ledger.EntryInput) error { view, _ := NewTxView(entries); for _, r := range v.Rules { if err := r.Validate(view); err != nil { return err } } }`)
**Gate-then-check rule shape** — Rules first gate on relevance (e.g. `if !tx.HasAccountTypes(r.From, r.To) { return nil }` or `len(accountTypes) != 1`) and return nil when not applicable, only validating when the targeted account-type shape is present. (`if !tx.HasAccountTypes(r.Left, r.Right) { return nil }`)
**Violations via ledger.ErrRoutingRuleViolated.WithAttrs** — Every failure returns ledger.ErrRoutingRuleViolated decorated with models.Attributes carrying a snake_case `reason` plus account/field context. Never return raw errors.New for rule violations. (`return ledger.ErrRoutingRuleViolated.WithAttrs(models.Attributes{"reason": "invalid_flow_direction", "account_type": r.From})`)
**Route comparison via RouteField enum** — Cross-entry field matching goes through the RouteField string enum and sameRouteField switch; add a new comparable dimension by extending RouteField and the switch, then reference it in RequireSameRouteRule.Fields. Unknown fields return an error. (`case RouteFieldCurrency: return left.Currency == right.Currency, nil`)
**Optional-field equality helpers** — Pointer route fields are compared with optional*Equal helpers (optionalStringEqual, optionalDecimalEqual, etc.) that treat both-nil as equal and one-nil as unequal; reuse these instead of inlining nil checks. (`func optionalDecimalEqual(l, r *alpacadecimal.Decimal) bool { if l==nil||r==nil { return l==nil&&r==nil }; return l.Equal(*r) }`)
**Immutable TxView accessors** — TxView/EntryView are read-only projections; Entries() returns slices.Clone, EntriesOf filters by account type, AccountTypes() dedups+sorts. Rules query these accessors and never mutate entries. (`func (t TxView) Entries() []EntryView { return slices.Clone(t.entries) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `routingrules.go` | All RoutingRule struct types, the Validator, RouteField enum, and the sign/route helper functions (entriesBySign, requireMatchingRouteFields, requireKnownToUnknownCostBasisTranslationEitherDirection). | Cost-basis translation rules use requireKnownToUnknownCostBasisTranslationEitherDirection (tries both directions) — preserve the either-direction semantics so both attribute and correction flows validate. Argument order to that helper matters per rule (FBO passes positiveEntries, negativeEntries). |
| `defaults.go` | DefaultValidator instance wiring the full production rule chain: allowed account-type sets, flow directions, auth-status, cost-basis translations, and per-pair RequireSameRouteRule field requirements. | This is the live ruleset used everywhere (testutils, historical ledger). Adding/removing an account-type combination requires editing AllowedAccountSetsRule.Sets AND likely the matching RequireFlowDirectionRule/RequireSameRouteRule, or otherwise-valid flows get rejected. |
| `view.go` | EntryView/TxView projections (decode PostingAddress into accountType + ledger.Route) plus all optional*Equal / stringSliceEqual comparison helpers. | newEntryView decodes the route eagerly via entry.PostingAddress().Route().Route(); a malformed route surfaces here. AccountTypes() sorts — don't rely on insertion order. |
| `routingrules_test.go` | Table of TestDefaultValidator_* cases asserting allowed/rejected flows; the canonical spec of what each rule permits. | Uses transactionstestutils.AnyEntryInput + addressForRoute helper. New rules MUST add both an Allows* and a Rejects* case; rejection assertions match on `require.ErrorContains(t, err, "ledger routing rule violated")`. |

## Anti-Patterns

- Returning raw errors (errors.New/fmt.Errorf) for rule violations instead of ledger.ErrRoutingRuleViolated.WithAttrs — callers and tests rely on that error identity/message.
- Adding DB access, context.Context, or service calls into a rule — this package is pure validation over already-decoded entries.
- Comparing pointer route fields with == or direct deref instead of the optional*Equal helpers, breaking nil-vs-nil semantics.
- Mutating the slice returned by TxView.Entries()/EntriesOf() expecting it to affect the view — they are clones/copies.
- Adding a new account-type combination to one rule (e.g. a flow direction) without updating AllowedAccountSetsRule.Sets, so the combination is rejected before the new rule runs.

## Decisions

- **Rules are structs satisfying RoutingRule and composed in a Validator list rather than one monolithic validate function.** — Each accounting invariant (flow direction, cost-basis translation, tax-dimension scope) is independently testable and reorderable, and the production ruleset is declared as data in DefaultValidator.
- **Validation operates on a decoded TxView/EntryView projection built once by ValidateEntries.** — Decoding PostingAddress→accountType+Route once and exposing query accessors (EntriesOf, HasAccountTypes, AccountTypes) keeps every rule simple and avoids repeated decoding per rule.

## Example: Add a new account-flow invariant as a rule and register it

```
type RequireFlowDirectionRule struct {
	From ledger.AccountType
	To   ledger.AccountType
}

func (r RequireFlowDirectionRule) Validate(tx TxView) error {
	if !tx.HasAccountTypes(r.From, r.To) {
		return nil
	}
	fromEntries := tx.EntriesOf(r.From)
	toEntries := tx.EntriesOf(r.To)
	if allEntriesPositive(fromEntries) && allEntriesNegative(toEntries) {
		return nil
	}
	return ledger.ErrRoutingRuleViolated.WithAttrs(models.Attributes{
// ...
```

<!-- archie:ai-end -->
