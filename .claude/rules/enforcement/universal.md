# Universal Enforcement (30 rules)

Anti-patterns shipped with Archie that apply to every project regardless of stack. These come from `platform_rules.json`, not from your project's `rules.json`.

## Mechanical Violations (block)

### `erosion-god-function` — God-function: CC>15. Complexity concentrating here — SlopCodeBench shows this is the #1 agent failure mode. Split before it grows further.

*check: `complexity_threshold`*

### `decay-empty-catch` — Empty catch/except block — error silently swallowed. SlopCodeBench shows error handling degrades first while core functionality stays.

*check: `forbidden_content`*

### `security-hardcoded-secret` — Possible hardcoded secret/API key in source code.

*check: `forbidden_content`*

### `security-debug-left-behind` — Debug breakpoint left in code. Will halt execution in production.

*check: `forbidden_content`*

### `android-layer-viewmodel-context` — 

*check: `architectural_constraint`*

**Why:** ViewModel must be lifecycle-independent. Referencing Context/View from ViewModel creates memory leaks and breaks testability.

### `android-layer-fragment-network` — 

*check: `architectural_constraint`*

**Why:** Fragments must not make network calls directly. All data flows through Repository → ViewModel → Fragment.

### `android-layer-fragment-db` — 

*check: `architectural_constraint`*

**Why:** Fragments must not access persistence directly. Data layer is the repository's responsibility.

### `android-layer-activity-db` — 

*check: `architectural_constraint`*

**Why:** Activities must not access persistence directly.

### `android-lifecycle-globalscope` — GlobalScope ignores lifecycle — coroutines leak on configuration change or process death. Use viewModelScope, lifecycleScope, or inject a supervised scope.

*check: `forbidden_content`*

### `swift-layer-view-network` — 

*check: `architectural_constraint`*

**Why:** SwiftUI Views must not make network calls. Data fetching belongs in ViewModel or Repository.

### `swift-layer-view-userdefaults` — 

*check: `architectural_constraint`*

**Why:** Views must not access persistence directly. Use a repository or data manager.

### `typescript-react-dom-manipulation` — 

*check: `architectural_constraint`*

**Why:** Direct DOM manipulation in React components breaks the virtual DOM model and causes subtle rendering bugs.

### `python-safety-bare-except` — Bare except catches SystemExit, KeyboardInterrupt, and GeneratorExit. Use except Exception: at minimum.

*check: `forbidden_content`*

### `python-safety-eval-exec` — eval/exec executes arbitrary code — critical security risk if input is user-controlled.

*check: `forbidden_content`*

## Tradeoff Signals (warn)

### `erosion-growing-complexity` — Complex function approaching god-function territory (CC>10). Track this — if CC grows between scans, it's eroding.

*check: `complexity_threshold`*

### `erosion-god-class` — Class has 20+ methods — likely accumulating responsibilities. Agents patch new logic here instead of creating focused classes.

*check: `size_threshold`*

### `erosion-monster-file` — File exceeds 600 lines. In agent-assisted codebases, large files grow because agents append rather than refactor.

*check: `size_threshold`*

### `erosion-many-params` — Function with 7+ parameters. Signal of a function doing too much or missing a data class/struct.

*check: `forbidden_content`*

### `decay-disabled-test` — Disabled/skipped test. Tests get disabled when agents can't fix them — this hides regressions. The paper shows error-mode tests fail first.

*check: `forbidden_content`*

### `decay-todo-fixme-hack` — FIXME/HACK/XXX marker — acknowledged technical debt. Track these: if count grows between scans, quality is degrading.

*check: `forbidden_content`*

### `decay-catch-log-only` — Catch block only logs the error without re-throwing or propagating. Error is visible in logs but callers don't know it failed.

*check: `forbidden_content`*

### `android-di-service-locator` — Service locator pattern (KoinJavaComponent.get) breaks constructor injection and makes dependencies invisible. Use constructor injection via Koin modules.

*check: `forbidden_content`*

### `swift-safety-force-unwrap` — Force unwrap (!) crashes at runtime if nil. Use guard let, if let, or ?? default.

*check: `forbidden_content`*

### `swift-safety-force-try` — Force try crashes at runtime if throwing function fails. Use do/catch or try?.

*check: `forbidden_content`*

### `typescript-layer-component-fetch` — 

*check: `architectural_constraint`*

**Why:** Components should not fetch data directly. Use hooks, services, or state management (React Query, SWR, etc.).

### `typescript-safety-any-type` — Type escape via 'any' — defeats TypeScript's type system. Use unknown, generics, or proper types.

*check: `forbidden_content`*

### `typescript-react-index-key` — Array index as React key causes rendering bugs when list items are reordered, inserted, or deleted.

*check: `forbidden_content`*

### `python-safety-mutable-default` — Mutable default argument (list/dict) — shared across all calls. Use None default with assignment in body.

*check: `forbidden_content`*

### `python-layer-star-import` — Star import pollutes namespace and hides dependencies. Makes it impossible to trace where a symbol comes from.

*check: `forbidden_content`*

### `python-layer-circular-import` — TYPE_CHECKING guard suggests circular import was encountered. The cycle should be resolved structurally, not worked around.

*check: `forbidden_content`*
