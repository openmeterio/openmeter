# resources

<!-- archie:ai-start -->

> Declares shared Benthos cache resources (dedupe_cache, memory backend, 1h TTL) referenced by name from stream processors in the quickstart collector pipeline. This folder contains declarations only — no processors or business logic.

## Patterns

**Named cache label convention** — Cache resources must use a label that exactly matches the string in dedupe processor `cache:` fields elsewhere in the pipeline. A label mismatch causes a runtime reference error when the stream starts. (`cache_resources:
  - label: dedupe_cache
    memory:
      default_ttl: 1h`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `dedupe-cache.yaml` | Defines the in-memory deduplication cache shared by input.yaml's dedupe processor. | Renaming `dedupe_cache` breaks the `cache: dedupe_cache` reference in streams/input.yaml. Changing TTL affects deduplication window and memory footprint. |

## Anti-Patterns

- Adding processors or business logic inside a resource file — resources are declarations only.
- Using a persistent cache backend (Redis, etc.) without updating collector deployment configuration.
- Renaming `dedupe_cache` without updating all `cache:` references in stream processors.

## Decisions

- **In-memory cache with 1h TTL for deduplication.** — Quickstart is a lightweight local demo; persistent cache would require additional infrastructure. 1h TTL matches expected event replay windows.

<!-- archie:ai-end -->
