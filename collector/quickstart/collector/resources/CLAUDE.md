# resources

<!-- archie:ai-start -->

> Declares shared Benthos cache resources for the quickstart collector pipeline. The single dedupe cache resource (label: dedupe_cache, memory backend, 1h TTL) is referenced by name from stream processors to deduplicate incoming CloudEvents.

## Patterns

**Named cache label convention** — Cache resources must be declared with a label that exactly matches the string referenced in dedupe processor `cache:` fields elsewhere in the pipeline. (`cache_resources:
  - label: dedupe_cache
    memory:
      default_ttl: 1h`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `dedupe-cache.yaml` | Defines the in-memory deduplication cache shared by the input stream's dedupe processor. | Changing the label breaks the dedupe processor reference in streams/input.yaml; changing TTL affects dedup window and memory usage. |

## Anti-Patterns

- Adding business logic or processors directly in a resource file — resources are declarations only.
- Using a persistent cache backend (Redis, etc.) here without updating the collector's deployment configuration.
- Renaming `dedupe_cache` without updating all `cache:` references in stream processors.

## Decisions

- **In-memory cache with 1h TTL for deduplication.** — Quickstart is a lightweight local demo; a persistent cache would require additional infrastructure. The 1h TTL matches expected event replay windows.

<!-- archie:ai-end -->
