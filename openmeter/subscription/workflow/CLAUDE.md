# workflow

<!-- archie:ai-start -->

> Orchestration layer (package subscriptionworkflow) above the subscription/addon/customer services: defines the Service interface (CreateFromPlan, EditRunning, ChangeToPlan, Restore, AddAddon, ChangeAddonQuantity) and shared helpers. service/ holds the concrete spec-centric, single-transaction implementation; this root holds the interface, input types, error mapping, and edit annotations.

## Patterns

**Spec-centric workflow signatures** — All methods take/return subscription.SubscriptionView, Spec, Patch, Timing or Plan — never raw DB rows. Mutations flow view -> AsSpec -> patch -> Update -> re-read view. (`EditRunning(ctx, subscriptionID models.NamespacedID, customizations []subscription.Patch, timing subscription.Timing) (subscription.SubscriptionView, error)`)
**Domain-error mapping at the boundary** — MapSubscriptionErrors translates SpecValidationError -> GenericValidationError and AlignmentError -> GenericConflictError before returning to callers. (`if sErr, ok := lo.ErrorsAs[*subscription.SpecValidationError](err); ok { return models.NewGenericValidationError(sErr) }`)
**Validate() on workflow inputs** — Input structs (AddAddonWorkflowInput, ChangeAddonQuantityWorkflowInput) implement Validate(); AddAddon requires AddonID and InitialQuantity > 0. (`if i.InitialQuantity <= 0 { return errors.New("initialQuantity must be greater than 0") }`)
**Edit-patch unique annotation stamping** — AnnotationParser.SetUniquePatchID stamps a ulid under subscription.workflow.patchid so edits are idempotently identifiable. (`annotations[AnnotationEditUniqueKey] = ulid.Make().String()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service interface and all workflow input structs (Create/Change/AddAddon/ChangeAddonQuantity). | ChangeToPlan returns both the cancelled current Subscription and the new SubscriptionView — plan change is cancel+create, not in-place. |
| `errors.go` | MapSubscriptionErrors translating spec/alignment errors into client-facing models errors. | Use this at the workflow boundary; don't leak raw SpecValidationError/AlignmentError to handlers. |
| `annotations.go` | AnnotationParser.SetUniquePatchID for edit idempotency. | Allocates the annotations map if nil; do not assume the input map is non-nil. |

## Anti-Patterns

- Persisting or calling underlying services outside a single transaction.Run wrapper — breaks atomicity of multi-step workflows.
- Mutating a SubscriptionView's items directly instead of view.AsSpec() -> spec mutation -> Service.Update.
- Editing a subscription that has addons via EditRunning instead of rejecting with NewGenericForbiddenError.
- Returning raw spec/alignment errors instead of routing through MapSubscriptionErrors.
- Hand-patching addon items instead of building addondiff.Diffable and re-syncing the full before/after addon set.

## Decisions

- **Workflows are spec-centric and each wraps its steps in one transaction.Run.** — Multi-step composition of subscription/addon/customer services must be atomic and validated against the spec model.
- **Plan changes are cancel-current + create-new with linked annotations.** — Avoids fragile in-place mutation of a live subscription and preserves history/linkage.

## Example: Mapping subscription domain errors at the workflow boundary

```
func MapSubscriptionErrors(err error) error {
    if err == nil { return nil }
    if sErr, ok := lo.ErrorsAs[*subscription.SpecValidationError](err); ok {
        return models.NewGenericValidationError(sErr)
    } else if sErr, ok := lo.ErrorsAs[*subscription.AlignmentError](err); ok {
        return models.NewGenericConflictError(sErr)
    }
    return err
}
```

<!-- archie:ai-end -->
