# workflow

<!-- archie:ai-start -->

> Public interface package for the subscription workflow layer: defines the Service interface (CreateFromPlan, EditRunning, ChangeToPlan, Restore, AddAddon, ChangeAddonQuantity), its input/output types, AnnotationParser for workflow-scoped annotations, and MapSubscriptionErrors for converting spec errors to HTTP-mappable generic errors. The concrete orchestration lives in workflow/service/.

## Patterns

**MapSubscriptionErrors wraps spec errors** — Every workflow operation wraps spec-level errors through MapSubscriptionErrors before returning — SpecValidationError → GenericValidationError (400), AlignmentError → GenericConflictError (409); all other errors pass through unchanged. (`return subscriptionworkflow.MapSubscriptionErrors(err)`)
**AnnotationParser for workflow annotations** — AnnotationParser.SetUniquePatchID stamps a ULID under the stable AnnotationEditUniqueKey; never write annotation keys by string literal. (`annotations = subscriptionworkflow.AnnotationParser.SetUniquePatchID(annotations)`)
**Input types with Validate()** — AddAddonWorkflowInput and ChangeAddonQuantityWorkflowInput implement Validate(); call it before passing to service methods. (`if err := inp.Validate(); err != nil { return models.NewGenericValidationError(err) }`)
**lockCustomer before any customer-scoped write** — The concrete workflow/service/ implementation acquires a per-customer advisory lock before entering any transaction that creates or modifies subscriptions. (`if err := s.lockCustomer(ctx, customerID); err != nil { return err }`)
**Addon sync via restore+apply diff cycle** — Updating a subscription with active addons always runs a full restore-then-apply diff cycle (syncWithAddons) rather than incremental patching, keeping the addon-extended spec coherent. (`return s.syncWithAddons(ctx, sub, patches)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service interface and all input/output types — single source of truth for the workflow API surface. | EditRunning takes a Timing parameter; a zero-value Timing resolves to clock.Now(). |
| `errors.go` | MapSubscriptionErrors — must be called on every error returned from a workflow operation. | Only SpecValidationError and AlignmentError are remapped; all other errors pass through. |
| `annotations.go` | AnnotationParser with SetUniquePatchID for stamping edit operations. | AnnotationEditUniqueKey is persisted in the DB — keep it stable. |
| `service/service.go` | Concrete Service implementation with transaction.Run, lockCustomer, syncWithAddons, and compile-time interface assertion. | New mutating methods must call lockCustomer before transaction.Run, or concurrent subscription mutations per customer race. |

## Anti-Patterns

- Returning spec errors without wrapping via MapSubscriptionErrors — they won't map to 400/409.
- Writing annotation keys by string literal instead of AnnotationParser.
- Calling s.Service.Update directly without the restore+apply diff cycle when addons are present — always use syncWithAddons.
- Skipping lockCustomer in new mutating workflow methods — concurrent customer mutations race.
- Returning raw fmt.Errorf for validation/conflict/forbidden instead of models.NewGeneric* constructors.

## Decisions

- **Workflow interface separated from subscription.Service.** — Workflow operations are multi-step (lock+tx+addon sync+spec update) composing core service methods; combining them would violate single responsibility and make the core service untestable in isolation.
- **MapSubscriptionErrors lives in the workflow package, not the service package.** — The workflow layer is the HTTP-facing boundary; mapping domain errors to HTTP status codes here keeps the core service pure.
- **Addon sync uses a full restore-then-apply diff cycle.** — Ensures the addon-extended spec is always coherent after any spec update, avoiding the partial state incremental patching could produce.

## Example: Call EditRunning and map spec errors to HTTP status codes

```
import subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"

view, err := h.workflowSvc.EditRunning(ctx, subID, patches, subscription.Timing{Custom: &now})
if err != nil {
    return nil, subscriptionworkflow.MapSubscriptionErrors(err)
}
```

<!-- archie:ai-end -->
