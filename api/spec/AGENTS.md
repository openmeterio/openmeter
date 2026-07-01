# OpenMeter API Spec & SDK Generator

This workspace holds the TypeSpec API definitions and the TypeScript SDK
generator. For repo-wide guidance see the root [AGENTS.md](../../AGENTS.md); this
file covers only what is specific to `api/spec`.

## Layout

```
packages/
  aip/                      # AIP TypeSpec source (api definitions, linter rules)
  legacy/                   # legacy OpenAPI output
  typespec-typescript/      # the SDK generator (TypeSpec emitter, Alloy-based)
  aip-client-javascript/    # generator OUTPUT: the emitted TypeScript SDK
```

The **baseline** (the frozen hand-written SDK + conformance tests the generator
reproduces) is NOT kept in the repo — its vitest/vite devDeps trip the workspace
`minimumReleaseAge` constraint. Its content is already embedded in
`typespec-typescript/src/runtime-templates.ts` (base64) and emitted into the
generated SDK, so the pipeline does not need it. To edit the runtime templates
or tests, restore the baseline (e.g. from `/tmp/om-aip-sdk-baseline` or git
history) and re-run `gen-runtime-templates.mjs` with `BASELINE_DIR` pointing at
it.

- `typespec-typescript` is a TypeSpec **emitter** built on `@alloy-js` +
  `@typespec/emitter-framework`. It walks HTTP operations and emits the full
  SDK.
- `aip-client-javascript` is its **output directory** (`emitter-output-dir` in
  `packages/aip/tspconfig.yaml` points here). Everything it contains is
  regenerable — never hand-edit it. A single `generate` emits the complete SDK
  (schemas, runtime, per-namespace surface, barrel) plus the conformance tests.

### How the emitter is structured

- `emitter.tsx` — `$onEmit`: emits `schemas.ts` (Alloy components, the original
  path), the static runtime files, and the per-namespace surface files, all as
  sibling `<ts.SourceFile>` children of one `<Output>`.
- `runtime-templates.ts` — base64-embedded copies of the fixed runtime files
  (`core.ts`, `lib/*`, `models/errors.ts`) and the conformance tests, generated
  from `baseline/` by `scripts/gen-runtime-templates.mjs`. Re-run that script
  when the baseline runtime or tests change.
- `sdk-operations.ts` — operation discovery: namespace grouping, per-op metadata
  (path/query/body/response), and naming (func name, facade method name via
  resource-noun stripping, namespace names).
- `sdk-files.ts` — string generators for the spec-derived surface files
  (operations types, funcs, facades, root client, barrels).
- `readme.ts` — builds the package `README.md` (emitted at the package root, not
  under `src/`) from the same grouped `SdkOperation[]` the SDK files use, so its
  documented call paths and routes always match the emitted client.

### Grouping (reproduce this)

The `OpenMeter` service surfaces every operation through an `*Endpoints`
interface that `extends` the resource's interface in its **source** namespace
(e.g. `OpenMeter.PlansEndpoints extends ProductCatalog.PlanOperations`). The op
walked lives on the `*Endpoints` interface, so its own `namespace` is
`OpenMeter` — the meaningful grouping is on `op.interface.sourceInterfaces[0]`.

Group by the **top of the source namespace chain** so multi-interface
namespaces stay one client: `MetersEndpoints` + `MetersQueryEndpoints` →
`meters` (keeps `meters.query`); all `Customer*Endpoints` → `customers`.
`ProductCatalog` is the exception (in `SPLIT_BY_INTERFACE`): it splits by source
interface → `plans`, `addons`, `planAddons`. Do NOT group by `@tag` (the tag is
a display string like "Metering Events" → stutter) or by `op.namespace` (always
`OpenMeter`).

### Nested sub-clients (reproduce this)

The SDK nests sub-clients (`customers.charges.list()`,
`customers.credits.grants.list()`) from the **source namespace chain** below the
group's top namespace. Walking `sourceInterfaces[0].namespace` up to the global
root yields e.g. `['Customers', 'Charges']` → group `customers`, nest path
`['charges']`; `['Customers', 'Credits', 'Grants']` → `customers.credits.grants`.
`facadeFile` builds a tree from these paths, emitting one class per node with
lazy sub-client getters that share the parent's `Client`.

Because grouping follows the source namespace (not the route), an operation
routed under one resource but defined in another's namespace lands under the
latter — by design. `list-customer-entitlement-access` is routed under
`/customers/` but its interface lives in `Entitlements`, so it is
`entitlements.listCustomerAccess()`, NOT `customers.entitlements.*`. This is a
deliberate decision (the op is genuinely an Entitlements operation); do not
"fix" it to nest under customers.

This nesting is **driven by the TypeSpec source structure, not API routes** — to
nest a resource, wrap its `*Operations` interface in a sub-namespace
(`namespace Charges { interface CustomerChargesOperations { … } }` inside a file
that declares `namespace Customers;`), and update the `extends` reference in
**both** `openmeter.tsp` and `konnect.tsp` to the nested path. Wrap ONLY the
operation interface — leave models in the parent namespace so their schema names
(and OpenAPI output) are unchanged. The method-name strip set includes the nest
segments, so `create-credit-grant` under `credits.grants` → `create`.

**OpenAPI invariance is the hard gate** for any `.tsp` change: regenerate and
confirm the `output/definitions/.../*.yaml` hashes are unchanged. Namespace
nesting of operation interfaces is OpenAPI-neutral (paths/tags/operationIds are
explicit); moving a _model_ is not. Watch for namespace collisions — nesting
`Customers.Billing` shadows the global `Billing` namespace for unqualified refs
in `Customers`-scoped files; alias around it (`Common.BillingRoot`) rather than
renaming.

### Naming rules (reproduce these)

- **func name** = full camelCase operationId: `get-meter` → `getMeter`.
- **facade method** = operationId with the group's resource noun(s) and the
  cross-cutting `metering` qualifier stripped: `get-meter` → `get`;
  `get-customer-billing` → `getBilling`; `ingest-metering-events` → `ingest`.
  Singular/plural folded. The resource name is split into strip-words on
  separators **and** case boundaries — both camelCase (`PlanAddons` → `plan`,
  `addons`) and acronym→word (`LLMCost` → `llm`, `cost`) — so multi-word and
  acronym-prefixed namespaces strip fully (`create-plan-addon` → `create`,
  `create-llm-cost-override` → `createOverride`). When the operationId noun is
  not the namespace's own resource word it is kept as a disambiguator
  (`llmCost.listPrices`/`listOverrides`, `subscriptions.listAddons`).
- **namespace** = source namespace (already plural, e.g. `Meters`, `Events`,
  `Customers`) or a pluralized split interface resource (`Plan` → `Plans`);
  PascalCase class / camelCase getter.
- **request type** composed from direct-TS parts (no `z.input`): query-only →
  `<Base>Query`; body-only → the body interface (its `…Input` variant when the
  body diverges on input); path-only → `{ id: string }`; path+body →
  `{ id; body }`; path+query → `<Base>Query & { id }`; body+query →
  `{ body } & <Base>Query` (body nested so query fields don't leak into the JSON
  body). Path params are ULIDs, typed `string`. See "Request types" below.

### TypeSpec style constraints

- When adding query decorators (for example `@query`) to a TypeSpec file that
  does not already use HTTP decorators, import `@typespec/http` and add
  `using TypeSpec.Http;` in that file; otherwise compilation fails with
  `Unknown decorator @query`.

## Commands

| Task                          | Command                                                  |
| ----------------------------- | -------------------------------------------------------- |
| Build the emitter             | `pnpm --filter @openmeter/typespec-typescript run build` |
| Regenerate SDK from TypeSpec  | `pnpm --filter @openmeter/api-spec-aip run generate`     |
| Run the SDK conformance tests | `pnpm run test:sdk`                                      |
| Install / refresh lockfile    | `pnpm install --config.confirmModulesPurge=false`        |

The emitter is bound by **package name** (`@openmeter/typespec-typescript`) in
`packages/aip/tspconfig.yaml` (both the `emit:` list and the `options:` key). The
internal lib name in `src/lib.ts` and its `…:` state keys are a separate
identity used for diagnostics/state and have no cross-package references.

## The emitted SDK: conventions the generator must reproduce

The hand-written baseline (kept under a temporary reference folder) defines the
exact shape the generator must reproduce. Its tests are the conformance target —
the generated SDK is "done" when it passes them.

### Casing: camelCase public surface, snake_case wire

The AIP API is **snake_case on the wire** (TypeSpec, OpenAPI, and the casing lint
rule stay snake). The generated JS SDK exposes a **camelCase** public surface — the
TS interfaces and zod schemas are camelCase — and a boundary mapper
(`src/lib/wire.ts`) translates at the edge: `toWire` (camelCase → snake_case) on
request bodies and query objects, `fromWire` (snake_case → camelCase) on responses.

camelCase is the **TypeScript-specific** public surface, not a wire change — the
wire stays snake_case for every SDK. Other language generators are expected to apply
their own idiomatic surface transformation over the same snake_case wire: a Go SDK
would use exported UpperCamelCase fields with `json:"snake_case"` tags, a Python SDK
would keep snake_case (already idiomatic), etc. Keep casing decisions in the
per-language emitter; do not push a language's casing into TypeSpec, OpenAPI, or the
wire.

The translation is a **deterministic casing rule**, not a per-field map: every wire
name round-trips through `toSnakeCase(toCamelCase(name))`, enforced at codegen by a
gate (`assertCasingDerivable`) that fails the build for any non-derivable name. The
public key is `toCamelCase(resolveEncodedName(...))`, so the wire key the mapper
emits is exactly the OpenAPI name. The mapper is **schema-driven**: it walks the zod
schema alongside the data so `Record<string, …>` keys that are user data (label
names, meter dimension names) are preserved verbatim, while typed field keys
(including AIP `filter[field]` names and `sort.by`) are translated.

The same gate (`assertCasingDerivable`) also **fails the build for a non-discriminated
union with two or more object variants reachable from a request body or success
response** — the mapper cannot pick a variant without a discriminator, and does not
guess. Use `@discriminated` for such unions (scalar-vs-object unions, and `T | T[]`
single-or-batch bodies, are fine — distinguished at runtime by JS type). Discriminated
unions dispatch via a memoized literal→variant map keyed on the (camel public / snake
wire) discriminator value.

### Response/request mapping drops unknown fields

`fromWire`/`toWire` **rename keys only** — they never call `schema.parse()`, never
apply zod defaults, and never coerce values. A field not present in the schema shape
is **dropped**, so the mapped object exactly matches the typed interface (a
server-added field is not in the type and does not survive). This is a deliberate
choice for strict typing over forward-compatibility. zod is retained for type
derivation (`z.input`/`z.output`), query/path coercion, mapper structure, and the
one `baseError.safeParse` in the error path. Error responses bypass the mapper
(`toError` reads the raw snake body; `HTTPError.getField` is a raw, untyped escape
hatch).

### Optional wire-payload validation (`validate` option)

`SDKOptions.validate` (default **off**) turns on schema validation of the actual
snake*case wire payload: the request body after `toWire` (before sending) and the
raw response body before `fromWire`. Validation uses the generated **`…Wire`
schemas** in `models/schemas.ts` — every model and per-op body/response is emitted a
second time in a snake_case "wire" pass (`WireModeContext` in the emitter), keyed by
the raw JSON wire name and made `z.strictObject`, so a wrong-shaped or
leaked-camelCase wire field is **rejected, not silently stripped**. Open models
(record spread, `emitsAsIntersection`, e.g. `baseError`) stay non-strict — strict
would defeat the record arm that exists to accept them. Because the wire pass is the
same emitter walk as the camelCase pass (parameterized by key-casing + strictness +
a separate refkey namespace), the two are structurally identical except for casing,
**by construction** — no runtime schema derivation. A failure throws
`ValidationError`, which `request()` surfaces as `Result.error` (request validation
runs \_inside* the `request()` closure so it does not throw synchronously).
**Enabling `validate` re-introduces exactly the rejection the default policy
avoids**: a strict wire schema rejects additive/unknown server fields and unknown
enum values. It is opt-in defense-in-depth, not the default, precisely because the
default contract must not break on additive fields.

### Documented types: generated from TypeSpec, verified against zod

**zod schemas and TypeScript types are separate artifacts with one source.** Both
are generated from the same TypeSpec, but neither is derived from the other at the
type level:

- `models/schemas.ts` — zod schemas (runtime validation in the error path, query/
  path coercion). The runtime artifact.
- `models/types.ts` — concrete TypeScript interfaces (the public surface that
  `.json<T>()` is typed against). **Self-contained: it imports neither `zod` nor
  `schemas.ts`.** Field types are walked directly from the TypeSpec AST by
  `tsTypeOf` in `ts-types.ts`, which mirrors the leaf decisions of
  `zodBaseSchemaParts` (the zod walker) so the two stay type-equivalent.
- `models/types.assert.ts` — the inferred types are used **only for verification**:
  a mutual-assignability guard ties `types.ts` to `schemas.ts` at build time.

Why not infer `types.ts` from `z.output<typeof schema>`? zod strips `.describe()`
at the type level, so an inferred type has the shape but no docs; and indexed
access (`Meter['name']`) couples the public types to the runtime schemas. Walking
TypeSpec directly gives clean concrete types (`id: string`,
`aggregation: "sum" | "count" | …`, `labels?: Labels`) with `/** … */` JSDoc from
the TypeSpec `@doc`, decoupled from zod.

`tsTypeOf` leaf mapping (must match `zodBaseSchemaParts` or the guard fails):

- scalars → `string` / `number` / `boolean`; **int64/uint64 → `bigint`** (zod uses
  `z.coerce.bigint()`); everything else numeric → `number`.
- **dates/times/durations → `string`** (wire-native; RFC 3339, never `Date`).
- enums and unions → inlined literal/variant unions (`"a" | "b"`, `A | B`); they
  are not collected as named interfaces.
- named models (incl. named records like `Labels`) → ref the interface; anonymous
  models → inlined object literal; arrays → `T[]` (parenthesized when `T` is a
  union: `(A | B)[]`); open records → `Record<string, V>`.

Structural rules the interface emitter follows:

- **Optionality follows OUTPUT**: a defaulted field is optional-in / required-out,
  so `prop.optional && prop.defaultValue === undefined` decides the `?`.
- **No-wire-prop models alias** to their mapped structure
  (`export type Labels = Record<string, string>`), never an empty permissive
  `interface {}`. The alias excludes the model from its own ref resolution so it
  does not become `type Labels = Labels`.
- **`extends`** the base interface when the model has a `baseModel`, so inherited
  fields/docs propagate (`BadRequest extends BaseError`).
- **Open records** (`...Record<…>`) get an index signature (`[key: string]: V`).
- **Unions stay on `z.output`** at the _response_ layer (e.g. `GetAppResponse`) — a
  discriminated union has no single interface, so wiring it to one would be wrong.

**Conformance guard (the oracle).** Every emitted type — both `interface`s **and**
the no-wire-prop `type` aliases — is paired with a mutual-assignability check in
`models/types.assert.ts`
(`[X] extends [z.output<…>] ? [z.output<…>] extends [X] ? true : {__error}`). This
is the _only_ place `types.ts` and `schemas.ts` meet: it proves the directly-walked
TS type is type-equivalent to the zod schema, turning any divergence (wrong leaf,
wrong optionality, header leak, open-record gap) into a **build error**. `tsc` is
the oracle. The alias branch must guard too: unlike a former `z.output` alias
(tautological), a `tsTypeOf`-walked alias like `LabelsFieldFilter` is an
independent claim that can diverge. One blind spot remains by nature: the check is
vacuous when either side is `any` — so the output is also grepped for `: any` (the
AIP spec uses `unknown`, never `any`, so no field hits it).

**Response wiring.** Per-operation `…Response` aliases point at the documented
interface when the success body resolves to a named model. The extracted HTTP body
of a list endpoint is **anonymous** (TypeSpec strips the envelope identity during
body extraction), so `sdkOperation` falls back to the 2xx **response envelope**
(`HttpOperationResponse.type`), whose `@friendlyName` survives — e.g.
`PagePaginatedResponse<Meter>` → `MeterPagePaginatedResponse`. This reuses the
already-emitted, already-guarded paginated interfaces (no synthesis). Net: ~72/83
responses wired to interfaces, 10 void, 1 union on `z.output`.

Compared to the `zod-to-ts` npm package (which also walks a zod schema to a TS
type with JSDoc from `.describe()`): that lib **inlines** nested objects and emits
`prop?: T | undefined` for optionals, sourcing docs from `.describe()`. This
generator instead **refs** named interfaces (better for a published SDK), emits
clean `prop?: T` output-shaped optionality (defaulted fields required, no
`| undefined`), and sources docs from TypeSpec `@doc` — so the emitter does not
depend on `.describe()` surviving into the runtime schemas.

### Factoring: what the generator emits, and how often

- **ONCE** (shared runtime in `lib/` + `core.ts`): the base `Client`/transport
  (one `ky.create`), the `request()` envelope, `Result`/`ok`/`err`/`unwrap`,
  the curated `RequestOptions`, the encoders (`encodePath`, `toURLSearchParams`,
  `encodeSort`, `querySerializer`), `toError`, and the `HTTPError` class.
- **PER-NAMESPACE** (per resource/tag): one façade class that **composes** a
  `Client` (holds a reference — it does **not** `extends Client`) plus one
  memoized lazy getter on the root `OpenMeter`.
- **PER-OPERATION** (×~83): one standalone func = path/query/body assembly +
  `request(() => http(client).<verb>(…).json<R>())`, plus a one-line façade
  wrapper. The request/response type aliases and per-op `…Query` types live in
  `models/operations/<ns>.ts` (their guards in `models/operations/<ns>.assert.ts`);
  `funcs/<ns>.ts` imports `…Request`/`…Response` from there and holds only
  functions, so the funcs modules stay free of type declarations and guards.

### Void responses must not call `.json()`

The 10 operations whose `Response` is `void` (`!op.hasResponse` — every
`delete*` plus `events.ingest`, which return `204 No Content` / `202 Accepted`
with an empty body) terminate with `request(async () => { await http(client).<verb>(…) })`,
**not** `.json<R>()`. ky's `.json()` throws `SyntaxError: Unexpected end of JSON input`
on an empty body (and explicitly on `204`), so calling it on a successful
void response rejects a request that actually succeeded server-side — `ingest`
(the product's hot path) and every delete. Awaiting the `ResponsePromise`
without parsing still rejects on non-2xx (ky's `throwHttpErrors` default is on),
so error propagation is preserved. `funcBody` branches on `op.hasResponse` for
this; non-void funcs keep the `.json<R>()` terminal unchanged.
`tests/void-responses.spec.ts` is the regression guard: success on empty
202/204, still-rejects on a 500 (status-only fallback), and **full problem+json
error fidelity** preserved on a void op (the ky fork populates `e.data` at throw
time regardless of `.json()`, so `to-error.ts` recovers `title`/`detail`/`type`
identically to non-void ops). Note `baseError.safeParse` requires `instance`, so
a problem+json mock without it falls through to the status-only error — include
`instance` to exercise the structured branch. The test is **hand-maintained**
(the baseline that feeds `runtime-templates.ts` is not in the repo), so it lives
directly in `tests/` and is not re-emitted by `generate` — do not delete it
expecting a regen to restore it.

### Request types: direct TS, input-variant interfaces

Request/response types are direct TS, not `z.input`/`z.output`, and live in
`models/operations/<ns>.ts` (re-exported from the barrel under their existing
public names). The split mirrors the model types:

- **Response** → the documented output interface (or `void`, or the one
  `z.output` union `GetAppResponse`).
- **Request body** → the body model's interface, or its **`…Input` variant**
  when the body's input shape diverges from its output. A body diverges iff a
  defaulted field — anywhere in its reachable subtree — flips from required
  (output) to optional (input). `computeDivergentModels` (in `input-variants.ts`)
  is the transitive fixpoint; `interface-types.ts` emits an `XInput` interface
  (relaxed optionality, refing child `YInput` variants) for each divergent model.
  ~12 request bodies diverge directly; their closure is ~51 `…Input` interfaces.
- **Query** → a per-op `<Base>Query` interface walked from the query parameter
  leaves in input mode (in `models/operations/<ns>.ts`).
- **Path** params are ULIDs → `string`.
- **Shared-route JSON body override** → a `@sharedRoute` endpoint declares one
  operation per content type (e.g. `events.ingest`: a single-event
  `cloudevents+json`, a batch `cloudevents-batch+json`, and a single-or-batch
  `application/json` union). `collectHttpOperations` keeps the **first** variant
  (for its doc/summary/response/202), which is the single-event one — so without
  intervention the request body would be `EventInput` only. `jsonBodyOverrides`
  (in `sdk-operations.ts`) maps such an endpoint to its `application/json` body
  type when that differs from the kept variant's; `request-types.ts` then renders
  the body with `tsTypeOf(..., 'input')` (→ `EventInput | EventInput[]`) instead
  of a single named-interface import. Trigger is narrow (only ingest today); the
  func/facade are unchanged — `json: req` serializes an object or array
  identically, so widening only the request **type** is sufficient.

Two traps the generator handles:

- **Name collision**: the op request type `<Base>Request` collides with a body
  model interface of the same name (e.g. `CreateMeterRequest`). The body is
  imported under a `<Name>Body` alias so the local request declaration owns the
  name (`import type { CreateMeterRequest as CreateMeterRequestBody }`).
- **Coerced leaves**: `z.input` of a `z.coerce.*` leaf is the loose `unknown`
  (zod 4). The emitted input type deliberately keeps the **strict** leaf
  (`bigint`/`number`/…) rather than propagate the `unknown` wart. So input
  variants and `…Query` types are guarded **one-directionally**
  (`[XInput] extends [z.input<…>]` — "is a valid input"), not bidirectionally.
  Output interfaces keep the full bidirectional guard. The `…Query` guards live
  in a sibling `models/operations/<ns>.assert.ts`, matching how model guards live
  in `types.assert.ts`.

**Selection is unguarded by the shipped guards — verify it separately.** A
too-strict request type (refing the output `X` where `XInput` was needed) still
satisfies the one-directional `[X] extends [z.input]`, so the shipped guards
can't catch picking the wrong variant or `computeDivergentModels` under-marking.
Two independent checks close this:

- The 20 conformance tests construct real requests (end-to-end).
- **Coverage probe** (the authoritative recipe — do NOT use a regex reachability
  tracer; it false-matches identifiers inside `.regex(/…/)` and `.describe()`):
  for every model reachable from a `*Body` schema that has an output interface,
  assert `[X] extends [z.input<typeof schemas.x>]`. If `X` is too strict (an
  `XInput` was needed but missing), this fails. Run it as a temporary probe file
  compiled by tsc; zero failures = every request-reachable model is covered.

### Dual surface

Every operation exists twice: a standalone func in `funcs/` returning
`Result<T>` (tree-shakeable, non-throwing) and a thin method on the namespace
façade in `sdk/` that `unwrap`s and throws. Both call the same func.

### README

`readme.ts` emits the package `README.md` at the package root (`emitter-output-dir`
is the package root, so non-`src/` paths land there; `package.json`/`tests/`
survive because `writeOutput` only writes listed paths). It is built from the
same grouped `SdkOperation[]` as the SDK files, in `groupOperations` insertion
order (matching `index.ts`), so the "Available Resources and Operations" table's
call paths (`getter` + `nestPath` + `methodName`, e.g. `customers.credits.grants.create`),
HTTP routes, and per-op summaries (`$.type.getDoc(op)`, carried on `SdkOperation.summary`)
always equal the emitted client. The install/import package name comes from the
**required** `package-name` emitter option (`context.options['package-name']`,
declared in `lib.ts` with `required: ['package-name']` and set in `aip/tspconfig.yaml`)
— never hardcode it in `readme.ts`, and there is no fallback: omitting the option
fails the whole compile with an `invalid-schema` diagnostic (verified by removing
it from tspconfig), so a missing name can never leak `undefined` into the README.
The example client variable is `client` (matching the table prefix `client.<path>`);
if you rename it, update both the fence declarations and `operationsTable`'s prefix
together. The table-of-contents anchors and the headings
are produced by one `slug()` so TOC links never break. Every code fence is
self-contained (constructs its own `client`) and typechecks against the real
generated types; the `meters.create` payload uses the camelCase public surface
(`eventType`, `valueProperty`) and the lowercase aggregation enum (`'sum'`),
matching `CreateMeterRequest`. The README is emitted raw (compact markdown
tables); the generated `aip-client-javascript` output and the emitter's own
`typespec-typescript/src` are **not** prettier-clean on HEAD (`prettier --check .`
is already red for both subtrees), so do not pre-align tables in the emitter or
single out the README in `.prettierignore`.

### RequestOptions is curated

`RequestOptions = Pick<Options, 'signal' | 'headers' | 'timeout' | 'retry'>`.
Do not widen it to the full ky `Options` — exposing `searchParams`/`json`/`hooks`/
`fetch`/`prefix` per call lets callers clobber transport internals.

### Errors

`toError` maps ky failures to the domain `HTTPError` (RFC7807 `problem+json`,
charset-tolerant Content-Type match; status-only fallback otherwise). `Result`'s
error type stays `Error` — the ky fork also throws `TimeoutError`/`NetworkError`,
so narrowing to `HTTPError` would be unsound. Callers narrow with
`instanceof HTTPError`. A single `HTTPError` class (no per-status hierarchy);
field-level validation errors are reachable via `getField('invalid_parameters')`.

### Server URL templating

`baseUrl` is **required** (no default). It may be a `ServerList` template with
`{region}`/`{port}` variables resolved via `encodePath(baseUrl, serverVariables)`,
a concrete URL, or a `URL` object. `region` is typed to the enumerated `Regions`.
Missing template variables throw (fail-loud, never a literal `{region}` on the
wire). The SDK owns URL construction: it pins `baseUrl` (trailing-slash
normalized) and `prefix: undefined` **after** spreading user options so a
user-supplied `prefix` cannot redirect requests; the auth hook is appended
**after** user `beforeRequest` hooks so SDK auth wins.

### ky is a fork — preserve its option names

The vendored `ky` uses `baseUrl`/`prefix`/`totalTimeout`/`retryOnTimeout` (not
mainline ky's `prefixUrl`). The emitter's runtime must use the fork's names; do
not "correct" them to mainline ky.

## Query serialization (verified against the server)

`api/v3/filters/parse.go` is the source of truth for filter encoding:

- deep objects: `page[size]`, `filter[key][eq]` (bracketed)
- scalar `filter[key]=v` is shorthand for `filter[key][eq]=v`
- array operands (`oeq`/`ocontains`) are **comma-joined into one param**; the
  server **rejects repeated** query params. Never emit `k=a&k=b`.
- `sort` serializes to a plain string `"<field> [asc|desc]"` (single space) on the
  wire; the SDK accepts a `{by, order}` object and `encodeSort` flattens it. `by` is
  a **camelCase** field name in the SDK and is `toSnakeCase`-translated to the wire
  field name (the server validates snake field names; see
  `api/v3/handlers/.../convert.go`).

## Tests

The conformance tests (Vitest + `@fetch-mock/vitest`, matching the legacy SDK's
stack) are embedded in `runtime-templates.ts` and emitted into the generated
SDK. They are the generator's spec: it is "done" when these tests pass against
the emitted `aip-client-javascript` output.

`pnpm run test:sdk` roots at `packages/aip-client-javascript` and runs the
**generated** tests against the **generated** SDK, so `generate` followed by
`test:sdk` is fully self-contained — no baseline needed. The generated package
is never hand-edited; to change the runtime or tests, edit the restored baseline
and re-run `gen-runtime-templates.mjs` (see the layout note above).

The meters namespace is behaviorally verified end-to-end by these 19 tests. The
other namespaces are generated and type-checked (`tsc` clean across all 13) but
not yet behaviorally tested — add a smoke test per namespace if broader runtime
coverage is wanted.
