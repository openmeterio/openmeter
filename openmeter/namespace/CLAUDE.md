# namespace

<!-- archie:ai-start -->

> Multi-tenancy infrastructure: Manager fans out CreateNamespace/DeleteNamespace to all registered Handler implementations (ClickHouse streaming, Kafka ingest, Ledger). Handlers are registered dynamically before startup completes; the namespacedriver sub-package provides the NamespaceDecoder abstraction for HTTP-layer namespace resolution.

## Patterns

**Handler registration before CreateDefaultNamespace** — All Handler implementations must be registered via RegisterHandler before CreateDefaultNamespace is called. Handlers registered after this call miss default-namespace provisioning for ClickHouse tables, Kafka topics, and Ledger accounts. (`manager.RegisterHandler(streamingConnector); manager.RegisterHandler(kafkaCollector); manager.CreateDefaultNamespace(ctx)`)
**Fan-out with errors.Join (no short-circuit)** — createNamespace and deleteNamespace iterate all handlers and accumulate errors with errors.Join. A failure in one handler does not prevent others from being called — no rollback on partial failure. (`for _, handler := range m.config.Handlers { if err := handler.CreateNamespace(ctx, name); err != nil { errs = append(errs, err) } }; return errors.Join(errs...)`)
**Default namespace protected from deletion** — DeleteNamespace rejects deletion of the default namespace by comparing name against m.config.DefaultNamespace before any handler call. (`if name == m.config.DefaultNamespace { return errors.New("cannot delete default namespace") }`)
**StaticNamespaceDecoder as a named string type** — namespacedriver.StaticNamespaceDecoder is a named string type — cast a string literal to use it, no struct initialisation required. Used in self-hosted single-tenant deployments. (`var decoder namespacedriver.NamespaceDecoder = namespacedriver.StaticNamespaceDecoder("default")`)
**RWMutex-guarded handler slice for dynamic registration** — RegisterHandler acquires a write lock; createNamespace/deleteNamespace acquire a read lock. Allows registration to race safely with in-flight namespace operations during startup. (`m.mu.Lock(); defer m.mu.Unlock(); m.config.Handlers = append(m.config.Handlers, handler)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `namespace.go` | Full Manager implementation: RWMutex-guarded handler slice, RegisterHandler, CreateNamespace, DeleteNamespace, CreateDefaultNamespace, GetDefaultNamespace, IsManagementDisabled. | No rollback on partial handler failure — a TODO comment documents this. Do not assume atomicity across handlers. Passing nil as a Handler returns an error; callers must check. |
| `namespacedriver/decoder.go` | NamespaceDecoder interface (GetNamespace(r *http.Request) (string, bool)) and StaticNamespaceDecoder named string type. Must stay dependency-free — no domain imports. | GetNamespace returning (true, "") silently scopes queries to an empty namespace. Returning (false, "") signals the caller to reject the request. |

## Anti-Patterns

- Registering a Handler after CreateDefaultNamespace is called — it will miss default namespace provisioning
- Returning (true, "") from NamespaceDecoder.GetNamespace — callers treat empty namespace as valid and misscope queries
- Importing domain packages (billing, customer, entitlement) in namespacedriver — must stay infrastructure-only
- Assuming createNamespace is atomic — failures in one handler do not roll back others
- Adding multi-tenant dynamic namespace resolution to StaticNamespaceDecoder — dynamic lookup belongs in a separate decoder implementation

## Decisions

- **Manager holds a RWMutex-guarded handler slice rather than a fixed set** — Handlers are registered dynamically at startup (after Manager construction) to support optional subsystems like Ledger that depend on the credits.enabled config flag; a fixed constructor-time set would force all handlers to be wired unconditionally.
- **StaticNamespaceDecoder is a named string type, not a struct** — Self-hosted single-tenant deployments always route to one namespace; a plain named string avoids initialisation ceremony while satisfying the NamespaceDecoder interface.

## Example: Registering a new handler that must receive default namespace provisioning

```
// In cmd/server/main.go — before initNamespace()
if err := app.NamespaceManager.RegisterHandler(myNewHandler); err != nil {
    return fmt.Errorf("register handler: %w", err)
}
// CreateDefaultNamespace fans out to all registered handlers including myNewHandler
if err := app.NamespaceManager.CreateDefaultNamespace(ctx); err != nil {
    return fmt.Errorf("create default namespace: %w", err)
}
```

<!-- archie:ai-end -->
