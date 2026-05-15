# workflow

<!-- archie:ai-start -->

> Public interface package for the subscription workflow layer. Defines the Service interface (CreateFromPlan, EditRunning, ChangeToPlan, Restore, AddAddon, ChangeAddonQuantity), input/output types, AnnotationParser for workflow-scoped annotations, and MapSubscriptionErrors for converting spec validation errors to HTTP-mappable generic errors. The concrete implementation lives in workflow/service/.

## Patterns

**MapSubscriptionErrors wraps spec errors** — All workflow operations must wrap spec-level errors through MapSubscriptionErrors before returning — SpecValidationError becomes GenericValidationError (400) and AlignmentError becomes GenericConflictError (409). (`return subscriptionworkflow.MapSubscriptionErrors(err)`)
**AnnotationParser for workflow annotations** — subscriptionworkflow.AnnotationParser.SetUniquePatchID stamps a ULID on annotations to identify an edit operation. Always use AnnotationParser instead of writing annotation keys by string literal. (`annotations = subscriptionworkflow.AnnotationParser.SetUniquePatchID(annotations)`)
**Input types with Validate()** — AddAddonWorkflowInput and ChangeAddonQuantityWorkflowInput implement Validate(). Call Validate() before passing to service methods. (`if err := inp.Validate(); err != nil { return models.NewGenericValidationError(err) }`)
**lockCustomer before any customer-scoped write** — The concrete workflow/service/ implementation acquires a per-customer advisory lock before entering any transaction that creates or modifies subscriptions. (`if err := s.lockCustomer(ctx, customerID); err != nil { return err }`)
**Addon sync via restore+apply diff cycle** — When updating a subscription with active addons, the workflow service always runs a full restore-then-apply diff cycle (syncWithAddons) rather than incremental patching to keep addon-extended spec in sync. (`return s.syncWithAddons(ctx, sub, patches)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service interface and all input/output types. Single source of truth for the workflow API surface. | EditRunning signature includes a Timing parameter — callers that omit it use zero-value Timing which resolves to clock.Now(). |
| `errors.go` | MapSubscriptionErrors — must be called on every error returned from a workflow operation. | Only SpecValidationError and AlignmentError are remapped; all other errors pass through unchanged. |
| `annotations.go` | AnnotationParser with SetUniquePatchID for stamping edit operations. | AnnotationEditUniqueKey constant must stay stable — it is persisted in the DB. |
| `workflow/service/service.go` | Concrete Service implementation with transaction.Run, lockCustomer, syncWithAddons, and compile-time interface assertion. | New mutating methods must call lockCustomer before entering transaction.Run; skipping it causes concurrent subscription races per customer. |

## Anti-Patterns

- Returning spec errors without wrapping via MapSubscriptionErrors — SpecValidationError will not map to 400 and AlignmentError will not map to 409.
- Writing annotation keys by string literal instead of using AnnotationParser.
- Calling s.Service.Update directly without the restore+apply diff cycle when addons are present — always use syncWithAddons for addon-aware spec updates.
- Skipping lockCustomer in new mutating workflow methods — concurrent subscription mutations for the same customer produce race conditions.
- Returning raw fmt.Errorf for validation, conflict, or forbidden errors — always use models.NewGenericValidationError/NewGenericConflictError/NewGenericForbiddenError.

## Decisions

- **Workflow interface separated from subscription.Service** — Workflow operations are multi-step (lock+tx+addon sync+spec update) and compose core service methods; combining them would violate single responsibility and make the core service untestable in isolation.
- **MapSubscriptionErrors in workflow package rather than service package** — The workflow layer is the HTTP-facing boundary; mapping domain errors to HTTP status codes at this boundary keeps the core service pure.
- **Addon sync uses full restore-then-apply diff cycle** — Ensures the addon-extended spec is always coherent after any spec update, avoiding partial or inconsistent addon state that incremental patching could produce.

## Example: Call EditRunning and map spec errors to HTTP status codes

```
import subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"

view, err := h.workflowSvc.EditRunning(ctx, subID, patches, subscription.Timing{Custom: &now})
if err != nil {
    return nil, subscriptionworkflow.MapSubscriptionErrors(err)
}
```

<!-- archie:ai-end -->
