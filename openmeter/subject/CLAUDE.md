# subject

<!-- archie:ai-start -->

> Domain package for subject lifecycle — subjects are the usage-measurement keys (analogous to users or devices), each with a stable user-facing Key and an internal ULID Id. Follows the standard three-layer split (Service/Adapter interfaces at the root, service/ implementation, adapter/ Ent persistence, httphandler/ v1 HTTP) and exposes ServiceHooks so entitlement/balanceworker can react to lifecycle events without import cycles.

## Patterns

**Service interface at root, concrete service in service/, Ent adapter in adapter/** — service.go and adapter.go at the package root define the Service and Adapter interfaces (Adapter embeds entutils.TxCreator); service/service.go implements orchestration, adapter/ implements persistence, httphandler/ does HTTP translation. (`type Service interface { models.ServiceHooks[Subject]; Create/Update/GetById/GetByKey/GetByIdOrKey/List/Delete }`)
**Validated <Verb>Input types before adapter delegation** — Every mutating Service method accepts a typed Input (CreateInput, UpdateInput) implementing Validate(); validation runs before the adapter is touched. Namespace is mandatory on every operation. (`input := subject.CreateInput{Namespace: ns, Key: "user@x.com"}; input.Validate(); svc.Create(ctx, input)`)
**Writes wrapped in transaction.Run, hooks fired inside the boundary** — service/ wraps Create/Update/Delete in transaction.Run/RunWithNoValue so adapter writes and ServiceHookRegistry fan-out commit or roll back atomically; read methods bypass transactions. The registry is embedded as a value, not an interface parameter. (`Create runs adapter write + ServiceHookRegistry PostCreate inside one transaction.Run closure.`)
**ServiceHooks for cross-domain callbacks, registered in app/common** — Subject.Service embeds models.ServiceHooks[Subject]; the service/hooks child bridges customer lifecycle to subject provisioning via a provisioner struct. Hooks are registered in app/common provider functions, never in the subject constructor. (`Service.RegisterHooks(...) called from app/common; hooks/ delegates logic to a provisioner.`)
**Soft-delete + OptionalNullable patch (legacy, do not replicate)** — Adapter list/get filters by DeletedAtIsNil/DeletedAtGTE; UpdateInput uses OptionalNullable[T]{Value,IsSet} to separate 'unset' from 'set-to-null'. This OptionalNullable pattern is explicitly FIXME'd as unique to subject — new code uses pkg/nullable. (`UpdateInput.DisplayName OptionalNullable[string] — IsSet=false keeps existing; IsSet=true+Value=nil clears.`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service interface, CreateInput/UpdateInput/ListParams/ListSortBy, and the OptionalNullable helper. | OptionalNullable is FIXME-marked unique to this package; new inputs use pkg/nullable. |
| `adapter.go` | Adapter interface embedding entutils.TxCreator — implementations must provide the full Tx/WithTx/Self triad. | GetByIdOrKey requires either id or key; honor the ctx-carried transaction. |
| `subject.go` | Subject domain type (Id, Key, Namespace, DisplayName, Metadata) and SubjectKey (key-only event payload). | StripeCustomerId is deprecated — use customer application entity; SubjectKey trims event payload size. |
| `service/ (child)` | Concrete Service: transaction-wrapped writes + ServiceHookRegistry fan-out; hooks/ bridges customer events. | Firing hooks after transaction.Run returns breaks atomicity. |
| `adapter/ (child)` | Ent persistence with TransactingRepo on every method, soft-delete, Ent→models.Generic*Error mapping. | Calling tx.db.* outside a TransactingRepo closure; missing the soft-delete filter. |
| `httphandler/ (child)` | v1 HTTP via httptransport (package httpdriver despite the folder name). | Skipping httptransport.WithOperationName; doing create-or-update logic outside the operation closure. |

## Anti-Patterns

- Using context.Background()/TODO() instead of the caller ctx in service or adapter methods — drops the Ent tx driver and OTel spans.
- Bypassing the Service interface to call adapter methods directly from outside the subject package.
- Importing app/common from subject testutils — build deps from raw constructors (subjecttestutils.NewTestEnv) to avoid import cycles.
- Adding new update fields as plain pointers instead of OptionalNullable/pkg/nullable when partial-null semantics are needed.
- Registering cross-domain hooks inside the subject constructor instead of in app/common provider functions.

## Decisions

- **Transactions wrap both adapter writes and hook execution.** — Hook side-effects (e.g. provisioning) stay atomic with the DB write and roll back together on failure.
- **ServiceHooks via models.ServiceHooks[Subject] for lifecycle callbacks.** — Lets entitlement/balanceworker subscribe to subject events without a circular import.
- **OptionalNullable[T] retained for UpdateInput.** — Historical, predates pkg/nullable; explicitly marked not to be replicated.

## Example: Creating a subject via the Service with validated input

```
import "github.com/openmeterio/openmeter/openmeter/subject"

input := subject.CreateInput{
    Namespace:   ns,
    Key:         "user@example.com",
    DisplayName: lo.ToPtr("Alice"),
}
if err := input.Validate(); err != nil {
    return nil, err
}
result, err := svc.Create(ctx, input)
```

<!-- archie:ai-end -->
