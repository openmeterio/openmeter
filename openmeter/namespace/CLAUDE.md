# namespace

<!-- archie:ai-start -->

> Foundational tenancy package: defines the Manager that fans namespace create/delete across registered Handlers (one per component) plus the namespacedriver seam that resolves the active tenant from request context. Imported by nearly every httpdriver, so it must stay dependency-light.

## Patterns

**Manager fans out to registered Handlers** — Manager.CreateNamespace/DeleteNamespace iterate config.Handlers, calling each Handler.CreateNamespace/DeleteNamespace and joining errors. Components register via RegisterHandler. (`for _, handler := range m.config.Handlers { err := handler.CreateNamespace(ctx, name); ... }`)
**Handler is a two-method per-component interface** — Each component that owns namespace-scoped state implements Handler{CreateNamespace, DeleteNamespace}; an empty name means the default namespace. (`type Handler interface { CreateNamespace(ctx, name) error; DeleteNamespace(ctx, name) error }`)
**Constructor + invariant guards** — NewManager requires a non-empty DefaultNamespace and rejects nil handlers; CreateNamespace rejects empty names; DeleteNamespace refuses to delete the default namespace. (`if name == m.config.DefaultNamespace { return errors.New("cannot delete default namespace") }`)
**Handler list guarded by RWMutex** — config.Handlers is read under m.mu.RLock during create/delete and written under m.mu.Lock in RegisterHandler, so registration is safe to call after construction. (`m.mu.RLock(); defer m.mu.RUnlock()`)
**Errors joined, not short-circuited** — createNamespace/deleteNamespace collect each handler's error and return errors.Join(errs...) so one failing component does not skip the others (resiliency TODO noted in code). (`errs = append(errs, err); ... return errors.Join(errs...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `namespace.go` | Manager, ManagerConfig, Handler interface, and CreateNamespace/DeleteNamespace/CreateDefaultNamespace/RegisterHandler orchestration. | No rollback/retry yet (TODO); a partial failure across handlers leaves namespaces half-created. The default namespace cannot be deleted. |
| `namespace_test.go` | Tests Manager create/delete/register fan-out using an in-memory fakeHandler. | fakeHandler guards its map with a mutex; keep handler implementations concurrency-safe. |

## Anti-Patterns

- Adding domain/DB/auth dependencies to this package — it is a leaf imported by nearly every httpdriver and must avoid import cycles.
- Short-circuiting handler iteration on the first error instead of joining all errors.
- Allowing deletion of the configured DefaultNamespace.
- Reading the active namespace from request body/query rather than from context via the namespacedriver decoder.

## Decisions

- **Namespace creation is orchestrated through per-component Handlers registered on a central Manager.** — Each component (billing, entitlement, etc.) provisions its own namespace-scoped state; the Manager decouples the orchestration from the components.
- **namespacedriver exposes namespace resolution as a one-method NamespaceDecoder interface returning (string, bool).** — Handlers depend on the interface so the source of the namespace (static config today, auth-derived later) can change without touching every httpdriver.

## Example: Fan-out create with joined errors

```
func (m *Manager) createNamespace(ctx context.Context, name string) error {
	var errs []error
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, handler := range m.config.Handlers {
		if err := handler.CreateNamespace(ctx, name); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
```

<!-- archie:ai-end -->
