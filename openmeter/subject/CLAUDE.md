# subject

<!-- archie:ai-start -->

> Domain package for subject lifecycle (create, update, get, list, delete) — subjects are the usage measurement keys (analogous to users or devices). Backed by an Ent adapter and exposes ServiceHooks for lifecycle events consumed by entitlement/balanceworker. Primary constraint: all mutations go through the Service interface with validated Input types.

## Patterns

**Service interface with <Verb>Input types** — All mutating operations accept typed input structs (CreateInput, UpdateInput) that implement Validate(). Service methods call input.Validate() before delegating to the adapter. (`type CreateInput struct { Namespace string; Key string; DisplayName *string }; func (i CreateInput) Validate() error { ... }`)
**OptionalNullable[T] for partial updates (legacy, do not replicate)** — UpdateInput fields use OptionalNullable[T]{Value *T; IsSet bool} to distinguish 'not provided' from 'explicitly set to null'. This pattern is unique to this package — new packages use pkg/nullable instead. (`UpdateInput.DisplayName OptionalNullable[string] — IsSet=false means keep existing, IsSet=true+Value=nil means clear.`)
**ServiceHooks registry for cross-domain callbacks** — Subject.Service embeds models.ServiceHooks[Subject], allowing other packages (entitlement/balanceworker) to subscribe to lifecycle events without import cycles. Register hooks in app/common, not inside domain constructors. (`Service.RegisterHooks(hooks SubjectServiceHooks) — called from app/common, not from subject package constructor.`)
**Namespace-scoped operations** — Every operation requires a non-empty Namespace field. GetByIdOrKey supports both ID and key lookup within a namespace. (`GetByIdOrKey(ctx, models.NamespacedID{Namespace: ns, ID: id}) or GetByKey(ctx, ns, key).`)
**Subject has both Id and Key — never conflate them** — Key is the user-facing stable identifier (e.g. user email or device ID); Id is the internal ULID primary key. GetByIdOrKey accepts either form. (`subject.Key is used in entitlement events to reduce payload size (SubjectKey struct); subject.Id is the DB primary key.`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/subject/service.go` | Defines the Service interface, CreateInput, UpdateInput, ListParams, ListSortBy, and the OptionalNullable helper. | OptionalNullable is explicitly marked FIXME as unique to this adapter — do not replicate. New update inputs should use nullable.Nullable from pkg/nullable. |
| `openmeter/subject/adapter.go` | Defines the Adapter interface including entutils.TxCreator embedding — all adapter implementations must provide the full TxCreator triad. | Adapter embeds entutils.TxCreator; concrete implementation in adapter/ sub-package must provide Tx/WithTx/Self. |
| `openmeter/subject/subject.go` | Defines the Subject domain type (Id, Key, Namespace, DisplayName, Metadata) and SubjectKey (key-only payload for events). | StripeCustomerId is deprecated — use customer application entity instead. SubjectKey is used in events to reduce payload size. |

## Anti-Patterns

- Using context.Background() instead of the caller's ctx in service or adapter methods — drops Ent transaction driver and OTel spans.
- Bypassing the Service interface to call adapter methods directly from outside the subject package.
- Importing app/common in subject testutils — build test deps from raw constructors (subjecttestutils.NewTestEnv) to avoid import cycles.
- Adding new update fields as plain pointer types instead of using OptionalNullable or pkg/nullable when partial-null semantics are needed.
- Registering cross-domain hooks inside the subject package constructor — always register in app/common provider functions.

## Decisions

- **OptionalNullable[T] for UpdateInput instead of nullable.Nullable** — Historical decision predating pkg/nullable; the code comment explicitly marks it as unique to this adapter and not to be replicated.
- **ServiceHooks for cross-domain lifecycle callbacks via models.ServiceHooks[Subject]** — Prevents circular imports between subject and entitlement/balanceworker by using the generic hook registry from pkg/models.
- **Service interface at package root (service.go), concrete implementation in service/ sub-package, Ent adapter in adapter/ sub-package** — Standard three-layer pattern used across all openmeter/* domains — keeps persistence separate from orchestration and HTTP translation.

## Example: Creating a subject via Service with validated input

```
import (
    "github.com/openmeterio/openmeter/openmeter/subject"
)

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
