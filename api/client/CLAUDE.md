# client

<!-- archie:ai-start -->

> Public SDK surface for OpenMeter — wires generated and thin hand-authored client code into stable, externally-importable packages for Go (api/client/go), the @openmeter/sdk npm package (api/client/javascript), and Python (api/client/python). All API types and operations originate from generated artefacts produced by make gen-api off the TypeSpec source; this folder only adds ergonomic wrappers and auth helpers. api/client/node and api/client/web are deprecated tombstones.

## Patterns

**Generated-first, wrapper-second** — Every API type, request/response struct, and operation method lives in a generated file (client.gen.go, src/client/schemas.ts, openmeter/_generated/). Hand-authored files add only ergonomic wrappers and auth helpers — never new types. (`client.go adds WithAPIKey() and ergonomic List* helpers atop the generated ClientWithResponses; never adds new types.`)
**Regeneration via make gen-api only** — client.gen.go, src/client/schemas.ts, src/zod/index.ts, and openmeter/_generated/ are overwritten on every make gen-api run; manual edits do not survive. Each language has its own codegen entry: codegen.yaml (oapi-codegen, Go), scripts/generate.ts (openapi-typescript + orval, JS), @typespec/http-client-python (Python). (`make gen-api -> oapi-codegen (Go) + openapi-typescript/orval (JS) + http-client-python (Python)`)
**Auth injected via RequestEditorFn / ClientOption, not embedded** — Bearer tokens and credentials are injected through the option chain (Go RequestEditorFn/ClientOption; Python _patch_sdk() subclass) — never hardcoded inside generated operation methods. (`func WithAPIKey(token string) ClientOption // wraps the generated client with a bearer RequestEditorFn`)
**Externally importable — no app-internal imports** — Go and Python SDKs must remain importable without the full monorepo; do not import app-internal packages beyond those already referenced. Tests must use httptest.NewServer mocks, never live HTTP. (`client_test.go spins httptest.NewServer; client.go imports only public/generated packages.`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `api/client/go/client.gen.go` | Generated Go SDK (oapi-codegen output from api/openapi.cloud.yaml). Never edit. | Enum types carry Valid(); the always-prefix-enum-values codegen option is set in codegen.yaml — overwritten every make gen-api. |
| `api/client/go/client.go` | Hand-authored ergonomic wrappers and auth helpers atop client.gen.go. | Must not import app-internal monorepo packages and must stay externally importable; add wrappers here, never new API types. |
| `api/client/javascript/package.json` | Declares the four named sub-package exports (default, /portal, /react, /zod) and the dual ESM/CJS duel build. | New resource classes must also be registered as public fields on the root OpenMeter class in src/client/index.ts; portal export is intentionally scoped to portal-token operations. |
| `api/client/javascript/patches/openapi-typescript.patch` | Single consolidated patch applied to openapi-typescript dist/ at install time. | pnpm patchedDependencies expects exactly one patch file per package; adding a second breaks install; patch dist/ not src/. |
| `api/client/python/openmeter/_client.py` | Hand-authored Python Client subclassing generated OpenMeterClient; _patch_sdk() called at module end. | All operations must come from openmeter/_generated/; never define new operation methods here; never commit _version.py/_commit.py (written transiently by release.sh). |
| `api/client/node/README.md` | Tombstone — Node SDK deprecated in favour of api/client/javascript. | Do not add code here. |
| `api/client/web/README.md` | Tombstone — Web SDK deprecated in favour of api/client/javascript. | Do not add code here. |

## Anti-Patterns

- Manually editing any *.gen.go, src/client/schemas.ts, src/zod/index.ts, or openmeter/_generated/ file — overwritten by make gen-api.
- Adding new API types or request/response structs to hand-authored wrapper files instead of to the TypeSpec source under api/spec/.
- Importing app-internal monorepo packages into the Go or Python SDK — they must remain externally importable.
- Adding new SDK code to api/client/node or api/client/web — both are deprecated tombstones pointing to api/client/javascript.
- Writing Go tests that make live HTTP calls instead of httptest.NewServer, or running scripts/generate.ts from outside the JS package root.

## Decisions

- **Go SDK is generated from api/openapi.cloud.yaml, not api/openapi.yaml.** — The cloud spec includes cloud-specific auth and endpoint variants; SDK consumers target the hosted service.
- **Python Client subclasses the generated OpenMeterClient rather than wrapping it.** — Lets _patch_sdk() augment generated methods while keeping the public surface identical to the generated class.
- **Four named JS sub-package exports (default, /portal, /react, /zod) instead of a single entry point.** — Enables tree-shaking and keeps the portal client scoped strictly to portal-token operations.

<!-- archie:ai-end -->
