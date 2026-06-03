# api

<!-- archie:ai-start -->

> The public contract surface of OpenMeter: TypeSpec source under api/spec is the single source of truth that compiles to OpenAPI YAMLs, Go server stubs (api/api.gen.go for v1, api/v3/api.gen.go for v3), and the Go/JS/Python SDKs under api/client. Its primary constraint is that every downstream artefact is generated — any change starts in TypeSpec and flows through make gen-api, never by hand-editing generated files.

## Patterns

**TypeSpec-first, generated-everything-else** — spec/ (TypeSpec) is authored; the OpenAPI YAMLs (openapi.yaml, openapi.cloud.yaml, v3/openapi.yaml), Go stubs (*.gen.go), and all api/client SDKs are regenerated artefacts. New endpoints begin in spec/, then make gen-api, then make generate. (`Add op in api/spec/packages/aip/src/openmeter.tsp -> make gen-api -> make generate -> implement in api/v3/handlers/foo/handler.go`)
**v1 vs v3 package split** — spec/packages/legacy compiles v1/v2 + Cloud to api/openapi.yaml + api/openapi.cloud.yaml + api/api.gen.go; spec/packages/aip compiles the v3 AIP API to api/v3/openapi.yaml + api/v3/api.gen.go. The two never mix content. (`v1 content in packages/legacy/src/main.tsp; v3 content in packages/aip/src/openmeter.tsp`)
**Route/tag bindings only at root namespace files** — @route and @tag decorators live only in root openmeter.tsp (v3) / main.tsp (v1); domain sub-folder operation files declare only operation signatures. (`@route("/api/v1/meters") interface MeterRoutes {} in the root tsp, not in a domain operations.tsp`)
**RFC 7807 error responses via apierrors / models.NewStatusProblem** — All v3 error responses go through the apierrors package; direct http.Error / w.WriteHeader breaks application/problem+json compliance. (`return nil, apierrors.NewValidationError("invalid input")`)
**nullable.Nullable[T] for optional v3 JSON fields** — v3 codegen uses nullable-type:true and always-prefix-enum-values; optional JSON fields use nullable.Nullable[string], never *string, to preserve the null-vs-absent distinction. (`type UpdateInput struct { Name nullable.Nullable[string] }`)
**api/types gap-fill via x-go-type only** — Manually-authored Go types (oneOf/anyOf cases oapi-codegen cannot generate) live in api/types and are referenced from the spec via x-go-type; that package holds no business logic and is consumed only by generated code. (`api/types/doc.go pure type definitions referenced from the spec via x-go-type`)
**SDKs are generated-first, wrapper-second and externally importable** — api/client packages add only ergonomic wrappers and auth helpers (RequestEditorFn / ClientOption); the Go SDK is generated from openapi.cloud.yaml, and no app-internal monorepo packages may be imported. api/client/node and api/client/web are deprecated tombstones. (`api/client/go/client.gen.go (from openapi.cloud.yaml) wrapped by hand-authored client.go`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `api/spec/` | Single source of truth: TypeSpec authored in packages/aip (v3) and packages/legacy (v1) that compiles to every OpenAPI YAML and SDK | Register new legacy sub-domain files in both main.tsp and cloud/main.tsp or the endpoint is silently absent; patches/ apply to dist/ not src/ |
| `api/api.gen.go` | Generated v1 server stubs, enums, and request/response types from openapi.yaml via oapi-codegen | DO NOT EDIT — hand edits are overwritten by make gen-api |
| `api/v3/api.gen.go` | Generated v3 ServerInterface (chi-server, nullable types) from api/v3/openapi.yaml | DO NOT EDIT; always-prefix-enum-values means enum consts are prefixed; business logic belongs only in api/v3/handlers/* |
| `api/v3/codegen.yaml` | oapi-codegen config for v3 (nullable-type, always-prefix-enum-values, custom chi-middleware.tmpl for deepObject filters) | Keep the custom chi-middleware.tmpl patch aligned with codegen.yaml when upgrading oapi-codegen |
| `api/client/go/client.gen.go` | Generated Go SDK from api/openapi.cloud.yaml (cloud variant, additional cloud-only endpoints) | Must not import internal monorepo packages — stays externally importable; regenerated from the cloud spec only |
| `api/types/doc.go` | Manually-authored types filling oapi-codegen gaps, referenced via x-go-type | No business logic, helpers, or methods; do not duplicate types already in *.gen.go or import this from domain code |
| `api/openapi.yaml / api/openapi.cloud.yaml / api/v3/openapi.yaml` | Generated OpenAPI specs (v1, cloud, v3) — downstream artefacts of make gen-api | Never hand-edit; regenerate from TypeSpec |
| `api/spec/packages/aip/scripts/flatten-allof.mjs` | Post-processing that flattens allOf in the v3 OpenAPI output | Must run as part of the make gen-api pipeline; bypassing leaves unflattened allOf in v3/openapi.yaml |

## Anti-Patterns

- Hand-editing any OpenAPI YAML or *.gen.go file instead of editing api/spec and running make gen-api
- Declaring @route or @tag inside a domain sub-folder TypeSpec file rather than the root namespace files
- Mixing v3 content into packages/legacy or v1/v2 content into packages/aip — they compile to separate targets
- Adding business logic at api/v3 level outside handlers/, or adding new API types to hand-authored SDK wrappers instead of TypeSpec
- Using *string instead of nullable.Nullable[T] for optional v3 JSON fields, or adding SDK code to the deprecated api/client/node and api/client/web tombstones

## Decisions

- **TypeSpec as single source of truth compiling to OpenAPI plus Go/JS/Python SDKs and two server versions** — One spec feeds three SDK languages and two API versions, making cross-language drift structurally impossible and forcing breaking changes to surface at TypeSpec compile time
- **Go SDK generated from openapi.cloud.yaml rather than openapi.yaml** — The cloud variant exposes additional cloud-only endpoints absent from the self-hosted spec, keeping self-hosted and cloud SDKs correctly scoped
- **Two independently compiled TypeSpec packages (aip and legacy) with vendored emitter patches and a flatten-allof post-process** — v1 and v3 evolve on different schemas and emit to separate targets; vendoring patches on dist/ and post-processing the output avoids forking the TypeSpec compiler

## Example: Add a v3 API endpoint: author TypeSpec, regenerate, then implement the handler

```
// 1. In api/spec/packages/aip/src/openmeter.tsp declare the op (route/tag at root namespace)
// 2. make gen-api && make generate
// 3. Implement in api/v3/handlers/foo/handler.go:
import (
    api "github.com/openmeterio/openmeter/api/v3"
)
func (h *Handler) ListFoos(ctx context.Context, req api.ListFoosRequestObject) (api.ListFoosResponseObject, error) {
    items, err := h.svc.List(ctx, req.Params.Namespace)
    if err != nil { return nil, err }
    return api.ListFoos200JSONResponse{Items: toAPI(items)}, nil
}
```

<!-- archie:ai-end -->
