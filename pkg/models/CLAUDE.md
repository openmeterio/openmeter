# models

<!-- archie:ai-start -->

> Foundational domain-primitive library shared by every openmeter/* and api/* package: typed GenericXxxError sentinels, the immutable ValidationIssue with-chain, ServiceHookRegistry[T] with re-entrancy prevention, RFC 7807 StatusProblem, FieldDescriptor trees, and CadencedModel half-open intervals. Its one hard constraint is zero imports from openmeter/* — it must remain a pure leaf.

## Patterns

**Typed GenericXxxError sentinels** — Wrap every domain error in a NewGenericXxxError(err) constructor and detect via IsGenericXxxError(err) (errors.As). Each type has a constructor + checker pair and maps to one HTTP status in the GenericErrorEncoder. (`return nil, models.NewGenericNotFoundError(fmt.Errorf("customer %s not found", id))`)
**ValidationIssue copy-on-write builder** — Construct via NewValidationIssue/NewValidationError/NewValidationWarning; every With* (WithPathString, WithComponent, WithSeverity, WithAttr) clones and returns a new value. AsValidationIssues(err) extracts only ValidationIssue nodes from a mixed error tree — plain errors.New() leaves are skipped. (`models.NewValidationError("invalid_amount", "must be positive").WithPathString("lineItems", "amount")`)
**FieldDescriptor tree for field paths** — NewFieldSelector(field) is a leaf; NewFieldSelectorGroup(selectors...) groups siblings (returns nil if all selectors nil); WithExpression attaches a filter, WithPrefix prepends a subtree. String()/JSONPath() DFS only leaf nodes via pkg/treex. (`models.NewFieldSelector("phases").WithExpression(models.NewFieldAttrValue("key", "trial"))`)
**ServiceHookRegistry re-entrancy guard** — ServiceHookRegistry[T] fans out PreCreate/PostCreate/PreUpdate/PostUpdate/PreDelete/PostDelete to registered hooks; a pointer-identity context key (loopKey from fmt.Sprintf('%p', r)) blocks infinite recursion when a hook calls back through the same service. Domain services embed *ServiceHookRegistry[T] and expose RegisterHooks. (`registry := models.NewServiceHookRegistry[Customer](); registry.RegisterHooks(ledgerHook)`)
**CadencedModel half-open intervals** — Entities active over [ActiveFrom, ActiveTo) embed CadencedModel; IsActiveAt(t) is false when ActiveTo is non-nil and not After(t). CadenceList[T] validates sorted/non-overlapping/continuous sequences via GetOverlaps (O(n) adjacent scan) and IsContinuous (exact ActiveTo==next.ActiveFrom). (`if !entity.CadencedModel.IsActiveAt(clock.Now()) { return nil, models.NewGenericNotFoundError(...) }`)
**Validator interface at service boundaries** — Input types implement models.Validator (Validate() error). Call input.Validate() at service entry, then wrap failure in NewGenericValidationError. Compose multiple validators via models.Validate[T]. (`if err := input.Validate(); err != nil { return nil, models.NewGenericValidationError(err) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `errors.go` | All Generic*Error types plus ErrorWithFieldPrefix/ErrorWithComponent wrappers for structured cross-boundary propagation. | Adding an error type without a matching IsXxxError checker or without wiring it into GenericErrorEncoder leaves it mapped to 500. |
| `validationissue.go` | Immutable ValidationIssue value; AsValidationIssues traverses error trees; Clone() sets wraps=self so errors.Is chains hold. | Never set the wraps field manually; WithAttr panics on nil/non-comparable keys. |
| `servicehook.go` | Thread-safe generic hook registry with pointer-identity re-entrancy prevention; init() is lazy via sync.Once. | Do not compare registries by value — id depends on pointer identity. |
| `fielddescriptor.go` | FieldDescriptor tree (pkg/treex); WithPrefix/WithExpression are copy-on-write returning new *FieldDescriptor. | NewFieldSelectorGroup with all-nil selectors returns nil — always nil-check before use. |
| `model.go` | ManagedModel/NamespacedModel/ManagedResource base structs embedded by persistent entities; NewManagedResource normalises to UTC. | IsDeletedAt uses !DeletedAt.After(t) — an entity deleted exactly at t is deleted. Always store UTC. |
| `annotation.go` | Annotations map[string]interface{} with Clone (deep), Merge (right-wins), and GetBool/GetString/GetInt typed accessors. | GetInt accepts whole-number float64 (JSON numbers); fractional/out-of-range returns (0,false). Use Equal(), not reflect.DeepEqual. |
| `cadencelist.go` | CadenceList[T] sorted/overlap/continuity validation; GetOverlaps scans adjacent pairs only (assumes sorted input). | IsContinuous requires exact ActiveTo==next.ActiveFrom — nanosecond/timezone drift yields false negatives. |

## Anti-Patterns

- Importing any openmeter/* domain package — pkg/models must stay a leaf.
- Mutating a ValidationIssue or FieldDescriptor in place — all With* methods clone; direct field assignment corrupts shared instances.
- Returning plain errors.New() where ValidationIssue is expected — AsValidationIssues cannot extract raw errors.
- Using reflect.DeepEqual for Annotations/ValidationIssue equality instead of the provided Equal() methods.
- Constructing a 500 StatusProblem with Detail set — NewStatusProblem suppresses detail on 500 to avoid internal leaks.

## Decisions

- **ValidationIssue uses a private constructor plus copy-on-write With* methods rather than a mutable builder.** — Guarantees every instance is valid at construction (code+message) and prevents shared mutation as issues pass through error-wrapping layers.
- **ServiceHookRegistry blocks re-entrancy with a pointer-identity context key.** — A hook (e.g. ledger on customer.PostCreate) may call back into the same service; the sentinel stops infinite recursion without hooks knowing about each other.
- **FieldDescriptor is a tree (pkg/treex), not a dot-delimited string.** — Enables WithPrefix composition from outer layers, structured JSONPath generation for API error payloads, and per-node attribute attachment.

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
