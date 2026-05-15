### Patterns agent

> Read all source files. Analyze design patterns and communication across ALL platforms (backend AND frontend).
>
> ### 1. Structural Patterns (identify with concrete examples)
> **Backend:**
> - **Dependency Injection**: How are dependencies wired? Container? Manual? Framework? (@inject, providers, etc.)
> - **Repository**: How is data access abstracted? Interface + implementation? Active Record?
> - **Factory**: How are complex objects created?
> - **Registry/Plugin**: How are multiple implementations managed?
>
> **Frontend:**
> - **Component Composition**: How are UI components composed? HOC? Render props? Hooks? Slots?
> - **Data Fetching**: How is server state managed? React Query? SWR? Apollo? Combine? Coroutines?
> - **State Management**: Global state approach? Context? Redux? Zustand? @Observable? ViewModel+StateFlow? Bloc?
> - **Routing**: File-based? Config-based? NavigationStack? NavGraph?
>
> For each pattern found: pattern name, platform (backend|frontend|shared), implementation description, example file paths.
>
> ### 2. Behavioral Patterns
> - **Service Orchestration**: How are multi-step workflows coordinated?
> - **Streaming**: How are long-running responses handled? SSE? WebSockets? gRPC streams?
> - **Event-Driven**: Are there publish/subscribe patterns? Event buses?
> - **Optimistic Updates**: How are UI updates handled before server confirmation?
> - **State Machines**: Any explicit state machine patterns?
>
> ### 3. Cross-Cutting Patterns
> - **Error Handling**: Custom exceptions? Error boundaries? Global handler? Error mapping? What errors map to what status codes?
> - **Validation**: Where? How? What library? Client-side vs server-side?
> - **Authentication**: JWT? Session? OAuth? Where validated? How propagated to frontend?
> - **Logging**: Structured? What logger? What's logged?
> - **Caching**: What's cached? TTL strategy? Browser cache? Server cache?
>
> For each: concern, approach, location (actual file paths).
>
> ### 4. Internal Communication
> - **Backend**: Direct method calls between layers, in-process events, message buses
> - **Frontend**: Props, Context, event emitters, pub/sub, state management stores
> - **Cross-Platform**: API calls from frontend to backend, shared types/contracts
> - **Runtime boundaries** (flag whenever present): separate processes/workers/threads/iframes/subprocesses, their lifecycle (who spawns, how they die), and the protocol crossing each boundary (IPC channel names, stdio framing like NDJSON/length-prefix, RPC shape). Record the exact files where each side is implemented.
> - **Seams and invariants** (flag whenever present): abstract interfaces/classes/traits with multiple concrete implementations (note what varies vs. what's stable); any wrapper/middleware/interceptor/guard that conditionally allows, denies, transforms, or sequences other operations based on state. Wave 2 will upgrade these into key decisions — just surface the raw signal here.
>
> ### 5. External Communication
> - **HTTP/REST**: External API calls (both backend-to-external and frontend-to-backend)
> - **Message Queue**: Async job processing (Redis, RabbitMQ, etc.)
> - **Streaming**: SSE, WebSockets, gRPC streams
> - **Database**: Query patterns, transactions, ORM usage
> - **Real-time**: Push notifications, live updates
>
> ### 6. Third-Party Integrations
> List ALL external services with: service name, purpose, integration point (file path).
> Categories: AI/LLM providers, payment processors, auth providers, storage services, analytics/monitoring, CDN/asset hosting.
> Also call out any **extension protocols** — mechanisms by which third-party or user-supplied capabilities can be plugged in at runtime (plugin systems, protocol-based tool/resource loading, hook/callback registries, config-driven dispatch). For each: the protocol, the registration point, and how contributed capabilities stitch into the core's catalog.
>
> ### 7. Frontend-Backend Contract
> - How do frontend and backend communicate? (REST, GraphQL, tRPC, WebSocket, etc.)
> - Are types shared between frontend and backend?
> - How are API errors propagated to the UI?
>
> ### 8. Pattern Selection Guide
> For common scenarios in this codebase, which pattern should be used and why?
>
> ### 9. Pattern preconditions (REQUIRED — do not skip)
>
> A pattern that "looks right" in one part of a codebase may silently break in another part that has the same call shape but a different invariant (e.g. an advisory lock keyed on `(namespace, X)` where the schema's index on `(namespace, X)` is unique in one entity but non-unique in another — the same call serializes unrelated rows in the second case). The defense is to make the precondition explicit and code-grounded.
>
> For each pattern that depends on a structural invariant (schema constraint, type-system guarantee, lifecycle state, ownership rule, concurrency primitive, structural contract), populate:
>
> - `applicable_when`: the **verifiable invariant** that makes this pattern correct in THIS codebase. MUST cite a concrete code artifact at `<file>:<line>` — pick whichever invariant shape fits the language and paradigm. Common shapes:
>   - **Schema annotation:** a unique/foreign-key/NOT-NULL/index constraint that the pattern relies on
>   - **Type signature:** a function returning Result/Option/Either; an exhaustive enum or sealed type; a generic bound that constrains callers
>   - **Lifecycle / framework state:** a hook called under a required Provider; a handler registered before bus start; a composable inside a known scope
>   - **Ownership / concurrency:** borrow scope, lock-held interval, transaction-active context
>   - **Structural:** single registration point + iterating consumer; sealed hierarchy + exhaustive match; conventional placement enforced by build/lint config
>   The requirement is that the citation is **falsifiable against the corpus**, not the invariant *shape*. Do NOT use prose like "per-customer", "per-user", "in the auth flow" — those pattern-match across contexts where the invariant doesn't hold.
>
>   **Empty by default — fill ONLY when misapplying the pattern in a context where the cited invariant does NOT hold would produce silently incorrect code.** That's the openmeter lockr test: cited `(namespace, customerID)` UNIQUE index ⇒ applying to a non-unique key silently serializes unrelated rows. The failure mode is articulable in one sentence and verifiable by reading the cited file:line.
>
>   **REQUIRED SHAPE — category-then-evidence, never raw description.** The predicate must name the **class of callers, components, or situations** the invariant guards or excludes — a categorical noun phrase a future agent can pattern-match against an unfamiliar component. The citation grounds the category by showing where the invariant is declared (or where it would be violated for `do_not_apply_when`); the citation does NOT replace the category. Read every entry as: `<categorical predicate> — <file:line>: <evidence that the predicate holds / fails here>`. If you can substitute any other component name into the predicate and the sentence still parses meaningfully, the shape is right; if it only makes sense about ONE specific file, you wrote trivia, not a rule.
>
>   **Reject these as `applicable_when` (leave empty instead) — they read as fuzzy circular text:**
>     - **Pattern-name restatements**: e.g. *"A class needs to receive a dependency without constructing it directly"* (= "use DI when you want DI"); *"A feature is locked behind the 'pro' entitlement and you need to render the gated UI"* (= "use Pro gating when you want to gate by Pro"); *"Any user-facing string"* (= "use the i18n system for all strings"). Restating `when_to_use` adds nothing.
>     - **Use-case lists masquerading as preconditions**: e.g. *"A service must react to whole-app foreground/background — GPS start/stop, analytics events, RevenueCat refresh"* — that's enumerating consumers, not stating an invariant.
>     - **"Any X" universals**: e.g. *"Any ViewModel that participates in Loading/Success state"* — if the answer to "when does this apply?" is "everywhere this kind of thing exists," it's not a precondition.
>     - **Recipe pointers**: e.g. *"The destination Fragment is registered in navigation_main.xml"* (without that, the recipe doesn't even compile — it's not a precondition, it's a step). Citing a registration site is OK ONLY if there's a real misapplication failure mode tied to that site.
>     - **Per-instance trivia masquerading as a category** (the most common failure mode — read this carefully): a description of what ONE specific file does, with a citation tacked on. The citation is concrete, but the predicate names a single entity instead of a class of cases, so a reader looking at any other file cannot decide whether the rule applies.
>       *BAD:* *"BabyWeatherAnalyticsManager uses an internal AtomicBoolean singleton and its own initialize() — exposed through analyticsModule but not constructed via Koin injection."* — true, falsifiable, useless. It's a fact about one file, not a category. A new component reading this can't ask "do I match?" because the predicate is the entity itself.
>       *GOOD (same content, categorical shape):* *"Component manages its own initialization lifecycle (manual `initialize()` outside DI; Koin only exposes the already-built instance) — `BabyWeatherAnalyticsManager` (analyticsModule) and `LocalisationHelper` (`DomainModules.kt:18-21`) both follow this shape."* The predicate is now a class of cases ("manages its own init lifecycle"); the citations are evidence the class is real in the corpus. Any future component asking "is my init lifecycle DI-managed?" can answer.
>
>   **Fill applicable_when when there's a real boundary** — patterns where one type of consumer must use this and another type must NOT. Citing `APIService.kt:13 declares @Header("Authorization")` works because anonymous endpoints (which lack that header) MUST NOT route through `tokenCheck` — applying it would block public calls behind auth. That's the openmeter shape.
>
> - `do_not_apply_when`: array of concrete anti-indicators. Each entry follows the same **category-then-evidence** shape as `applicable_when`: lead with a categorical noun phrase describing the kind of caller/context where the pattern misapplies, then back it with a citation. Each MUST be falsifiable against schema/code. Include any places in the corpus where the same shape is used WITHOUT the invariant holding — those are the real-world `do_not_apply_when` anchors. Reject the same per-instance trivia shape: *"`FooManager` does X in `Foo.kt:42`"* without a class of cases is a fact, not a rule.
>
> - `scope`: array of component names from `components.components[].name` where this pattern is **relevant when editing** — including producers, consumers, and boundary participants, NOT just where the source file lives. Example: a registry pattern defined in component `core` but consumed by `feature-A` and `feature-B` has scope `["core", "feature-A", "feature-B"]`. Empty array means "applies repo-wide". **Conservative default:** if uncertain, leave `[]` — the rule loads everywhere (no regression vs. today). Only narrow scope when there is verifiable evidence the pattern is component-bound.
>
> For **generic** patterns (REST, WebSocket, Event Bus, generic logging) where no codebase-specific invariant applies, leave `applicable_when` as `""`, `do_not_apply_when` as `[]`, `scope` as `[]`. Do NOT invent preconditions.
>
> **Two illustrative examples** — each grounds `applicable_when` in a different invariant shape, to show the field is paradigm-agnostic. Pick whichever shape fits the codebase you are analyzing; do not force a schema annotation if there is no schema. Placeholder paths (`<schema>/<entity>.<ext>`, `<domain-A>`) are shown to keep these examples language-neutral — substitute concrete files and components for the actual codebase.
>
> *Example 1 (lifecycle / framework state — typical for UI codebases):*
> ```json
> {
>   "name": "Context-backed session hook",
>   "when_to_use": "Read the current user inside any component below the root provider",
>   "how_it_works": "useSession() reads from a context populated by SessionProvider; provider hydrates from cookie on mount",
>   "examples": ["src/auth/use-session.<ext>", "src/auth/provider.<ext>"],
>   "applicable_when": "src/app/layout.<ext>:14 wraps the entire tree in <SessionProvider> — the hook is therefore safe to call from any descendant client component",
>   "do_not_apply_when": ["Caller is a server component or middleware — context is unavailable; the server-side equivalent (e.g. getServerSession) is the correct API", "Test renders the component in isolation without SessionProvider — context is null and the hook throws"],
>   "scope": []
> }
> ```
>
> *Example 2 (schema-bound — typical for DB-backed services):*
> ```json
> {
>   "name": "Per-key advisory lock",
>   "when_to_use": "Serialize concurrent mutations of the same logical entity",
>   "how_it_works": "Acquire a database advisory lock keyed on (namespace, entityID) inside a transaction; lock auto-releases on commit/rollback",
>   "examples": ["<lib>/lock.<ext>", "<domain>/service/write.<ext>"],
>   "applicable_when": "<schema>/<entity>.<ext>:<line> declares a UNIQUE index on the lock-key columns — the (namespace, entityID) tuple maps to at most one row in the locked entity's table",
>   "do_not_apply_when": ["Schema index for the proposed key is NOT unique — multiple rows can legitimately share the key, so a single advisory lock would serialize unrelated rows", "Operation mutates one row only — built-in row-level locking (e.g. SELECT FOR UPDATE) suffices, no cross-row invariant to protect"],
>   "scope": ["<domain-A>", "<domain-B>"]
> }
> ```
>
> Both shapes are valid. The requirement isn't a specific invariant *type* — it's that the citation lets a reader verify the precondition by looking at the cited code.
>
> Return JSON:
> ```json
> {
>   "communication": {
>     "patterns": [
>       {"name": "", "when_to_use": "", "how_it_works": "", "examples": [], "applicable_when": "", "do_not_apply_when": [], "scope": []}
>     ],
>     "integrations": [
>       {"service": "", "purpose": "", "integration_point": ""}
>     ],
>     "pattern_selection_guide": [
>       {"scenario": "", "pattern": "", "rationale": ""}
>     ]
>   },
>   "quick_reference": {
>     "pattern_selection": [
>       {"scenario": "", "pattern": "", "scope": []}
>     ],
>     "error_mapping": [{"error": "", "status_code": 0, "description": ""}]
>   }
> }
> ```

