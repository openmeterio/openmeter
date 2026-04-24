# transport

<!-- archie:ai-start -->

> Organisational folder owning the HTTP transport layer for all domain httpdriver packages. Its sole child (httptransport) provides the single generic Handler[Request,Response] pipeline that every v1 and v3 endpoint in the monorepo is built on — nothing else lives here.

## Patterns

**Child-delegated architecture** — All source code lives in pkg/framework/transport/httptransport; this folder is a namespace wrapper only. New transport sub-packages (e.g. grpctransport) would sit here as siblings. (`pkg/framework/transport/httptransport/handler.go — the only current child`)

## Anti-Patterns

- Placing source files directly in pkg/framework/transport/ — all code belongs in named sub-packages
- Adding a second child package that duplicates the decode/operate/encode pipeline instead of reusing httptransport.Handler
- Importing pkg/framework/transport directly (no Go files here); always import the sub-package

## Decisions

- **Single child package rather than a flat file set** — Keeps the transport abstraction extensible (future grpc/websocket siblings) while letting all current consumers import a stable sub-package path (httptransport) without churn.

<!-- archie:ai-end -->
