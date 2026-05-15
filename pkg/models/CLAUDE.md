# models

<!-- archie:ai-start -->

> Foundational domain primitive library shared by all openmeter/* and api/* packages — provides typed error sentinels, immutable ValidationIssue builder, ServiceHookRegistry with re-entrancy prevention, RFC 7807 StatusProblem, FieldDescriptor trees for structured field paths, and time-bounded CadencedModel helpers. The primary constraint is zero imports from openmeter/* domain packages; this package must remain a pure leaf.

## Patterns

**Generic typed error sentinels** — All domain errors are wrapped in typed GenericXxxError structs (GenericNotFoundError, GenericValidationError, GenericConflictError, GenericForbiddenError, GenericUnauthorizedError, GenericNotImplementedError, GenericPreConditionFailedError, GenericStatusFailedDependencyError). Use NewGenericXxxError(err) constructors and check with IsGenericXxxError(err) via errors.As. HTTP layer maps each type to its status code via GenericErrorEncoder. (`return nil, models.NewGenericNotFoundError(fmt.Errorf("customer %s not found", id))`)
**ValidationIssue immutable with-chain builder** — ValidationIssue is constructed via NewValidationIssue(code, message) or NewValidationError/NewValidationWarning. All With* methods (WithPathString, WithComponent, WithSeverity, WithAttr) clone and return a new value — never mutate directly. AsValidationIssues(err) traverses a mixed error tree extracting ValidationIssue nodes; plain errors.New() leaves are not extracted. (`return models.NewValidationError("invalid_amount", "must be positive").WithPathString("lineItems", "amount").WithAttr(httpStatusCodeAttr, 422)`)
**FieldDescriptor tree for structured field paths** — Field paths use FieldDescriptor trees: NewFieldSelector(field) creates a leaf; NewFieldSelectorGroup(selectors...) groups siblings; WithExpression attaches a filter (NewFieldAttrValue, NewMultiFieldAttrValue, NewFieldArrIndex); WithPrefix prepends a subtree. String() and JSONPath() walk leaf nodes via DFS. NewFieldSelectorGroup with all-nil selectors returns nil — always nil-check. (`models.NewFieldSelectorGroup(models.NewFieldSelector("phases").WithExpression(models.NewFieldAttrValue("key", "trial")), models.NewFieldSelector("rateCards"))`)
**ServiceHookRegistry cross-domain lifecycle callbacks** — ServiceHookRegistry[T] is a thread-safe, re-entrancy-preventing hook fan-out (PreCreate/PostCreate/PreUpdate/PostUpdate/PreDelete/PostDelete). Re-entrancy is blocked by a pointer-identity context key (loopKey from fmt.Sprintf('%p', r)) so a hook calling back through the same service does not recurse infinitely. Domain services embed *ServiceHookRegistry[DomainType] and expose RegisterHooks externally. (`registry := models.NewServiceHookRegistry[Customer]()
registry.RegisterHooks(ledgerHook, notificationHook)
// Before delete:
if err := registry.PreDelete(ctx, &customer); err != nil { return err }`)
**CadencedModel for half-open time intervals** — Entities active over [ActiveFrom, ActiveTo) embed CadencedModel. IsActiveAt(t) returns false if ActiveTo is non-nil and not After(t) — entity deleted exactly at t is considered inactive. CadenceList[T] validates sorted, non-overlapping, continuous sequences via GetOverlaps() (O(n) adjacent scan) and IsContinuous() (exact ActiveTo==next.ActiveFrom equality). Use NewSortedCadenceList for auto-sorted input. (`if !entity.CadencedModel.IsActiveAt(clock.Now()) { return nil, models.NewGenericNotFoundError(fmt.Errorf("entity not active")) }`)
**RFC 7807 StatusProblem for HTTP error responses** — NewStatusProblem(ctx, err, statusCode) builds an application/problem+json response. It extracts request-id from Chi middleware context, maps 'context canceled' substring to 408, and suppresses err detail for 500 responses. Extensions map is always non-nil so callers can always assign to it. Never produce 500 StatusProblem with Detail set. (`problem := models.NewStatusProblem(ctx, err, http.StatusBadRequest)
problem.Extensions = models.EncodeValidationIssues(err)
problem.Respond(w)`)
**Validator interface on input types** — Service input types implement models.Validator (Validate() error). Call input.Validate() at service entry points before any adapter interaction. Use models.Validate[T](v, ...ValidatorFunc[T]) to compose multiple validators. Wrap validation failure in NewGenericValidationError. (`if err := input.Validate(); err != nil { return nil, models.NewGenericValidationError(err) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `pkg/models/errors.go` | Defines all generic domain error types and ErrorWithFieldPrefix/ErrorWithComponent wrappers for structured propagation across service layers. | Adding a new error type without an IsXxxError(err) checker or without ensuring GenericErrorEncoder maps it to a status code. |
| `pkg/models/validationissue.go` | ValidationIssue immutable value type; AsValidationIssues traverses error trees extracting ValidationIssue nodes; Clone() sets wraps=self so errors.Is chains work. | WithAttr/WithAttrs panic on nil or non-comparable keys. Do not set wraps field manually — Clone() manages it. |
| `pkg/models/servicehook.go` | Thread-safe generic hook registry with re-entrancy prevention via pointer-identity context key. | init() is lazy via sync.Once; the id field is set on first use. Do not compare registries by value — id depends on pointer identity. |
| `pkg/models/fielddescriptor.go` | FieldDescriptor forms a tree via pkg/treex for nested field paths. WithPrefix/WithExpression return new *FieldDescriptor (copy-on-write). | NewFieldSelectorGroup with all-nil selectors returns nil — always nil-check the result before use. String() and JSONPath() walk leaf nodes only via DFS. |
| `pkg/models/model.go` | ManagedModel/NamespacedModel/ManagedResource base structs embedded by all persistent entities. NewManagedResource converts times to UTC. | IsDeletedAt uses !DeletedAt.After(t) — entity deleted exactly at t is considered deleted. Always store UTC times. |
| `pkg/models/cadencelist.go` | CadenceList[T] helpers validating sorted, non-overlapping, continuous cadence sequences. GetOverlaps is O(n) scanning adjacent pairs only (assumes sorted input). | IsContinuous requires exact ActiveTo == next.ActiveFrom equality — timezone or nanosecond differences produce false negatives. GetOverlaps returns empty slice, not nil, for no overlaps. |
| `pkg/models/annotation.go` | Annotations is map[string]interface{} with Clone (deep copy), Merge (right-wins), GetBool/GetString/GetInt typed accessors. | GetInt accepts float64 whole numbers because JSON numbers deserialize as float64. Non-comparable or fractional float64 returns (0, false). Use Equal() not reflect.DeepEqual for comparison. |

## Anti-Patterns

- Importing from openmeter/* domain packages — pkg/models must remain a leaf with no openmeter/* imports.
- Mutating a ValidationIssue in place — all With* methods clone; direct struct field assignment bypasses the immutability contract and corrupts shared instances.
- Returning plain errors.New() from a service layer where ValidationIssue is expected — AsValidationIssues cannot extract raw errors, only ValidationIssue nodes.
- Using reflect.DeepEqual for Annotations or ValidationIssue equality — use the provided Equal() methods which handle nil and non-comparable values correctly.
- Constructing StatusProblem with Detail set for 500 responses — NewStatusProblem intentionally suppresses detail on 500 to prevent internal leak.

## Decisions

- **ValidationIssue uses a private constructor and copy-on-write With* methods instead of a mutable builder.** — Ensures every ValidationIssue instance is valid at construction (code + message required) and prevents accidental shared mutation when issues are passed through multiple error-wrapping layers.
- **ServiceHookRegistry uses a pointer-identity-based context key (loopKey from fmt.Sprintf('%p', r)) to prevent re-entrant hook invocations.** — Domain hooks (e.g., ledger hook on customer.PostCreate) may call back into the customer service; the context sentinel blocks infinite recursion without requiring hooks to be aware of each other.
- **FieldDescriptor forms a tree (via pkg/treex) rather than a simple dot-delimited string.** — Enables WithPrefix composition (prepend path segments from outer service layer), structured JSONPath generation for API error payloads, and attribute attachment at any node level — all impossible with a flat string.

## Example: Return a structured ValidationIssue with field path and HTTP status from a domain service

```
import (
    "github.com/openmeterio/openmeter/pkg/models"
    "github.com/openmeterio/openmeter/pkg/framework/commonhttp"
    "net/http"
)

func validateAmount(amount float64) error {
    if amount <= 0 {
        return models.NewValidationError("invalid_amount", "amount must be positive").
            WithPathString("lineItems", "amount").
            WithAttr(commonhttp.HTTPStatusCodeAttribute, http.StatusUnprocessableEntity)
    }
    return nil
}
```

<!-- archie:ai-end -->
