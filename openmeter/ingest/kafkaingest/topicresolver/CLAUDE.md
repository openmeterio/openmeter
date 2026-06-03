# topicresolver

<!-- archie:ai-start -->

> Provides the Resolver interface and its NamespacedTopicResolver implementation, mapping a namespace string to a Kafka topic name via a printf-style template (e.g. 'om_%s_events'). Used by the Kafka ingest collector to pick the per-namespace target topic.

## Patterns

**Resolver interface for topic lookup** — All topic resolution goes through Resolver.Resolve(ctx, namespace) (string, error). Callers must never hard-code topic names. (`type Resolver interface { Resolve(ctx context.Context, namespace string) (string, error) }`)
**Template-based namespace-to-topic mapping** — NamespacedTopicResolver uses fmt.Sprintf(template, namespace). Template must contain exactly one %s; the constructor validates non-empty but not placeholder count. (`NewNamespacedTopicResolver("om_%s_events")`)
**Compile-time interface assertion** — var _ Resolver = (*NamespacedTopicResolver)(nil) ensures the concrete type satisfies the interface at compile time. (`var _ Resolver = (*NamespacedTopicResolver)(nil)`)
**Context threading** — Resolve accepts context.Context even though NamespacedTopicResolver ignores it (_); always pass the caller's ctx to support future async/DB-backed implementations. (`func (r NamespacedTopicResolver) Resolve(_ context.Context, namespace string) (string, error)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `resolver.go` | Pure interface definition only. New resolver implementations belong in separate files. | Do not add business logic to resolver.go — keep it a pure interface. |
| `namespacedtopic.go` | Single-template resolver; constructor rejects empty templates. The only current Resolver implementation. | Template must have exactly one %s; the constructor does not validate placeholder count, so zero or two %s causes a runtime fmt.Sprintf format error. |

## Anti-Patterns

- Hard-coding topic names in the collector instead of using Resolver.Resolve
- Passing a template with zero or multiple %s placeholders to NewNamespacedTopicResolver
- Implementing Resolver without accepting context.Context
- Calling Resolve with context.Background() in application code — propagate the caller's ctx

## Decisions

- **Topic name derived from a template rather than a static config string** — Enables per-namespace Kafka topic isolation (multi-tenant ingest) without config changes per new namespace.
- **Resolver is an interface, not a function type** — Allows future implementations (e.g. DB-backed dynamic routing) to be swapped in via DI without changing the collector.

<!-- archie:ai-end -->
