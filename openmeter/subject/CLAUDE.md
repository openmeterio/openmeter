# subject

<!-- archie:ai-start -->

> Domain package for subject lifecycle (create, update, get, list, delete) — subjects are the usage measurement keys (analogous to users or devices). Backed by an Ent adapter and exposes ServiceHooks for lifecycle events consumed by entitlement/balanceworker.

## Patterns

**Service interface with <Verb>Input types** — All mutating operations accept typed input structs (CreateInput, UpdateInput) that implement Validate(). Service methods call input.Validate() before delegating to the adapter. (`type CreateInput struct { Namespace string; Key string; DisplayName *string; ... }; func (i CreateInput) Validate() error { ... }`)
**OptionalNullable[T] for partial updates** — UpdateInput fields use OptionalNullable[T]{Value *T; IsSet bool} to distinguish between 'not provided' and 'explicitly set to null'. This pattern is unique to this package — do not replicate in new packages. (`UpdateInput.DisplayName OptionalNullable[string] — IsSet=false means keep existing, IsSet=true+Value=nil means clear.`)
**ServiceHooks registry for cross-domain callbacks** — Subject.Service exposes RegisterHooks to allow other packages (entitlement/balanceworker) to subscribe to lifecycle events without creating import cycles. Uses the generic models.ServiceHooks[T] registry. (`Service.RegisterHooks(hooks SubjectServiceHooks) allows balanceworker to react to subject deletion.`)
**Namespace-scoped operations** — Every operation requires a non-empty Namespace field in its input. GetByIdOrKey supports both ID and key lookup within a namespace — always prefer GetByIdOrKey over separate Get methods when the caller may have either identifier. (`GetByIdOrKey(ctx, models.NamespacedID{Namespace: ns, ID: id}) or GetByKey(ctx, ns, key).`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/subject/service.go` | Defines the Service interface and all Input/Output types including the OptionalNullable helper and ListSortBy enum. | OptionalNullable is explicitly marked as a unique pattern not to be reused elsewhere; new update inputs should use nullable.Nullable from pkg/nullable instead. |
| `openmeter/subject/subject.go` | Defines the Subject domain type embedding models.ManagedModel with Namespace, Id, Key, DisplayName, Metadata fields. | Subject has both Id and Key — never conflate them; Key is the user-facing stable identifier, Id is the internal ULID. |

## Anti-Patterns

- Using context.Background() instead of the caller's ctx in service or adapter methods.
- Bypassing the Service interface to call adapter methods directly from outside the package.
- Importing app/common in subject testutils — build test deps from raw constructors to avoid import cycles.
- Adding new update fields as plain pointer types instead of OptionalNullable when partial-null semantics are needed (though note this pattern is discouraged for new code).

## Decisions

- **OptionalNullable[T] for UpdateInput instead of nullable.Nullable** — Historical decision predating the pkg/nullable package; the code comment explicitly marks it as unique to this adapter and not to be replicated.
- **ServiceHooks for cross-domain lifecycle callbacks** — Prevents circular imports between subject and entitlement/balanceworker by using the generic hook registry pattern from pkg/models.

<!-- archie:ai-end -->
