# namespace

<!-- archie:ai-start -->

> Multi-tenancy infrastructure: Manager fans out CreateNamespace/DeleteNamespace to all registered Handler implementations (ClickHouse, Kafka ingest, Ledger). namespacedriver sub-package provides the HTTP-layer NamespaceDecoder abstraction.

## Patterns

**Handler registration before CreateDefaultNamespace** — Handlers must be registered via RegisterHandler before CreateDefaultNamespace is called at server startup, otherwise the handler misses default namespace provisioning. (`manager.RegisterHandler(streamingConnector); manager.RegisterHandler(kafkaCollector); manager.CreateDefaultNamespace(ctx)`)
**Fan-out with errors.Join (no short-circuit)** — createNamespace/deleteNamespace iterate all handlers and accumulate errors with errors.Join. A failure in one handler does not prevent others from being called. (`for _, handler := range m.config.Handlers { if err := handler.CreateNamespace(ctx, name); err != nil { errs = append(errs, err) } }; return errors.Join(errs...)`)
**Default namespace protected from deletion** — DeleteNamespace rejects deletion of the default namespace by comparing name == m.config.DefaultNamespace before any handler call. (`if name == m.config.DefaultNamespace { return errors.New("cannot delete default namespace") }`)
**StaticNamespaceDecoder as plain string type** — namespacedriver.StaticNamespaceDecoder is a named string type, not a struct — cast a string literal to use it. (`var decoder namespacedriver.NamespaceDecoder = namespacedriver.StaticNamespaceDecoder("default")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `namespace.go` | Full Manager implementation including RWMutex-guarded handler list, RegisterHandler, CreateNamespace, DeleteNamespace, CreateDefaultNamespace. | No rollback on partial handler failure — a TODO comment documents this. Do not assume atomicity. |
| `namespacedriver/decoder.go` | NamespaceDecoder interface + StaticNamespaceDecoder. Must stay dependency-free (no domain imports). | GetNamespace returning (empty string, true) silently scopes queries to an unintended tenant. |

## Anti-Patterns

- Registering a Handler after CreateDefaultNamespace is called — it will miss default namespace provisioning
- Returning true with an empty string from NamespaceDecoder.GetNamespace
- Importing domain packages (billing, customer, entitlement) in namespacedriver — it must stay infrastructure-only
- Assuming createNamespace is atomic — failures in one handler do not roll back others
- Passing nil as a Handler to NewManager or RegisterHandler — both return an error; callers must check

## Decisions

- **Manager holds a RWMutex-guarded handler slice rather than fixed set** — Handlers are registered dynamically at startup (after Manager construction) to support optional subsystems like Ledger that depend on credits.enabled config flag.
- **StaticNamespaceDecoder is a string type, not a struct** — Self-hosted deployments always route to one namespace; a plain named string avoids unnecessary initialization ceremony while still satisfying the NamespaceDecoder interface.

<!-- archie:ai-end -->
