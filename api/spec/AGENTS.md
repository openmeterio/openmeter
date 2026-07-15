# OpenMeter API Spec & SDK Generator

This workspace holds the TypeSpec API definitions and SDK generators. For
repo-wide guidance see the root [AGENTS.md](../../AGENTS.md); this
file covers only what is specific to `api/spec`.

## Layout

```
packages/
  aip/                      # AIP TypeSpec source (api definitions, linter rules)
  legacy/                   # legacy OpenAPI output
  typespec-typescript/      # the SDK generator (TypeSpec emitter, Alloy-based)
  typespec-go/              # the Go SDK generator (TypeSpec emitter, Alloy-based)
  aip-client-javascript/    # generator OUTPUT: the emitted TypeScript SDK
```

The **runtime templates** (the fixed SDK runtime files + conformance tests the
generator reproduces verbatim) live as real, reviewable files under
`typespec-typescript/templates/` — not embedded blobs and not a separate
baseline directory. `typespec-typescript/src/runtime-templates.ts` reads them
via `readFileSync` at build time and emits them into the generated SDK. To
edit the runtime templates or tests, edit the files under `templates/`
directly, then run `make -C api/spec generate`.

- `typespec-typescript` is a TypeSpec **emitter** built on `@alloy-js` +
  `@typespec/emitter-framework`. It walks HTTP operations and emits the full
  SDK.
- `aip-client-javascript` is its **output directory** (`emitter-output-dir` in
  `packages/aip/tspconfig.yaml` points here). Everything it contains is
  regenerable — never hand-edit it. A single `generate` emits the complete SDK
  (schemas, runtime, per-namespace surface, barrel) plus the conformance tests.
- `typespec-go` is a TypeSpec **emitter** built on `@alloy-js/go` +
  `@typespec/emitter-framework`. It emits the Go SDK into `api/v3/client`, which
  is also fully regenerable generated output.

### How the emitter is structured

- `emitter.tsx` — `$onEmit`: emits `schemas.ts` (Alloy components, the original
  path), the static runtime files, and the per-namespace surface files, all as
  sibling `<ts.SourceFile>` children of one `<Output>`.
- `runtime-templates.ts` — reads the fixed runtime files (`core.ts`, `lib/*`,
  `models/errors.ts`) and the conformance tests verbatim, via `readFileSync`,
  from the committed `templates/runtime/` and `templates/tests/` directories
  at build time. Edit those files directly to change the runtime or tests;
  `templates/` is excluded from this package's own `tsconfig.json` `include`
  (only checked downstream, as part of the generated `aip-client-javascript`
  package's typecheck/test suite).
- New runtime helpers that don't fit the fixed `templates/` set (e.g.
  `lib/wire.ts`) are authored as a real `.ts` file under `src/runtime/`
  instead (type-checked and unit-tested by the emitter package's own tooling)
  and embedded verbatim via `readFileSync` at build time (see
  `src/wire-runtime.ts`), not as a template-string constant — backticks/`${`
  inside the runtime source collide with the template-literal delimiters.
- `sdk-operations.ts` — operation discovery: namespace grouping, per-op metadata
  (path/query/body/response), and naming (func name, facade method name via
  resource-noun stripping, namespace names).
- `pagination.ts` — structural detection of page-number vs cursor list
  operations (see "Pagination companions" below).
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

| Task                          | Command                                              |
| ----------------------------- | ---------------------------------------------------- |
| Build all TypeSpec emitters   | `pnpm run build`                                     |
| Regenerate SDK from TypeSpec  | `pnpm --filter @openmeter/api-spec-aip run generate` |
| Run the SDK conformance tests | `pnpm run test:sdk`                                  |
| Install / refresh lockfile    | `pnpm install --config.confirmModulesPurge=false`    |

The emitters are bound by **package name** (`@openmeter/typespec-typescript`,
`@openmeter/typespec-go`) in `packages/aip/tspconfig.yaml` (both the `emit:` list
and the `options:` keys). The internal lib names in each `src/lib.ts` and their
`…:` state keys are separate identities used for diagnostics/state and have no
cross-package references.

## The emitted SDK: conventions the generator must reproduce

The hand-written runtime files and conformance tests under
`typespec-typescript/templates/` define the exact shape the generator must
reproduce. The tests are the conformance target — the generated SDK is "done"
when it passes them.

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

### Dates: `Date` public surface, RFC 3339 wire, requests also take strings

Every date-time in the AIP spec is the shared `DateTime` scalar (`utcDateTime` with
`@encode(rfc3339)`). The wire stays the RFC 3339 string; the generated TS surface
types these fields **`Date`** — in interfaces, query types, and the camelCase zod
schemas (`z.date()`) — while the `…Wire` schemas keep `z.string().datetime()`.
The boundary mapper converts alongside the casing pass: `toWire` serializes any
`Date` instance to `toISOString()` wherever it sits (bodies and query objects alike,
and before `…Wire` validation, so `validate` checks the wire form), and `fromWire`
revives strings into `Date`s at date-typed schema nodes, including record/array
values. A datetime behind a union (`DateTime | null` on `event.time`,
enum-or-`DateTime` on subscription `timing`) is revived only when the date variant is
the string's sole plausible owner — enum literals, matching string literals, and
plain-string variants pass through untouched (fail-open, same policy as unmatched
union variants).

**Requests additionally accept RFC 3339 strings**: each body/query-bearing
`…Request` alias is wrapped in `AcceptDateStrings<T>` (exported from `lib/wire.ts`),
a recursive mapped type turning every `Date` into `Date | DateString`, where
`DateString = string & Record<never, never>` — assignable from any string but
immune to union absorption, so literal siblings of a `Date` (subscription
`timing`'s `'immediate' | 'next_billing_cycle'`) keep their autocomplete instead
of collapsing into `string`. The widening
lives on the request alias only — domain interfaces and `…Query` interfaces stay
`Date`, because they also describe responses and are pinned to the schemas by the
model conformance guard and the per-op query input guards; widening them (or forking
input variants per model) is exactly what this avoids. At runtime a request string
passes through the mapper verbatim (never re-parsed or normalized — a non-UTC
offset or malformed string reaches the server as-is unless `validate` is on, where
the wire schema's UTC `datetime()` check rejects it).

### Response/request mapping drops unknown fields

`fromWire`/`toWire` **rename keys and map date values only** (`Date` ↔ RFC 3339
string, see above) — they never call `schema.parse()`, never apply zod defaults, and
never coerce any other value. A field not present in the schema shape
is **dropped**, so the mapped object exactly matches the typed interface (a
server-added field is not in the type and does not survive). This is a deliberate
choice for strict typing over forward-compatibility. zod is retained for type
derivation (`z.input`/`z.output`), query/path coercion, mapper structure, and the
one `baseError.safeParse` in the error path. Error responses bypass the mapper
(`toError` reads the raw snake body; `HTTPError.getField` is a raw, untyped escape
hatch).

### Optional wire-payload validation (`validate` option)

`SDKOptions.validate` (default **off**) turns on schema validation of the actual
`snake_case` wire payload: the request body after `toWire` (before sending) and the
raw response body before `fromWire`. Validation uses the generated **`…Wire`
schemas** in `models/schemas.ts` — every model and per-op body/response is emitted a
second time in a `snake_case` "wire" pass (`WireModeContext` in the emitter), keyed by
the raw JSON wire name and made `z.strictObject`, so a wrong-shaped or
leaked-camelCase wire field is **rejected, not silently stripped**. Open models
(record spread, `emitsAsIntersection`, e.g. `baseError`) stay non-strict — strict
would defeat the record arm that exists to accept them. Because the wire pass is the
same emitter walk as the camelCase pass (parameterized by key-casing + strictness +
a separate refkey namespace), the two are structurally identical except for casing,
**by construction** — no runtime schema derivation. A failure throws
`ValidationError`, which `request()` surfaces as `Result.error` (request validation
runs _inside_ the `request()` closure so it does not throw synchronously).
**Enabling `validate` re-introduces exactly the rejection the default policy
avoids**: a strict wire schema rejects additive/unknown server fields and unknown
enum values. It is opt-in defense-in-depth, not the default, precisely because the
default contract must not break on additive fields.

Models decorated with `@useRef` still need a local TypeSpec shape that matches the
referenced OpenAPI schema. The TypeScript and Go emitters walk the local TypeSpec
AST; `@useRef` only changes the emitted OpenAPI reference and does not import the
referenced schema's requiredness or nullability into language-specific SDKs.

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
- enums → inlined literal unions (`"a" | "b"`); never collected as named
  interfaces. A **named** TypeSpec `union` (`union Price { free: PriceFree, … }`)
  refs its own `types.ts` alias when reachable (see "Named union aliases"
  below); an anonymous union expression (`A | B` written inline) still inlines
  its variants.
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
- **Named union aliases.** Every named TypeSpec `union` that is reachable from
  an operation on an included service gets its own
  `export type <Name> = <Variant1> | <Variant2> | …` in `types.ts` (`interface-types.ts`,
  `unionVariantsType` in `ts-types.ts`) — variants resolve through the same
  `RefName`/`refNameInput` machinery as model properties, so a model-variant is
  named (`PriceFree`) and an anonymous-object variant inlines. The alias gets the
  same conformance guard as a model interface, and an `…Input` variant
  (`computeDivergentUnions` in `input-variants.ts`) only when at least one variant
  is itself a divergent model (e.g. `WorkflowPaymentSettingsInput`, because
  `WorkflowPaymentSendInvoiceSettings` has a defaulted field) — a union with no
  divergent variant (e.g. `WorkflowCollectionAlignment`) has no `…Input` alias.
  **Reachability gate:** a union can be declared in TypeSpec (and still get a zod
  schema, since `getAllDataTypes` walks the whole namespace tree) without
  anything in the actual SDK surface referencing it — `computeReachableUnions` in
  `emitter.tsx` walks every collected operation's request body, query
  parameters, and response body (success and error) and only aliases unions it
  reaches. Every operation counts as a reachability root: `x-internal` and
  `x-private` operations are emitted under the `client.internal.*` surface, so
  the unions they reach are aliased too (`Invoice`/`InvoiceLine`/
  `UpdateInvoiceRequest` via the `x-private` invoice operations, `Currency` via
  the `x-internal` currency operations). `PriceUsageBased`,
  `ULIDOrResourceKey`, and `ULIDOrExternalResourceKey` are declared but never
  referenced by anything, so they stay zod-only (aliasing them would export a
  degenerate type like `string | string`); models are never reachability-gated
  — only the union alias pass is. This is a deliberately narrower policy
  than models', to avoid exporting unions nothing in the shipped client can
  ever produce or accept.
- **Response wiring picks up named unions too.** Because a named union now
  resolves through the same `resolveInterface`/`emittedInterfaceNames` path as a
  model, an operation whose success body is directly a reachable named union
  (e.g. `get-app` → `App`) wires its `…Response` alias to the union alias instead
  of falling back to `z.output<typeof schemas...>` — see "Response wiring" below.

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
interface when the success body resolves to a named model **or named union**
(e.g. `get-app` → `App`; see "Named union aliases" above). The extracted HTTP body
of a list endpoint is **anonymous** (TypeSpec strips the envelope identity during
body extraction), so `sdkOperation` falls back to the 2xx **response envelope**
(`HttpOperationResponse.type`), whose `@friendlyName` survives — e.g.
`PagePaginatedResponse<Meter>` → `MeterPagePaginatedResponse`. This reuses the
already-emitted, already-guarded paginated interfaces (no synthesis). Net: ~70/81
responses wired to interfaces, 10 void, 1 text (CSV) — none fall back to
`z.output<typeof schemas...>` now that a directly-returned named union resolves
to its own alias instead.

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
(it isn't part of the `templates/tests/` set `runtime-templates.ts` emits), so
it lives directly in `tests/` and is not re-emitted by `generate` — do not
delete it expecting a regen to restore it.

### Request types: direct TS, input-variant interfaces

Request/response types are direct TS, not `z.input`/`z.output`, and live in
`models/operations/<ns>.ts` (re-exported from the barrel under their existing
public names). The split mirrors the model types:

- **Response** → the documented output interface (a model or, since named
  unions are aliased too, a union like `App`), `void`, or `string` (text/CSV).
- **Request body** → the body model's (or named union's) interface, or its
  **`…Input` variant** when the body's input shape diverges from its output. A
  model diverges iff a defaulted field — anywhere in its reachable subtree —
  flips from required (output) to optional (input); `computeDivergentModels` (in
  `input-variants.ts`) is the transitive fixpoint. A union diverges iff at least
  one of its own variants is a divergent model (`computeDivergentUnions`, same
  file — shallow, not transitive, since a union carries no properties of its
  own). `interface-types.ts` emits an `XInput` interface/alias (relaxed
  optionality, refing child `YInput` variants) for each divergent model or union
  — e.g. `create-customer-charges`'s body resolves to the union alias
  `CreateChargeRequest`. ~12 request bodies diverge directly; their closure is
  ~51 `…Input` interfaces.
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

### Pagination companions (`<method>All`)

Every page-number or cursor **list** operation gets a companion facade method
— `<method>All` alongside `<method>` (e.g. `client.meters.listAll()` next to
`client.meters.list()`) — that returns `AsyncIterable<Item>` and fetches
following pages lazily as the iterable is consumed. This is purely additive:
existing `list()`/`funcs.listX()` signatures and behavior are untouched; only
the facade layer (`sdk-files.ts`) gains the extra method. No standalone-func
equivalent is emitted — the companion is facade-only, matching the "thin
codegen, shared runtime" split below.

**Detection is structural, by AST node identity, not by name.** Both
pagination styles are TypeSpec generic response templates in
`shared/responses.tsp`: `Shared.PagePaginatedResponse<T>` (`meta:
Common.PageMeta`, i.e. `{ page: { number, size, total } }`) and
`Shared.CursorPaginatedResponse<T>` (`meta: Common.CursorMeta`, i.e. `{ page:
{ next?, previous?, first?, last?, size? } }`). `pagination.ts` resolves
these two template declarations once per emit
(`program.getGlobalNamespaceType().namespaces.get('Shared')`, then
`.models.get('PagePaginatedResponse'|'CursorPaginatedResponse')`) and matches
each operation's success response envelope (`successResponseEnvelope`,
exported from `sdk-operations.ts`) against them by `.node` identity — every
instantiation of a TypeSpec generic model shares the declaration's syntax
node, so this is exact regardless of the instantiation's own
(`@friendlyName`-interpolated) name. `getPagingOperation`/`@pageItems` from
`@typespec/compiler` was evaluated and rejected: in this spec `@pageItems` is
the only paging decorator actually used, so it can confirm "this operation is
paginated" but cannot distinguish the two styles — node identity subsumes it
and is the only structural signal that does distinguish them. The item type
`T` comes from `envelope.templateMapper.args[0]`, resolved to its documented
interface name via the same `resolveInterface` every other response uses — an
item type with no documented interface (should never happen for a real list
op) gets no companion rather than an untyped one. `Shared` is looked up by
name because TypeSpec has no other way to name "the two templates this
emitter builds pagination around" (same precedent as `SPLIT_BY_INTERFACE`);
the per-operation match itself is never name-based.

**Runtime helpers, not per-operation loop bodies.** The iteration logic lives
once in `templates/runtime/paginate.ts` → generated `src/lib/paginate.ts`:
`paginatePages` advances `request.page.number`, stopping on a page shorter
than the server's own reported `meta.page.size` (including an empty page) or
once the running item count reaches `meta.page.total`; `paginateCursor`
follows `meta.page.next` — an **opaque cursor token** fed back verbatim as
`page.after` (despite `next`/`previous`/`first`/`last` carrying a `format:
uri` annotation in the spec — confirmed against the server's own handlers,
e.g. `api/v3/handlers/customers/credits/list_transactions.go`: "We
intentionally expose opaque cursor tokens instead of URI links" — do not
"fix" this by having the helper fetch `next` as a URL). Both helpers accept a
generic `fetchPage: (req, options) => Promise<Result<Envelope>>` and unwrap
each page internally (facades throw `HTTPError`, matching every other
facade method), cap iteration at `MAX_PAGINATION_PAGES` (10,000) and throw
`PaginationLimitExceededError` rather than loop forever on a misbehaving
server (mirroring `DepthLimitExceededError` in `wire.ts`), and forward the
caller's `RequestOptions` (including `signal`) to every page fetch. The
generated companion only wires the right helper to the right func, binding
`this._client` in a closure — `sdk-files.ts`'s `emitPaginationMethod`; no
per-operation loop code is emitted. `PaginationLimitExceededError` is
exported from the package root (`indexFile` in `sdk-files.ts`) alongside the
other typed runtime errors.

Coverage: `paginate.ts` joins `wire.ts` in the generated package's
`vitest.config.ts` coverage `include` at the same 100%
statement/function/line threshold (85% branch, matching `wire.ts`) — it has
no compile-time guard either, so its behavior must be covered entirely by
`tests/paginate.spec.ts` (both helpers: multi-page iteration, early-break
fires no extra requests, empty/short/exact-total page termination, absent-
next-cursor termination, filter/sort/page-size preserved across pages,
`AbortSignal` propagation, and the `PaginationLimitExceededError` cap for
both styles — the cap tests drive `paginatePages`/`paginateCursor` directly
with an in-memory stub `fetchPage`, not through `fetch-mock`, so 10,000
iterations stay fast).

### Method/function JSDoc

Every emitted facade method (`sdk/*.ts`) and standalone function (`funcs/*.ts`)
carries a JSDoc comment, built by `operationJsDoc` in `sdk-operations.ts`: the
`@summary` decorator text (`SdkOperation.summary`, short one-liner) followed by
the `@doc` description body (`SdkOperation.doc`, longer prose) when it differs
from the summary, and always a final line naming the HTTP route
(`POST /openmeter/meters`). The route line is unconditional, so every operation
gets a useful IDE hover even the rare one with neither a TypeSpec `@doc` nor a
`@summary` — the generator never emits a hollow JSDoc block. Summary and
description appear only when the TypeSpec source declares them; the generator
never fabricates prose, so a method whose JSDoc lacks a description is a
spec-authoring gap (add `@doc` to the operation), not an emitter bug. `*Input` variant
interfaces in `models/types.ts` (`interface-types.ts`) inherit the base
interface's doc comment verbatim (no doc on the base → none on the variant).
The shared `jsdoc()` helper (`utils.tsx`) escapes any literal `*/` in
doc/summary text so it cannot prematurely close the emitted comment; do not
bypass this helper when adding new doc-emitting call sites.

### README

`readme.ts` emits the package `README.md` at the package root (`emitter-output-dir`
is the package root, so non-`src/` paths land there; `package.json`/`tests/`
survive because `writeOutput` only writes listed paths). It is built from the
same grouped `SdkOperation[]` as the SDK files, in `groupOperations` insertion
order (matching `index.ts`), so the "Available Resources and Operations" table's
call paths (`getter` + `nestPath` + `methodName`, e.g. `customers.credits.grants.create`),
HTTP routes, and per-op summaries (`$.type.getDoc(op)`, carried on `SdkOperation.doc`)
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
stack) live under `typespec-typescript/templates/tests/` and are emitted into
the generated SDK by `runtime-templates.ts`. They are the generator's spec: it
is "done" when these tests pass against the emitted `aip-client-javascript`
output.

`pnpm run test:sdk` roots at `packages/aip-client-javascript` and runs the
**generated** tests against the **generated** SDK, so `generate` followed by
`test:sdk` is fully self-contained. The generated package is never
hand-edited; to change the runtime or tests, edit the files under
`typespec-typescript/templates/` and re-run `generate` (see the layout note
above).

Vitest strips types without checking them, so the package `typecheck` script
runs twice: `tsc --noEmit` (the build tsconfig, `src/` only, keeps declaration
diagnostics) and `tsc -p tsconfig.tests.json` (adds `tests/`, no emit,
`skipLibCheck` because `@fetch-mock/vitest`'s own d.ts imports the undeclared
jest `expect` package). Without the second run, test files are never
type-checked by any gate — type-level probes placed in `tests/` prove nothing.
`tsconfig.tests.json` is hand-maintained at the package root (like
`package.json`/`vitest.config.ts`, it survives regeneration) and is
`.npmignore`d.

The meters namespace is behaviorally verified end-to-end by these 19 tests. The
other namespaces are generated and type-checked (`tsc` clean across all 13) but
not yet behaviorally tested — add a smoke test per namespace if broader runtime
coverage is wanted.

### Emitter-level tests (in-memory compile harness)

`typespec-typescript/test/emit.ts` builds an `EmitterTester` with
`createTester` from `@typespec/compiler/testing`: it compiles a fixture
TypeSpec program in-memory, runs the emitter through the compiler's real emit
pipeline, and returns the emitted files as `outputs: Record<path, content>`
(paths relative to the emitter output dir, e.g. `src/sdk/internal.ts`). Use it
to pin generator behavior that should be caught before regenerating the real
client — `test/internal-surface.test.ts` (the x-private/x-internal routing to
the `client.internal.*` surface) is the model. Constraints:

- The tester resolves the emitter by its package name through `package.json`
  exports, i.e. it runs the **built** `dist/` — the package `test` script runs
  `alloy build` first for exactly this reason. A stale manual `vitest` run
  tests stale code.
- Fixture specs must author operations via the same `extends` pattern the real
  spec uses (`interface Endpoints extends Domain.Operations {}` inside a
  `@service` namespace) or grouping falls into `ungrouped-operation`.
  Pagination detection requires a top-level `Shared` namespace declaring
  `PagePaginatedResponse`/`CursorPaginatedResponse`.
- The harness is what surfaced the unawaited-`writeOutput` race in
  `$onEmit`: the tsp CLI keeps the process alive past the pending writes, but
  in-memory compilation returns immediately, observing a partial output dir.
  Keep `writeOutput` awaited.

`make -C api/spec test` runs `pnpm --filter @openmeter/typespec-typescript run
check` (typecheck + these tests) alongside `test:sdk:coverage`, and the
`aip-npm-release` workflow runs that target before publishing.

## Go SDK emitter

### Output and wiring

- `typespec-go` emits a single-package Go SDK (`package openmeter`) into
  `api/v3/client` at the **repo root** — not under `api/spec/packages/`. It is
  a standalone nested Go module, `github.com/openmeterio/openmeter/api/v3/client`,
  with its own `go.mod`/`go.sum` (sole dependency:
  `github.com/oapi-codegen/nullable`). The root `go test ./...` never reaches
  it; use `make test-go-sdk` at the repo root.
- Wiring lives in `packages/aip/tspconfig.yaml` under `@openmeter/typespec-go`:
  `emitter-output-dir: '{output-dir}/../../../v3/client'` plus the options
  `module-path`, `package-name: 'openmeter'`, `include-services: ['OpenMeter']`,
  `strip-name-prefixes`, and `readme-note`. `sdk-version` is deliberately not
  set there, so day-to-day regeneration stamps the `0.0.0-dev` placeholder; the
  release process sets it (see Releases below). The full option surface is
  declared in `typespec-go/src/lib.ts`.
- Never hand-edit generated files in `api/v3/client`. Change `typespec-go`
  emitter components or `src/runtime-templates.ts`, then regenerate. The output
  cleaner deletes previously generated entries before emission (so file renames
  cannot leave duplicate declarations) but preserves `*_test.go` files and
  `testdata/`: hand-written Go wire tests live in `api/v3/client` alongside the
  generated files and survive regeneration.
- Grouping and nesting follow the same TypeSpec source-namespace rules as the
  TypeScript SDK. Public Go names use PascalCase fields and methods with
  `json:"snake_case"` tags; there is no runtime casing mapper.
- Static Go runtime files live as reviewable TypeScript template strings in
  `typespec-go/src/runtime-templates.ts`. Do not place Go files, `go.mod`, or
  `go.sum` under a `typespec-go/runtime/` directory; that makes the emitter
  source tree look like a standalone Go package.
- Every generated `.go` file carries the
  `// Code generated by @openmeter/typespec-go. DO NOT EDIT.` header **before**
  the package clause, and generation gofmt-formats the output (a runnable
  `gofmt` on PATH is a hard requirement of generation).

### Model projection rules

- Model emission is payload-context aware. The response reachability walk
  filters properties by `Lifecycle.Read` visibility, so create-only fields do
  not leak into read models. A model reachable only from requests emits its
  input projection under its natural name (e.g. `CreateMeterRequest`); a model
  reachable from both requests and responses emits one declaration when the
  projections agree, or a read declaration plus an `Input` twin when they
  diverge (e.g. `Event` and `EventInput`). See `src/projections.ts`.
- Structural dedupe collapses visibility-projection twins onto canonical types:
  a `Create`/`Update`/`Upsert`-prefixed declaration whose rendered shape is
  structurally identical to another emitted declaration is dropped and every
  reference is redirected to the canonical name, so read-modify-write flows
  need no type mapping (`computeStructuralAliases` in `src/projections.ts`).
- Anonymous inline models are promoted to deterministic names derived from the
  enclosing type plus field (`SubscriptionCreate.customer` →
  `SubscriptionCreateCustomer`); a promoted-name collision is a generation
  error, resolved with `@friendlyName`.
- Named `*FieldFilter` unions (the `StringFieldFilter` family) are
  runtime-backed: an exact-name map in `src/go-types.tsx`
  (`runtimeFilterTypesByUnionName`) resolves them to the static runtime filter
  types (`StringFilter`, `StringExactFilter`, `DateTimeFilter`, `NumericFilter`,
  `BooleanFilter`), and they are excluded from the model reachability walk so
  their variants never emit dead declarations. An unmapped `*FieldFilter` union
  name fails generation instead of guessing.
- Formatless TypeSpec `integer` (and `safeint`) map to `int64`; neither fits a
  narrower sized Go integer by declaration.

### Wire-shape rules

- Shared-route representations are retained when media type or body shape
  differs. Events ingest intentionally emits `Events.IngestEvent`,
  `Events.IngestEvents`, and `Events.IngestEventsJSON`, each with its own
  request `Content-Type`. Response-only siblings such as meter CSV can reuse the
  JSON request body while keeping a distinct response `Accept`.
- TypeSpec `T | null` emits value-typed `Nullable[T]` backed by
  `github.com/oapi-codegen/nullable`, not `*Nullable[T]`. Optional nullable
  fields rely on `omitempty` for the unspecified state while still preserving
  explicit `null` and concrete values on marshal/unmarshal.
- Optional maps and slices in request input models emit as pointers
  (`*map[...]...`, `*[]...`) so callers can distinguish omission from an explicit
  empty object/array. Keep this input-only through the projection rules above
  so response models remain ergonomic value maps/slices.
- Go string enum constants stay prefixed as `<Type><Value>` and every generated
  enum exposes `Valid() bool`; unknown wire values must still decode and
  re-encode unchanged for forward compatibility.
- Union wrappers are raw-preserving: `UnmarshalJSON` and `MarshalJSON` copy the
  payload with cloned buffers (`append([]byte(nil), ...)`), the zero-value
  union marshals as JSON `null`, and unknown discriminator values round-trip
  unchanged. `<Union>From<Variant>` constructors stamp the variant's
  discriminator field before marshaling, keeping request construction ergonomic
  without weakening unknown-discriminator round-tripping.
- `<List>All` iterator methods are emitted only for list responses with the
  canonical `{data, meta}` page envelope. A paginated response carrying any
  extra top-level field gets only the plain method returning the full envelope,
  because the iterator surfaces page elements alone.

### Releases

- The `sdk-version` emitter option stamps `const Version` in
  `api/v3/client/option.go` (also the default `User-Agent` version); it
  defaults to `0.0.0-dev`.
- A release is an `api/v3/client/vX.Y.Z` git tag (`-dev.N`/`-beta.N` prerelease
  suffixes are also accepted). `.github/workflows/release-go-sdk.yaml` gates
  the tag: it verifies the stamped `Version` constant matches the tag version,
  runs `make test-go-sdk`, and creates a GitHub release for visibility.
- Release steps: set `sdk-version` under the `@openmeter/typespec-go` options
  in `packages/aip/tspconfig.yaml`, regenerate (`make gen-api`), commit the
  stamped output, then push the matching `api/v3/client/vX.Y.Z` tag.

### Verification

Verify Go emitter changes with (first two from `api/spec`, third from the repo
root):

```bash
pnpm --filter @openmeter/typespec-go run check
pnpm --filter @openmeter/api-spec-aip run generate   # or: make gen-api (repo root)
(cd api/v3/client && gofmt -l . && go build ./... && go vet ./... && go test ./...)
```

`make test-go-sdk` at the repo root is the build/vet/test part of the last
line. In CI, the `generators-openapi` job runs the generated-output drift check
(`make update-openapi` + clean git diff) and the emitter's `check` script, and
the `go-sdk` job runs `make test-go-sdk`.
