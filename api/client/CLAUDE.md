# client

<!-- archie:ai-start -->

> Structural root for the published, multi-language OpenMeter client SDKs. Every sub-package here is generated from the TypeSpec/OpenAPI contract in api/spec and is consumed by external users; only the Go SDK is imported by the Go backend itself (e.g. e2e, quickstart).

## Patterns

**Split by language, generated per language** — Each child owns one SDK language with its own generator/toolchain: go/ (oapi-codegen via codegen.yaml), javascript/ (orval via orval.config.ts), python/ (TypeSpec http-client-python). node/ and web/ are README-only stubs. (`api/client/go/codegen.yaml drives oapi-codegen; api/client/javascript/orval.config.ts drives orval`)
**Hand-edited only at the seams** — Generated code (client.gen.go, javascript/src/) is never edited; thin hand-written wrappers add auth/ergonomics. In go/, client.go adds NewAuthClient / IngestEvent helpers over the generated Client. (`api/client/go/client.go: NewAuthClientWithResponses and IngestEventBatch wrap generated *Client methods`)
**Generation source is api/spec, not these folders** — All client code derives from the OpenAPI specs produced by api/spec; regenerate via `make gen-api` at the repo root. Editing a child's generated output is always wrong. (`go/client.go carries a //go:generate oapi-codegen directive consuming ../../openapi.cloud.yaml`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `go/client.gen.go` | oapi-codegen output: full v1/cloud Client, ClientWithResponses, and request/response types (~1.7MB). | DO NOT EDIT; regenerate via the //go:generate directive in client.go (make gen-api) |
| `go/client.go` | Hand-written auth + event-ingest convenience wrappers over the generated client. | Only place for Go-side ergonomics; keep wrappers as thin pass-throughs to generated methods |
| `javascript/orval.config.ts` | orval generator config for the TypeScript SDK (output under src/). | src/ is generated from the spec; change generation here, not the src output |

## Anti-Patterns

- Editing any generated artifact (go/client.gen.go, javascript/src/, python/openmeter/) instead of the api/spec .tsp source
- Importing a non-Go client (javascript/python) from the Go backend
- Adding language-SDK logic outside the per-language child folder

## Decisions

- **One generated SDK per language, each with its own native generator/toolchain.** — Idiomatic per-language clients beat one forced abstraction; all stay in sync because they share a single OpenAPI source of truth.

<!-- archie:ai-end -->
