# namespace

<!-- archie:ai-start -->

> Multi-tenancy infrastructure: Manager fans out CreateNamespace/DeleteNamespace to all registered Handler implementations (ClickHouse streaming, Kafka ingest, Ledger). Handlers are registered dynamically before startup completes; the namespacedriver/ sub-package provides the NamespaceDecoder abstraction for HTTP-layer namespace resolution.

## Patterns

**Handler registration before CreateDefaultNamespace** — All Handler implementations must be registered via RegisterHandler before CreateDefaultNamespace; handlers registered later miss default-namespace provisioning. (`manager.RegisterHandler(streamingConnector); manager.RegisterHandler(kafkaCollector); manager.CreateDefaultNamespace(ctx)`)
**Fan-out with errors.Join (no short-circuit)** — createNamespace/deleteNamespace iterate all handlers and accumulate errors with errors.Join; a failure in one handler does not stop others and there is no rollback. (`for _, handler := range m.config.Handlers { if err := handler.CreateNamespace(ctx, name); err != nil { errs = append(errs, err) } }; return errors.Join(errs...)`)
**Default namespace protected from deletion** — DeleteNamespace rejects deletion of the default namespace before any handler call. (`if name == m.config.DefaultNamespace { return errors.New("cannot delete default namespace") }`)
**RWMutex-guarded handler slice** — RegisterHandler takes a write lock; createNamespace/deleteNamespace take a read lock, allowing registration to race safely with in-flight operations during startup. (`m.mu.Lock(); defer m.mu.Unlock(); m.config.Handlers = append(m.config.Handlers, handler)`)
**StaticNamespaceDecoder as a named string type** — namespacedriver.StaticNamespaceDecoder is a named string type — cast a string literal, no struct init. Used in self-hosted single-tenant deployments. (`var decoder namespacedriver.NamespaceDecoder = namespacedriver.StaticNamespaceDecoder("default")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `namespace.go` | Full Manager: RWMutex-guarded handler slice, RegisterHandler, Create/Delete/CreateDefaultNamespace, GetDefaultNamespace, IsManagementDisabled. | No rollback on partial handler failure (TODO documented) — do not assume atomicity. NewManager and RegisterHandler reject nil handlers; callers must check the error. |
| `namespacedriver/decoder.go` | NamespaceDecoder interface (GetNamespace(r) (string, bool)) and StaticNamespaceDecoder named string type; dependency-free. | Returning ("", true) silently scopes queries to an empty namespace; ("", false) signals the caller to reject the request. No domain imports. |

## Anti-Patterns

- Registering a Handler after CreateDefaultNamespace is called — it will miss default namespace provisioning
- Returning (true, "") from NamespaceDecoder.GetNamespace — callers treat an empty namespace as valid and misscope queries
- Importing domain packages (billing, customer, entitlement) in namespacedriver — must stay infrastructure-only
- Assuming createNamespace is atomic — failures in one handler do not roll back others
- Adding multi-tenant dynamic namespace resolution to StaticNamespaceDecoder — dynamic lookup belongs in a separate decoder

## Decisions

- **Manager holds a RWMutex-guarded handler slice rather than a fixed constructor-time set** — Handlers are registered dynamically at startup to support optional subsystems like Ledger that depend on credits.enabled; a fixed set would force all handlers to be wired unconditionally.
- **StaticNamespaceDecoder is a named string type, not a struct** — Self-hosted single-tenant deployments always route to one namespace; a plain named string avoids initialisation ceremony.

<!-- archie:ai-end -->
