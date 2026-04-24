# client

<!-- archie:ai-start -->

> Public SDK surface for OpenMeter — organises generated and hand-authored client code for Go, JavaScript (npm @openmeter/sdk), and Python. The primary constraint is that all business logic and API types originate from generated artefacts; this folder only wires them into stable, externally-importable packages.

## Patterns

**Generated-first, wrapper-second** — All API types, request/response structs, and operation methods live in generated files (client.gen.go, src/client/schemas.ts, openmeter/_generated/). Hand-authored files (client.go, src/client/index.ts, openmeter/_client.py) add ergonomic wrappers and auth helpers only. (`client.go adds WithAPIKey() and ergonomic List* helpers atop the generated ClientWithResponses — never adds new types.`)
**Regeneration via make gen-api only** — client.gen.go, src/client/schemas.ts, and openmeter/_generated/ are overwritten on every `make gen-api` run. No manual edits survive. (`codegen.yaml drives oapi-codegen for Go; scripts/generate.ts drives openapi-typescript + orval for JS.`)
**Auth via RequestEditorFn / ClientOption (Go)** — Authentication tokens are injected via the RequestEditorFn / ClientOption pattern in client.go, never embedded inside generated operation methods. (`WithAPIKey(token string) ClientOption wraps the generated client with a bearer-token RequestEditorFn.`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `api/client/go/client.gen.go` | Generated Go SDK — oapi-codegen output from api/openapi.cloud.yaml. Never edit. | Enum types carry Valid(); always-prefix-enum-values codegen option is set in codegen.yaml. |
| `api/client/go/client.go` | Hand-authored ergonomic wrappers and auth helpers on top of client.gen.go. | Must not import app-internal monorepo packages; must remain externally importable. |
| `api/client/javascript/package.json` | Declares four named sub-package exports (default, /portal, /react, /zod) and the dual ESM/CJS duel build. | New resource classes must also be registered as public fields on the root OpenMeter class in src/client/index.ts. |
| `api/client/javascript/patches/openapi-typescript.patch` | Single consolidated patch applied to openapi-typescript dist/ at install time. | pnpm patchedDependencies expects exactly one patch file; adding a second breaks install. |
| `api/client/python/openmeter/_client.py` | Hand-authored Python client subclassing generated OpenMeterClient; _patch_sdk() called at module end. | All operations must come from openmeter/_generated/; never define new operation methods here. |
| `api/client/node/README.md` | Tombstone — Node SDK deprecated in favour of api/client/javascript. | Do not add code here. |
| `api/client/web/README.md` | Tombstone — Web SDK deprecated in favour of api/client/javascript. | Do not add code here. |

## Anti-Patterns

- Manually editing any *.gen.go, src/client/schemas.ts, or openmeter/_generated/ file — overwritten by make gen-api
- Adding new API types or request/response structs to the hand-authored wrapper files instead of api/spec/ TypeSpec
- Importing app-internal monorepo packages into Go or Python SDK — must remain externally importable
- Adding new SDK code to api/client/node or api/client/web — both are deprecated tombstones

## Decisions

- **Go SDK generated from api/openapi.cloud.yaml (not openapi.yaml)** — Cloud spec includes cloud-specific auth and endpoint variants; SDK consumers target the hosted service.
- **Python client subclasses generated OpenMeterClient rather than wrapping it** — Allows _patch_sdk() to augment generated methods cleanly while keeping the public API surface identical to the generated class.
- **Four named JS sub-package exports instead of a single entry** — Lets consumers tree-shake; portal client is intentionally scoped to portal-token operations.

<!-- archie:ai-end -->
