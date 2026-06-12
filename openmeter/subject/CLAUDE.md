# subject

<!-- archie:ai-start -->

> Subject domain: subjects are usage-attribution identities (a key plus optional display name / stripe customer id) kept loosely in sync with customers. Root files define the Subject model, the Adapter and Service interfaces, and the OptionalNullable update helper. service/ is the only write path (transactional + hooks), adapter/ is Ent-backed, httphandler/ is HTTP transport, testutils/ wires a real Postgres harness.

## Patterns

**Adapter/Service split with TxCreator** — Adapter embeds entutils.TxCreator and exposes raw CRUD/List; Service (in service/) wraps every mutation in transaction.Run and fires models.ServiceHooks[Subject]. The Service interface embeds models.ServiceHooks[Subject]. (`type Service interface { models.ServiceHooks[Subject]; Create(...); Update(...); Delete(...) }`)
**Input struct Validate()** — CreateInput/UpdateInput/GetSubjectAdapterInput each have Validate() that accumulates into errs and returns errors.Join; the service wraps these as models.NewGenericValidationError. (`func (i CreateInput) Validate() error { ... if len(errs)>0 { return errors.Join(errs...) } }`)
**OptionalNullable for tri-state updates** — UpdateInput uses OptionalNullable[T]{Value, IsSet} to distinguish unset vs explicit-null vs value; the adapter must honor IsSet rather than calling SetX unconditionally. The file comment marks this pattern as intentionally local to this adapter. (`DisplayName OptionalNullable[string]`)
**Key-only event payload** — SubjectKey is a minimal {Key} projection used in entitlement events to shrink payload size. (`type SubjectKey struct { Key string }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `subject.go` | Subject domain model (ManagedModel + namespace/key/displayName/metadata/stripeCustomerId) and SubjectKey | StripeCustomerId is deprecated in favor of the customer application entity; key is the only required field in Validate() |
| `service.go` | Service interface, ListParams/ListSortBy, CreateInput/UpdateInput, OptionalNullable helper | OptionalNullable is flagged 'unique to this adapter, should not be reused'; updates depend on IsSet semantics |
| `adapter.go` | Adapter interface (CRUD + GetByIdOrKey/GetByKey/GetById/List) embedding entutils.TxCreator | GetByIdOrKey is a convenience method — prefer GetById/GetByKey; all reads are namespace-scoped |

## Anti-Patterns

- Writing through the Adapter directly instead of the Service (skips transaction.Run, hooks, and validation)
- On Update, calling SetX unconditionally instead of branching on OptionalNullable.IsSet (silently nulls or overwrites fields)
- Hard-deleting subjects or cascading into entitlements — Delete is soft and entitlements outlive the subject
- Cross-importing customer/entitlement services into the Service — it depends only on subject.Adapter plus a hook registry
- Constructing Service via a struct literal instead of New, or passing a nil adapter

## Decisions

- **Service depends only on subject.Adapter and a hook registry, not on customer/entitlement services** — Subjects are an attribution identity; keeping the service dependency-light avoids cycles and lets customer-sync run as a hook rather than an inline cross-service call
- **Subject delete is soft and never touches entitlements** — Usage data and entitlements must survive subject removal for billing/audit

## Example: Adapter interface embedding TxCreator with namespaced reads

```
type Adapter interface {
	Create(ctx context.Context, input CreateInput) (Subject, error)
	Update(ctx context.Context, input UpdateInput) (Subject, error)
	GetByKey(ctx context.Context, key models.NamespacedKey) (Subject, error)
	List(ctx context.Context, namespace string, params ListParams) (pagination.Result[Subject], error)
	Delete(ctx context.Context, id models.NamespacedID) error
	entutils.TxCreator
}
```

<!-- archie:ai-end -->
