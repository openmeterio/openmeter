# Emit Format — httpYac `.http` files

How to emit an httpYac scenario file from an NL spec scenario.
Runner-aware, format-strict, generator-friendly.

This reference is the format contract for the `e2e-httpyac` skill.
Worked examples of every shape class live in `examples.md`.

---

## Goal

A `.http` file that:

1. Reads top-to-bottom as the scenario's narrative — every request
   region maps 1:1 to a numbered Step in the NL spec.
2. Is **executable as-is** by `httpyac send` against a live OpenMeter
   server, with no helper scripts, no `package.json`, no fixture
   loaders.
3. Produces clean JUnit / JSON output where each `r__` request is its
   own reporter entry, naming the scenario id and step verbatim.
4. Is mechanically traceable back to the NL spec — the emitted file's
   request names embed the scenario id, and the file lives at
   `e2e/http/<family>/<scenario_id>.http`.

---

## File layout

```http
# <scenario id>
#
# Source: e2e/specs/<family>.md ## Scenario: <scenario id>
# (any NEEDS-VERIFY markers from the spec, copied here verbatim)

@api_base = {{process.env.OPENMETER_ADDRESS}}/api/v3
@d__<scenario>__<key1> = <literal or {{$dynamic}} value>
@d__<scenario>__<key2> = …

###
# @name r__<scenario>__<step1_slug>
# @title <step 1 title from NL spec>
<METHOD> {{api_base}}<path>
<headers>

<body>

?? <assertion 1>
?? <assertion 2>
{{
  exports.v__<scenario>__<capture> = response.parsedBody.<path>;
}}

###
# @name r__<scenario>__<step2_slug>
# @title <step 2 title>
# @ref r__<scenario>__<step1_slug>
…
```

Constraints:

- **First content block** is a comment naming the scenario id and the
  spec source path.
- **File-global aliases** (`@<name> = <value>`) live before the first
  `###`. `@api_base = {{process.env.OPENMETER_ADDRESS}}/api/v3` is
  always present. **`OPENMETER_ADDRESS` stays the bare host** (e.g.
  `http://localhost:8888`) to match the `/e2e` Go suite's convention
  exactly. The version prefix `/api/v3` lives in the `.http` file —
  not in the env var — so a single shell-exported `OPENMETER_ADDRESS`
  works for both runners.
- **One request region per Step** in the NL spec. Don't merge or
  split steps relative to the spec.
- **Request region order** matches Step order. httpYac runs requests
  in file order, and captures only flow forward.

---

## Naming convention (hard rule)

| Prefix | Used for | Pattern |
|---|---|---|
| `r__` | `# @name` for a request region | `r__<scenario>__<step_slug>` |
| `v__` | Exported variable from a post-response script | `v__<scenario>__<capture_name>` |
| `d__` | File-global immutable data variable | `d__<scenario>__<field_name>` |
| upper-snake | Environment variable from `process.env` | `OPENMETER_ADDRESS`, `OPENMETER_TOKEN` |

`<scenario>` is the spec's `id`, lowercased and snake-cased.

**Step slug derivation rule** (deterministic, so family-regenerate
is reproducible):

1. Take the NL spec Step's bold leading phrase (the text between
   `**` markers at the start of the step, e.g.
   `**Create feature.**` → `Create feature`).
2. Lowercase, replace any spaces with `_`, drop trailing
   punctuation, strip articles (`a`, `the`).
3. Compress to one or two words — drop generic verbs/objects when
   the meaning stays clear (`Create feature` → `create_feature`;
   `Get feature after deletion` → `get_after_delete`).
4. For matrix scenarios, prefix with `row<N>_` and suffix with
   the row's outcome verb (`row3_negative_amount_rejected`,
   `row1_manual_baseline_succeeds`).

Same NL spec → same slug. The skill is allowed to ask the user for
disambiguation when two steps would produce identical slugs, but
this should be vanishingly rare for sane spec authoring.

All names are valid JavaScript identifiers — letters, digits,
underscores only. No hyphens. No dynamic content in names; dynamic
values (`$timestamp`, `$uuid`) appear in **data values**, not in
identifiers.

Why this is non-negotiable:

- httpYac's `@name` registry has no scope across imports — first
  match wins. Without scenario-prefixed names, two files importing a
  shared baseline can silently capture each other's responses.
- Script variables live on the global scope of the script context —
  variable names must be unambiguous across all files in a single
  `httpyac send` invocation.
- The `r__` / `v__` / `d__` distinction makes a file grep-able: a
  reviewer can scan one file and see request boundaries, runtime
  state, and fixture data at a glance.

---

## Capture pattern — explicit `exports`, not implicit `@name`

httpYac's `@name` captures the parsed JSON body of a response and
exposes it under `{{<name>.<field>}}`. This is **insufficient** for
this skill: the body alone doesn't carry headers, status, or computed
values, and the implicit binding is awkward to debug when something
goes wrong.

**Always emit an explicit post-response script** for any captured
value:

```http
###
# @name r__feature_lifecycle__create_feature
POST {{api_base}}/openmeter/features
Content-Type: application/json

{ "key": "{{d__feature_lifecycle__feature_key}}", "name": "Test Feature" }

?? status == 201
{{
  exports.v__feature_lifecycle__feature_id = response.parsedBody.id;
}}
```

Rules:

- **The newline after `{{` is required.** httpYac distinguishes
  `{{var}}` (substitution) from `{{ block }}` (script) by the line
  break — emitting `{{exports.foo = …;}}` on one line silently turns
  the whole thing into a substitution attempt.
- **Capture only what later steps reference.** Don't pre-emptively
  export every field — emit one `exports.v__...` per field that
  appears in a downstream `{{v__...}}` substitution or assertion.
- **Use `response.parsedBody`** (auto-parsed JSON) for body fields.
  Use `response.headers` for header values. Use `response.statusCode`
  for status (rarely needed for captures; assertions cover it).
- **No async work.** If a capture genuinely requires async, export
  the Promise (`exports.v__... = (async () => {...})();`); but for
  v1, every emitted scenario stays synchronous.

### `?? js` is for static expressions only — use `test()` for any runtime value

httpyac 6.16.7's `?? js` line parser is **brittle around runtime
values**. Two failure modes verified against the runner:

1. **`{{d__...}}` substitution inside `?? js`.** Data variables
   resolve correctly in request bodies, headers, URLs, and query
   strings, but `?? js
   response.parsedBody.detail.includes('with key
   {{d__scn__feature_key}} already exists')` raises `SyntaxError:
   Invalid or unexpected token`. In `?? js` context httpyac evaluates
   the contents of `{{...}}` as a JS expression rather than a string
   substitution, and `d__...` has no JS-scope binding (only
   `exports.v__...` values do).

2. **JS `+` concatenation with an exported `v__`.** Even rewriting
   the assertion to `?? js
   response.parsedBody.detail.includes('with key ' +
   v__scn__captured_key + ' already exists')` — which **is** valid
   JS — still throws `SyntaxError`. The `?? js` parser doesn't reliably
   forward multi-operator JS expressions into the underlying
   evaluator.

**Rule.** Reach for `?? js` only when the assertion's right-hand side
is a **constant literal** (status code, fixed string, fixed number).
Any time a runtime-captured value participates in the comparison —
exported `v__`, parsed body field substring containing the captured
value, computed expression — emit a JS `test()` block instead. This
is the format's documented fallback (see "Relationship / nested /
array checks → JS `test()`" below); the brittleness above is what
makes that fallback non-optional rather than just stylistic.

Canonical pattern when the assertion needs to mention a runtime
value:

```http
###
# @name r__scn__create_feature
POST {{api_base}}/openmeter/features
Content-Type: application/json

{ "key": "{{d__scn__feature_key}}", "name": "Test Feature" }

?? status == 201
{{
  // Echo the data-var into a JS-accessible v__ for downstream
  // assertions; {{d__...}} does not resolve in ?? js context.
  exports.v__scn__captured_key = response.parsedBody.key;
}}

###
# @name r__scn__check_detail_mentions_key
# @ref r__scn__create_feature
…
?? status == 409
{{
  const { ok } = require('assert');
  test('detail mentions the duplicate key', () => {
    const detail = response.parsedBody.detail || '';
    const key = v__scn__captured_key;
    ok(
      detail.includes('with key ' + key + ' already exists'),
      'expected detail to contain "with key <key> already exists", got: ' + detail
    );
  });
}}
```

`?? js` *does* work fine when the substitution resolves to a real JS
expression that's effectively constant at parse time — `?? js
response.parsedBody.id == {{v__scn__feature_id}}` (where the right
side is just a bare ULID with no spaces or operators) is consistently
reliable. The brittleness shows up when `?? js` has to forward more
than one operator or a string-concat chain.

---

## Assertion vocabulary

Every Step in the NL spec maps to a fixed set of `??` lines and at
most one `{{ test(...) }}` block. Pick by **shape of the assertion**,
not by intent.

### DSL-first principle

**Prefer the `??` DSL over JS `test()` blocks whenever an idiom
exists.** The DSL is denser, more grep-able, and the JUnit reporter
treats each `??` line as a separate test case (a `test()` block
collapses to one entry per `test(...)` call). Reach for `test()`
only when the DSL genuinely can't express the check — typically:

- array membership / length / ordering,
- multi-field invariants (e.g. "field A and field B both present"),
- assertions that need conditional JS logic,
- structurally-loose checks for `NEEDS-VERIFY` markers,
- distinguishing `undefined` from `null` from `false` (where the
  JS-side semantics matter).

Order of preference, top to bottom:

1. **Plain `??`** (status, body substring, header, duration, hash).
2. **`?? js <expr> <op> <literal>`** (any field whose value is a
   primitive, including via `typeof`).
3. **`{{ test(...) }}` JS block** as a fallback for anything the
   above can't express.

### Cheap declarative checks → `??`

| NL phrasing | Emitted line |
|---|---|
| `Expect 201 Created` | `?? status == 201` |
| `Expect 204 No Content` | `?? status == 204` |
| `Expect detail equals "<full>"` | `?? js response.parsedBody.detail == <full>` |
| `Expect detail contains "<sub>"` | `?? js response.parsedBody.detail.includes('<sub>')` ([see warning](#why-not-body-contains)) |
| `Expect <field> equals <literal>` (string/number) | `?? js response.parsedBody.<field> == <literal>` |
| `Expect <field> is absent` | `?? js typeof response.parsedBody.<field> == undefined` |
| `Expect <field> is null` | **JS `test()` block** — see "Null and strict-equality" below |
| `Expect <field> is falsy` (any of undefined/null/false/0/"") | `?? js response.parsedBody.<field> isFalse` |

Prefer `?? js response.parsedBody.<path> == <value>` over
`?? body contains "<value>"` whenever the field is structured —
substring matches on serialized JSON are brittle (decimal
normalization, whitespace, key ordering).

#### Why not `body contains`?

`?? body contains "<sub>"` is **unreliable on httpyac 6.16.7 for
`application/problem+json` response bodies**. Even when the
substring is verbatim inside the response `detail`, the assertion
throws an internal stack frame at `httpyac/dist/index.js:2:48611`
rather than evaluating cleanly. The call site fires for every
affected row, so it's a code-path quirk, not data-dependent.

**Always reach for `?? js response.parsedBody.<field> == <value>`
or `.includes('<sub>')` instead.** The DSL-native form is more
precise (operates on the parsed structured field, not raw bytes),
DSL-first principle aligned, and the JUnit reporter shows one
entry per `??` line. `body contains` stays in the table for
genuinely unstructured bodies (HTML, plain text, non-`+json`
content types) — but those are rare in this API.

### How the absence idiom works

`?? js typeof response.parsedBody.<field> == undefined` works because
**both sides are strings** at compare time:

- LHS: `typeof <expr>` evaluates JS-side and returns the string
  `"undefined"` when the property is missing.
- RHS: `undefined` is a 9-character literal, parsed as the string
  `"undefined"`.
- httpYac compares string `"undefined"` to string `"undefined"`. ✓

This is the DSL-native idiom for absence and avoids a `test()`
block. The semantics are slightly looser than `assert.strictEqual(x,
undefined)` — `typeof` returns `"undefined"` for both undeclared
variables and explicitly-undefined properties — but for response
bodies that distinction never matters, so the loose form is safe.

### Null and strict-equality checks → JS `test()`

The DSL has no clean idiom for "field is exactly `null`". Writing
`?? js response.parsedBody.<field> == null` is unsafe: httpYac's
parser likely treats the right side as the string `"null"`, in
which case the JS comparison `null == "null"` is `false` and the
assertion silently fails. Use a `test()` block:

```http
{{
  const { strictEqual } = require('assert');
  test('field is null', () => {
    strictEqual(response.parsedBody.<field>, null);
  });
}}
```

Same fallback applies any time you need to distinguish `undefined`
from `null`, or assert anything `===`-strict that the DSL can't
express. The forbidden directives table flags `?? js <expr> ===
<value>` directly because httpYac's parser splits operators on the
first `==` and leaks the trailing `=` into the expected literal,
making the assertion fail even when the underlying values match.

### Relationship / nested / array checks → JS `test()`

```http
{{
  const { ok, strictEqual } = require('assert');
  test('<human-readable assertion>', () => {
    const x = response.parsedBody.<path>;
    ok(<condition>, '<failure message>');
    strictEqual(<actual>, <expected>);
  });
}}
```

Use a JS block when:

- An assertion checks an array (membership, length, ordering).
- An assertion combines multiple fields (mutual presence, derived
  equality).
- An assertion is conditional (a value is acceptable as either form
  X or form Y, like `cost: "0"` or `cost: null`).
- An assertion has a `NEEDS-VERIFY` marker — emit a structurally
  loose check rather than a literal match.

Don't use a JS block for a single status-or-substring check. Don't
use a JS block to set up state for the next request — that's what
post-response `exports` are for.

---

## Error-shape templates

Every 4xx step in the NL spec names one of three shapes. The mapping
is mechanical.

### Detail substring or equality — `problem.detail`

NL spec (substring):
```
- Expect 404 Not Found.
- Expect detail contains "feature not found: {feature.id}".
```

Emit (substring):
```http
?? status == 404
?? js response.parsedBody.detail.includes('feature not found: {{v__<scenario>__feature_id}}')
```

NL spec (equality — preferred when the detail is fully
deterministic, e.g. the OpenMeter framework's
`"validation error: <bare msg>"` wrapper):
```
- Expect 400 Bad Request.
- Expect detail equals "validation error: manual unit cost amount must be non-negative".
```

Emit (equality):
```http
?? status == 400
?? js response.parsedBody.detail == validation error: manual unit cost amount must be non-negative
```

Notes:

- The spec's `{feature.id}` placeholder maps to the corresponding
  `{{v__<scenario>__feature_id}}` exported variable. If the spec
  references a fixture-bound id rather than a captured one, map to
  the corresponding `{{d__...}}` data variable.
- **Do not use `?? body contains "<sub>"`** — see the warning in
  "Cheap declarative checks → `??`" above. Use
  `?? js response.parsedBody.detail.includes('<sub>')` for
  substring matching, or `?? js response.parsedBody.detail ==
  <full>` for full equality. The `==` form parses everything to
  the right of `==` (until end of line) as a string literal, so
  no quoting is needed for the right-hand side.

### Domain code — `extensions.validationErrors[].code`

NL spec:
```
- Expect 400 Bad Request.
- Expect domain code "currency_invalid".
```

Emit:
```http
?? status == 400
{{
  const { ok } = require('assert');
  test('domain code currency_invalid is present', () => {
    const errors = (response.parsedBody.extensions || {}).validationErrors || [];
    ok(
      errors.some(e => e.code === 'currency_invalid'),
      'expected validationErrors[].code to include "currency_invalid", got: ' + JSON.stringify(errors)
    );
  });
}}
```

### Schema rule — `invalid_parameters[].rule`

NL spec:
```
- Expect 400 Bad Request.
- Expect schema rule "format" on field referencing "meter_id".
```

Emit:
```http
?? status == 400
{{
  const { ok } = require('assert');
  test('invalid_parameters references meter_id with format rule', () => {
    const ips = response.parsedBody.invalid_parameters || [];
    ok(ips.length > 0, 'invalid_parameters must be non-empty');
    ok(
      ips.some(p => (p.field || '').includes('meter_id') && p.rule === 'format'),
      'expected invalid_parameters[] entry with field~"meter_id" and rule="format", got: ' + JSON.stringify(ips)
    );
  });
}}
```

If the NL spec marks the rule string `NEEDS-VERIFY`, drop the
`p.rule === 'format'` condition and assert only field presence —
plus a `# NEEDS-VERIFY: <reason>` comment line above the JS block.

#### DSL-first form for single-element arrays

When the response consistently has exactly one
`invalid_parameters[]` entry (which is the common case for current
OpenMeter rejection responses), prefer the DSL-first form with
indexed access — terser, one JUnit entry per assertion, no
`test()` block needed:

```http
?? status == 400
?? js response.parsedBody.invalid_parameters[0].field == <field>
?? js response.parsedBody.invalid_parameters[0].rule == <rule>
```

Use the `test()` + `some()` form (above) when the response can
return multiple entries and ordering isn't guaranteed.

#### Common binder-rejection patterns spec authors should expect

When the NL spec pins a `field` or `rule` value and a real run
surfaces something different, the deviation is usually one of two
recurring patterns. Both arise from how TypeSpec compiles to the
OpenAPI schema the binder layer enforces.

**Pattern 1 — TypeSpec `union` → OpenAPI `anyOf`.** If the API
field is typed as a TypeSpec `union` (e.g., a filter that accepts
either a bare ULID or an `{eq?, neq?, ...}` operator object), the
generated schema uses `anyOf`. When the input matches no branch,
the binder reports `rule: "anyOf"` — *not* the more specific
format/pattern rule that an individual branch carries. The binder
can't tell you which branch was closest; that's by design of
`anyOf`.

If the spec guessed `"format"` or `"pattern"` for a union-typed
field and the run shows `"anyOf"`, this is the cause. Pin
`"anyOf"` and add a one-line note in the scenario's Notes section.

**Pattern 2 — umbrella-field binding for unknown sub-operators.**
For deep-object query params (e.g.
`filter[meter_id][zz]=<value>`), an unknown sub-operator (`zz`)
binds the rejection at the **umbrella query parameter** (`field:
"filter"`), not at the sub-key (`field: "meter_id"`). The
offending sub-key surfaces only in `reason` (a string like
`"filter[meter_id][zz]: unsupported operator"`).

If the spec assumed `field: "meter_id"` and the run shows `field:
"filter"`, this is the cause. Pin `field: "filter"` plus a
`?? body contains "<sub-key reference>"` (or
`?? js response.parsedBody.invalid_parameters[0].reason.includes(...)`)
to anchor the assertion to the right sub-key.

Both patterns also belong in the `/e2e-nl` skill's spec-authoring
guidance — once the spec gets them right at draft time, the emit
side becomes mechanical.

---

## Matrix scenarios — N regions, not a loop

The NL spec's matrix shape uses a Markdown rows table. Each row
becomes its own request region. **Do not emit `# @loop`** — it
collapses every row into a single reporter entry, masks per-row
failure details, and has historical bugs in the JUnit emitter.

Pattern:

```http
# Shared prerequisite (created once, referenced by every row that needs it)
###
# @name r__<scenario>__create_shared_<dep>
…

###
# @name r__<scenario>__row1_<descriptive_slug>
# @title row 1: <row's intent> → <expected outcome>
<request>
?? status == <row's expected status>
[?? js response.parsedBody.detail == <full detail> if 4xx with detail-equality shape]
[?? js response.parsedBody.detail.includes('<sub>') if 4xx with detail-substring shape]
[?? js response.parsedBody.invalid_parameters[0].field == <field> if 4xx with schema-rule shape]
[?? js response.parsedBody.invalid_parameters[0].rule == <rule> if 4xx with schema-rule shape]

###
# @name r__<scenario>__row2_<descriptive_slug>
…
```

Rules:

- Row slugs are descriptive, not numeric — `row3_negative_amount_rejected`,
  not `row3`. The JUnit reporter shows the slug; numeric-only row
  names tell the reader nothing on failure.
- Shared prerequisites (a meter the matrix re-uses, an addon the
  matrix mutates against) are created once at the top of the file
  and referenced via `# @ref` on each row that needs them.
- Heterogeneous bodies are written out fully per row. Do not try
  to share a base body and merge per-row deltas — JSON's lack of
  trailing-comma tolerance turns conditional fields into a
  template-engine nightmare.
- If the NL spec lists a row without an explicit detail substring
  (a 2xx success row), emit only `?? status == 201` (or whatever
  the row expects). Don't invent assertions.

The cost is verbosity — a 10-row matrix is ~150 lines of `.http`.
The benefit is per-row clarity in failure reports and a 1:1 mapping
between NL spec rows and emitted regions, which makes spec drift
trivial to spot.

---

## Skipped scenarios

NL spec scenarios with `status: skipped` in their YAML frontmatter
become `# @disabled` regions plus a comment explaining the gate.
Every step is still emitted — the skill never silently drops
content — but every region carries `# @disabled`.

```http
# Source: e2e/specs/<family>.md ## Scenario: <id>
#
# STATUS: SKIPPED — <reason copied verbatim from the spec>
# Re-enable by removing the @disabled lines once <gate condition> is met.

###
# @disabled
# @name r__<scenario>__<step>
…
```

`@disabled` is static. Do not emit conditional disable expressions
(`# @disabled !{{env_seed_present}}`) — they're brittle and tend to
silently re-enable on env changes.

---

## NEEDS-VERIFY markers

When the NL spec annotates a row or step with `NEEDS-VERIFY: <reason>`,
emit:

1. A comment in the `.http` directly above the relevant assertion(s),
   copying the `NEEDS-VERIFY:` text verbatim.
2. A **structurally loose** assertion that pins what's verifiable
   (field presence, status code) without locking in the un-pinned
   detail.

Example — schema rule with un-pinned `rule` string:

```http
###
# @name r__<scenario>__<row_slug>
GET {{api_base}}/openmeter/features?filter[meter_id][eq]=not-a-ulid

?? status == 400
# NEEDS-VERIFY: confirm exact `rule` string against a live server.
# Tighten the JS block below once pinned.
{{
  const { ok } = require('assert');
  test('invalid_parameters references meter_id', () => {
    const ips = response.parsedBody.invalid_parameters || [];
    ok(ips.length > 0, 'invalid_parameters must be non-empty');
    ok(
      ips.some(p => (p.field || '').includes('meter_id')),
      'expected invalid_parameters[] entry referencing meter_id'
    );
  });
}}
```

When the user pins the value via a real run, they (or a follow-up
skill invocation) tighten the assertion and remove the comment.

---

## Cleanup ordering

The NL spec is the source of truth on cleanup. If the spec includes
explicit delete steps (e.g. the lifecycle scenario's "Delete
feature"), emit them in the order the spec lists them. If the spec
doesn't include cleanup, **don't invent it** — the OpenMeter e2e DB
is shared and unique-suffixed keys make orphaned records benign.

If a scenario creates resources for setup but doesn't clean them up
in the spec (a common pattern for matrix rows), emit the file as-is.
A future cleanup convention can be added without revisiting every
existing file.

---

## Forbidden directives (v1)

These are off-limits in emitted files. They either break in CLI
mode, introduce non-determinism, or have unresolved reporter bugs.

| Directive | Why forbidden |
|---|---|
| `# @loop` | Collapses matrix rows into a single reporter entry; historical JUnit emitter bugs. Use N regions instead. |
| `# @disabled <expression>` (dynamic) | Conditional disabling tends to silently re-enable on env changes. Use static `@disabled` only. |
| `# @note` | Pops a confirmation dialog in editor mode; behavior in CLI is undefined. |
| `# @openWith`, `# @save`, `# @extension` | Editor-only output controls; ignored or warning in CLI. |
| `$input`, `$password`, `$pick` | Interactive prompts; deadlock CLI runs. |
| `exports.$cancel`, `$httpyac.cancel(…)` | Dynamic cancellation; surprises reporters. |
| `# @import <other-scenario>.http` | Cross-scenario state leaks via the shared `@name` namespace. Each scenario file is self-contained. |
| `@<var> := <expr>` (lazy) | Lazy variables resolve at last-possible moment, which makes debug-by-grepping harder. Use fixed `@<var> = <value>` plus explicit `exports`. |
| `?? js <expr> === <value>` (triple-equals) | httpYac's `??` DSL parses the right side as a literal, so `=== undefined` becomes `== "= undefined"`. Always use a JS `test()` block for strict comparisons (see "Absence and strict-equality checks"). |

`# @ref <name>` and `# @forceRef <name>` **are** allowed and
encouraged when one region depends on another's setup running first.
Use them on every region that references a `v__...` exported by an
earlier region.

---

## Project root files

The skill bootstraps these on first run if they don't exist.

### `e2e/http/httpyac.config.js`

```js
// Project-root marker for httpYac. The presence of this file scopes
// httpYac's project root to e2e/http/.
//
// Configuration is shell-env based, mirroring the Go e2e convention
// (`OPENMETER_ADDRESS=... go test ./e2e/...`). Each .http file
// references {{process.env.OPENMETER_ADDRESS}}; httpYac resolves it from
// process.env at send time. No env files, no `--env <name>`.
//
// Add response-log scrubbing here if real auth tokens ever flow
// through tests. The skill does not bake auth scrubbing into v1 —
// add it intentionally when the auth surface is settled.

module.exports = {};
```

The skill emits this file **once**. Subsequent runs leave it
untouched — it is user-edited configuration, not generated output.

---

## Emit checklist

Before declaring a scenario file done:

- [ ] First-line comment names the scenario id and spec source path.
- [ ] `@api_base = {{process.env.OPENMETER_ADDRESS}}/api/v3` is present.
- [ ] Every `# @name` follows `r__<scenario>__<step_slug>`.
- [ ] Every captured value uses `exports.v__<scenario>__<name>`,
      with the required newline after `{{`.
- [ ] Every Step in the NL spec maps to exactly one request region.
- [ ] Every 4xx step has a status assertion **and** an error-shape
      assertion matching the spec's named shape.
- [ ] No forbidden directives (see table above).
- [ ] Skipped scenarios (`status: skipped`) emit `# @disabled`
      regions, not redacted bodies.
- [ ] `NEEDS-VERIFY` markers carry over as comments + loose
      assertions.
- [ ] File parses with `httpyac send --dry-run` (when the user
      validates).
