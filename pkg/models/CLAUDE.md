# models

<!-- archie:ai-start -->

> Foundational domain primitive library shared by all openmeter/* and api/* packages. Provides namespace-scoped identity types (NamespacedID, NamespacedKey), lifecycle base models (ManagedModel, CadencedModel, VersionedModel), generic service hook registry, structured validation errors (ValidationIssue/ValidationIssues), RFC 7807 problem types, and field-path descriptors for error attribution. The primary constraint is zero-import from openmeter/* domain packages — this package must remain a leaf.

## Patterns

**Generic error wrapping with typed sentinels** — All domain errors are wrapped in typed GenericXxxError structs (GenericNotFoundError, GenericValidationError, GenericConflictError, etc.) that implement GenericError (error + Unwrap). Use NewGenericXxxError(err) constructors; check with IsGenericXxxError(err). HTTP layer maps these types to status codes. (`return nil, models.NewGenericNotFoundError(fmt.Errorf("customer %s not found", id))`)
**ValidationIssue builder pattern (immutable with-chains)** — ValidationIssue is constructed via NewValidationIssue(code, message, opts...) or NewValidationError/NewValidationWarning. Mutations return new copies: issue.WithField(...), .WithAttr(key, val), .WithSeverity(...), .WithPathString(...). Never mutate a ValidationIssue directly — all With* methods clone first. (`issue := models.NewValidationError("invalid_field", "must be positive").WithPathString("lineItems", "amount").WithAttr(httpStatusCodeAttr, 422)`)
**FieldDescriptor tree for structured field paths** — Field paths in ValidationIssue use FieldDescriptor trees built via NewFieldSelector(field), NewFieldSelectorGroup(selectors...), and .WithPrefix(). They produce both human-readable strings (plan[key=pro].phases[key=trial]) and JSONPath expressions ($.plan[?(@.key=='pro')].phases[?(@.key=='trial')]) used in error serialization. (`models.NewFieldSelectorGroup(models.NewFieldSelector("phases").WithExpression(models.NewFieldAttrValue("key", "trial")), models.NewFieldSelector("rateCards"))`)
**ServiceHookRegistry for cross-domain lifecycle callbacks** — ServiceHookRegistry[T] implements both ServiceHooks[T] (RegisterHooks) and ServiceHook[T] (Pre/Post Create/Update/Delete). It uses a per-registry context key to prevent re-entrant hook invocations. Domain services embed *ServiceHookRegistry[DomainType] and expose RegisterHooks externally so other packages register callbacks without import cycles. (`registry := models.NewServiceHookRegistry[Customer]()
registry.RegisterHooks(ledgerHook, notificationHook)
// Before delete: registry.PreDelete(ctx, &customer)`)
**CadencedModel for time-bounded entities** — Entities active over a half-open interval [ActiveFrom, ActiveTo) embed CadencedModel. Use IsActiveAt(t) to test point-in-time membership. CadenceList[T] validates sorted, non-overlapping, and continuous cadence sequences via GetOverlaps() and IsContinuous(). (`if !entity.CadencedModel.IsActiveAt(clock.Now()) { return nil, models.NewGenericNotFoundError(...) }`)
**Validator interface on all input types** — Service input types and model types implement models.Validator (Validate() error). NamespacedID.Validate(), ManagedResource.Validate(), and NamespacedModel.Validate() are called at service entry points before any adapter interaction. Use models.Validate[T](v, ...ValidatorFunc[T]) for composing multiple validators. (`if err := input.Validate(); err != nil { return nil, models.NewGenericValidationError(err) }`)
**RFC 7807 Problem via NewStatusProblem** — HTTP error responses use StatusProblem serialized as application/problem+json. NewStatusProblem(ctx, err, statusCode) extracts request-id from Chi middleware context, maps 'context canceled' to 408, and suppresses detail on 500. Extensions map carries validationErrors array when applicable. (`problem := models.NewStatusProblem(ctx, err, http.StatusBadRequest)
problem.Extensions = models.EncodeValidationIssues(err)
problem.Respond(w)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `pkg/models/errors.go` | Defines all generic domain error types (NotFound, Validation, Conflict, Forbidden, Unauthorized, NotImplemented, PreConditionFailed, StatusFailedDependency) plus ErrorWithFieldPrefix and ErrorWithComponent wrappers for structured error propagation. | Do not add new error types without also adding an IsXxxError(err) checker and ensuring the HTTP error encoder maps it to a status code. |
| `pkg/models/validationissue.go` | ValidationIssue is the structured error type used for API validation responses. AsValidationIssues(err) traverses a mixed error tree and converts componentWrapper/fieldPrefixedWrapper/ValidationIssue nodes into a flat ValidationIssues slice; unknown leaf errors return an error only if no component context was seen. | WithAttr/WithAttrs panic on nil or non-comparable keys. Clone() sets wraps=self so errors.Is chains work — don't set wraps manually. |
| `pkg/models/servicehook.go` | Thread-safe generic hook registry with re-entrancy prevention. The per-registry context key (a loopKey pointer-based string) ensures a hook that calls back through the same service does not trigger infinite recursion. | init() is lazy via sync.Once — the id field is set on first use. Do not compare registries by value; the id field depends on pointer identity. |
| `pkg/models/fielddescriptor.go` | FieldDescriptor forms a tree via treex.Node for representing nested field paths. WithPrefix() prepends a subtree; WithExpression() attaches array/condition filter. String() and JSONPath() walk leaf nodes only via DFS. | WithPrefix and WithExpression return new *FieldDescriptor values (copy-on-write); original is not mutated. NewFieldSelectorGroup with all-nil selectors returns nil — callers must nil-check. |
| `pkg/models/model.go` | ManagedModel (CreatedAt/UpdatedAt/DeletedAt), NamespacedModel (Namespace), ManagedResource, ManagedUniqueResource, and VersionedModel base structs that all persistent domain entities embed. | NewManagedResource converts all times to UTC via .UTC() — always store UTC. IsDeletedAt uses !DeletedAt.After(t), meaning entity deleted exactly at t is considered deleted. |
| `pkg/models/problem.go` | RFC 7807 HTTP problem response. NewStatusProblem suppresses the error detail for 500 responses to avoid leaking internals. Extensions is initialized as an empty map (never nil) so callers can always assign to it. | context canceled substring check is string-based — not errors.Is — to catch wrapped context errors. Do not produce 500 problems with Detail set. |
| `pkg/models/cadencelist.go` | CadenceList[T] helpers for validating time-ordered entity sequences. GetOverlaps is O(n) scanning adjacent pairs only (assumes sorted input). IsContinuous requires exact ActiveTo == next.ActiveFrom equality. | GetOverlaps returns empty slice (not nil) for no overlaps. Use NewSortedCadenceList for auto-sorted input. |
| `pkg/models/annotation.go` | Annotations is map[string]interface{} with Clone (deep copy via brunoga/deep), Merge (right-wins shallow merge of deep-copied values), and typed getters GetBool/GetString/GetInt with float64 integer coercion. | GetInt accepts float64 whole numbers (JSON numbers deserialize as float64). Non-comparable or fractional float64 returns (0, false). |

## Anti-Patterns

- Importing from openmeter/* domain packages inside pkg/models — it must remain a leaf package with no openmeter/* imports.
- Mutating a ValidationIssue in place — always use With* methods which clone; direct struct field assignment bypasses the immutability contract.
- Returning a plain errors.New() from a service layer where ValidationIssue is expected — callers use AsValidationIssues which can only extract ValidationIssue nodes, not raw errors.
- Using reflect.DeepEqual for Annotations or ValidationIssue equality — use the provided Equal() methods which handle nil and non-comparable values correctly.
- Creating a custom Paginator type instead of using NewPaginator[T](fn) — the paginator[T] unexported struct is the only implementation; wrap your list function via NewPaginator.

## Decisions

- **ValidationIssue uses a private constructor (newValidationIssue with option functions) and copy-on-write With* methods instead of a builder pattern.** — Ensures every ValidationIssue instance is valid at construction (code + message required) and prevents accidental shared mutation when issues are passed through multiple error-wrapping layers.
- **ServiceHookRegistry uses a pointer-identity-based context key (loopKey derived from fmt.Sprintf("%p", r)) to prevent re-entrant hook invocations.** — Domain hooks (e.g. ledger hook on customer.PostCreate) may themselves call back into the customer service; the context sentinel blocks infinite recursion without requiring hooks to be aware of each other.
- **FieldDescriptor forms a tree (via pkg/treex) rather than a simple dot-delimited string.** — Enables WithPrefix composition (prepend path segments from outer service layer), structured JSONPath generation for API error payloads, and attribute attachment at any node level — all impossible with a flat string.

## Example: Returning a structured ValidationIssue with field path from a domain service

```
import (
    "github.com/openmeterio/openmeter/pkg/models"
)

// Build a validation error with a nested field path and custom attribute
func validateAmount(amount float64) error {
    if amount <= 0 {
        return models.NewValidationError("invalid_amount", "amount must be positive").
            WithPathString("lineItems", "amount").
            WithAttr(httpStatusCodeAttr, 422)
    }
    return nil
}

// In HTTP handler, convert to API response
// ...
```

<!-- archie:ai-end -->
