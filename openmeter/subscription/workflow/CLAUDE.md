# workflow

<!-- archie:ai-start -->

> Public interface package for the subscription workflow layer. Defines the Service interface (CreateFromPlan, EditRunning, ChangeToPlan, Restore, AddAddon, ChangeAddonQuantity), input/output types, AnnotationParser for workflow-scoped annotations, and MapSubscriptionErrors for converting spec validation errors to HTTP-mappable generic errors. The concrete implementation lives in workflow/service/.

## Patterns

**MapSubscriptionErrors wraps spec errors** — All workflow operations wrap spec-level errors through MapSubscriptionErrors before returning so SpecValidationError becomes GenericValidationError (400) and AlignmentError becomes GenericConflictError (409). (`return subscriptionworkflow.MapSubscriptionErrors(err)`)
**AnnotationParser for workflow annotations** — subscriptionworkflow.AnnotationParser.SetUniquePatchID stamps a ULID on annotations to identify an edit operation. Always use AnnotationParser instead of writing annotation keys by string literal. (`annotations = subscriptionworkflow.AnnotationParser.SetUniquePatchID(annotations)`)
**Input types with Validate()** — AddAddonWorkflowInput and ChangeAddonQuantityWorkflowInput implement Validate(). Call Validate() before passing to service methods. (`if err := inp.Validate(); err != nil { return models.NewGenericValidationError(err) }`)
**EditRunning accepts timing parameter** — Service.EditRunning takes a subscription.Timing parameter (as of the updated interface) — the concrete implementation in workflow/service/ uses this to resolve the effective edit timestamp. (`svc.EditRunning(ctx, subID, patches, subscription.Timing{Custom: &now})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service interface and all input/output types. Single source of truth for the workflow API surface. | EditRunning signature includes timing — callers that omit it will use zero-value Timing which resolves to clock.Now(). |
| `errors.go` | MapSubscriptionErrors — must be called on every error returned from a workflow operation. | Only SpecValidationError and AlignmentError are remapped; all other errors pass through unchanged. |
| `annotations.go` | AnnotationParser with SetUniquePatchID for stamping edit operations. | Key constant AnnotationEditUniqueKey must stay stable — it is persisted in the DB. |

## Anti-Patterns

- Returning spec errors without wrapping via MapSubscriptionErrors — SpecValidationError will not map to 400 and AlignmentError will not map to 409.
- Writing annotation keys by string literal instead of using AnnotationParser.
- Implementing workflow operations directly on subscription.Service instead of through the workflow layer — workflow operations require locking, transactions, and addon sync that the core service does not provide.

## Decisions

- **Workflow interface separated from subscription.Service** — Workflow operations are multi-step (lock+tx+addon sync+spec update) and compose core service methods; combining them would violate single responsibility and make the core service untestable in isolation.
- **MapSubscriptionErrors in workflow package rather than service package** — The workflow layer is the HTTP-facing boundary; mapping domain errors to HTTP status codes at this boundary keeps the core service pure.

## Example: Call EditRunning and map spec errors to HTTP status codes

```
import subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"

view, err := h.workflowSvc.EditRunning(ctx, subID, patches, subscription.Timing{Custom: &now})
if err != nil {
    return nil, subscriptionworkflow.MapSubscriptionErrors(err)
}
```

<!-- archie:ai-end -->
