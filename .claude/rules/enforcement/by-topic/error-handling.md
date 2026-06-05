# Enforcement: error-handling (3 rules)

Topic file. Loaded on demand when an agent works on something in the `error-handling` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Decision Violations (block)

### `dec-validate-001` — Validate() must collect all issues and return models.NewNillableGenericValidationError(errors.Join(...)), not fail on the first field

*source: `deep_scan`*

**Why:** Validation must report all problems at once (not fail-fast) and surface as a single 400/ValidationIssue at the HTTP boundary. Validate() methods collect issues into var errs []error, wrap each with field context (fmt.Errorf("field: %w", err)), and return models.NewNillableGenericValidationError(errors.Join(errs...)) so a nil join yields nil; single-field checks use models.NewGenericValidationError(...) directly. Returning on the first invalid field, or a bespoke per-domain validation error, breaks the uniform contract the error encoders map to 400.

**Example:**

```
func (i Input) Validate() error {
    var errs []error
    if i.Name == "" { errs = append(errs, errors.New("name: required")) }
    if err := i.Address.Validate(); err != nil { errs = append(errs, fmt.Errorf("address: %w", err)) }
    return models.NewNillableGenericValidationError(errors.Join(errs...))
}
```

**Path glob:** `openmeter/**`, `api/**`, `pkg/**`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "func \\([^)]*\\) Validate\\(\\) error"
    ],
    "must_not_match": [
      "NewNillableGenericValidationError",
      "NewGenericValidationError",
      "_test\\.go"
    ]
  }
]
```

</details>

## Pitfalls (block)

### `pf-ledger-panic-001` — Never panic in production code; ledger TemplateCode and every new TransactionTemplate must declare a non-empty code()

*source: `deep_scan`*

**Why:** Pitfall pf_0013: TemplateCode(template) panics when a TransactionTemplate's code() returns an empty string (openmeter/ledger/transactions/codes.go:113), so a newly added template that forgets to declare its code() crashes the ledger collector at runtime instead of failing the build or returning an error. AGENTS.md: never use panic in non-test code paths; if a new failure mode is possible, change the function signature to return an error and propagate it explicitly.

**Path glob:** `openmeter/**`, `pkg/**`, `app/**`, `cmd/**`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "\\bpanic\\("
    ],
    "must_not_match": [
      "_test\\.go",
      "testutils",
      "testutil"
    ]
  }
]
```

</details>

## Pattern Divergence (inform)

### `sem-httptransport-001` — Build v1 endpoints with httptransport.Handler[Req,Resp] and map domain errors via typed errorEncoders

*source: `deep_scan`*

**Why:** v1 endpoints use httptransport.NewHandler[Request,Response](decode, service-op, encode, ...opts) with an errorEncoder that maps domain error types to status codes. Error mapping uses an ordered HandleErrorIfTypeMatches[T] chain (errors.As short-circuit: NotFoundError→404, GenericValidationError→400, UpdateAfterDeleteError→409) plus the ValidationIssue openmeter.http.status_code attribute (singular mapping). The v3 surface mirrors the same logic in apierrors.NewV3ErrorHandlerFunc.
