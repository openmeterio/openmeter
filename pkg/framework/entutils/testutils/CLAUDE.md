# testutils

<!-- archie:ai-start -->

> Structural folder holding two standalone Ent codegen fixtures (ent1, ent2) used exclusively to test pkg/framework/entutils transaction and mixin helpers (TransactingRepo, TxDriver, mixins, cursor/paginate) in isolation from the production openmeter/ent schema set. It owns the test-only generated Ent surface, not production code.

## Patterns

**Two-client transaction test fixtures** — Children ent1 and ent2 each generate an independent Ent db/ client (Example1, Example2 schemas) so entutils helpers can be exercised across two separate clients in the same test without touching the real schema. (`transaction_test.go imports both testutils/ent1/db and testutils/ent2/db`)
**Deliberate fixture divergence** — ent1 is the feature-complete fixture (entpaginate + entcursor + entexpose); ent2 is intentionally leaner (entexpose only). entcursor/cursor_test.go and entpaginate/paginate_test.go depend on ent1's fuller generated surface. (`entpaginate/paginate_test.go imports testutils/ent1/db only`)

## Anti-Patterns

- Adding production schema concerns to these fixtures — they exist only to test entutils helpers, not to mirror openmeter/ent.
- Hand-editing any generated db/ client under ent1 or ent2 instead of editing schema/ and regenerating.
- Collapsing ent1 and ent2 into one fixture — two independent clients are required to test cross-client transaction behavior.
- Forcing ent2 to match ent1's extension list — the leaner ent2 is intentional.

## Decisions

- **Keep test-only Ent clients here, fully separate from openmeter/ent.** — Lets entutils transaction/mixin helpers be unit-tested against a tiny stable schema without coupling to the large production schema or its migrations.
- **Maintain two fixtures of differing completeness (ent1 full, ent2 lean).** — ent1 covers paginate/cursor-dependent tests; the second client enables cross-client transaction tests while staying minimal.

<!-- archie:ai-end -->
