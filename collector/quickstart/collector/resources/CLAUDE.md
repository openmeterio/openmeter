# resources

<!-- archie:ai-start -->

> Benthos/Redpanda Connect shared resource definitions for the quickstart collector — currently holds only the in-memory dedupe cache that the events ingestion stream references by label. Loaded by the Benthos CLI alongside the stream configs as part of the quickstart docker-compose stack.

## Patterns

**Resources declared, not inlined** — Reusable Benthos components live here as labeled top-level resources (cache_resources, rate_limit_resources, etc.) so streams can reference them by label rather than redefining them inline. (`cache_resources:
  - label: dedupe_cache
    memory:
      default_ttl: 1h`)
**Label is the cross-file contract** — The `label` value is the only handle a stream has to a resource. The label `dedupe_cache` here must exactly match the `cache:` reference in streams/input.yaml's dedupe processor. (`label: dedupe_cache  # referenced as cache: "dedupe_cache" in input.yaml`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `dedupe-cache.yaml` | Defines the `dedupe_cache` memory cache (1h TTL) used by the input stream's dedupe processor to drop duplicate CloudEvents by xxhash64 content hash. | Memory cache is per-process and non-durable — restarts lose dedupe state, and it does not work across horizontally-scaled collector replicas. Renaming the label silently breaks the input stream's dedupe step. TTL bounds the dedupe window. |

## Anti-Patterns

- Renaming the `dedupe_cache` label without updating the `cache:` reference in streams/input.yaml — the dedupe processor will fail to resolve the cache.
- Assuming the memory cache provides durable or cluster-wide dedupe — it is process-local and cleared on restart.

## Decisions

- **Dedupe cache kept as a shared resource rather than inlined in the stream.** — Benthos resources are referenced by label across stream files, keeping cache config in one place and reusable by multiple processors.

<!-- archie:ai-end -->
