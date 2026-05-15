# docs

<!-- archie:ai-start -->

> Documentation-only root split into ADRs (docs/decisions/) recording immutable architectural choices and user-facing migration guides (docs/migration-guides/) for breaking changes between versions. Contains no Go source; changes here affect operator understanding and historical record, not compile-time behavior.

## Patterns

**ADR immutability** — Existing ADR files in docs/decisions/ record a past decision and must never be edited to change that decision. Create a new sequentially numbered ADR to supersede. (`// 0005-clickhouse.md documents the switch from ksqlDB
// A reversal requires creating 0006-revert-to-ksqldb.md — never edit 0005`)
**One migration guide per breaking change** — Each file in docs/migration-guides/ covers exactly one breaking change with SQL audit queries before any destructive statements, Before/After API examples, and a deprecation timeline table. (`// 2025-08-12-subject-customer-consolidation.md covers only the subject→customer consolidation
// SQL: SELECT count(*) FROM subjects WHERE ... (audit before DELETE)`)
**Audit queries before destructive SQL** — Migration guides that include SQL must show a SELECT audit query before any UPDATE/DELETE statement so operators can verify scope before executing. (`-- Audit first:
SELECT id, key FROM subjects WHERE customer_id IS NULL;
-- Then migrate:
UPDATE subjects SET customer_id = ... WHERE ...`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `docs/decisions/0001-event-streaming-platform.md` | ADR: choice of Kafka as the event streaming platform | Do not edit to reflect a different decision; create a new 000N-*.md ADR to supersede |
| `docs/decisions/0005-clickhouse.md` | ADR: ClickHouse replaces ksqlDB for analytics — authoritative source for why ClickHouse is the analytics store | Any ClickHouse architectural change requires a new superseding ADR, not an edit here |
| `docs/database-migration.md` | Operator guide covering autoMigrate modes (ent, migration, false) and the Atlas workflow | Keep in sync with actual atlas.hcl and Makefile targets when migration commands change |
| `docs/event-ingestion.md` | User-facing explanation of CloudEvents format, JSONPath meter configuration, and deduplication semantics | Any change to ingest API contract or deduplication window must be reflected here |
| `docs/stripe-dev.md` | Developer guide for working against a Stripe test account including webhook setup and psql commands | Keep webhook URL patterns in sync with actual API routes; update psql column names if Ent schema changes |

## Anti-Patterns

- Editing existing ADR files to change the recorded decision — always create a new sequentially numbered ADR
- Combining multiple unrelated breaking changes in a single migration guide file — one breaking change per file
- Writing a migration guide without audit SELECT queries before destructive UPDATE/DELETE statements
- Using docs/decisions for operational runbooks or migration SQL — those belong in docs/migration-guides
- Omitting the 'Considered Options' and 'Cons' sections when writing a new ADR — every decision must document its tradeoffs

## Decisions

- **ADRs in docs/decisions/ are append-only historical records** — Immutable records prevent revisionist history and provide an auditable trail of why the current architecture exists; developers can trace constraints back to their original context
- **SQL migration scripts embedded directly in docs/migration-guides/ rather than as separate .sql files** — Self-contained guides are easier for operators to follow without cross-referencing separate files; the inline context explains what each statement does and why

<!-- archie:ai-end -->
