# topicresolver

<!-- archie:ai-start -->

> Maps a namespace to its Kafka topic name. Defines the Resolver interface and NamespacedTopicResolver, the indirection that lets the ingest/sink pipeline derive per-namespace topics from a single template.

## Patterns

**Resolver interface** — All topic resolution goes through Resolver.Resolve(ctx, namespace) (string, error). Inject the interface, not the concrete type, into ingest/sink wiring. (`type Resolver interface { Resolve(ctx context.Context, namespace string) (string, error) }`)
**Compile-time interface assertion** — Implementations declare a static assertion so the build fails if the interface drifts. (`var _ Resolver = (*NamespacedTopicResolver)(nil)`)
**Template-driven topic naming** — NamespacedTopicResolver formats namespace into a single-parameter template via fmt.Sprintf (e.g. "om_%s_events"). The constructor rejects an empty template. (`fmt.Sprintf(r.template, namespace)`)
**Validating constructor returning pointer + error** — NewNamespacedTopicResolver validates input and returns (*NamespacedTopicResolver, error); the resolver itself never errors on a configured template. (`func NewNamespacedTopicResolver(template string) (*NamespacedTopicResolver, error)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `resolver.go` | Declares the Resolver interface — the abstraction consumed by ingest publishers and the sink worker. | Resolve takes ctx but the namespaced impl ignores it; any DB/config-backed resolver must honor ctx for cancellation. |
| `namespacedtopic.go` | NamespacedTopicResolver + its constructor; the default template-based implementation. | template must contain exactly one %s for fmt.Sprintf; a template with no/multiple verbs produces malformed topic names. Empty template is rejected at construction, not at Resolve time. |

## Anti-Patterns

- Hard-coding topic names in publishers/consumers instead of resolving through a Resolver.
- Depending on the concrete *NamespacedTopicResolver in wiring rather than the Resolver interface.
- Ignoring ctx in a new resolver that performs I/O (e.g. config or DB lookups).

## Decisions

- **Topic resolution is an interface with a trivial template-based default.** — Keeps namespace→topic mapping swappable (template, lookup table, or dynamic) without touching ingest/sink call sites.

## Example: Build a namespaced topic resolver and resolve a topic

```
import "github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/topicresolver"

r, err := topicresolver.NewNamespacedTopicResolver("om_%s_events")
if err != nil { return err }
topic, err := r.Resolve(ctx, namespace) // "om_<namespace>_events"
```

<!-- archie:ai-end -->
