# transport

<!-- archie:ai-start -->

> Structural namespace owning the codebase's transport-layer handler primitives. Its sole child, httptransport, supplies the generic decode->operate->encode HTTP handler abstraction that every v3 handler and most openmeter/*/httpdriver packages build on (47 in-edges); the constraint is that everything here stays domain-agnostic.

## Patterns

**Transport primitives live one level down** — This folder has no direct source files; all handler machinery lives in the httptransport sub-package. New transport abstractions (e.g. a future grpctransport) belong as sibling sub-packages, not as files at this level. (`pkg/framework/transport/httptransport/{handler.go,argshandler.go,options.go}`)
**Domain-agnostic transport layer** — Code under this tree depends only on framework primitives (pkg/framework/operation, pkg/framework/commonhttp, pkg/contextx, pkg/models) and the encoder subpackage, never on openmeter/* domain packages, so it stays a reusable foundation. (`httptransport wraps an operation.Operation; it never imports billing/customer/etc.`)

## Anti-Patterns

- Adding domain-specific handler code directly under transport/ instead of inside httptransport or a new transport sub-package — pollutes the foundational layer.
- Importing openmeter/* domain packages from anything in this tree — the transport primitive must remain decoupled from concrete domains.

## Decisions

- **Transport is a structural namespace with one concrete child (httptransport) rather than a flat package.** — Leaves room for additional transports (gRPC, etc.) as siblings while keeping the generic HTTP decode->operate->encode handler isolated and reusable across v3 handlers and httpdriver packages.

<!-- archie:ai-end -->
