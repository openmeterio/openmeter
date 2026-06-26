# ledger

<!-- archie:ai-start -->

> Double-entry-style ledger for customer credit/business accounts, feature-gated by credits.enabled. The root declares the account taxonomy (customer FBO/receivable/accrued + business wash/earnings/brokerage/breakage), posting primitives (Account/SubAccount/PostingAddress/Entry/Transaction/Ledger), versioned routing (Route/RoutingKey), balance and impact queries, annotation vocabulary, and ValidationIssue error codes. Subpackages provide account, transactions, historical, resolvers, routingrules, recognizer, breakage and a noop/ used when credits are off.

## Patterns

**Accounts vs SubAccounts vs PostingAddress** — Account describes ownership/purpose (Type/ID); SubAccount is the concrete postable address derived from a Route via GetSubAccountForRoute; PostingAddress is the routing-only handle (SubAccountID/AccountType/Route) used in entries. Per-type route param structs (CustomerFBORouteParams, CustomerReceivableRouteParams, BusinessRouteParams) build a Route and Validate it. (`fboSub, err := customerAccounts.FBOAccount.GetSubAccountForRoute(ctx, ledger.CustomerFBORouteParams{Currency: "USD", CreditPriority: ledger.DefaultCustomerFBOPriority})`)
**Versioned routing keys** — Route.Normalize() picks the minimum RoutingKeyVersion via selectRoutingKeyVersion (V2 when TaxBehavior set, else V1); BuildRoutingKey dispatches per version (BuildRoutingKeyV1/V2). A new identity-affecting field means a new version, not a mutated key format. (`normalized, _ := route.Normalize(); key, _ := ledger.BuildRoutingKey(normalized)`)
**Atomic transaction groups balanced to zero** — TransactionInput/EntryInput build a TransactionGroupInput committed atomically via Ledger.CommitGroup; ValidateTransactionInput enforces entries sum to 0 (ErrInvalidTransactionTotal). Transactions order by TransactionCursor (BookedAt, then CreatedAt, then ID). (`l.CommitGroup(ctx, txInput.AsGroupInput("namespace", nil))`)
**RouteFilter with mo.Option tri-state matching** — RouteFilter uses mo.Option[*T] for TaxCode/TaxBehavior/Features/CostBasis so a filter can require 'this nil value' vs 'do not filter'; CreditPriority/TransactionAuthorizationStatus are plain pointers meaningful only for fbo/receivable queries. EntryMatchesImpactFilter / TransactionImpact apply this. (`ledger.RouteFilter{TaxCode: mo.Some[*string](nil)} // matches only entries with nil tax code`)
**Annotations as cross-domain linkage vocabulary** — Charge/subscription/breakage/transaction linkage rides in models.Annotations under stable keys (AnnotationChargeID, AnnotationTransactionTemplateCode, AnnotationBreakageKind...) built by helper constructors (ChargeTransactionAnnotations, TransactionAnnotations, BreakageAnnotations) and read back via typed extractors. (`annotations := ledger.TransactionAnnotations("customer.receivable.issue", ledger.TransactionDirectionForward)`)
**Errors are ValidationIssues with attrs** — All ledger errors are models.NewValidationIssue(code, msg) constants; callers attach context via .WithAttrs(models.Attributes{...}) and tests assert by issue.Code() == ErrCode... (`var ErrInvalidTransactionTotal = models.NewValidationIssue(ErrCodeInvalidTransactionTotal, "...credits and debits must sum to 0")`)
**Required route params enforced by the type system** — CreditPriority on CustomerFBORouteParams and TransactionAuthorizationStatus on CustomerReceivableRouteParams are non-pointer/required so the compiler forces an explicit choice; Validate adds value-range checks (ValidateCreditPriority). (`type CustomerFBORouteParams struct { Currency currencyx.Code; CreditPriority int; ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `account.go` | AccountType enum, Customer/Business account groupings, AccountResolver/Reader/Provisioner/Locker interfaces, Create*Input | CustomerAccountTypes vs BusinessAccountTypes drive provisioning; AccountType.Validate is a closed switch. |
| `accounts.go` | Per-type SubAccount interfaces + route param structs (FBO/Receivable/Accrued/Business) with Route()+Validate() | Each param struct maps to a Route; FBO requires CreditPriority, Receivable requires TransactionAuthorizationStatus. |
| `primitives.go` | PostingAddress/SubAccount/Account/Entry/Transaction/Ledger interfaces, RouteFilter, TransactionCursor.Compare | Cursor order is BookedAt then CreatedAt then ID; entries must reference a valid SubAccount address. |
| `routing.go` | RoutingKeyVersion, Route, Normalize, selectRoutingKeyVersion, BuildRoutingKey(V1/V2) | Never change an existing version's key format; add a new RoutingKeyVersion and extend BuildRoutingKey's switch. |
| `impact.go` | TransactionImpact / EntryMatchesImpactFilter — sum entry amounts matching an ImpactFilter (AccountType + RouteFilter) | Route matching uses the normalized route's Matches; mo.Option nil-required semantics matter. |
| `annotations.go` | Annotation key constants + builder/extractor helpers (charge/transaction/breakage linkage, direction, breakage kinds) | Use the helper constructors and *FromAnnotations extractors instead of literal keys; extractors error on missing/invalid. |
| `errors.go` | All ledger ValidationIssue error codes/constants | Return these issues (with .WithAttrs) rather than bare errors so API mapping and tests by Code() work. |
| `balance.go` | BalanceQuerier: GetAccountBalance/GetSubAccountBalance over a RouteFilter + BalanceQuery (After cursor / AsOf) | Balance distinguishes Settled() vs Pending(). |

## Anti-Patterns

- Mutating an existing RoutingKey format instead of introducing a new RoutingKeyVersion (breaks historical routing-key matching).
- Posting an unbalanced transaction (entries not summing to 0) — ValidateTransactionInput rejects with ErrInvalidTransactionTotal.
- Writing literal annotation key strings instead of the AnnotationX constants and their builder/extractor helpers.
- Building a SubAccount/posting address by hand instead of GetSubAccountForRoute on the typed account, skipping route Validate (e.g. omitting CreditPriority/TransactionAuthorizationStatus).
- Returning bare errors instead of the ledger ValidationIssue constants with .WithAttrs context.

## Decisions

- **Routing keys are explicitly versioned and selected by the minimum version a route needs.** — Lets the route schema grow (e.g. adding tax_behavior in V2) without rewriting historical keys; balance/impact matching stays stable across versions.
- **Account ownership/purpose (Account) is separated from the concrete postable address (SubAccount/PostingAddress).** — A single account fans out to many sub-accounts parameterized by currency/priority/tax route, so postings can be filtered and balanced per route while accounts stay coarse.
- **Some route params (FBO CreditPriority, Receivable authorization status) are non-pointer required fields.** — Forces callers to make an explicit routing decision at the type level rather than defaulting silently.

## Example: Resolving a customer FBO sub-account and committing a balanced transaction group

```
fboSub, err := customerAccounts.FBOAccount.GetSubAccountForRoute(ctx, ledger.CustomerFBORouteParams{
	Currency:       currencyx.Code("USD"),
	CreditPriority: ledger.DefaultCustomerFBOPriority,
})
if err != nil { return err }

inputs, err := transactions.ResolveTransactions(ctx, deps, scope, tx1, tx2, tx3)
if err != nil { return err }
_, err = histLedger.CommitGroup(ctx, transactions.GroupInputs(namespace, nil, inputs...))
```

<!-- archie:ai-end -->
