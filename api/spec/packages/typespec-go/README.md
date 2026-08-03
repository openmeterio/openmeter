# @openmeter/typespec-go

A TypeSpec **emitter** that generates the OpenMeter **Go** SDK from the AIP
TypeSpec specs, mirroring [`@openmeter/typespec-typescript`](../typespec-typescript)
but targeting Go.

## Output

The emitter writes to `api/v3/client` at the **repo root** (not a sibling of
this package): a single flat `package openmeter` that is its own nested Go
module, `github.com/openmeterio/openmeter/api/v3/client`. The output directory
is set in [`packages/aip/tspconfig.yaml`](../aip/tspconfig.yaml) via
`emitter-output-dir: '{output-dir}/../../../v3/client'`.

The generated files are fully regenerable — **never hand-edit them**. Change
the emitter (for spec-derived files) or `src/runtime-templates.ts` (for the
static runtime files), then regenerate. The output cleaner wipes previously
generated files before emission but preserves `*_test.go` and `testdata/`:
hand-written wire tests live in `api/v3/client` and survive regeneration.

## How it works

`src/emitter.tsx` discovers and groups HTTP operations, validates codec/name
exhaustiveness, computes payload-context reachability (read vs input model
projections), and renders models, services, the root client, and the package
`README.md` from one operation IR — so documented call paths and routes always
match the emitted SDK. Unions retain their raw JSON for forward-compatible
round-tripping. The wire format is snake_case and the Go surface is PascalCase
fields with `json:"snake_case"` tags, so — unlike the TS emitter — there is
**no casing translation layer**. See [PLAN.md](./PLAN.md) for the design
history and the full architecture.

## Options

Declared in `src/lib.ts`, configured in `packages/aip/tspconfig.yaml`:

| Option                | Required | Purpose                                                                                |
| --------------------- | -------- | -------------------------------------------------------------------------------------- |
| `module-path`         | yes      | Go module path of the generated SDK (`github.com/openmeterio/openmeter/api/v3/client`) |
| `package-name`        | yes      | Go package name (`openmeter`)                                                          |
| `sdk-version`         | no       | Fallback version used when Go build info is unavailable; defaults to `0.0.0-dev`       |
| `include-services`    | no       | Service namespaces to emit (`['OpenMeter']`); all services when omitted                |
| `strip-name-prefixes` | no       | PascalCase type-name prefixes stripped when unambiguous                                |
| `include-resources`   | no       | Operation groups to emit; every discovered group when omitted                          |
| `readme-note`         | no       | Markdown callout inserted after the generated README intro                             |
| `go-version`          | no       | Stamped into the go.mod `go` directive; defaults to `1.23`, the generated code's floor |

## Commands

| Task                | Command                                                                                                  |
| ------------------- | -------------------------------------------------------------------------------------------------------- |
| Build the emitter   | `pnpm build` (`alloy build`)                                                                             |
| Watch               | `pnpm watch`                                                                                             |
| Typecheck           | `pnpm typecheck`                                                                                         |
| Emitter tests       | `pnpm test` (vitest over `test/`)                                                                        |
| Emitter checks      | `pnpm check` (typecheck + tests)                                                                         |
| Regenerate the SDK  | `pnpm --filter @openmeter/api-spec-aip run generate` (or `make gen-api`, repo root)                      |
| Generated SDK check | `make test-go-sdk` (repo root), or in `api/v3/client`: `go build ./... && go vet ./... && go test ./...` |

## Wiring

Registered in [`packages/aip/tspconfig.yaml`](../aip/tspconfig.yaml) under
`emit:` and declared as a `workspace:*` devDependency of
`@openmeter/api-spec-aip` so `tsp` resolves it. One `pnpm generate` produces
the OpenAPI document and every SDK.

## Releases

Releases are `api/v3/client/vX.Y.Z` git tags, gated by
`.github/workflows/release-go-sdk.yaml`. `Version` resolves at runtime via
`debug.ReadBuildInfo()` to the module version consumers pulled in through their
own `go.mod`, so no stamping commit is needed before tagging: push the paired
root and nested-module tags and the gate runs `make test-go-sdk` against them.
`sdk-version` in `packages/aip/tspconfig.yaml` only sets the fallback baked
into `option.go` for builds without resolvable module build info (the module
itself, replace directives, vendored trees).
