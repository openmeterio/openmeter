# migration-guides

<!-- archie:ai-start -->

> User-facing migration guides for breaking changes between OpenMeter versions. Each file covers a specific breaking change: what changed, SQL/API migration steps, and a deprecation timeline. These documents are operator-facing, not developer patterns.

## Patterns

**Date-prefixed filename** — Files are named YYYY-MM-DD-<slug>.md where the date is the release date of the breaking change, not the writing date. (`2025-06-26-subscription-alignment.md, 2025-08-12-subject-customer-consolidation.md`)
**SQL migration scripts included inline** — Guides that require database changes embed the SQL directly in fenced code blocks (```sql) so operators can run them against their Postgres instance. Scripts are read-only SELECTs first, then UPDATE statements. (`2025-06-26-subscription-alignment.md provides SELECT COUNT(*) audit queries before the UPDATE plans SET billables_must_align = true statement.`)
**Before/After API examples for SDK migrations** — API-level changes show TypeScript before/after code blocks using the @openmeter/sdk client so SDK consumers can update their integration code directly. (`2025-08-12-subject-customer-consolidation.md shows openmeter.subjects.createEntitlement → openmeter.customers.createEntitlement.`)
**Deprecation timeline table** — Guides with phased deprecations include a Markdown table with Date and Change columns documenting when deprecated APIs are removed. (`2025-08-12-subject-customer-consolidation.md: September 01, 2025 (deprecated) → November 01, 2025 (removed).`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `2025-06-26-subscription-alignment.md` | Migration from unaligned to aligned subscriptions (billables_must_align flag). References openmeter/productcatalog/errors.go#L411 and alignment.go#L22 for the validation logic. | The SQL targets plans and subscriptions tables directly — the billables_must_align column must still exist in the schema for these queries to work. |
| `2025-08-12-subject-customer-consolidation.md` | Documents the consolidation of subject and customer entities — subjects are deprecated as primary billing entities in favor of customers. Explains automatic usage attribution by customer ID/Key. | The usageAttribution.subjects field is now optional; code that assumes it is required will need updating. Subject APIs targeted for removal November 2025. |
| `2025-11-04-entitlement-events-v1.md` | Removal notice for V1 entitlement events. V2 events replaced V1 after commit 85f7ec90. Operators must drain V1 events before upgrading. | If adding new entitlement event versions, follow the same pattern: introduce V(N+1), stop producing VN, drain VN, then remove VN in a later release. |

## Anti-Patterns

- Writing a migration guide without audit SELECT queries before destructive UPDATE/DELETE statements
- Referencing internal Go file paths without also explaining the user-visible behavior change
- Combining multiple unrelated breaking changes into a single guide file — one breaking change per file
- Using these files as ADRs for architectural decisions — those belong in docs/decisions

## Decisions

- **SQL scripts embedded directly in migration guides rather than as separate .sql files** — Keeps the migration context (what, why, SQL) together in one document operators read linearly; separate files risk being run in wrong order or without context.
- **Phased deprecation timeline with explicit removal dates** — API consumers need advance notice; a two-phase approach (deprecate then remove) gives them time to migrate without blocking the internal codebase cleanup.

<!-- archie:ai-end -->
